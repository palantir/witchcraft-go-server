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
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/internal/negroni"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wtracing/propagation/b3"
)

func NewRouteTelemetry(mr metrics.Registry) wrouter.RouteHandlerMiddleware {
	return func(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
		if reqVals.DisableTelemetry {
			next(rw, req, reqVals)
			return
		}

		lrw := toLoggingResponseWriter(rw)
		req, span := newRouteLogTraceSpan(req, reqVals)
		start := now()
		defer func() {
			if span != nil {
				defer span.Finish()
			}
			duration := now().Sub(start)
			newRouteRequestLog(req, reqVals, lrw, duration)
			markRequestMetricRequestMeter(mr, req, reqVals, lrw, duration)
		}()

		// Add a panic recovery after the context is fully configured to maximize traceability.
		if err := wapp.RunWithFatalLogging(req.Context(), func(context.Context) error {
			next(lrw, req, reqVals)
			return nil
		}); err != nil {
			if lrw.Written() {
				svc1log.FromContext(req.Context()).Error("Panic recovered in request handler. This is a bug. HTTP response status already written.", svc1log.Stacktrace(err))
			} else {
				// Only write to 500 response if we have not written anything yet
				cerr := errors.WrapWithInternal(err)
				svc1log.FromContext(req.Context()).Error("Panic recovered in request handler. This is a bug. Responding 500 Internal Server Error.", svc1log.Stacktrace(cerr))
				errors.WriteErrorResponse(lrw, cerr)
			}
		}
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

func newRouteRequestLog(req *http.Request, reqVals wrouter.RequestVals, lrw loggingResponseWriter, duration time.Duration) {
	var pathPerms, queryPerms, headerPerms req2log.ParamPerms
	if reqVals.ParamPerms != nil {
		pathPerms = reqVals.ParamPerms.PathParamPerms()
		queryPerms = reqVals.ParamPerms.QueryParamPerms()
		headerPerms = reqVals.ParamPerms.HeaderParamPerms()
	}
	req2log.FromContext(req.Context()).Request(req2log.Request{
		Request: req,
		RouteInfo: req2log.RouteInfo{
			Template:   reqVals.Spec.PathTemplate,
			PathParams: reqVals.PathParamVals,
		},
		ResponseStatus:   lrw.Status(),
		ResponseSize:     int64(lrw.Size()),
		Duration:         duration,
		PathParamPerms:   pathPerms,
		QueryParamPerms:  queryPerms,
		HeaderParamPerms: headerPerms,
	})
}

func newRouteLogTraceSpan(req *http.Request, reqVals wrouter.RequestVals) (*http.Request, wtracing.Span) {
	tracer := wtracing.TracerFromContext(req.Context())
	if reqVals.DisableTelemetry || tracer == nil {
		return req, nil
	}

	// create new span and set it on the context and header
	spanName := req.Method
	if reqVals.Spec.PathTemplate != "" {
		spanName += " " + reqVals.Spec.PathTemplate
	}
	reqSpanCtx := b3.SpanExtractor(req)()
	span := tracer.StartSpan(spanName, wtracing.WithParentSpanContext(reqSpanCtx))

	ctx := req.Context()
	ctx = wtracing.ContextWithSpan(ctx, span)

	req = req.WithContext(ctx)
	b3.SpanInjector(req)(span.Context())

	return req, span
}

func markRequestMetricRequestMeter(mr metrics.Registry, req *http.Request, reqVals wrouter.RequestVals, lrw loggingResponseWriter, duration time.Duration) {
	mr.Timer("server.response", reqVals.MetricTags...).Update(duration)
	mr.Histogram("server.request.size", reqVals.MetricTags...).Update(req.ContentLength)
	mr.Histogram("server.response.size", reqVals.MetricTags...).Update(int64(lrw.Size()))
	if lrw.Status()/100 == 5 {
		mr.Meter("server.response.error", reqVals.MetricTags...).Mark(1)
	}
}
