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

// Deprecated: Use github.com/palantir/witchcraft-go-router/wresource instead.
package wresource

import (
	routerwresource "github.com/palantir/witchcraft-go-router/wresource"
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
)

// Deprecated: Use github.com/palantir/witchcraft-go-router/wresource.Resource instead.
type Resource = routerwresource.Resource

const (
	// Deprecated: Use github.com/palantir/witchcraft-go-router/wresource.ResourceTagName instead.
	ResourceTagName = routerwresource.ResourceTagName
	// Deprecated: Use github.com/palantir/witchcraft-go-router/wresource.MethodTagName instead.
	MethodTagName = routerwresource.MethodTagName
	// Deprecated: Use github.com/palantir/witchcraft-go-router/wresource.EndpointTagName instead.
	EndpointTagName = routerwresource.EndpointTagName
)

// Deprecated: Use github.com/palantir/witchcraft-go-router/wresource.New instead.
func New(resourceName string, router wrouter.Router) Resource {
	return routerwresource.New(resourceName, router)
}
