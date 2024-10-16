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
	"net/http"
	"net/url"
	"time"

	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"golang.org/x/net/http2"
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

	TLS TLSParams
}

func NewRefreshableTransport(ctx context.Context, p RefreshableTransportParams, tlsProvider TLSProvider, dialer ContextDialer) http.RoundTripper {
	return &RefreshableTransport{
		Refreshable: p.MapTransportParams(func(p TransportParams) interface{} {
			return newTransport(ctx, p, tlsProvider, dialer)
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

func (r *RefreshableTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return r.Current().(*http.Transport).RoundTrip(req)
}

func newTransport(ctx context.Context, p TransportParams, tlsProvider TLSProvider, dialer ContextDialer) *http.Transport {
	svc1log.FromContext(ctx).Debug("Reconstructing HTTP Transport")

	var transportProxy func(*http.Request) (*url.URL, error)
	if p.HTTPProxyURL != nil {
		transportProxy = func(*http.Request) (*url.URL, error) { return p.HTTPProxyURL, nil }
	} else if p.ProxyFromEnvironment {
		transportProxy = http.ProxyFromEnvironment
	}

	tlsConfig := tlsProvider.GetTLSConfig(ctx)
	transport := &http.Transport{
		Proxy:                 transportProxy,
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

	if !p.DisableHTTP2 {
		// Attempt to configure net/http HTTP/1 Transport to use HTTP/2.
		http2Transport, err := http2.ConfigureTransports(transport)
		if err != nil {
			// ConfigureTransport's only error as of this writing is the idempotent "protocol https already registered."
			// It should never happen in our usage because this is immediately after creation.
			// In case of something unexpected, log it and move on.
			svc1log.FromContext(ctx).Error("failed to configure transport for http2", svc1log.Stacktrace(err))
		} else {
			// ReadIdleTimeout is the amount of time to wait before running periodic health checks (pings)
			// after not receiving a frame from the HTTP/2 connection.
			// Setting this value will enable the health checks and allows broken idle
			// connections to be pruned more quickly, preventing the client from
			// attempting to re-use connections that will no longer work.
			// ref: https://github.com/golang/go/issues/36026
			http2Transport.ReadIdleTimeout = p.HTTP2ReadIdleTimeout

			// PingTimeout configures the amount of time to wait for a ping response (health check)
			// before closing an HTTP/2 connection. The PingTimeout is only valid if
			// the above ReadIdleTimeout is > 0.
			http2Transport.PingTimeout = p.HTTP2PingTimeout
		}
	}

	return transport
}
