// Copyright (c) 2018 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package httpclient provides round trippers/transport wrappers for http clients.
package httpclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient/internal"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient/internal/refreshingclient"
	"github.com/palantir/pkg/bytesbuffers"
	"github.com/palantir/pkg/refreshable"
	werror "github.com/palantir/witchcraft-go-error"
)

const (
	defaultDialTimeout           = 10 * time.Second
	defaultHTTPTimeout           = 60 * time.Second
	defaultKeepAlive             = 30 * time.Second
	defaultIdleConnTimeout       = 90 * time.Second
	defaultTLSHandshakeTimeout   = 10 * time.Second
	defaultExpectContinueTimeout = 1 * time.Second
	defaultMaxIdleConns          = 200
	defaultMaxIdleConnsPerHost   = 100
	defaultHTTP2ReadIdleTimeout  = 30 * time.Second
	defaultHTTP2PingTimeout      = 15 * time.Second
	defaultInitialBackoff        = 250 * time.Millisecond
	defaultMaxBackoff            = 2 * time.Second
)

var (
	// ErrEmptyURIs is returned when the client expects to have base URIs configured to make requests, but the URIs are empty.
	// This check occurs in two places: when the client is constructed and when a request is executed.
	// To avoid the construction validation, use WithAllowCreateWithEmptyURIs().
	ErrEmptyURIs = fmt.Errorf("httpclient URLs must not be empty")
)

type clientBuilder struct {
	HTTP *httpClientBuilder

	URIScorerBuilder func([]string) internal.URIScoringMiddleware

	// If false, NewClient() will return an error when URIs.Current() is empty.
	// This allows for a refreshable URI slice to be populated after construction but before use.
	AllowEmptyURIs bool

	ErrorDecoder ErrorDecoder

	BytesBufferPool bytesbuffers.Pool
}

type httpClientBuilder struct {
	// A set of functions that modify the ClientConfig before it is used to build the client.
	// Populated by client params and applied using Map on the underlying refreshable config.
	ConfigOverride []func(c *ClientConfig)

	TLSConfig           *tls.Config // Optionally set by WithTLSConfig(). If unset, config.Security is used.
	Middlewares         []Middleware
	MetricsTagProviders []TagsProvider

	// These middleware options are not refreshed anywhere because they are not in ClientConfig,
	// but they could be made refreshable if ever needed.
	DisableRequestSpan  bool
	DisableRecovery     bool
	DisableTraceHeaders bool
}

func (b *httpClientBuilder) Build(ctx context.Context, config RefreshableClientConfig, reloadErrorSubmitter func(error), params ...HTTPClientParam) (RefreshableHTTPClient, refreshingclient.RefreshableValidatedClientParams, error) {
	for _, p := range params {
		if p == nil {
			continue
		}
		if err := p.applyHTTPClient(b); err != nil {
			return nil, nil, err
		}
	}
	finalConfig := newRefreshingClientConfigWithConfigOverrides(config, b.ConfigOverride...)

	validParams, err := newRefreshingValidatedClientParams(ctx, finalConfig, reloadErrorSubmitter)
	if err != nil {
		return nil, nil, err
	}

	// Build the transport with a refreshable dialer and tls config.
	// Each component attempts to rebuild as infrequently as possible.
	serviceName := validParams.ServiceName()
	transportParams := validParams.Transport()
	metricsTagProviders := append(b.MetricsTagProviders, refreshableTagsProvider{validParams.MetricsTags()})

	var tlsProvider refreshingclient.TLSProvider
	if b.TLSConfig != nil {
		tlsProvider = refreshingclient.NewStaticTLSConfigProvider(b.TLSConfig)
	} else {
		refreshableProvider, err := refreshingclient.NewRefreshableTLSConfig(ctx, transportParams.TLS())
		if err != nil {
			return nil, nil, err
		}
		tlsProvider = refreshableProvider
	}

	dialer := refreshingclient.NewRefreshableDialer(ctx, validParams.Dialer())
	transport := refreshingclient.NewRefreshableTransport(ctx, transportParams, tlsProvider, dialer)
	transport = wrapTransport(transport, newMetricsMiddleware(serviceName, metricsTagProviders, validParams.DisableMetrics()))
	transport = wrapTransport(transport, newTraceMiddleware(serviceName, b.DisableRequestSpan, b.DisableTraceHeaders))
	if !b.DisableRecovery {
		transport = wrapTransport(transport, recoveryMiddleware{})
	}
	transport = wrapTransport(transport, b.Middlewares...)

	client := refreshingclient.NewRefreshableHTTPClient(transport, validParams.Timeout())
	return client, validParams, nil
}

// NewClient returns a configured client ready for use.
// We apply "sane defaults" before applying the provided params.
func NewClient(params ...ClientParam) (Client, error) {
	return NewClientFromRefreshableConfig(context.TODO(), nil, params...)
}

