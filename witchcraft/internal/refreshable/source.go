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

	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/sources"
	healthstatus "github.com/palantir/witchcraft-go-health/status"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/refreshable"
)

type validatingRefreshableHealthCheckSource struct {
	healthstatus.HealthCheckSource

	healthCheckType health.CheckType
	refreshable     refreshable.ValidatingRefreshable
}

func (v *validatingRefreshableHealthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	healthCheckResult := sources.HealthyHealthCheckResult(v.healthCheckType)

	if err := v.refreshable.LastValidateErr(); err != nil {
		svc1log.FromContext(ctx).Error("Refreshable validation failed", svc1log.Stacktrace(err))
		healthCheckResult = sources.UnhealthyHealthCheckResult(v.healthCheckType,
			"Refreshable validation failed, please look at service logs for more information.")
	}

	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			v.healthCheckType: healthCheckResult,
		},
	}
}

// NewValidatingRefreshableHealthCheckSource returns a status.HealthCheckSource that returns an Error health check whenever
// the provided ValidatingRefreshable is failing its validation.
func NewValidatingRefreshableHealthCheckSource(healthCheckType health.CheckType, refreshable refreshable.ValidatingRefreshable) healthstatus.HealthCheckSource {
	return &validatingRefreshableHealthCheckSource{
		healthCheckType: healthCheckType,
		refreshable:     refreshable,
	}
}
