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
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-health/v2/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/v2/sources"
	healthstatus "github.com/palantir/witchcraft-go-health/v2/status"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

type validatingRefreshableHealthCheckSource struct {
	healthstatus.HealthCheckSource

	healthCheckType health.CheckType
	validations     []func() error
}

func (v *validatingRefreshableHealthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	var errParams []map[string]any
	for _, validation := range v.validations {
		if err := validation(); err != nil {
			svc1log.FromContext(ctx).Error("Encountered config reload error.", svc1log.Stacktrace(err))
			errParams = append(errParams, map[string]any{
				"error":  err.Error(),
				"params": werror.Convert(err).(werror.Werror).SafeParams(),
			})
		}
	}
	switch len(errParams) {
	case 0:
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				v.healthCheckType: sources.HealthyHealthCheckResult(v.healthCheckType),
			},
		}
	case 1:
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				v.healthCheckType: sources.UnhealthyHealthCheckResult(v.healthCheckType, "Config reload error. See service logs for more information.", errParams[0]),
			},
		}
	default:
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				v.healthCheckType: sources.UnhealthyHealthCheckResult(v.healthCheckType, "Multiple config reload errors. See service logs for more information.", map[string]any{"errors": errParams}),
			},
		}
	}
}

func ValidationErrFunc[T any](v refreshable.Validated[T]) func() error {
	return func() error {
		_, err := v.Validation()
		return err
	}
}

// NewValidatingRefreshableHealthCheckSource returns a status.HealthCheckSource that returns an Error health check whenever
// the provided ValidatingRefreshable is failing its validation.
func NewValidatingRefreshableHealthCheckSource(healthCheckType health.CheckType, validations ...func() error) healthstatus.HealthCheckSource {
	return &validatingRefreshableHealthCheckSource{
		healthCheckType: healthCheckType,
		validations:     validations,
	}
}