// NewClientFromRefreshableConfig returns a configured client ready for use.
// We apply "sane defaults" before applying the provided params.
func NewClientFromRefreshableConfig(ctx context.Context, config RefreshableClientConfig, params ...ClientParam) (Client, error) {
	b := &clientBuilder{
		HTTP:         &httpClientBuilder{},
		ErrorDecoder: restErrorDecoder{},
	}
	return newClient(ctx, config, b, nil, params...)
}

func newClient(ctx context.Context, config RefreshableClientConfig, b *clientBuilder, reloadErrorSubmitter func(error), params ...ClientParam) (Client, error) {
	for _, p := range params {
		if p == nil {
			continue
		}
		if err := p.apply(b); err != nil {
			return nil, err
		}
	}

	var edm Middleware
	if b.ErrorDecoder != nil {
		edm = errorDecoderMiddleware{errorDecoder: b.ErrorDecoder}
	}

	middleware := b.HTTP.Middlewares
	b.HTTP.Middlewares = nil

	httpClient, validParams, err := b.HTTP.Build(ctx, config, reloadErrorSubmitter)
	if err != nil {
		return nil, err
	}

	if !b.AllowEmptyURIs {
		// Validate that the URIs are not empty.
		if curr := validParams.CurrentValidatedClientParams(); len(curr.URIs) == 0 {
			return nil, werror.ErrorWithContextParams(ctx, "httpclient URIs must be set in configuration or by constructor param",
				werror.SafeParam("serviceName", curr.ServiceName))
		}
	}

	var recovery Middleware
	if !b.HTTP.DisableRecovery {
		recovery = recoveryMiddleware{}
	}
	uriScorer := internal.NewRefreshableURIScoringMiddleware(validParams.URIs(), func(uris []string) internal.URIScoringMiddleware {
		if b.URIScorerBuilder == nil {
			return internal.NewBalancedURIScoringMiddleware(uris, func() int64 { return time.Now().UnixNano() })
		}
		return b.URIScorerBuilder(uris)
	})

	middleware = append(middleware,
		newAuthTokenMiddlewareFromRefreshable(validParams.APIToken()),
		newBasicAuthMiddlewareFromRefreshable(validParams.BasicAuth()))

	return &clientImpl{
		serviceName:            validParams.ServiceName(),
		client:                 httpClient,
		uriScorer:              uriScorer,
		maxAttempts:            validParams.MaxAttempts(),
		backoffOptions:         validParams.Retry(),
		middlewares:            middleware,
		errorDecoderMiddleware: edm,
		recoveryMiddleware:     recovery,
		bufferPool:             b.BytesBufferPool,
	}, nil
}

// NewHTTPClient returns a configured http client ready for use.
// We apply "sane defaults" before applying the provided params.
func NewHTTPClient(params ...HTTPClientParam) (*http.Client, error) {
	provider, err := NewHTTPClientFromRefreshableConfig(context.TODO(), nil, params...)
	if err != nil {
		return nil, err
	}
	return provider.CurrentHTTPClient(), nil
}

// RefreshableHTTPClient exposes the internal interface
type RefreshableHTTPClient = refreshingclient.RefreshableHTTPClient

// NewHTTPClientFromRefreshableConfig returns a configured http client ready for use.
// We apply "sane defaults" before applying the provided params.
//
// The RefreshableClientConfig is not accepted as a client param because there must be exactly one
// subscription used to build the ValidatedClientParams in Build().
func NewHTTPClientFromRefreshableConfig(ctx context.Context, config RefreshableClientConfig, params ...HTTPClientParam) (RefreshableHTTPClient, error) {
	client, _, err := new(httpClientBuilder).Build(ctx, config, nil, params...)
	return client, err
}

// Map the final config to a set of validated client params used to build the dialer, retrier, tls config, and transport.
func newRefreshingValidatedClientParams(ctx context.Context, config RefreshableClientConfig, reloadErrorSubmitter func(error)) (refreshingclient.RefreshableValidatedClientParams, error) {
	refreshingParams, err := refreshable.NewMapValidatingRefreshable(config, func(i interface{}) (interface{}, error) {
		p, err := newValidatedClientParamsFromConfig(ctx, i.(ClientConfig))
		if reloadErrorSubmitter != nil {
			reloadErrorSubmitter(err)
		}
		return p, err
	})
	if err != nil {
		return nil, err
	}
	validParams := refreshingclient.NewRefreshingValidatedClientParams(refreshingParams)
	return validParams, nil
}

func newRefreshingClientConfigWithConfigOverrides(config RefreshableClientConfig, configOverride ...func(c *ClientConfig)) RefreshableClientConfig {
	// Build the final config refreshable by applying the overrides.
	if config == nil {
		config = NewRefreshingClientConfig(refreshable.NewDefaultRefreshable(ClientConfig{}))
	}
	return NewRefreshingClientConfig(config.MapClientConfig(func(c ClientConfig) interface{} {
		for _, o := range configOverride {
			o(&c)
		}
		return c
	}))
}
