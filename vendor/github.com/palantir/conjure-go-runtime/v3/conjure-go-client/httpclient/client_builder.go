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

	"github.com/palantir/conjure-go-runtime/v3/conjure-go-client/httpclient/internal"
	"github.com/palantir/conjure-go-runtime/v3/conjure-go-client/httpclient/internal/refreshingclient"
	"github.com/palantir/pkg/bytesbuffers"
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/refreshable/v2"
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

	URIs             refreshable.Refreshable[[]string]
	URIScorerBuilder func([]string) internal.URIScoringMiddleware

	// If false, NewClient() will return an error when URIs.Current() is empty.
	// This allows for a refreshable URI slice to be populated after construction but before use.
	AllowEmptyURIs bool

	ErrorDecoder ErrorDecoder

	BytesBufferPool bytesbuffers.Pool
	MaxAttempts     refreshable.Refreshable[*int]
	RetryParams     refreshable.Refreshable[refreshingclient.RetryParams]
}

type httpClientBuilder struct {
	ServiceName     refreshable.Refreshable[string]
	Timeout         refreshable.Refreshable[time.Duration]
	DialerParams    refreshable.Refreshable[refreshingclient.DialerParams]
	TLSConfig       *tls.Config // If unset, config in TransportParams will be used.
	TransportParams refreshable.Refreshable[refreshingclient.TransportParams]
	TLSCABytes      refreshable.Refreshable[[][]byte] // Optional refreshable CA bytes to combine with TLSParams.
	Middlewares     []Middleware

	DisableMetrics      refreshable.Refreshable[bool]
	MetricsTagProviders []TagsProvider

	// These middleware options are not refreshed anywhere because they are not in ClientConfig,
	// but they could be made refreshable if ever needed.
	DisableRequestSpan  bool
	DisableRecovery     bool
	DisableTraceHeaders bool
}

func (b *httpClientBuilder) Build(ctx context.Context, params ...HTTPClientParam) (refreshable.Refreshable[*http.Client], error) {
	for _, p := range params {
		if p == nil {
			continue
		}
		if err := p.applyHTTPClient(b); err != nil {
			return nil, err
		}
	}
	refreshableConfig, err := b.getRefreshableTLSConfig(ctx)
	if err != nil {
		return nil, err
	}

	// Create dialer and transport
	dialer := refreshingclient.NewRefreshableDialer(ctx, b.DialerParams)
	transport := refreshingclient.NewRefreshableTransport(ctx, b.TransportParams, refreshableConfig, dialer)
	transport = wrapTransport(transport, newMetricsMiddleware(b.ServiceName, b.MetricsTagProviders, b.DisableMetrics))
	transport = wrapTransport(transport, newTraceMiddleware(b.ServiceName, b.DisableRequestSpan, b.DisableTraceHeaders))
	if !b.DisableRecovery {
		transport = wrapTransport(transport, recoveryMiddleware{})
	}
	transport = wrapTransport(transport, b.Middlewares...)

	return refreshingclient.NewRefreshableHTTPClient(transport, b.Timeout), nil
}

func (b *httpClientBuilder) getRefreshableTLSConfig(ctx context.Context) (refreshable.Validated[*tls.Config], error) {
	if b.TLSConfig != nil {
		refreshableOfStaticTLSConfig := refreshable.New(b.TLSConfig)
		validatedStaticTLSConfig, _, err := refreshable.Validate(refreshableOfStaticTLSConfig, func(cfg *tls.Config) error {
			// No validation needed given validation is done when setting config
			return nil
		})
		if err != nil {
			return nil, err
		}
		return validatedStaticTLSConfig, nil
	}
	fileSlices, _ := refreshable.Map(b.TransportParams, func(t refreshingclient.TransportParams) map[string]struct{} {
		toReturn := map[string]struct{}{}
		for _, file := range t.TLSConfigurationParams.CAFiles {
			toReturn[file] = struct{}{}
		}
		return toReturn
	})
	multiFileRefreshable := refreshable.NewMultiFileRefreshable(ctx, fileSlices)
	if _, err := multiFileRefreshable.Validation(); err != nil {
		return nil, werror.WrapWithContextParams(ctx, err, "failed to read CA files")
	}
	tlsParams, _ := refreshable.Merge(b.TransportParams, multiFileRefreshable, func(t1 refreshingclient.TransportParams, t2 map[string][]byte) refreshingclient.TLSParams {
		var caBytes [][]byte
		for _, caSlice := range t2 {
			caBytes = append(caBytes, caSlice)
		}
		return refreshingclient.TLSParams{
			CABytes:            caBytes,
			CertFile:           t1.TLSConfigurationParams.CertFile,
			KeyFile:            t1.TLSConfigurationParams.KeyFile,
			InsecureSkipVerify: t1.TLSConfigurationParams.InsecureSkipVerify,
		}
	})
	if b.TLSCABytes != nil {
		tlsParams, _ = refreshable.Merge(tlsParams, b.TLSCABytes, func(tlsParams refreshingclient.TLSParams, caByteSlices [][]byte) refreshingclient.TLSParams {
			for _, caByteSlice := range caByteSlices {
				tlsParams.CABytes = append(tlsParams.CABytes, caByteSlice)
			}
			return tlsParams
		})
	}
	return refreshingclient.NewRefreshableTLSConfig(ctx, tlsParams)
}

