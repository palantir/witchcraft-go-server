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

package httpclient

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient/internal"
	"github.com/palantir/pkg/bytesbuffers"
	"github.com/palantir/pkg/refreshable"
	werror "github.com/palantir/witchcraft-go-error"
)

// ClientParam is an interface that seals optional parameters used by NewClient and its variants.
type ClientParam interface {
	apply(builder *clientBuilder) error
}

// HTTPClientParam is an interface that seals optional parameters used by NewHTTPClient and its variants.
// These params (generally) operate on the http.Transport and do not modify the http.Request itself.
type HTTPClientParam interface {
	applyHTTPClient(builder *httpClientBuilder) error
}

// ClientOrHTTPClientParam is a param that can be used to build a Client or an http.Client
type ClientOrHTTPClientParam interface {
	ClientParam
	HTTPClientParam
}

// clientParamFunc is a convenience type that helps build a ClientParam. Use when you want a param that can be used to
// build a Client and *not* an http.Client
type clientParamFunc func(builder *clientBuilder) error

func (f clientParamFunc) apply(b *clientBuilder) error {
	return f(b)
}

// httpClientParamFunc is a convenience type that helps build a HTTPClientParam. Use when you want a param that can be used to
// build an http.Client and *not* a Client
type httpClientParamFunc func(builder *httpClientBuilder) error

func (f httpClientParamFunc) applyHTTPClient(b *httpClientBuilder) error {
	return f(b)
}

// clientOrHTTPClientParamFunc is a convenience type that helps build a ClientOrHTTPClientParam. Use when you want a param that can be used to
// either as an Client or a http.Client
type clientOrHTTPClientParamFunc func(builder *httpClientBuilder) error

func (f clientOrHTTPClientParamFunc) apply(b *clientBuilder) error {
	return f(b.HTTP)
}

func (f clientOrHTTPClientParamFunc) applyHTTPClient(b *httpClientBuilder) error {
	return f(b)
}

// configOverrideClientParamFunc constructs a ClientOrHTTPClientParam that modifies a ClientConfig pointer.
// If provided to NewClient or NewHTTPClient, these parameters will be applied to the ClientConfig before the client is built
// and will override any values set in the refreshable configuration.
func configOverrideClientParamFunc(override func(c *ClientConfig)) clientOrHTTPClientParamFunc {
	return func(b *httpClientBuilder) error {
		b.ConfigOverride = append(b.ConfigOverride, override)
		return nil
	}
}

// WithConfig merges all the values from the provided config into the final config used by the client.
// These masked values take precedence over those from a refreshable configuration used as the basis
// for NewClient or NewHTTPClient.
func WithConfig(in ClientConfig) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		*c = MergeClientConfig(in, *c)
	})
}

// WithConfigForHTTPClient merges all the values from the provided config into the final config used by the http.Client.
//
// Deprecated: Use WithConfig instead.
func WithConfigForHTTPClient(in ClientConfig) HTTPClientParam {
	return WithConfig(in)
}

// WithServiceName sets the service name for the client, used in telemetry.
func WithServiceName(serviceName string) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.ServiceName = serviceName
	})
}

// WithMiddleware will be invoked for custom HTTP behavior after the
// underlying transport is initialized. Each handler added "wraps" the previous
// round trip, so it will see the request first and the response last.
func WithMiddleware(h Middleware) ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		b.Middlewares = append(b.Middlewares, h)
		return nil
	})
}

// WithAddHeader adds the key-value pair to the request headers.
// If the key is already set, the value is appended.
func WithAddHeader(key, value string) ClientOrHTTPClientParam {
	return WithMiddleware(MiddlewareFunc(func(req *http.Request, next http.RoundTripper) (*http.Response, error) {
		req.Header.Add(key, value)
		return next.RoundTrip(req)
	}))
}

// WithSetHeader sets the key-value pair to the request headers.
// If the key is already set, the value is overwritten.
func WithSetHeader(key, value string) ClientOrHTTPClientParam {
	return WithMiddleware(MiddlewareFunc(func(req *http.Request, next http.RoundTripper) (*http.Response, error) {
		req.Header.Set(key, value)
		return next.RoundTrip(req)
	}))
}

// WithAuthToken sets the Authorization header to a static bearerToken.
func WithAuthToken(bearerToken string) ClientOrHTTPClientParam {
	return WithAuthTokenProvider(func(context.Context) (string, error) {
		return bearerToken, nil
	})
}

