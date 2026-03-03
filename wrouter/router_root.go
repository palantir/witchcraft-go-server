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
	routerwrouter "github.com/palantir/witchcraft-go-router/wrouter"
)

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RootRouter instead.
type RootRouter = routerwrouter.RootRouter

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RootRouterParam instead.
type RootRouterParam = routerwrouter.RootRouterParam

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.New instead.
func New(impl RouterImpl, params ...RootRouterParam) RootRouter {
	return routerwrouter.New(impl, params...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RootRouterParamAddRequestHandlerMiddleware instead.
func RootRouterParamAddRequestHandlerMiddleware(reqHandler ...RequestHandlerMiddleware) RootRouterParam {
	return routerwrouter.RootRouterParamAddRequestHandlerMiddleware(reqHandler...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RootRouterParamAddRouteHandlerMiddleware instead.
func RootRouterParamAddRouteHandlerMiddleware(routeReqHandler ...RouteHandlerMiddleware) RootRouterParam {
	return routerwrouter.RootRouterParamAddRouteHandlerMiddleware(routeReqHandler...)
}
