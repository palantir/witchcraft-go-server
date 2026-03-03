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

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter instead.
package wrouter

import (
	"net/http"

	routerwrouter "github.com/palantir/witchcraft-go-router/wrouter"
)

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.Router instead.
type Router = routerwrouter.Router

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RequestHandlerMiddleware instead.
type RequestHandlerMiddleware = routerwrouter.RequestHandlerMiddleware

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RouteHandlerMiddleware instead.
type RouteHandlerMiddleware = routerwrouter.RouteHandlerMiddleware

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RouteRequestHandler instead.
type RouteRequestHandler = routerwrouter.RouteRequestHandler

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RouteSpec instead.
type RouteSpec = routerwrouter.RouteSpec

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RequestVals instead.
type RequestVals = routerwrouter.RequestVals

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.ResponseVals instead.
type ResponseVals = routerwrouter.ResponseVals

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.PathParams instead.
func PathParams(r *http.Request) map[string]string {
	return routerwrouter.PathParams(r)
}
