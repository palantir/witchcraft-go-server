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
	"net/http"

	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wtracing/propagation/b3"
)

// traceMiddleware injects tracing information from the request's context into the request headers.
// If there is no wtracing.Tracer on the context, this middleware is a no-op.
// Only if the RPC method name is set does the middleware create a new span (with that name) for the
// duration of the request.
type traceMiddleware struct {
	ServiceName string
}

func (t *traceMiddleware) RoundTrip(req *http.Request, next http.RoundTripper) (*http.Response, error) {
	ctx := req.Context()
	span := wtracing.SpanFromContext(ctx)

	// Create a child span if a method name is set. Otherwise, fall through and just inject the parent span's headers.
	if method := getRPCMethodName(req.Context()); method != "" {
		span, ctx = wtracing.StartSpanFromContext(ctx, wtracing.TracerFromContext(ctx), method,
			wtracing.WithKind(wtracing.Client),
			wtracing.WithRemoteEndpoint(&wtracing.Endpoint{ServiceName: t.ServiceName}))
		if span != nil {
			defer span.Finish()
		}
		req = req.WithContext(ctx)
	}

	if span != nil {
		b3.SpanInjector(req)(span.Context())
	}

	return next.RoundTrip(req)
}
