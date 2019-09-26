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
			Name: "No runs within grace period, last was error, with a message",
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
							Type:    checkType,
							State:   health.HealthStateError,
							Message: stringPtr("something went wrong"),
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
						Message: stringPtr("No completed checks during 1m0s grace period: something went wrong"),
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

// Test does the following:
//   * Starts health check source with 10ms retry interval and 100ms grace period
//   * First health check returns healthy (t=10ms, counter=0)
//   * Second health check returns unhealthy (t=20ms, counter=1)
//   * When third health check is run, pause the health check routine and signal that health should be checked (t=30ms)
//   * TEST: health status should be healthy, counter=0 (checking at roughly t=30ms, so there was a healthy check within grace period)
//   * Wait until grace period has elapsed since healthy check was returned (roughly t=130ms)
//   * Third check returns unhealthy (t=130ms, counter=2)
//   * TEST: health status should be unhealthy due to no success within grace period (checking at roughly t=130ms, so there was a check that occurred within the grace period, but no successful check within the grace period)
//   * Wait until grace period has elapsed (t=230ms)
//   * TEST: health status should be unhealthy due to no check within grace period (checking at roughly t=230ms, so there is no check that occurred within the grace period)
func TestFromHealthCheckSource(t *testing.T) {
	// health check sends on this channel on its third run (after it has returned healthy and then error)
	doneChan := make(chan struct{})
	defer close(doneChan)

	// health check waits on this channel on its third run (after it has sent on doneChan)
	pauseChan := make(chan struct{})
	defer close(pauseChan)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	gracePeriod := 100 * time.Millisecond
	retryInterval := 10 * time.Millisecond
	counter := 0

	source := FromHealthCheckSource(ctx, gracePeriod, retryInterval, Source{
		Checks: map[health.CheckType]CheckFunc{
			checkType: func(ctx context.Context) (rVal *health.HealthCheckResult) {
				defer func() {
					counter++
				}()

				switch counter {
				// return healthy state on first run
				case 0:
					return &health.HealthCheckResult{
						Type:    checkType,
						State:   health.HealthStateHealthy,
						Message: stringPtr("Healthy state"),
						Params: map[string]interface{}{
							"counter": counter,
						},
					}
				// return error state on second run
				case 1:
					return &health.HealthCheckResult{
						Type:    checkType,
						State:   health.HealthStateError,
						Message: stringPtr("Error state"),
						Params: map[string]interface{}{
							"counter": counter,
						},
					}
				// on third run, send on doneChan and read from pauseChan
				case 2:
					// signal that health can be checked
					doneChan <- struct{}{}
					// pause until health check has occurred
					<-pauseChan
					// return unhealthy
					return &health.HealthCheckResult{
						Type:    checkType,
						State:   health.HealthStateError,
						Message: stringPtr("Error state"),
						Params: map[string]interface{}{
							"counter": counter,
						},
					}
				case 3:
					// signal that health can be checked
					doneChan <- struct{}{}
					// pause (do not return until test has completed)
					<-pauseChan
				}
				return nil
			},
		},
	})

	// wait until health check has returned healthy and then unhealthy
	<-doneChan
	status := source.HealthStatus(ctx)

	// health check should be healthy: even though health source returned error state most recently, it returned
	// healthy state within the grace period
	assert.Equal(t, map[health.CheckType]health.HealthCheckResult{
		checkType: {
			Type:    checkType,
			State:   health.HealthStateHealthy,
			Message: stringPtr("Healthy state"),
			Params: map[string]interface{}{
				"counter": 0,
			},
		},
	}, status.Checks)

	// health has been checked: wait for grace period to pass and then unpause the health routine
	time.Sleep(gracePeriod)
	pauseChan <- struct{}{}
	<-doneChan

	// health check should be unhealthy: the most recent check ran within the grace period, but the last success was before grace period
	status = source.HealthStatus(ctx)
	assert.Equal(t, map[health.CheckType]health.HealthCheckResult{
		checkType: {
			Type:    checkType,
			State:   health.HealthStateError,
			Message: stringPtr("No successful checks during 100ms grace period: Error state"),
			Params: map[string]interface{}{
				"counter": 2,
			},
		},
	}, status.Checks)

	// wait for grace period
	time.Sleep(gracePeriod)

	// health check should be unhealthy: no check ran within grace period, and last known status was unhealthy
	status = source.HealthStatus(ctx)
	assert.Equal(t, map[health.CheckType]health.HealthCheckResult{
		checkType: {
			Type:    checkType,
			State:   health.HealthStateError,
			Message: stringPtr("No completed checks during 100ms grace period: Error state"),
			Params: map[string]interface{}{
				"counter": 2,
			},
		},
	}, status.Checks)
}
