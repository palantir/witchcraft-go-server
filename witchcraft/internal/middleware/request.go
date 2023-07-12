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

package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/extractor"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wtracing/propagation/b3"
	"github.com/palantir/witchcraft-go-tracing/wzipkin"
)

// now is a local copy of time.Now() for testing purposes.
var now = time.Now

// NewRequestPanicRecovery returns a middleware which recovers panics in the wrapped handler.
// It accepts loggers as arguments, as we are not guaranteed they have been set on the request context.
// These loggers are only used in the case of a panic.
// When this is the outermost middleware, some request information (e.g. trace ids) will not be set.
func NewRequestPanicRecovery(svcLogger svc1log.Logger, evtLogger evt2log.Logger) wrouter.RequestHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, next http.Handler) {
		lrw := toLoggingResponseWriter(rw)
		panicRecoveryMiddleware(lrw, req, svcLogger, evtLogger, func() {
			next.ServeHTTP(lrw, req)
		})
	}
}

// NewRequestContextLoggers is request middleware that sets loggers that can be retrieved from a context on the request
// context.
func NewRequestContextLoggers(
	svcLogger svc1log.Logger,
	evtLogger evt2log.Logger,
	auditLogger audit2log.Logger,
	metricLogger metric1log.Logger,
	diagLogger diag1log.Logger,
	reqLogger req2log.Logger,
) wrouter.RequestHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, next http.Handler) {
		ctx := req.Context()
		if svcLogger != nil {
			ctx = svc1log.WithLogger(ctx, svcLogger)
		}
		if evtLogger != nil {
			ctx = evt2log.WithLogger(ctx, evtLogger)
		}
		if auditLogger != nil {
			ctx = audit2log.WithLogger(ctx, auditLogger)
		}
		if metricLogger != nil {
			ctx = metric1log.WithLogger(ctx, metricLogger)
		}
		if diagLogger != nil {
			ctx = diag1log.WithLogger(ctx, diagLogger)
		}
		if reqLogger != nil {
			ctx = req2log.WithLogger(ctx, reqLogger)
		}
		next.ServeHTTP(rw, req.WithContext(ctx))
	}
}

// NewRequestContextMetricsRegistry is request middleware that sets the metrics registry on the request context.
func NewRequestContextMetricsRegistry(metricsRegistry metrics.Registry) wrouter.RequestHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, next http.Handler) {
		ctx := req.Context()
		if metricsRegistry != nil {
			ctx = metrics.WithRegistry(ctx, metricsRegistry)
		}
		next.ServeHTTP(rw, req.WithContext(ctx))
	}
}

func NewRequestExtractIDs(
	svcLogger svc1log.Logger,
	trcLogger trc1log.Logger,
	tracerOptions []wtracing.TracerOption,
	idsExtractor extractor.IDsFromRequest,
) wrouter.RequestHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, next http.Handler) {
		// extract all IDs from request
		ids := idsExtractor.ExtractIDs(req)
		uid := ids[extractor.UIDKey]
		sid := ids[extractor.SIDKey]
		tokenID := ids[extractor.TokenIDKey]

		// set IDs on context for loggers
		ctx := req.Context()
		if uid != "" {
			ctx = wlog.ContextWithUID(ctx, uid)
		}
		if sid != "" {
			ctx = wlog.ContextWithSID(ctx, sid)
		}
		if tokenID != "" {
			ctx = wlog.ContextWithTokenID(ctx, tokenID)
		}

		// create tracer and set on context. Tracer logs to trace logger if it is non-nil or is a no-op if nil.
		traceReporter := wtracing.NewNoopReporter()
		if trcLogger != nil {
			// add trc1logger with params set
			// TODO(nmiyake): there is currently ongoing discussion about whether or not these fields are required for trace logs. If they are not, it would cleaner to put the logic that extracts the IDs into its own request middleware layer.
			ctx = trc1log.WithLogger(ctx, trc1log.WithParams(trcLogger, trc1log.UID(uid), trc1log.SID(sid), trc1log.TokenID(tokenID)))
			traceReporter = trcLogger
		}
		tracer, err := wzipkin.NewTracer(traceReporter, tracerOptions...)
		if err != nil && svcLogger != nil {
			svcLogger.Error("Failed to create tracer", svc1log.Stacktrace(err))
		}
		ctx = wtracing.ContextWithTracer(ctx, tracer)

		// retrieve existing trace info from request and create a span
		reqSpanContext := b3.SpanExtractor(req)()
		span := tracer.StartSpan("witchcraft-go-server request middleware",
			wtracing.WithParentSpanContext(reqSpanContext),
			wtracing.WithSpanTag("http.method", req.Method),
			wtracing.WithSpanTag("http.useragent", req.UserAgent()),
		)
		defer span.Finish()

		ctx = wtracing.ContextWithSpan(ctx, span)
		b3.SpanInjector(req)(span.Context())

		// update request with new context
		req = req.WithContext(ctx)

		// delegate to the next handler
		lrw := toLoggingResponseWriter(rw)
		next.ServeHTTP(lrw, req)
		// tag the status_code
		span.Tag("http.status_code", strconv.Itoa(lrw.Status()))
	}
}

// staticRootSpanIDGenerator returns the stored TraceID as its TraceID and SpanID.
type staticRootSpanIDGenerator wtracing.TraceID

func (s staticRootSpanIDGenerator) TraceID() wtracing.TraceID {
	return wtracing.TraceID(s)
}

func (s staticRootSpanIDGenerator) SpanID(traceID wtracing.TraceID) wtracing.SpanID {
	return wtracing.SpanID(s)
}

func NewRequestMetricRequestMeter(mr metrics.RootRegistry) wrouter.RouteHandlerMiddleware {
	const (
		serverResponseMetricName      = "server.response"
		serverResponseErrorMetricName = "server.response.error"
		serverRequestSizeMetricName   = "server.request.size"
		serverResponseSizeMetricName  = "server.response.size"
	)
	return func(rw http.ResponseWriter, r *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
		if reqVals.DisableTelemetry {
			next(rw, r, reqVals)
			return
		}
		// add capability to store tags on the context
		r = r.WithContext(metrics.AddTags(r.Context()))

		start := now()
		lrw := toLoggingResponseWriter(rw)
		next(lrw, r, reqVals)
		end := now()

		tags := reqVals.MetricTags
		// record metrics for call
		mr.Timer(serverResponseMetricName, tags...).Update(end.Sub(start))
		mr.Histogram(serverRequestSizeMetricName, tags...).Update(r.ContentLength)
		mr.Histogram(serverResponseSizeMetricName, tags...).Update(int64(lrw.Size()))
		if lrw.Status()/100 == 5 {
			mr.Meter(serverResponseErrorMetricName, tags...).Mark(1)
		}
	}
}

func NewStrictTransportSecurityHeader() wrouter.RequestHandlerMiddleware {
	const (
		strictTransportSecurityHeader = "Strict-Transport-Security"
		strictTransportSecurityValue  = "max-age=31536000"
	)
	return func(rw http.ResponseWriter, r *http.Request, next http.Handler) {
		rw.Header().Set(strictTransportSecurityHeader, strictTransportSecurityValue)
		next.ServeHTTP(rw, r)
	}
}
