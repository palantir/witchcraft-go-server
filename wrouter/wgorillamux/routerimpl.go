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

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/wgorillamux instead.
package wgorillamux

import (
	"net/http"

	"github.com/palantir/witchcraft-go-server/v3/wrouter"
	routerwgorillamux "github.com/palantir/witchcraft-go-router/wrouter/wgorillamux"
)

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/wgorillamux.Param instead.
type Param = routerwgorillamux.Param

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/wgorillamux.New instead.
func New(params ...Param) wrouter.RouterImpl {
	return routerwgorillamux.New(params...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/wgorillamux.NotFoundHandler instead.
func NotFoundHandler(h http.Handler) Param {
	return routerwgorillamux.NotFoundHandler(h)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/wgorillamux.StrictSlash instead.
func StrictSlash(value bool) Param {
	return routerwgorillamux.StrictSlash(value)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/wgorillamux.SkipClean instead.
func SkipClean(value bool) Param {
	return routerwgorillamux.SkipClean(value)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter/wgorillamux.UseEncodedPath instead.
func UseEncodedPath() Param {
	return routerwgorillamux.UseEncodedPath()
}
