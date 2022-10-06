// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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

package refreshingclient

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

type TransportParams struct {
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	DisableHTTP2          bool
	DisableKeepAlives     bool
	IdleConnTimeout       time.Duration
	ExpectContinueTimeout time.Duration
	ResponseHeaderTimeout time.Duration
	TLSHandshakeTimeout   time.Duration
	HTTPProxyURL          *url.URL `refreshables:",exclude"`
	ProxyFromEnvironment  bool
	HTTP2ReadIdleTimeout  time.Duration
	HTTP2PingTimeout      time.Duration
}

func NewRefreshableTransport(ctx context.Context, p RefreshableTransportParams, tlsConfig *tls.Config, dialer ContextDialer) http.RoundTripper {
	return &RefreshableTransport{
		Refreshable: p.MapTransportParams(func(p TransportParams) interface{} {
			return newTransport(ctx, p, tlsConfig, dialer)
		}),
	}
}

// ConfigureTransport accepts a mapping function which will be applied to the params value as it is evaluated.
// This can be used to layer/overwrite configuration before building the RefreshableTransportParams.
func ConfigureTransport(r RefreshableTransportParams, mapFn func(p TransportParams) TransportParams) RefreshableTransportParams {
	return NewRefreshingTransportParams(r.MapTransportParams(func(params TransportParams) interface{} {
		return mapFn(params)
	}))
}

// RefreshableTransport implements http.RoundTripper backed by a refreshable *http.Transport.
// The transport and internal dialer are each rebuilt when any of their respective parameters are updated.
type RefreshableTransport struct {
	refreshable.Refreshable // contains *http.Transport
}

func (r RefreshableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.Current().(*http.Transport).RoundTrip(req)
}

func newTransport(ctx context.Context, p TransportParams, tlsConfig *tls.Config, dialer ContextDialer) *http.Transport {
	svc1log.FromContext(ctx).Debug("Reconstructing HTTP Transport")
	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          p.MaxIdleConns,
		MaxIdleConnsPerHost:   p.MaxIdleConnsPerHost,
		TLSClientConfig:       tlsConfig,
		DisableKeepAlives:     p.DisableKeepAlives,
		ExpectContinueTimeout: p.ExpectContinueTimeout,
		IdleConnTimeout:       p.IdleConnTimeout,
		TLSHandshakeTimeout:   p.TLSHandshakeTimeout,
		ResponseHeaderTimeout: p.ResponseHeaderTimeout,
	}

	if p.HTTPProxyURL != nil {
		transport.Proxy = func(*http.Request) (*url.URL, error) { return p.HTTPProxyURL, nil }
	} else if p.ProxyFromEnvironment {
		transport.Proxy = http.ProxyFromEnvironment
	}

	if !p.DisableHTTP2 {
		if err := configureHTTP2(transport, p.HTTP2ReadIdleTimeout, p.HTTP2PingTimeout); err != nil {
			// ConfigureTransport's only error as of this writing is the idempotent "protocol https already registered."
			// It should never happen in our usage because this is immediately after creation.
			// In case of something unexpected, log it and move on.
			svc1log.FromContext(ctx).Error("failed to configure transport for http2", svc1log.Stacktrace(err))
		}
	}
	return transport
}
