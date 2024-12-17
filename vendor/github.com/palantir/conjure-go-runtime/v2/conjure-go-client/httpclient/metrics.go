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
	"errors"
	"net"
	"net/http"
	"net/http/httptrace"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/refreshable"
	werror "github.com/palantir/witchcraft-go-error"
)

const (
	MetricTagServiceName = "service-name"
	metricClientResponse = "client.response"
	metricTagFamily      = "family"
	metricTagMethod      = "method"
	metricRPCMethodName  = "method-name"

	MetricTLSHandshakeAttempt = "tls.handshake.attempt"
	MetricTLSHandshakeFailure = "tls.handshake.failure"
	MetricTLSHandshake        = "tls.handshake"
	CipherTagKey              = "cipher"
	NextProtocolTagKey        = "next_protocol"
	TLSVersionTagKey          = "tls_version"

	MetricConnCreate      = "client.connection.create" // monotonic counter of each new request, tagged with reused:true or reused:false
	MetricRequestInFlight = "client.request.in-flight"
)

var (
	MetricTagConnectionNew    = metrics.MustNewTag("reused", "false")
	MetricTagConnectionReused = metrics.MustNewTag("reused", "true")

	metricTagFamily1xx     = metrics.MustNewTag(metricTagFamily, "1xx")
	metricTagFamily2xx     = metrics.MustNewTag(metricTagFamily, "2xx")
	metricTagFamily3xx     = metrics.MustNewTag(metricTagFamily, "3xx")
	metricTagFamily4xx     = metrics.MustNewTag(metricTagFamily, "4xx")
	metricTagFamily5xx     = metrics.MustNewTag(metricTagFamily, "5xx")
	metricTagFamilyOther   = metrics.MustNewTag(metricTagFamily, "other")
	metricTagFamilyTimeout = metrics.MustNewTag(metricTagFamily, "timeout")
)

// A TagsProvider returns metrics tags based on an http round trip.
// The 'error' argument is that returned from the request (if any).
type TagsProvider interface {
	Tags(*http.Request, *http.Response, error) metrics.Tags
}

// TagsProviderFunc is a convenience type that implements TagsProvider.
type TagsProviderFunc func(*http.Request, *http.Response, error) metrics.Tags

func (f TagsProviderFunc) Tags(req *http.Request, resp *http.Response, respErr error) metrics.Tags {
	return f(req, resp, respErr)
}

type StaticTagsProvider metrics.Tags

func (s StaticTagsProvider) Tags(_ *http.Request, _ *http.Response, _ error) metrics.Tags {
	return metrics.Tags(s)
}

// MetricsMiddleware updates the "client.response" timer metric on every request.
// By default, metrics are tagged with 'service-name', 'method', and 'family' (of the
// status code). This metric name and tag set matches http-remoting's DefaultHostMetrics:
// https://github.com/palantir/http-remoting/blob/develop/okhttp-clients/src/main/java/com/palantir/remoting3/okhttp/DefaultHostMetrics.java
func MetricsMiddleware(serviceName string, tagProviders ...TagsProvider) (Middleware, error) {
	refreshableName := refreshable.NewString(refreshable.NewDefaultRefreshable(serviceName))
	return newMetricsMiddleware(refreshableName, tagProviders, nil), nil
}

func newMetricsMiddleware(serviceName refreshable.String, tagProviders []TagsProvider, disabled refreshable.Bool) Middleware {
	return &metricsMiddleware{
		Disabled:    disabled,
		ServiceName: serviceName,
		Tags: append(
			tagProviders,
			TagsProviderFunc(tagStatusFamily),
			TagsProviderFunc(tagRequestMethod),
			TagsProviderFunc(tagRequestMethodName),
		),
	}
}

type metricsMiddleware struct {
	Disabled    refreshable.Bool
	ServiceName refreshable.String
	Tags        []TagsProvider
}

// RoundTrip will emit counter and timer metrics with the name 'mariner.k8sClient.request'
// and k8s for API group, API version, namespace, resource kind, request method, and response status code.
func (h *metricsMiddleware) RoundTrip(req *http.Request, next http.RoundTripper) (*http.Response, error) {
	if h.Disabled != nil && h.Disabled.CurrentBool() {
		// If we have a Disabled refreshable and it is true, no-op.
		return next.RoundTrip(req)
	}
	serviceNameTag := metrics.NewTagWithFallbackValue(MetricTagServiceName, h.ServiceName.CurrentString(), "unknown")

	metrics.FromContext(req.Context()).Counter(MetricRequestInFlight, serviceNameTag).Inc(1)
	start := time.Now()
	tlsMetricsContext := h.tlsTraceContext(req.Context(), serviceNameTag)
	resp, err := next.RoundTrip(req.WithContext(tlsMetricsContext))
	duration := time.Since(start)
	metrics.FromContext(req.Context()).Counter(MetricRequestInFlight, serviceNameTag).Dec(1)

	tags := []metrics.Tag{serviceNameTag}
	for _, tagProvider := range h.Tags {
		tags = append(tags, tagProvider.Tags(req, resp, err)...)
	}

	metrics.FromContext(req.Context()).Timer(metricClientResponse, tags...).Update(duration / time.Microsecond)
	return resp, err
}

