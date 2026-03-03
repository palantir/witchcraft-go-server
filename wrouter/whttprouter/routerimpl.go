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

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/whttprouter instead.
package whttprouter

import (
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
	routerwhttprouter "github.com/palantir/witchcraft-go-router/wrouter/whttprouter"
)

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/whttprouter.Param instead.
type Param = routerwhttprouter.Param

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/whttprouter.New instead.
func New(params ...Param) wrouter.RouterImpl {
	return routerwhttprouter.New(params...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/whttprouter.RedirectTrailingSlash instead.
func RedirectTrailingSlash(redirect bool) Param {
	return routerwhttprouter.RedirectTrailingSlash(redirect)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/whttprouter.RedirectFixedPath instead.
func RedirectFixedPath(redirect bool) Param {
	return routerwhttprouter.RedirectFixedPath(redirect)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/whttprouter.HandleMethodNotAllowed instead.
func HandleMethodNotAllowed(notAllowed bool) Param {
	return routerwhttprouter.HandleMethodNotAllowed(notAllowed)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/whttprouter.HandleOPTIONS instead.
func HandleOPTIONS(handle bool) Param {
	return routerwhttprouter.HandleOPTIONS(handle)
}
