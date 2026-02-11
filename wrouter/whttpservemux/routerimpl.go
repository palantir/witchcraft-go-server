// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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

package whttpservemux

import (
	"net/http"
	"strings"

	"github.com/palantir/witchcraft-go-server/v3/wrouter"
)

// New returns a wrouter.RouterImpl backed by a new http.ServeMux configured using the provided parameters.
func New(params ...Param) wrouter.RouterImpl {
	r := &router{
		mux: http.NewServeMux(),
	}
	for _, p := range params {
		p.apply(r)
	}
	return r
}

// Param is a parameter that configures the behavior of a router.
type Param interface {
	apply(*router)
}

type paramFunc func(*router)

func (f paramFunc) apply(r *router) {
	f(r)
}

type router struct {
	mux             *http.ServeMux
	notFoundHandler http.Handler
}

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// [http.ServeMux.Handler] returns an empty pattern when no registered handler
	// applies to the request (both 404 not found and 405 method not allowed cases).
	// We must call r.mux.ServeHTTP (not the returned handler h directly) because
	// [http.ServeMux.Handler] does not populate named path wildcards on the request;
	// only [http.ServeMux.ServeHTTP] sets req.PathValue, which PathParams relies on.
	_, pattern := r.mux.Handler(req)
	if pattern == "" && r.notFoundHandler != nil {
		r.notFoundHandler.ServeHTTP(w, req)
		return
	}
	r.mux.ServeHTTP(w, req)
}

func (r *router) Register(method string, pathSegments []wrouter.PathSegment, handler http.Handler) {
	r.mux.Handle(convertPathSegments(method, pathSegments), handler)
}

func (r *router) RegisterNotFoundHandler(handler http.Handler) {
	r.notFoundHandler = handler
}

func (r *router) PathParams(req *http.Request, pathVarNames []string) map[string]string {
	if len(pathVarNames) == 0 {
		return nil
	}
	params := make(map[string]string)
	for _, name := range pathVarNames {
		params[name] = req.PathValue(name)
	}
	return params
}

// convertPathSegments converts wrouter path segments and an HTTP method into an http.ServeMux pattern string.
func convertPathSegments(method string, pathSegments []wrouter.PathSegment) string {
	pathParts := make([]string, len(pathSegments))
	for i, segment := range pathSegments {
		switch segment.Type {
		case wrouter.PathParamSegment:
			pathParts[i] = "{" + segment.Value + "}"
		case wrouter.TrailingPathParamSegment:
			pathParts[i] = "{" + segment.Value + "...}"
		default:
			pathParts[i] = segment.Value
		}
	}
	path := "/" + strings.Join(pathParts, "/")
	// Append {$} when path ends with "/" to prevent subtree matching.
	// This is critical for the root path "/" which would otherwise match all requests.
	if strings.HasSuffix(path, "/") {
		path += "{$}"
	}
	return method + " " + path
}
