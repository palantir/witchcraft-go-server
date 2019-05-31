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

package periodic

import (
	"context"
	"testing"
	"time"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/stretchr/testify/assert"
)

const (
	checkType      = "TEST_CHECK"
	otherCheckType = "OTHER_TEST_CHECK"
)

func TestHealthCheckSource_HealthStatus(t *testing.T) {
	for _, test := range []struct {
		Name     string
		State    *healthCheckSource
		Expected health.HealthStatus
	}{
		{
			Name: "Last result successful",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType: nil,
					},
				},
				gracePeriod: time.Minute,
				checkStates: map[health.CheckType]*checkState{
					checkType: {
						lastResult: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastResultTime: time.Now(),
						lastSuccess: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastSuccessTime: time.Now(),
					},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:  checkType,
						State: health.HealthStateHealthy,
					},
				},
			},
		},
		{
			Name: "Last success within grace period",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType: nil,
					},
				},
				gracePeriod: time.Hour,
				checkStates: map[health.CheckType]*checkState{
					checkType: {
						lastResult: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateError,
						},
						lastResultTime: time.Now(),
						lastSuccess: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastSuccessTime: time.Now().Add(-5 * time.Minute),
					},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:  checkType,
						State: health.HealthStateHealthy,
					},
				},
			},
		},
		{
			Name: "Last success outside grace period",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType: nil,
					},
				},
				gracePeriod: time.Minute,
				checkStates: map[health.CheckType]*checkState{
					checkType: {
						lastResult: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateError,
						},
						lastResultTime: time.Now(),
						lastSuccess: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastSuccessTime: time.Now().Add(-5 * time.Minute),
					},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:    checkType,
						State:   health.HealthStateError,
						Message: stringPtr("No successful checks during 1m0s grace period"),
					},
				},
			},
		},
		{
			Name: "No runs within grace period, last was success",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType: nil,
					},
				},
				gracePeriod: time.Minute,
				checkStates: map[health.CheckType]*checkState{
					checkType: {
						lastResult: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastResultTime: time.Now().Add(-5 * time.Minute),
						lastSuccess: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastSuccessTime: time.Now().Add(-5 * time.Minute),
					},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:    checkType,
						State:   health.HealthStateRepairing,
						Message: stringPtr("No completed checks during 1m0s grace period"),
					},
				},
			},
		},
		{
			Name: "No runs within grace period, last was error",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType: nil,
					},
				},
				gracePeriod: time.Minute,
				checkStates: map[health.CheckType]*checkState{
					checkType: {
						lastResult: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateError,
						},
						lastResultTime: time.Now().Add(-3 * time.Minute),
						lastSuccess: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastSuccessTime: time.Now().Add(-5 * time.Minute),
					},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:    checkType,
						State:   health.HealthStateError,
						Message: stringPtr("No completed checks during 1m0s grace period"),
					},
				},
			},
		},
		{
			Name: "Never started",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType: nil,
					},
				},
				gracePeriod: time.Minute,
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:    checkType,
						State:   health.HealthStateRepairing,
						Message: stringPtr("Check has not yet run"),
					},
				},
			},
		},
		{
			Name: "Two checks, one last result successful, one last success outside grace period",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType:      nil,
						otherCheckType: nil,
					},
				},
				gracePeriod: time.Minute,
				checkStates: map[health.CheckType]*checkState{
					checkType: {
						lastResult: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastResultTime: time.Now(),
						lastSuccess: &health.HealthCheckResult{
							Type:  checkType,
							State: health.HealthStateHealthy,
						},
						lastSuccessTime: time.Now(),
					},
					otherCheckType: {
						lastResult: &health.HealthCheckResult{
							Type:  otherCheckType,
							State: health.HealthStateError,
						},
						lastResultTime: time.Now(),
						lastSuccess: &health.HealthCheckResult{
							Type:  otherCheckType,
							State: health.HealthStateHealthy,
						},
						lastSuccessTime: time.Now().Add(-5 * time.Minute),
					},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:  checkType,
						State: health.HealthStateHealthy,
					},
					otherCheckType: {
						Type:    otherCheckType,
						State:   health.HealthStateError,
						Message: stringPtr("No successful checks during 1m0s grace period"),
					},
				},
			},
		},
		{
			Name: "Two checks, neither started",
			State: &healthCheckSource{
				source: Source{
					Checks: map[health.CheckType]CheckFunc{
						checkType:      nil,
						otherCheckType: nil,
					},
				},
				gracePeriod: time.Minute,
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					checkType: {
						Type:    checkType,
						State:   health.HealthStateRepairing,
						Message: stringPtr("Check has not yet run"),
					},
					otherCheckType: {
						Type:    otherCheckType,
						State:   health.HealthStateRepairing,
						Message: stringPtr("Check has not yet run"),
					},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			result := test.State.HealthStatus(context.Background())
			assert.Equal(t, test.Expected, result)
		})
	}
}
