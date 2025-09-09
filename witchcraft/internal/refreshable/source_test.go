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
	"os"
	"testing"

	"github.com/palantir/pkg/refreshable/v2"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/stretchr/testify/require"
)

func TestNewValidatingRefreshableHealthCheckSource_HealthStatus(t *testing.T) {
	ctx := svc1log.WithLogger(context.Background(), svc1log.NewFromCreator(os.Stdout, wlog.DebugLevel, wlog.NewJSONMarshalLoggerProvider().NewLeveledLogger))
	testHealthCheckType := health.CheckType("TEST_HEALTH_CHECK")
	testRefreshable := refreshable.New("initial-value")
	validatingRefreshable, _, err := refreshable.Validate(testRefreshable, func(i string) error {
		if i == "validation-failing-value" {
			return werror.Error("fail validation", werror.SafeParam("key", "value"))
		}
		return nil
	})
	require.NoError(t, err)
	healthCheckSource := NewValidatingRefreshableHealthCheckSource(testHealthCheckType, ValidationErrFunc(validatingRefreshable))

	// check initial state is healthy
	require.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testHealthCheckType: {
				Type:  testHealthCheckType,
				State: health.New_HealthState(health.HealthState_HEALTHY),
			},
		},
	}, healthCheckSource.HealthStatus(ctx))

	// change underyling refreshable to value that fails validation
	testRefreshable.Update("validation-failing-value")
	errorMsg := "Config reload error. See service logs for more information."
	require.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testHealthCheckType: {
				Type:    testHealthCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &errorMsg,
				Params: map[string]interface{}{
					"error":  "fail validation",
					"params": map[string]interface{}{"key": "value"},
				},
			},
		},
	}, healthCheckSource.HealthStatus(ctx))

	// change underyling refreshable to value that passes validation
	testRefreshable.Update("other-value")
	require.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testHealthCheckType: {
				Type:  testHealthCheckType,
				State: health.New_HealthState(health.HealthState_HEALTHY),
			},
		},
	}, healthCheckSource.HealthStatus(ctx))
}
