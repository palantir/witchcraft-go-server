// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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

package internal

import (
	"github.com/palantir/pkg/refreshable"
)

type RefreshableURIScoringMiddleware interface {
	CurrentURIScoringMiddleware() URIScoringMiddleware
}

func NewRefreshableURIScoringMiddleware(uris refreshable.StringSlice, constructor func([]string) URIScoringMiddleware) RefreshableURIScoringMiddleware {
	return refreshableURIScoringMiddleware{uris.MapStringSlice(func(uris []string) interface{} {
		return constructor(uris)
	})}
}

type refreshableURIScoringMiddleware struct{ refreshable.Refreshable }

func (r refreshableURIScoringMiddleware) CurrentURIScoringMiddleware() URIScoringMiddleware {
	return r.Current().(URIScoringMiddleware)
}
