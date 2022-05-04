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
	"net/http"
	"strconv"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	werror "github.com/palantir/witchcraft-go-error"
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
	r := &debugResource{SharedSecret: sharedSecret}
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
	if sharedSecret := r.SharedSecret.CurrentString(); sharedSecret != "" {
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
	diagnosticType := DiagnosticType(diagnosticTypeStr)

	handler, ok := diagnosticHandlers[diagnosticType]
	if !ok {
		return errors.WrapWithInvalidArgument(werror.ErrorWithContextParams(ctx, "unsupported diagnosticType", werror.SafeParam("diagnosticType", diagnosticType)))
	}

	rw.Header().Set(headerKeyContentType, handler.ContentType())
	rw.Header().Set(headerKeySafeLoggable, strconv.FormatBool(handler.SafeLoggable()))
	return handler.WriteDiagnostic(ctx, rw)
}