// WithAuthTokenProvider calls provideToken() and sets the Authorization header.
func WithAuthTokenProvider(provideToken TokenProvider) ClientOrHTTPClientParam {
	return WithMiddleware(&authTokenMiddleware{provideToken: provideToken})
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(userAgent string) ClientOrHTTPClientParam {
	return WithSetHeader("User-Agent", userAgent)
}

// WithOverrideRequestHost overrides the request Host from the default URL.Host
func WithOverrideRequestHost(host string) ClientOrHTTPClientParam {
	return WithMiddleware(MiddlewareFunc(func(req *http.Request, next http.RoundTripper) (*http.Response, error) {
		req.Host = host
		return next.RoundTrip(req)
	}))
}

// WithMetrics enables the "client.response" metric. See MetricsMiddleware for details.
// The serviceName will appear as the "service-name" tag.
func WithMetrics(tagProviders ...TagsProvider) ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		b.ConfigOverride = append(b.ConfigOverride, func(c *ClientConfig) {
			c.Metrics.Enabled = newPtr(true)
		})
		b.MetricsTagProviders = append(b.MetricsTagProviders, tagProviders...)
		return nil
	})
}

// WithBytesBufferPool stores a bytes buffer pool on the client for use in encoding request bodies.
// This prevents allocating a new byte buffer for every request.
func WithBytesBufferPool(pool bytesbuffers.Pool) ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.BytesBufferPool = pool
		return nil
	})
}

// WithDisablePanicRecovery disables the enabled-by-default panic recovery middleware.
// If the request was otherwise succeeding (err == nil), we return a new werror with
// the recovered object as an unsafe param. If there's an error, we werror.Wrap it.
// If errMiddleware is not nil, it is invoked on the recovered object.
func WithDisablePanicRecovery() ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		b.DisableRecovery = true
		return nil
	})
}

// WithDisableTracing disables the enabled-by-default tracing middleware which
// instructs the client to propagate trace information using the go-zipkin libraries
// method of attaching traces to requests. The server at the other end of such a request should
// be instrumented to read zipkin-style headers
//
// If a trace is already attached to a request context, then the trace is continued. Otherwise, no
// trace information is propagate. This will not create a span if one does not exist.
func WithDisableTracing() ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		b.DisableRequestSpan = true
		return nil
	})
}

// WithDisableTraceHeaderPropagation disables the enabled-by-default traceId header propagation
// By default, if witchcraft-logging has attached a traceId to the context of the request (for service and request logging),
// then the client will attach this traceId as a header for future services to do the same if desired
func WithDisableTraceHeaderPropagation() ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		b.DisableTraceHeaders = true
		return nil
	})
}

// WithHTTPTimeout sets the timeout on the http client.
// If unset, the client defaults to 1 minute.
func WithHTTPTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.ReadTimeout = &timeout
		c.WriteTimeout = &timeout
	})
}

// WithDisableHTTP2 skips the default behavior of configuring
// the transport with http2.ConfigureTransport.
func WithDisableHTTP2() ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.DisableHTTP2 = newPtr(true)
	})
}

// WithHTTP2ReadIdleTimeout configures the HTTP/2 ReadIdleTimeout.
// A ReadIdleTimeout > 0 will enable health checks and allows broken/idle
// connections to be pruned more quickly, preventing the client from
// attempting to re-use connections that will no longer work.
// If the HTTP/2 connection has not received any frames after the ReadIdleTimeout period,
// then periodic pings (health checks) will be sent to the server before attempting to close the connection.
// The amount of time to wait for the ping response can be configured by the WithHTTP2PingTimeout param.
// If unset, the client defaults to 30 seconds, if HTTP2 is enabled.
func WithHTTP2ReadIdleTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.HTTP2ReadIdleTimeout = &timeout
	})
}

// WithHTTP2PingTimeout configures the amount of time to wait for a ping response
// before closing an HTTP/2 connection. The PingTimeout is only valid when
// the ReadIdleTimeout is > 0 otherwise pings (health checks) are not enabled.
// If unset, the client defaults to 15 seconds, if HTTP/2 is enabled and the ReadIdleTimeout is > 0.
func WithHTTP2PingTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.HTTP2PingTimeout = &timeout
	})
}

// WithMaxIdleConns sets the number of reusable TCP connections the client
// will maintain. If unset, the client defaults to 200.
func WithMaxIdleConns(conns int) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.MaxIdleConns = &conns
	})
}

// WithMaxIdleConnsPerHost sets the number of reusable TCP connections the client
// will maintain per destination. If unset, the client defaults to 100.
func WithMaxIdleConnsPerHost(conns int) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.MaxIdleConnsPerHost = &conns
	})
}

// WithNoProxy nils out the Proxy field of the http.Transport,
// ignoring any proxy set in the process's environment.
// If unset, the default is http.ProxyFromEnvironment.
func WithNoProxy() ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.ProxyURL = nil
		c.ProxyFromEnvironment = newPtr(false)
	})
}