func tagStatusFamily(_ *http.Request, resp *http.Response, respErr error) metrics.Tags {
	switch {
	case isTimeoutError(respErr):
		return metrics.Tags{metricTagFamilyTimeout}
	case resp == nil, resp.StatusCode < 100, resp.StatusCode > 599:
		return metrics.Tags{metricTagFamilyOther}
	case resp.StatusCode < 200:
		return metrics.Tags{metricTagFamily1xx}
	case resp.StatusCode < 300:
		return metrics.Tags{metricTagFamily2xx}
	case resp.StatusCode < 400:
		return metrics.Tags{metricTagFamily3xx}
	case resp.StatusCode < 500:
		return metrics.Tags{metricTagFamily4xx}
	case resp.StatusCode < 600:
		return metrics.Tags{metricTagFamily5xx}
	}
	// unreachable
	return metrics.Tags{}
}

func tagRequestMethod(req *http.Request, _ *http.Response, _ error) metrics.Tags {
	return metrics.Tags{metrics.MustNewTag(metricTagMethod, req.Method)}
}

func tagRequestMethodName(req *http.Request, _ *http.Response, _ error) metrics.Tags {
	rpcMethodName := getRPCMethodName(req.Context())
	if rpcMethodName == "" {
		return metrics.Tags{metrics.MustNewTag(metricRPCMethodName, "RPCMethodNameMissing")}
	}
	tag, err := metrics.NewTag(metricRPCMethodName, rpcMethodName)
	if err == nil {
		return metrics.Tags{tag}
	}
	return metrics.Tags{metrics.MustNewTag(metricRPCMethodName, "RPCMethodNameInvalid")}
}

func (h *metricsMiddleware) tlsTraceContext(ctx context.Context, serviceNameTag metrics.Tag) context.Context {
	return httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{
		GotConn: func(info httptrace.GotConnInfo) {
			if info.Reused {
				metrics.FromContext(ctx).Counter(MetricConnCreate, serviceNameTag, MetricTagConnectionReused).Inc(1)
			} else {
				metrics.FromContext(ctx).Counter(MetricConnCreate, serviceNameTag, MetricTagConnectionNew).Inc(1)
			}
		},
		TLSHandshakeStart: func() {
			metrics.FromContext(ctx).Meter(MetricTLSHandshakeAttempt, serviceNameTag).Mark(1)
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			tags := []metrics.Tag{serviceNameTag}
			cipherSuite := tls.CipherSuiteName(state.CipherSuite)
			if cipherSuite != "" {
				tags = append(tags, metrics.MustNewTag(CipherTagKey, cipherSuite))
			}
			if state.NegotiatedProtocol != "" {
				tags = append(tags, metrics.MustNewTag(NextProtocolTagKey, state.NegotiatedProtocol))
			}
			if tlsVersion := tlsVersionString(state.Version); tlsVersion != "" {
				tags = append(tags, metrics.MustNewTag(TLSVersionTagKey, tlsVersion))
			}
			if err != nil {
				metrics.FromContext(ctx).Meter(MetricTLSHandshakeFailure, tags...).Mark(1)
			} else {
				metrics.FromContext(ctx).Meter(MetricTLSHandshake, tags...).Mark(1)
			}
		},
	})
}

func tlsVersionString(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS10"
	case tls.VersionTLS11:
		return "TLS11"
	case tls.VersionTLS12:
		return "TLS12"
	case tls.VersionTLS13:
		return "TLS13"
	}
	return ""
}

func isTimeoutError(respErr error) bool {
	if respErr == nil {
		return false
	}
	rootErr := werror.RootCause(respErr)
	if rootErr == nil {
		return false
	}

	if nerr, ok := rootErr.(net.Error); ok && nerr.Timeout() {
		return true
	}
	if errors.Is(rootErr, context.Canceled) || errors.Is(rootErr, context.DeadlineExceeded) {
		return true
	}
	// N.B. the http package does not expose these error types
	if rootErr.Error() == "net/http: request canceled" || rootErr.Error() == "net/http: request canceled while waiting for connection" {
		return true
	}
	return false
}
