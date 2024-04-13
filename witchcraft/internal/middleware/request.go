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

const (
	strictTransportSecurityHeader = "Strict-Transport-Security"
	strictTransportSecurityValue  = "max-age=31536000"
)

// NewRequestTelemetry is request middleware that configures instrumentation and emits telemetry that applies to all requests.
//
// * Recover panics in the wrapped handler
// * Set loggers, metrics registry, and tracer on context
// * Extract IDs from request and set on context
// * Create trace span for request
// * Set HSTS headers
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
	return func(rw http.ResponseWriter, req *http.Request, next http.Handler) {
		// This context only used in the case of a panic.
		// When this is the outermost middleware, some request information (e.g. trace ids) will not be set.
		panicCtx := req.Context()
		if svcLogger != nil {
			panicCtx = svc1log.WithLogger(panicCtx, svcLogger)
		}
		if evtLogger != nil {
			panicCtx = evt2log.WithLogger(panicCtx, evtLogger)
		}

		lrw := toLoggingResponseWriter(rw)
		if err := wapp.RunWithFatalLogging(panicCtx, func(context.Context) error {
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
			if metricsRegistry != nil {
				ctx = metrics.WithRegistry(ctx, metricsRegistry)
			}
			// add capability to store tags on the context
			ctx = metrics.AddTags(ctx)

			// extract all IDs from request
			var uid, sid, tokenID string
			if idsExtractor != nil {
				ids := idsExtractor.ExtractIDs(req)
				uid = ids[extractor.UIDKey]
				sid = ids[extractor.SIDKey]
				tokenID = ids[extractor.TokenIDKey]

				// set IDs on context for loggers
				if uid != "" {
					ctx = wlog.ContextWithUID(ctx, uid)
				}
				if sid != "" {
					ctx = wlog.ContextWithSID(ctx, sid)
				}
				if tokenID != "" {
					ctx = wlog.ContextWithTokenID(ctx, tokenID)
				}
			}

			// create tracer and set on context. Tracer logs to trace logger if it is non-nil or is a no-op if nil.
			traceReporter := wtracing.NewNoopReporter()
			if trcLogger != nil {
				// add trc1logger with params set
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

			// enforce setting HSTS headers per RFC 6797
			lrw.Header().Set(strictTransportSecurityHeader, strictTransportSecurityValue)

			// delegate to the next handler
			next.ServeHTTP(lrw, req)
			// tag the status_code
			span.Tag("http.status_code", strconv.Itoa(lrw.Status()))

			return nil
		}); err != nil {
			if lrw.Written() {
				svc1log.FromContext(panicCtx).Error("Panic recovered in server handler. This is a bug.", svc1log.Stacktrace(err))
			} else {
				// Only write to 500 response if we have not written anything yet
				cerr := errors.WrapWithInternal(err)
				svc1log.FromContext(panicCtx).Error("Panic recovered in server handler. This is a bug.", svc1log.Stacktrace(cerr))
				errors.WriteErrorResponse(lrw, cerr)
			}
		}
	}
}
