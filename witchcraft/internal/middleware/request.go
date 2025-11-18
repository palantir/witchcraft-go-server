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
	"sync/atomic"
	"time"

	"github.com/palantir/conjure-go-runtime/v3/conjure-go-contract/errors"
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/uuid"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/extractor"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wtracing/propagation/b3"
	"github.com/palantir/witchcraft-go-tracing/wzipkin"
)

const (
	strictTransportSecurityHeader = "Strict-Transport-Security"
	strictTransportSecurityValue  = "max-age=31536000"
)

// now is a local copy of time.Now() for testing purposes.
var now = time.Now

// NewRequestTelemetry is request middleware that configures instrumentation and emits telemetry that applies to all requests.
//
// * Set loggers, metrics registry, and tracer on context
// * Extract IDs from request and set on context
// * Create outer trace span for request
// * Set HSTS headers
func NewRequestTelemetry(
	svcLogger svc1log.Logger,
	evtLogger evt2log.Logger,
	audit2Logger audit2log.Logger,
	audit3Logger *atomic.Pointer[audit3log.Logger],
	metricLogger metric1log.Logger,
	diagLogger diag1log.Logger,
	reqLogger req2log.Logger,
	trcLogger trc1log.Logger,
	tracerOptions []wtracing.TracerOption,
	idsExtractor extractor.IDsFromRequest,
	metricsRegistry metrics.Registry,
) wrouter.RequestHandlerMiddleware {
	withLoggers := newRequestContextLoggers(svcLogger, evtLogger, audit2Logger, audit3Logger, metricLogger, diagLogger, reqLogger, trcLogger, metricsRegistry)
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

		lrw.Header().Set(strictTransportSecurityHeader, strictTransportSecurityValue) // set HSTS headers per RFC 6797

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
	audit2Logger audit2log.Logger,
	audit3Logger *atomic.Pointer[audit3log.Logger],
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
		if audit2Logger != nil {
			ctx = audit2log.WithLogger(ctx, audit2Logger)
		}
		if audit3Logger != nil {
			if loadedAudit3Logger := audit3Logger.Load(); loadedAudit3Logger != nil && *loadedAudit3Logger != nil {
				ctx = audit3log.WithLogger(ctx, *loadedAudit3Logger)
			}
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
			ctx = metrics.WithRegistry(metrics.AddTags(ctx), metricsRegistry)
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

		// extract IDs from request
		ids := idsExtractor.ExtractIDs(req)

		// Set IDs on context and loggers as appropriate.
		// Note that treatment between trace and audit 3 loggers is different for historical purposes.
		//
		// trc1log.FromContext does not populate any parameters from the context, so this function creates trace logger
		// parameters for every relevant value and sets the trace logger in the context to one with the parameters
		// applied.
		//
		// audit3log.FromContext extracts parameters defined at the wlog context level and sets them on the returned
		// logger. As such, this function sets these values on the context, but does not add these values to the logger
		// itself, as this would be duplicative (since the "FromContext" call will add the same parameters again).
		// However, there are also request-scoped audit.3 log parameters that are not set on the context (because they
		// are not wlog-level parameters), and for those values this function creates parameters for the relevant values
		// and sets the audit.3 logger in the context to one with the parameters applied.
		var (
			trc1LogParams   []trc1log.Param
			audit3LogParams []audit3log.Param
		)

		if uid := ids[extractor.UIDKey]; uid != "" {
			ctx = wlog.ContextWithUID(ctx, uid)
			trc1LogParams = append(trc1LogParams, trc1log.UID(uid))
			audit3LogParams = append(audit3LogParams, audit3log.Users([]audit3log.ContextualizedUser{
				{
					UID: uid,
				},
			}))
		}
		if sid := ids[extractor.SIDKey]; sid != "" {
			ctx = wlog.ContextWithSID(ctx, sid)
			trc1LogParams = append(trc1LogParams, trc1log.SID(sid))
		}
		if tokenID := ids[extractor.TokenIDKey]; tokenID != "" {
			ctx = wlog.ContextWithTokenID(ctx, tokenID)
			trc1LogParams = append(trc1LogParams, trc1log.TokenID(tokenID))
		}
		if orgID := ids[extractor.OrgIDKey]; orgID != "" {
			ctx = wlog.ContextWithOrgID(ctx, orgID)
			trc1LogParams = append(trc1LogParams, trc1log.OrgID(orgID))
		}
		if userAgent := ids[extractor.UserAgentKey]; userAgent != "" {
			audit3LogParams = append(audit3LogParams, audit3log.UserAgent(userAgent))
		}
		audit3SourceIP := ids[extractor.SourceIPKey]
		if audit3SourceIP != "" {
			audit3LogParams = append(audit3LogParams, audit3log.SourceOrigin(audit3SourceIP))
		}
		var audit3Origins []string
		if forwardedForHeaderValue := ids[extractor.ForwardIPs]; forwardedForHeaderValue != "" {
			audit3Origins = audit3log.OriginsFromXForwardedForHeaderValue(forwardedForHeaderValue)
			if len(audit3Origins) > 0 {
				audit3LogParams = append(audit3LogParams, audit3log.Origins(audit3Origins))
			}
		}
		if audit3Origin := audit3log.OriginFromForwardedOrSourceOrigin(audit3Origins, audit3SourceIP); audit3Origin != "" {
			audit3LogParams = append(audit3LogParams, audit3log.Origin(audit3SourceIP))
		}
		// set unique event ID on a per-request basis
		audit3LogParams = append(audit3LogParams, audit3log.EventID(uuid.NewUUID().String()))

		if len(trc1LogParams) > 0 {
			ctx = trc1log.WithLogger(ctx, trc1log.WithParams(trc1log.FromContext(ctx), trc1LogParams...))
		}
		if len(audit3LogParams) > 0 {
			ctx = audit3log.WithLogger(ctx, audit3log.WithParams(audit3log.FromContext(ctx), audit3LogParams...))
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