// WithProxyFromEnvironment can be used to set the HTTP(s) proxy to use
// the Go standard library's http.ProxyFromEnvironment.
func WithProxyFromEnvironment() ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.ProxyFromEnvironment = newPtr(true)
	})
}

// WithProxyURL can be used to set a socks5 or HTTP(s) proxy.
func WithProxyURL(proxyURLString string) ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		proxyURL, err := url.Parse(proxyURLString)
		if err != nil {
			return werror.Wrap(err, "failed to parse proxy url")
		}
		switch proxyURL.Scheme {
		case "http", "https":
		case "socks5", "socks5h":
		default:
			return werror.Error("unrecognized proxy scheme", werror.SafeParam("scheme", proxyURL.Scheme))
		}
		b.ConfigOverride = append(b.ConfigOverride, func(c *ClientConfig) {
			c.ProxyURL = &proxyURLString
		})
		return nil
	})
}

// WithTLSConfig sets the SSL/TLS configuration for the HTTP client's Transport using a copy of the provided config.
// The palantir/pkg/tlsconfig package is recommended to build a tls.Config from sane defaults.
func WithTLSConfig(conf *tls.Config) ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		if conf == nil {
			b.TLSConfig = nil
		} else {
			b.TLSConfig = conf.Clone()
		}
		return nil
	})
}

// WithTLSInsecureSkipVerify sets the InsecureSkipVerify field for the HTTP client's tls config.
// This option should only be used in clients that have way to establish trust with servers.
// If WithTLSConfig is used, the config's InsecureSkipVerify is set to true.
func WithTLSInsecureSkipVerify() ClientOrHTTPClientParam {
	return clientOrHTTPClientParamFunc(func(b *httpClientBuilder) error {
		if b.TLSConfig != nil {
			b.TLSConfig.InsecureSkipVerify = true
		}
		b.ConfigOverride = append(b.ConfigOverride, func(c *ClientConfig) {
			c.Security.InsecureSkipVerify = newPtr(true)
		})
		return nil
	})
}

// WithDialTimeout sets the timeout on the Dialer.
// If unset, the client defaults to 90 seconds.
func WithDialTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.ConnectTimeout = &timeout
	})
}

// WithIdleConnTimeout sets the timeout for idle connections.
// If unset, the client defaults to 90 seconds.
func WithIdleConnTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.IdleConnTimeout = &timeout
	})
}

// WithTLSHandshakeTimeout sets the timeout for TLS handshakes.
// If unset, the client defaults to 10 seconds.
func WithTLSHandshakeTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.TLSHandshakeTimeout = &timeout
	})
}

// WithExpectContinueTimeout sets the timeout to receive the server's first response headers after
// fully writing the request headers if the request has an "Expect: 100-continue" header.
// If unset, the client defaults to 1 second.
func WithExpectContinueTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.ExpectContinueTimeout = &timeout
	})
}

// WithResponseHeaderTimeout specifies the amount of time to wait for a server's response headers after fully writing
// the request (including its body, if any). This time does not include the time to read the response body. If unset,
// the client defaults to having no response header timeout.
func WithResponseHeaderTimeout(timeout time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.ResponseHeaderTimeout = &timeout
	})
}

// WithKeepAlive sets the keep alive frequency on the Dialer.
// If unset, the client defaults to 30 seconds.
func WithKeepAlive(keepAlive time.Duration) ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.KeepAlive = &keepAlive
	})
}

// WithBaseURLs sets the base URLs for every request. This is meant to be used in conjunction with WithPath.
func WithBaseURLs(urls []string) ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.HTTP.ConfigOverride = append(b.HTTP.ConfigOverride, func(c *ClientConfig) {
			c.URIs = urls
		})
		return nil
	})
}

// WithRefreshableBaseURLs sets the base URLs for every request. This is meant to be used in conjunction with WithPath.
func WithRefreshableBaseURLs(urls refreshable.StringSlice) ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.HTTP.ConfigOverride = append(b.HTTP.ConfigOverride, func(c *ClientConfig) {
			c.URIs = urls.CurrentStringSlice()
		})
		return nil
	})
}

// WithAllowCreateWithEmptyURIs prevents NewClient from returning an error when the URI slice is empty.
// This is useful when the URIs are not known at client creation time but will be populated by a refreshable.
// Requests will error if attempted before URIs are populated.
func WithAllowCreateWithEmptyURIs() ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.AllowEmptyURIs = true
		return nil
	})
}

