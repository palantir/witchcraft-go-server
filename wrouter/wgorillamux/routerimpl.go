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

package wgorillamux

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/palantir/witchcraft-go-server/wrouter"
)

// New returns a wrouter.RouterImpl backed by a new mux.Router configured using the provided parameters.
func New(params ...Param) wrouter.RouterImpl {
	r := mux.NewRouter()
	for _, p := range params {
		p.apply(r)
	}
	return (*router)(r)
}

type Param interface {
	apply(*mux.Router)
}

type paramFunc func(*mux.Router)

func (f paramFunc) apply(r *mux.Router) {
	f(r)
}

func NotFoundHandler(h http.Handler) Param {
	return paramFunc(func(r *mux.Router) {
		r.NotFoundHandler = h
	})
}

func StrictSlash(value bool) Param {
	return paramFunc(func(r *mux.Router) {
		r.StrictSlash(value)
	})
}

func SkipClean(value bool) Param {
	return paramFunc(func(r *mux.Router) {
		r.SkipClean(value)
	})
}

func UseEncodedPath() Param {
	return paramFunc(func(r *mux.Router) {
		r.UseEncodedPath()
	})
}

type router mux.Router

func (r *router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	(*mux.Router)(r).ServeHTTP(w, req)
}

func (r *router) Register(method string, pathSegments []wrouter.PathSegment, handler http.Handler) {
	(*mux.Router)(r).Path(r.convertPathParams(pathSegments)).Methods(method).Handler(handler)
}

func (r *router) PathParams(req *http.Request, pathVarNames []string) map[string]string {
	vars := mux.Vars(req)
	if len(vars) == 0 {
		return nil
	}
	params := make(map[string]string)
	for _, currVarName := range pathVarNames {
		params[currVarName] = vars[currVarName]
	}
	return params
}

func (r *router) convertPathParams(path []wrouter.PathSegment) string {
	pathParts := make([]string, len(path))
	for i, segment := range path {
		switch segment.Type {
		case wrouter.TrailingPathParamSegment:
			pathParts[i] = fmt.Sprintf("{%s:.+}", segment.Value)
		case wrouter.PathParamSegment:
			pathParts[i] = fmt.Sprintf("{%s}", segment.Value)
		default:
			pathParts[i] = segment.Value
		}
	}
	return "/" + strings.Join(pathParts, "/")
}
