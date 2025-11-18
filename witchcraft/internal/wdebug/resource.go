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
	"context"
	"net/http"
	"strconv"

	"github.com/palantir/conjure-go-runtime/v3/conjure-go-contract/errors"
	"github.com/palantir/conjure-go-runtime/v3/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable/v2"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft/wdebug"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft/wresource"
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
)

const (
	headerKeyContentType  = "Content-Type"
	headerKeySafeLoggable = "Safe-Loggable"
)

type debugResource struct {
	SharedSecret             refreshable.Refreshable[string]
	customDiagnosticHandlers map[wdebug.DiagnosticType]wdebug.DiagnosticHandler
}

func RegisterRoute(ctx context.Context, router wrouter.Router, sharedSecret refreshable.Refreshable[string], customDiagnosticHandlers ...wdebug.DiagnosticHandler) error {
	customHandlersByType := make(map[wdebug.DiagnosticType]wdebug.DiagnosticHandler, len(customDiagnosticHandlers))
	for _, handler := range customDiagnosticHandlers {
		if err := handler.Type().Validate(); err != nil {
			return werror.WrapWithContextParams(ctx, err, "failed to register WitchcraftDebugService")
		}
		customHandlersByType[handler.Type()] = handler
	}
	r := &debugResource{SharedSecret: sharedSecret, customDiagnosticHandlers: customHandlersByType}
	if err := wresource.New("witchcraftdebugservice", router).
		Get("GetDiagnostic", "/debug/diagnostic/{diagnosticType}",
			httpserver.NewJSONHandler(r.ServeHTTP, httpserver.StatusCodeMapper, httpserver.ErrHandler),
		); err != nil {
		return werror.Wrap(err, "failed to register WitchcraftDebugService")
	}
	return nil
}

func (r *debugResource) ServeHTTP(rw http.ResponseWriter, req *http.Request) error {
	ctx := req.Context()
	if sharedSecret := r.SharedSecret.Current(); sharedSecret != "" {
		token, err := httpserver.ParseBearerTokenHeader(req)
		if err != nil {
			return errors.WrapWithUnauthorized(err)
		}
		if !httpserver.SecretStringEqual(sharedSecret, token) {
			return errors.NewUnauthorized()
		}
	}

	pathParams := wrouter.PathParams(req)
	if pathParams == nil {
		return werror.WrapWithContextParams(ctx, errors.NewInternal(), "path params not found on request: ensure this endpoint is registered with wrouter")
	}
	diagnosticTypeStr, ok := pathParams["diagnosticType"]
	if !ok {
		return werror.WrapWithContextParams(ctx, errors.NewInvalidArgument(), "path param not present", werror.SafeParam("pathParamName", "diagnosticType"))
	}
	diagnosticType := wdebug.DiagnosticType(diagnosticTypeStr)

	handler, ok := r.resolveHandlerForType(diagnosticType)
	if !ok {
		return errors.WrapWithInvalidArgument(werror.ErrorWithContextParams(ctx, "unsupported diagnosticType", werror.SafeParam("diagnosticType", diagnosticType)))
	}

	rw.Header().Set(headerKeyContentType, handler.ContentType())
	rw.Header().Set(headerKeySafeLoggable, strconv.FormatBool(handler.SafeLoggable()))
	return handler.WriteDiagnostic(ctx, rw)
}

func (r *debugResource) resolveHandlerForType(diagnosticType wdebug.DiagnosticType) (wdebug.DiagnosticHandler, bool) {
	handler, ok := diagnosticHandlers[diagnosticType]
	if ok {
		return handler, true
	}
	handler, ok = r.customDiagnosticHandlers[diagnosticType]
	return handler, ok
}
