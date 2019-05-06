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

type RouteParam interface {
	perms() RouteParamPerms
	metricTags() metrics.Tags
}

type requestParamPermsFunc func() RouteParamPerms

func (f requestParamPermsFunc) perms() RouteParamPerms {
	return f()
}

func (f requestParamPermsFunc) metricTags() metrics.Tags {
	return nil
}

func RouteParamPermsParam(perms RouteParamPerms) RouteParam {
	return requestParamPermsFunc(func() RouteParamPerms {
		return perms
	})
}

func SafePathParams(safeParams ...string) RouteParam {
	return requestParamPermsFunc(func() RouteParamPerms {
		return &requestParamPermsImpl{
			pathParamPerms: newSafeParamPerms(safeParams...),
		}
	})
}

func ForbiddenPathParams(forbiddenParams ...string) RouteParam {
	return requestParamPermsFunc(func() RouteParamPerms {
		return &requestParamPermsImpl{
			pathParamPerms: newForbiddenParamPerms(forbiddenParams...),
		}
	})
}

func SafeQueryParams(safeParams ...string) RouteParam {
	return requestParamPermsFunc(func() RouteParamPerms {
		return &requestParamPermsImpl{
			queryParamPerms: newSafeParamPerms(safeParams...),
		}
	})
}

func ForbiddenQueryParams(forbiddenParams ...string) RouteParam {
	return requestParamPermsFunc(func() RouteParamPerms {
		return &requestParamPermsImpl{
			queryParamPerms: newForbiddenParamPerms(forbiddenParams...),
		}
	})
}

func SafeHeaderParams(safeParams ...string) RouteParam {
	return requestParamPermsFunc(func() RouteParamPerms {
		return &requestParamPermsImpl{
			headerParamPerms: newSafeParamPerms(safeParams...),
		}
	})
}

func ForbiddenHeaderParams(forbiddenParams ...string) RouteParam {
	return requestParamPermsFunc(func() RouteParamPerms {
		return &requestParamPermsImpl{
			headerParamPerms: newForbiddenParamPerms(forbiddenParams...),
		}
	})
}

func toRequestParamPerms(params []RouteParam) RouteParamPerms {
	var pathParamPerms []ParamPerms
	var queryParamPerms []ParamPerms
	var headerParamPerms []ParamPerms

	for _, p := range params {
		if p == nil {
			continue
		}

		perms := p.perms()
		if perms != nil {
			pathParamPerms = append(pathParamPerms, perms.PathParamPerms())
			queryParamPerms = append(queryParamPerms, perms.QueryParamPerms())
			headerParamPerms = append(headerParamPerms, perms.HeaderParamPerms())
		}
	}

	return &requestParamPermsImpl{
		pathParamPerms:   NewCombinedParamPerms(pathParamPerms...),
		queryParamPerms:  NewCombinedParamPerms(queryParamPerms...),
		headerParamPerms: NewCombinedParamPerms(headerParamPerms...),
	}
}

type requestMetricTagsFunc func() metrics.Tags

func (f requestMetricTagsFunc) perms() RouteParamPerms {
	return nil
}

func (f requestMetricTagsFunc) metricTags() metrics.Tags {
	return f()
}

func MetricTags(tags metrics.Tags) RouteParam {
	return requestMetricTagsFunc(func() metrics.Tags {
		return tags
	})
}

func toMetricTags(params []RouteParam) metrics.Tags {
	var tags metrics.Tags
	for _, param := range params {
		tags = append(tags, param.metricTags()...)
	}
	return tags
}
