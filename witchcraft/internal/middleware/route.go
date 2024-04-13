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
	"fmt"
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

const (
	serverResponseMetricName      = "server.response"
	serverResponseErrorMetricName = "server.response.error"
	serverRequestSizeMetricName   = "server.request.size"
	serverResponseSizeMetricName  = "server.response.size"
)

// now is a local copy of time.Now() for testing purposes.
var now = time.Now

// NewRouteTelemetry returns a middleware which logs the request and records metrics for the request.
// This is distinct from NewRequestTelemetry in that it handles telemetry for specific routes.
//
// * Recover panics in the wrapped handler
// * Record metrics for request
// * Create trace span for request
// * Emit request.2 logs
func NewRouteTelemetry(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
	if !reqVals.DisableTelemetry {
		span, ctx := wtracing.StartSpanFromTracerInContext(
			req.Context(),
			fmt.Sprintf("%s %s", req.Method, reqVals.Spec.PathTemplate),
			wtracing.WithParentSpanContext(b3.SpanExtractor(req)()))
		defer span.Finish()
		req = req.WithContext(ctx)
		b3.SpanInjector(req)(span.Context())
	}

	ctx := req.Context()
	lrw := toLoggingResponseWriter(rw)
	start := now()
	// add a second, inner panic recovery middleware so panics within handler logic are correctly configured with logging, trace IDs, etc.
	if err := wapp.RunWithFatalLogging(ctx, func(ctx context.Context) error {
		next(lrw, req, reqVals)
		return nil
	}); err != nil {
		if lrw.Written() {
			svc1log.FromContext(ctx).Error("Panic recovered in request handler. This is a bug.", svc1log.Stacktrace(err))
		} else {
			// Only write to 500 response if we have not written anything yet
			cerr := errors.WrapWithInternal(err)
			svc1log.FromContext(ctx).Error("Panic recovered in request handler. This is a bug.", svc1log.Stacktrace(cerr))
			errors.WriteErrorResponse(lrw, cerr)
		}
	}
	duration := now().Sub(start)

	if !reqVals.DisableTelemetry {
		var pathPerm, queryPerm, headerPerm wrouter.ParamPerms
		if reqVals.ParamPerms != nil {
			pathPerm = reqVals.ParamPerms.PathParamPerms()
			queryPerm = reqVals.ParamPerms.QueryParamPerms()
			headerPerm = reqVals.ParamPerms.HeaderParamPerms()
		}
		// Log the request
		req2log.FromContext(req.Context()).Request(req2log.Request{
			Request: req,
			RouteInfo: req2log.RouteInfo{
				Template:   reqVals.Spec.PathTemplate,
				PathParams: reqVals.PathParamVals,
			},
			ResponseStatus:   lrw.Status(),
			ResponseSize:     int64(lrw.Size()),
			Duration:         duration,
			PathParamPerms:   pathPerm,
			QueryParamPerms:  queryPerm,
			HeaderParamPerms: headerPerm,
		})

		// record metrics for call
		mr := metrics.FromContext(req.Context())
		mr.Timer(serverResponseMetricName, reqVals.MetricTags...).Update(duration)
		mr.Histogram(serverRequestSizeMetricName, reqVals.MetricTags...).Update(req.ContentLength)
		mr.Histogram(serverResponseSizeMetricName, reqVals.MetricTags...).Update(int64(lrw.Size()))
		if lrw.Status()/100 == 5 {
			mr.Meter(serverResponseErrorMetricName, reqVals.MetricTags...).Mark(1)
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
	// Written returns whether the ResponseWriter has been written.
	Written() bool
}
