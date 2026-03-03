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

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.PathTemplate instead.
type PathTemplate = routerwrouter.PathTemplate

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.NewPathTemplate instead.
func NewPathTemplate(in string) (PathTemplate, error) {
	return routerwrouter.NewPathTemplate(in)
}
