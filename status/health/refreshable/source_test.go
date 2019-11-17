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
	"reflect"
	"testing"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/witchcraft/refreshable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidatingRefreshableHealthCheckSource_HealthStatus(t *testing.T) {
	testHealthCheckType := health.CheckType("TEST_HEALTH_CHECK")
	testRefreshable := refreshable.NewDefaultRefreshable("initial-value")
	validatingRefreshable, err := refreshable.NewValidatingRefreshable(testRefreshable, func(i interface{}) error {
		if i.(string) == "validation-failing-value" {
			return werror.Error("fail validation")
		}
		return nil
	})
	require.NoError(t, err)
	healthCheckSource := NewValidatingRefreshableHealthCheckSource(testHealthCheckType, *validatingRefreshable)

	// check initial state is healthy
	assert.True(t, reflect.DeepEqual(healthCheckSource.HealthStatus(context.Background()), health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testHealthCheckType: {
				Type:  testHealthCheckType,
				State: health.HealthStateHealthy,
			},
		},
	}))

	// change underyling refreshable to value that fails validation
	err = testRefreshable.Update("validation-failing-value")
	require.NoError(t, err)
	errorMsg := "Refreshable validation failed, please look at service logs for more information."
	assert.True(t, reflect.DeepEqual(healthCheckSource.HealthStatus(context.Background()), health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testHealthCheckType: {
				Type:    testHealthCheckType,
				State:   health.HealthStateError,
				Message: &errorMsg,
			},
		},
	}))

	// change underyling refreshable to value that passes validation
	err = testRefreshable.Update("other-value")
	require.NoError(t, err)
	assert.True(t, reflect.DeepEqual(healthCheckSource.HealthStatus(context.Background()), health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testHealthCheckType: {
				Type:  testHealthCheckType,
				State: health.HealthStateHealthy,
			},
		},
	}))
}