// NewClient returns a configured client ready for use.
// We apply "sane defaults" before applying the provided params.
func NewClient(params ...ClientParam) (Client, error) {
	b := newClientBuilder()
	return newClient(context.TODO(), b, params...)
}

// NewClientFromRefreshableConfig returns a configured client ready for use.
// We apply "sane defaults" before applying the provided params.
func NewClientFromRefreshableConfig(ctx context.Context, config refreshable.Refreshable[ClientConfig], params ...ClientParam) (Client, error) {
	b := newClientBuilder()
	if err := newClientBuilderFromRefreshableConfig(ctx, config, b, nil); err != nil {
		return nil, err
	}
	return newClient(ctx, b, params...)
}

func newClient(ctx context.Context, b *clientBuilder, params ...ClientParam) (Client, error) {
	for _, p := range params {
		if p == nil {
			continue
		}
		if err := p.apply(b); err != nil {
			return nil, err
		}
	}
	if b.URIs == nil {
		return nil, werror.ErrorWithContextParams(ctx, "httpclient URLs must be set in configuration or by constructor param", werror.SafeParam("serviceName", b.HTTP.ServiceName.Current()))
	}
	if !b.AllowEmptyURIs && len(b.URIs.Current()) == 0 {
		return nil, werror.WrapWithContextParams(ctx, ErrEmptyURIs, "", werror.SafeParam("serviceName", b.HTTP.ServiceName.Current()))
	}

	var edm Middleware
	if b.ErrorDecoder != nil {
		edm = errorDecoderMiddleware{errorDecoder: b.ErrorDecoder}
	}

	middleware := b.HTTP.Middlewares
	b.HTTP.Middlewares = nil

	httpClient, err := b.HTTP.Build(ctx)
	if err != nil {
		return nil, err
	}

	var recovery Middleware
	if !b.HTTP.DisableRecovery {
		recovery = recoveryMiddleware{}
	}
	uriScorer := internal.NewRefreshableURIScoringMiddleware(b.URIs, func(uris []string) internal.URIScoringMiddleware {
		if b.URIScorerBuilder == nil {
			return internal.NewBalancedURIScoringMiddleware(uris, func() int64 { return time.Now().UnixNano() })
		}
		return b.URIScorerBuilder(uris)
	})
	return &clientImpl{
		serviceName:            b.HTTP.ServiceName,
		client:                 httpClient,
		uriScorer:              uriScorer,
		maxAttempts:            b.MaxAttempts,
		backoffOptions:         b.RetryParams,
		middlewares:            middleware,
		errorDecoderMiddleware: edm,
		recoveryMiddleware:     recovery,
		bufferPool:             b.BytesBufferPool,
	}, nil
}

// NewHTTPClient returns a configured http client ready for use.
// We apply "sane defaults" before applying the provided params.
func NewHTTPClient(params ...HTTPClientParam) (*http.Client, error) {
	b := newClientBuilder()
	provider, err := b.HTTP.Build(context.TODO(), params...)
	if err != nil {
		return nil, err
	}
	return provider.Current(), nil
}

// NewHTTPClientFromRefreshableConfig returns a configured http client ready for use.
// We apply "sane defaults" before applying the provided params.
func NewHTTPClientFromRefreshableConfig(ctx context.Context, config refreshable.Refreshable[ClientConfig], params ...HTTPClientParam) (refreshable.Refreshable[*http.Client], error) {
	b := newClientBuilder()
	if err := newClientBuilderFromRefreshableConfig(ctx, config, b, nil); err != nil {
		return nil, err
	}
	return b.HTTP.Build(ctx, params...)
}

