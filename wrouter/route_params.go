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
	"github.com/palantir/pkg/metrics"
	routerwrouter "github.com/palantir/witchcraft-go-router/wrouter"
)

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RouteParam instead.
type RouteParam = routerwrouter.RouteParam

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RouteParamPermsParam instead.
func RouteParamPermsParam(perms RouteParamPerms) RouteParam {
	return routerwrouter.RouteParamPermsParam(perms)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.SafePathParams instead.
func SafePathParams(safeParams ...string) RouteParam {
	return routerwrouter.SafePathParams(safeParams...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.ForbiddenPathParams instead.
func ForbiddenPathParams(forbiddenParams ...string) RouteParam {
	return routerwrouter.ForbiddenPathParams(forbiddenParams...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.SafeQueryParams instead.
func SafeQueryParams(safeParams ...string) RouteParam {
	return routerwrouter.SafeQueryParams(safeParams...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.ForbiddenQueryParams instead.
func ForbiddenQueryParams(forbiddenParams ...string) RouteParam {
	return routerwrouter.ForbiddenQueryParams(forbiddenParams...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.SafeHeaderParams instead.
func SafeHeaderParams(safeParams ...string) RouteParam {
	return routerwrouter.SafeHeaderParams(safeParams...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.ForbiddenHeaderParams instead.
func ForbiddenHeaderParams(forbiddenParams ...string) RouteParam {
	return routerwrouter.ForbiddenHeaderParams(forbiddenParams...)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.MetricTags instead.
func MetricTags(tags metrics.Tags) RouteParam {
	return routerwrouter.MetricTags(tags)
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.DisableTelemetry instead.
func DisableTelemetry() RouteParam {
	return routerwrouter.DisableTelemetry()
}

// Deprecated: Use github.com/palantir/witchcraft-go-router/wrouter.RouteMiddleware instead.
func RouteMiddleware(middleware RouteHandlerMiddleware) RouteParam {
	return routerwrouter.RouteMiddleware(middleware)
}
