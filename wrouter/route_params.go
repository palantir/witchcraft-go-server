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

package wrouter

import (
	"github.com/palantir/pkg/metrics"
)

type routeParamBuilder struct {
	middleware       []RouteHandlerMiddleware
	paramPerms       RouteParamPerms
	metricTags       metrics.Tags
	disableTelemetry bool
}

func (b *routeParamBuilder) toRequestParamPerms() RouteParamPerms {
	if b.paramPerms != nil {
		return b.paramPerms
	}
	return &requestParamPermsImpl{
		pathParamPerms:   newParamPerms(nil, nil),
		queryParamPerms:  newParamPerms(nil, nil),
		headerParamPerms: newParamPerms(nil, nil),
	}
}

func (b *routeParamBuilder) toMetricTags() metrics.Tags {
	var tags metrics.Tags
	if b.metricTags != nil {
		tags = append(tags, b.metricTags...)
	}
	return tags
}

type RouteParam interface {
	apply(*routeParamBuilder) error
}

type routeParamFunc func(*routeParamBuilder) error

func (f routeParamFunc) apply(b *routeParamBuilder) error {
	return f(b)
}

func RouteParamPermsParam(perms RouteParamPerms) RouteParam {
	return routeParamFunc(func(b *routeParamBuilder) error {
		var pathParamPerms []ParamPerms
		var queryParamPerms []ParamPerms
		var headerParamPerms []ParamPerms
		if b.paramPerms != nil {
			pathParamPerms = append(pathParamPerms, b.paramPerms.PathParamPerms())
			queryParamPerms = append(queryParamPerms, b.paramPerms.QueryParamPerms())
			headerParamPerms = append(headerParamPerms, b.paramPerms.HeaderParamPerms())
		}

		pathParamPerms = append(pathParamPerms, perms.PathParamPerms())
		queryParamPerms = append(queryParamPerms, perms.QueryParamPerms())
		headerParamPerms = append(headerParamPerms, perms.HeaderParamPerms())

		b.paramPerms = &requestParamPermsImpl{
			pathParamPerms:   NewCombinedParamPerms(pathParamPerms...),
			queryParamPerms:  NewCombinedParamPerms(queryParamPerms...),
			headerParamPerms: NewCombinedParamPerms(headerParamPerms...),
		}
		return nil
	})
}

func SafePathParams(safeParams ...string) RouteParam {
	return RouteParamPermsParam(&requestParamPermsImpl{
		pathParamPerms: newSafeParamPerms(safeParams...),
	})
}

func ForbiddenPathParams(forbiddenParams ...string) RouteParam {
	return RouteParamPermsParam(&requestParamPermsImpl{
		pathParamPerms: newForbiddenParamPerms(forbiddenParams...),
	})
}

func SafeQueryParams(safeParams ...string) RouteParam {
	return RouteParamPermsParam(&requestParamPermsImpl{
		queryParamPerms: newSafeParamPerms(safeParams...),
	})
}

func ForbiddenQueryParams(forbiddenParams ...string) RouteParam {
	return RouteParamPermsParam(&requestParamPermsImpl{
		queryParamPerms: newForbiddenParamPerms(forbiddenParams...),
	})
}

func SafeHeaderParams(safeParams ...string) RouteParam {
	return RouteParamPermsParam(&requestParamPermsImpl{
		headerParamPerms: newSafeParamPerms(safeParams...),
	})
}

func ForbiddenHeaderParams(forbiddenParams ...string) RouteParam {
	return RouteParamPermsParam(&requestParamPermsImpl{
		headerParamPerms: newForbiddenParamPerms(forbiddenParams...),
	})
}

func MetricTags(tags metrics.Tags) RouteParam {
	return routeParamFunc(func(b *routeParamBuilder) error {
		b.metricTags = tags
		return nil
	})
}

func DisableTelemetry() RouteParam {
	return routeParamFunc(func(b *routeParamBuilder) error {
		b.disableTelemetry = true
		return nil
	})
}

// RouteMiddleware configures the provided middleware to run on requests matching this specific route.
func RouteMiddleware(middleware RouteHandlerMiddleware) RouteParam {
	return routeParamFunc(func(b *routeParamBuilder) error {
		if middleware != nil {
			b.middleware = append(b.middleware, middleware)
		}
		return nil
	})
}