func newClientBuilder() *clientBuilder {
	return &clientBuilder{
		HTTP: &httpClientBuilder{
			ServiceName: refreshable.New(""),
			Timeout:     refreshable.New(defaultHTTPTimeout),
			DialerParams: refreshable.New(refreshingclient.DialerParams{
				DialTimeout:   defaultDialTimeout,
				KeepAlive:     defaultKeepAlive,
				SocksProxyURL: nil,
			}),
			TransportParams: refreshable.New(refreshingclient.TransportParams{
				MaxIdleConns:          defaultMaxIdleConns,
				MaxIdleConnsPerHost:   defaultMaxIdleConnsPerHost,
				DisableHTTP2:          false,
				DisableKeepAlives:     false,
				IdleConnTimeout:       defaultIdleConnTimeout,
				ExpectContinueTimeout: defaultExpectContinueTimeout,
				ResponseHeaderTimeout: 0,
				TLSHandshakeTimeout:   defaultTLSHandshakeTimeout,
				HTTPProxyURL:          nil,
				ProxyFromEnvironment:  true,
				HTTP2ReadIdleTimeout:  defaultHTTP2ReadIdleTimeout,
				HTTP2PingTimeout:      defaultHTTP2PingTimeout,
			}),
			Middlewares:         nil,
			DisableMetrics:      refreshable.New(false),
			MetricsTagProviders: nil,
			DisableRecovery:     false,
			DisableRequestSpan:  false,
			DisableTraceHeaders: false,
		},
		URIs:            nil,
		BytesBufferPool: nil,
		ErrorDecoder:    restErrorDecoder{},
		MaxAttempts:     nil,
		RetryParams: refreshable.New(refreshingclient.RetryParams{
			InitialBackoff: defaultInitialBackoff,
			MaxBackoff:     defaultMaxBackoff,
		}),
	}
}

func newClientBuilderFromRefreshableConfig(ctx context.Context, config refreshable.Refreshable[ClientConfig], b *clientBuilder, reloadErrorSubmitter func(error)) error {
	validParams, _, err := refreshable.MapWithError(config, func(c ClientConfig) (refreshingclient.ValidatedClientParams, error) {
		p, err := newValidatedClientParamsFromConfig(ctx, c)
		if reloadErrorSubmitter != nil {
			reloadErrorSubmitter(err)
		}
		return p, err
	})
	if err != nil {
		return err
	}

	// Extract individual fields from ValidatedClientParams using Map.
	// We discard the unsubscribe callbacks since these subscriptions persist for the HTTP client's lifetime.
	b.HTTP.ServiceName, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) string {
		return p.ServiceName
	})
	b.HTTP.DialerParams, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) refreshingclient.DialerParams {
		return p.Dialer
	})
	b.HTTP.TransportParams, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) refreshingclient.TransportParams {
		return p.Transport
	})
	b.HTTP.Timeout, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) time.Duration {
		return p.Timeout
	})
	b.HTTP.DisableMetrics, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) bool {
		return p.DisableMetrics
	})
	b.HTTP.MetricsTagProviders = append(b.HTTP.MetricsTagProviders,
		TagsProviderFunc(func(*http.Request, *http.Response, error) metrics.Tags {
			return validParams.Current().MetricsTags
		}))

	apiToken, _ := refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) *string {
		return p.APIToken
	})
	basicAuth, _ := refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) *BasicAuth {
		if p.BasicAuth == nil {
			return nil
		}
		return &BasicAuth{User: p.BasicAuth.User, Password: p.BasicAuth.Password}
	})
	b.HTTP.Middlewares = append(b.HTTP.Middlewares,
		newAuthTokenMiddlewareFromRefreshable(apiToken),
		newBasicAuthMiddlewareFromRefreshable(basicAuth))

	b.URIs, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) []string {
		return p.URIs
	})
	b.MaxAttempts, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) *int {
		return p.MaxAttempts
	})
	b.RetryParams, _ = refreshable.Map(validParams, func(p refreshingclient.ValidatedClientParams) refreshingclient.RetryParams {
		return p.Retry
	})
	return nil
}