// WithMaxBackoff sets the maximum backoff between retried calls to the same URI.
// Defaults to 2 seconds. <= 0 indicates no limit.
func WithMaxBackoff(maxBackoff time.Duration) ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.HTTP.ConfigOverride = append(b.HTTP.ConfigOverride, func(c *ClientConfig) {
			c.MaxBackoff = &maxBackoff
		})
		return nil
	})
}

// WithInitialBackoff sets the initial backoff between retried calls to the same URI. Defaults to 250ms.
func WithInitialBackoff(initialBackoff time.Duration) ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.HTTP.ConfigOverride = append(b.HTTP.ConfigOverride, func(c *ClientConfig) {
			c.InitialBackoff = &initialBackoff
		})
		return nil
	})
}

// WithMaxRetries sets the maximum number of retries on transport errors for every request. Backoffs are
// also capped at this.
// If unset, the client defaults to 2 * size of URIs
// TODO (#151): Rename to WithMaxAttempts and set maxAttempts directly using the argument provided to the function.
func WithMaxRetries(maxTransportRetries int) ClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.MaxNumRetries = &maxTransportRetries
	})
}

// WithUnlimitedRetries sets an unlimited number of retries on transport errors for every request.
// If set, this supersedes any retry limits set with WithMaxRetries.
func WithUnlimitedRetries() ClientParam {
	// hack: will have 1 added to create MaxAttempts of 0, which means unlimited retries.
	return WithMaxRetries(-1)
}

// WithDisableRestErrors disables the middleware which sets Do()'s returned
// error to a non-nil value in the case of >= 400 HTTP response.
func WithDisableRestErrors() ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.ErrorDecoder = nil
		return nil
	})
}

// WithDisableKeepAlives disables keep alives on the http transport
func WithDisableKeepAlives() ClientOrHTTPClientParam {
	return configOverrideClientParamFunc(func(c *ClientConfig) {
		c.KeepAlive = newPtr(time.Duration(0))
	})
}

func WithErrorDecoder(errorDecoder ErrorDecoder) ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.ErrorDecoder = errorDecoder
		return nil
	})
}

// WithBasicAuth sets the request's Authorization header to use HTTP Basic Authentication with the provided username and
// password.
func WithBasicAuth(user, password string) ClientOrHTTPClientParam {
	return WithBasicAuthProvider(func(context.Context) (BasicAuth, error) {
		return BasicAuth{User: user, Password: password}, nil
	})
}

// WithBasicAuthProvider sets the request's Authorization header to use HTTP Basic Authentication.
// The provider is expected to always return a nonempty BasicAuth value, or an error.
func WithBasicAuthProvider(provider BasicAuthProvider) ClientOrHTTPClientParam {
	return WithBasicAuthOptionalProvider(func(ctx context.Context) (*BasicAuth, error) {
		basicAuth, err := provider(ctx)
		if err != nil {
			return nil, err
		}
		return &basicAuth, nil
	})
}

// WithBasicAuthOptionalProvider sets the request's Authorization header to use HTTP Basic Authentication based on the
// return value of the provided BasicAuthOptionalProvider. If the provider returns a non-nil error, if the returned
// BasicAuth value is non-nil then its values are set on the header, while if the returned BasicAuth value is nil then
// no basic authentication header values are set.
func WithBasicAuthOptionalProvider(provider BasicAuthOptionalProvider) ClientOrHTTPClientParam {
	return WithMiddleware(MiddlewareFunc(func(req *http.Request, next http.RoundTripper) (*http.Response, error) {
		basicAuth, err := provider(req.Context())
		if err != nil {
			return nil, err
		}
		if basicAuth != nil {
			setBasicAuth(req.Header, basicAuth.User, basicAuth.Password)
		}
		return next.RoundTrip(req)
	}))
}

// WithBalancedURIScoring adds middleware that prioritizes sending requests to URIs with the fewest in-flight requests
// and least recent errors.
// Deprecated: This param is a no-op as balanced URI scoring is the default behavior.
func WithBalancedURIScoring() ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.URIScorerBuilder = func(uris []string) internal.URIScoringMiddleware {
			return internal.NewBalancedURIScoringMiddleware(uris, func() int64 {
				return time.Now().UnixNano()
			})
		}
		return nil
	})
}

// WithRandomURIScoring adds middleware that randomizes the order URIs are prioritized in for each request.
func WithRandomURIScoring() ClientParam {
	return clientParamFunc(func(b *clientBuilder) error {
		b.URIScorerBuilder = func(uris []string) internal.URIScoringMiddleware {
			return internal.NewRandomURIScoringMiddleware(uris, func() int64 {
				return time.Now().UnixNano()
			})
		}
		return nil
	})
}

func newPtr[T any](t T) *T { return &t }
