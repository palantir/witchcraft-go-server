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
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/internal/negroni"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wtracing/propagation/b3"
)

func NewRouteRequestLog() wrouter.RouteHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
		if reqVals.DisableTelemetry {
			next(rw, req, reqVals)
			return
		}

		lrw := toLoggingResponseWriter(rw)
		start := time.Now()
		next(lrw, req, reqVals)
		duration := time.Since(start)

		req2log.FromContext(req.Context()).Request(req2log.Request{
			Request: req,
			RouteInfo: req2log.RouteInfo{
				Template:   reqVals.Spec.PathTemplate,
				PathParams: reqVals.PathParamVals,
			},
			ResponseStatus:   lrw.Status(),
			ResponseSize:     int64(lrw.Size()),
			Duration:         duration,
			PathParamPerms:   reqVals.ParamPerms.PathParamPerms(),
			QueryParamPerms:  reqVals.ParamPerms.QueryParamPerms(),
			HeaderParamPerms: reqVals.ParamPerms.HeaderParamPerms(),
		})
	}
}

func toLoggingResponseWriter(rw http.ResponseWriter) loggingResponseWriter {
	if lrw, ok := rw.(loggingResponseWriter); ok {
		return lrw
	}
	// if provided responseWriter does not implement loggingResponseWriter, wrap in a negroniResponseWriter,
	// which implements the interface. There is no particular reason that the default implementation is
	// negroni's beyond the fact that negroni already provides an implementation of the required interface.
	return negroni.NewResponseWriter(rw)
}

// loggingResponseWriter defines the functions required by the logging handler to get information on the status and
// size of the response written by a writer.
type loggingResponseWriter interface {
	http.ResponseWriter
	// Status returns the status code of the response or 0 if the response has not been written.
	Status() int
	// Size returns the size of the response body.
	Size() int
	// Written returns whether or not the ResponseWriter has been written.
	Written() bool
}

func NewRouteLogTraceSpan() wrouter.RouteHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
		if reqVals.DisableTelemetry {
			next(rw, req, reqVals)
			return
		}

		tracer := wtracing.TracerFromContext(req.Context())
		if tracer == nil {
			next(rw, req, reqVals)
			return
		}

		// create new span and set it on the context and header
		spanName := req.Method
		if reqVals.Spec.PathTemplate != "" {
			spanName += " " + reqVals.Spec.PathTemplate
		}
		reqSpanCtx := b3.SpanExtractor(req)()
		span := tracer.StartSpan(spanName, wtracing.WithParentSpanContext(reqSpanCtx))
		defer span.Finish()

		ctx := req.Context()
		ctx = wtracing.ContextWithSpan(ctx, span)

		req = req.WithContext(ctx)
		b3.SpanInjector(req)(span.Context())

		next(rw, req, reqVals)
	}
}

// NewRoutePanicRecovery returns a middleware which recovers panics within the inner route handler.
// This is distinct from NewRequestPanicRecovery in that it runs when all logging/telemetry are configured on the request.
func NewRoutePanicRecovery() wrouter.RouteHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
		lrw := toLoggingResponseWriter(rw)
		panicRecoveryMiddleware(lrw, req, nil, nil, func() {
			next(lrw, req, reqVals)
		})
	}
}

func panicRecoveryMiddleware(lrw loggingResponseWriter, req *http.Request, svcLogger svc1log.Logger, evtLogger evt2log.Logger, nextFunc func()) {
	ctx := req.Context() // ctx changes are used within this middleware but not stored to the request
	if svcLogger != nil {
		ctx = svc1log.WithLogger(ctx, svcLogger)
	}
	if evtLogger != nil {
		ctx = evt2log.WithLogger(ctx, evtLogger)
	}

	if err := wapp.RunWithRecoveryLoggingWithError(ctx, func(ctx context.Context) error {
		nextFunc()
		return nil
	}); err != nil {
		cerr := errors.WrapWithInternal(err)
		httpserver.ErrHandler(ctx, cerr.Code().StatusCode(), cerr)

		// Only write to response if we have not written anything yet
		if !lrw.Written() {
			errors.WriteErrorResponse(lrw, cerr)
		}
	}
}
