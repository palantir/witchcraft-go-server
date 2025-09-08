// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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

package refreshable

import (
	"context"

	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/sources"
	healthstatus "github.com/palantir/witchcraft-go-health/status"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

type validatingRefreshableHealthCheckSource[T any] struct {
	healthstatus.HealthCheckSource

	healthCheckType health.CheckType
	refreshables    []refreshable.Validated[T]
}

func (v *validatingRefreshableHealthCheckSource[T]) HealthStatus(ctx context.Context) health.HealthStatus {
	healthCheckResult := sources.HealthyHealthCheckResult(v.healthCheckType)

	for _, refreshable := range v.refreshables {
		if _, err := refreshable.Validation(); err != nil {
			svc1log.FromContext(ctx).Error("Refreshable validation failed", svc1log.Stacktrace(err))
			healthCheckResult = sources.UnhealthyHealthCheckResult(v.healthCheckType,
				"Refreshable validation failed, please look at service logs for more information.",
				map[string]interface{}{},
			)
		}
	}

	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			v.healthCheckType: healthCheckResult,
		},
	}
}

// NewValidatingRefreshableHealthCheckSource returns a status.HealthCheckSource that returns an Error health check whenever
// the provided ValidatingRefreshable is failing its validation.
func NewValidatingRefreshableHealthCheckSource[T any](healthCheckType health.CheckType, refreshables ...refreshable.Validated[T]) healthstatus.HealthCheckSource {
	return &validatingRefreshableHealthCheckSource[T]{
		healthCheckType: healthCheckType,
		refreshables:    refreshables,
	}
}
