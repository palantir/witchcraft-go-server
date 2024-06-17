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
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
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
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wtracing/propagation/b3"
	"github.com/palantir/witchcraft-go-tracing/wzipkin"
)

// now is a local copy of time.Now() for testing purposes.
var now = time.Now

// NewRequestTelemetry is request middleware that configures instrumentation and emits telemetry that applies to all requests.
//
// * Set loggers, metrics registry, and tracer on context
// * Extract IDs from request and set on context
// * Create outer trace span for request
func NewRequestTelemetry(
	svcLogger svc1log.Logger,
	evtLogger evt2log.Logger,
	auditLogger audit2log.Logger,
	metricLogger metric1log.Logger,
	diagLogger diag1log.Logger,
	reqLogger req2log.Logger,
	trcLogger trc1log.Logger,
	tracerOptions []wtracing.TracerOption,
	idsExtractor extractor.IDsFromRequest,
	metricsRegistry metrics.Registry,
) wrouter.RequestHandlerMiddleware {
	withLoggers := newRequestContextLoggers(svcLogger, evtLogger, auditLogger, metricLogger, diagLogger, reqLogger, trcLogger, metricsRegistry)
	withIDs := newRequestExtractIDs(idsExtractor)
	withTracer := newRequestTracer(svcLogger, trcLogger, tracerOptions)
	withSpan := newRequestTraceSpan()
	return func(rw http.ResponseWriter, req *http.Request, next http.Handler) {
		lrw := toLoggingResponseWriter(rw)
		req = withLoggers(req)
		req = withIDs(req)
		req = withTracer(req)
		req, finishSpan := withSpan(req)
		defer finishSpan(lrw)

		if err := wapp.RunWithFatalLogging(req.Context(), func(context.Context) error {
			next.ServeHTTP(lrw, req)
			return nil
		}); err != nil {
			if lrw.Written() {
				svc1log.FromContext(req.Context()).Error("Panic recovered in server handler. This is a bug. HTTP response status already written.", svc1log.Stacktrace(err))
			} else {
				// Only write to 500 response if we have not written anything yet
				cerr := errors.WrapWithInternal(err)
				svc1log.FromContext(req.Context()).Error("Panic recovered in server handler. This is a bug. Responding 500 Internal Server Error.", svc1log.Stacktrace(cerr))
				errors.WriteErrorResponse(lrw, cerr)
			}
		}
	}
}

func newRequestContextLoggers(
	svcLogger svc1log.Logger,
	evtLogger evt2log.Logger,
	auditLogger audit2log.Logger,
	metricLogger metric1log.Logger,
	diagLogger diag1log.Logger,
	reqLogger req2log.Logger,
	trcLogger trc1log.Logger,
	metricsRegistry metrics.Registry,
) func(req *http.Request) *http.Request {
	return func(req *http.Request) *http.Request {
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
		if trcLogger != nil {
			ctx = trc1log.WithLogger(ctx, trcLogger)
		}
		if metricsRegistry != nil {
			ctx = metrics.WithRegistry(ctx, metricsRegistry)
		}
		return req.WithContext(ctx)
	}
}

func newRequestExtractIDs(idsExtractor extractor.IDsFromRequest) func(req *http.Request) *http.Request {
	if idsExtractor == nil {
		idsExtractor = extractor.NewDefaultIDsExtractor()
	}
	return func(req *http.Request) *http.Request {
		ctx := req.Context()
		// extract all IDs from request
		ids := idsExtractor.ExtractIDs(req)
		uid := ids[extractor.UIDKey]
		sid := ids[extractor.SIDKey]
		tokenID := ids[extractor.TokenIDKey]

		// set IDs on context for loggers
		if uid != "" {
			ctx = wlog.ContextWithUID(ctx, uid)
			ctx = trc1log.WithLogger(ctx, trc1log.WithParams(trc1log.FromContext(ctx), trc1log.UID(uid)))
		}
		if sid != "" {
			ctx = wlog.ContextWithSID(ctx, sid)
			ctx = trc1log.WithLogger(ctx, trc1log.WithParams(trc1log.FromContext(ctx), trc1log.SID(sid)))
		}
		if tokenID != "" {
			ctx = wlog.ContextWithTokenID(ctx, tokenID)
			ctx = trc1log.WithLogger(ctx, trc1log.WithParams(trc1log.FromContext(ctx), trc1log.TokenID(tokenID)))
		}
		return req.WithContext(ctx)
	}
}

func newRequestTracer(svcLogger svc1log.Logger, trcLogger trc1log.Logger, tracerOptions []wtracing.TracerOption) func(req *http.Request) *http.Request {
	return func(req *http.Request) *http.Request {
		ctx := req.Context()
		// create tracer and set on context. Tracer logs to trace logger if it is non-nil or is a no-op if nil.
		traceReporter := wtracing.NewNoopReporter()
		if trcLogger != nil {
			traceReporter = trcLogger
		}
		tracer, err := wzipkin.NewTracer(traceReporter, tracerOptions...)
		if err != nil && svcLogger != nil {
			svcLogger.Error("Failed to create tracer", svc1log.Stacktrace(err))
		}
		ctx = wtracing.ContextWithTracer(ctx, tracer)
		return req.WithContext(ctx)
	}
}

func newRequestTraceSpan() func(req *http.Request) (*http.Request, func(writer loggingResponseWriter)) {
	return func(req *http.Request) (*http.Request, func(writer loggingResponseWriter)) {
		ctx := req.Context()
		// retrieve existing trace info from request and create a span
		reqSpanContext := b3.SpanExtractor(req)()
		span, ctx := wtracing.StartSpanFromTracerInContext(ctx, "witchcraft-go-server request middleware",
			wtracing.WithParentSpanContext(reqSpanContext),
			wtracing.WithSpanTag("http.method", req.Method),
			wtracing.WithSpanTag("http.useragent", req.UserAgent()),
		)
		b3.SpanInjector(req)(span.Context())
		return req.WithContext(ctx), func(lrw loggingResponseWriter) {
			span.Tag("http.status_code", strconv.Itoa(lrw.Status()))
			span.Finish()
		}
	}
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
