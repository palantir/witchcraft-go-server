// Copyright (c) 2020 Palantir Technologies. All rights reserved.
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

package wdebug

import (
	"bytes"
	"context"
	"net/http"
	"runtime/pprof"
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/codecs"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/wresource"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

const (
	headerKeyContentType  = "Content-Type"
	headerKeySafeLoggable = "Safe-Loggable"
)

type DiagnosticType string

type debugResource struct {
	SharedSecret refreshable.String
}

func RegisterRoute(router wrouter.Router, sharedSecret refreshable.String) error {
	debugResource := debugResource{SharedSecret: sharedSecret}
	if err := wresource.New("witchcraftdebugservice", router).
		Get("GetDiagnostic", "/debug/diagnostic/{diagnosticType}",
			httpserver.NewJSONHandler(debugResource.ServeHTTP, httpserver.StatusCodeMapper, httpserver.ErrHandler),
		); err != nil {
		return werror.Wrap(err, "failed to register WitchcraftDebugService")
	}
	return nil
}

func (r *debugResource) ServeHTTP(rw http.ResponseWriter, req *http.Request) error {
	ctx := req.Context()
	authHeader, err := httpserver.ParseBearerTokenHeader(req)
	if err != nil {
		return errors.WrapWithNewError(err, errors.DefaultPermissionDenied)
	}
	if authHeader != r.SharedSecret.CurrentString() {
		return errors.NewPermissionDenied()
	}

	pathParams := wrouter.PathParams(req)
	if pathParams == nil {
		return werror.WrapWithContextParams(ctx, errors.NewInternal(), "path params not found on request: ensure this endpoint is registered with wrouter")
	}
	diagnosticTypeStr, ok := pathParams["diagnosticType"]
	if !ok {
		return werror.WrapWithContextParams(ctx, errors.NewInvalidArgument(), "path param not present", werror.SafeParam("pathParamName", "diagnosticType"))
	}
	diagnosticType := DiagnosticType(diagnosticTypeStr)

	return r.writeDiagnostic(ctx, diagnosticType, rw)
}

func (r *debugResource) writeDiagnostic(ctx context.Context, diagnosticType DiagnosticType, rw http.ResponseWriter) error {
	switch diagnosticType {
	case "threaddump.v1":
		return r.getThreadDumpV1(ctx, rw)
	case "go.profile.cpu.v1":
		return r.getCPUProfileV1(ctx, rw)
	case "go.profile.heap.v1":
		return r.getHeapProfileV1(ctx, rw)
	case "go.profile.allocs.v1":
		return r.getAllocsProfileV1(ctx, rw)
	default:
		return errors.WrapWithInvalidArgument(werror.ErrorWithContextParams(ctx, "unsupported diagnosticType", werror.SafeParam("diagnosticType", diagnosticType)))
	}
}

func (r *debugResource) getThreadDumpV1(ctx context.Context, rw http.ResponseWriter) error {
	var buf bytes.Buffer
	_ = pprof.Lookup("goroutine").WriteTo(&buf, 2) // bytes.Buffer's Write never returns an error, so we swallow it
	threads := diag1log.ThreadDumpV1FromGoroutines(buf.Bytes())

	rw.Header().Set(headerKeyContentType, codecs.JSON.ContentType())
	rw.Header().Set(headerKeySafeLoggable, "true")
	return codecs.JSON.Encode(rw, threads)
}

func (r *debugResource) getCPUProfileV1(ctx context.Context, rw http.ResponseWriter) error {
	// If diagnostics ever support parameters, this can be configurable
	const defaultDuration = 30 * time.Second

	rw.Header().Set(headerKeyContentType, codecs.Binary.ContentType())
	rw.Header().Set(headerKeySafeLoggable, "true")

	if err := pprof.StartCPUProfile(rw); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to start CPU profile")
	}
	defer pprof.StopCPUProfile()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(defaultDuration):
	}
	return nil
}

func (r *debugResource) getHeapProfileV1(ctx context.Context, rw http.ResponseWriter) error {
	rw.Header().Set(headerKeyContentType, codecs.Binary.ContentType())
	rw.Header().Set(headerKeySafeLoggable, "true")
	if err := pprof.Lookup("heap").WriteTo(rw, 0); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to write heap in-use profile")
	}
	return nil
}

func (r *debugResource) getAllocsProfileV1(ctx context.Context, rw http.ResponseWriter) error {
	rw.Header().Set(headerKeyContentType, codecs.Binary.ContentType())
	rw.Header().Set(headerKeySafeLoggable, "true")
	if err := pprof.Lookup("allocs").WriteTo(rw, 0); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to write heap allocs profile")
	}
	return nil
}
