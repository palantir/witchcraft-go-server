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

package window

import (
	"context"
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	whealth "github.com/palantir/witchcraft-go-server/status/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCheckType health.CheckType = "TEST_CHECK"
	windowSize                     = 100 * time.Millisecond
)

func TestUnhealthyIfAtLeastOneErrorSource(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		errors        []error
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no items",
			errors:        nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when there are only nil items",
			errors: []error{
				nil,
				nil,
				nil,
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when there is at least one err",
			errors: []error{
				nil,
				werror.ErrorWithContextParams(context.Background(), "Error #1"),
				nil,
				werror.ErrorWithContextParams(context.Background(), "Error #2"),
				nil,
			},
			expectedCheck: whealth.UnhealthyHealthCheckResult(testCheckType, "Error #2"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source, err := NewUnhealthyIfAtLeastOneErrorSource(testCheckType, time.Hour)
			require.NoError(t, err)

			for _, err := range testCase.errors {
				source.Submit(err)
			}
			actualStatus := source.HealthStatus(context.Background())
			expectedStatus := health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					testCheckType: testCase.expectedCheck,
				},
			}
			assert.Equal(t, expectedStatus, actualStatus)
		})
	}
}

func TestHealthyIfNotAllErrorsSource(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		errors        []error
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no items",
			errors:        nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when there are only nil items",
			errors: []error{
				nil,
				nil,
				nil,
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when there is at least one non nil err",
			errors: []error{
				nil,
				werror.ErrorWithContextParams(context.Background(), "Error #1"),
				nil,
				werror.ErrorWithContextParams(context.Background(), "Error #2"),
				nil,
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when there are only non nil items",
			errors: []error{
				werror.ErrorWithContextParams(context.Background(), "Error #1"),
				werror.ErrorWithContextParams(context.Background(), "Error #2"),
			},
			expectedCheck: whealth.UnhealthyHealthCheckResult(testCheckType, "Error #2"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			timeProvider := &offsetTimeProvider{}
			source, err := newHealthyIfNotAllErrorsSource(testCheckType, time.Hour, 0, false, timeProvider)

			require.NoError(t, err)
			for _, err := range testCase.errors {
				source.Submit(err)
			}
			actualStatus := source.HealthStatus(context.Background())
			expectedStatus := health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					testCheckType: testCase.expectedCheck,
				},
			}
			assert.Equal(t, expectedStatus, actualStatus)
		})
	}
}

// TestHealthyIfNotAllErrorsSource_ErrorInInitialWindowWhenFirstFullWindowRequired validates that error in the first window
// causes the check to report as repairing when first window is required.
func TestHealthyIfNotAllErrorsSource_ErrorInInitialWindowWhenFirstFullWindowRequired(t *testing.T) {
	timeProvider := &offsetTimeProvider{}
	anchoredWindow, err := newHealthyIfNotAllErrorsSource(testCheckType, windowSize, 0, true, timeProvider)
	assert.NoError(t, err)

	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))
	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateRepairing, checkResult.State)
}

// TestAnchoredHealthyIfNotAllErrorsSource_ErrorInInitialAnchoredWindow validates that error in the first window
// does not cause the health status to become unhealthy when anchored as well.
func TestAnchoredHealthyIfNotAllErrorsSource_ErrorInInitialAnchoredWindow(t *testing.T) {
	timeProvider := &offsetTimeProvider{}
	anchoredWindow, err := newHealthyIfNotAllErrorsSource(testCheckType, windowSize, windowSize, false, timeProvider)
	assert.NoError(t, err)

	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))
	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateRepairing, checkResult.State)
}

// TestAnchoredHealthyIfNotAllErrorsSource_GapThenRepairing validates that error in the first window
// does not cause the health status to become unhealthy when anchored as well.
func TestAnchoredHealthyIfNotAllErrorsSource_GapThenRepairing(t *testing.T) {
	timeProvider := &offsetTimeProvider{}
	anchoredWindow, err := newHealthyIfNotAllErrorsSource(testCheckType, windowSize, windowSize, true, timeProvider)
	assert.NoError(t, err)

	timeProvider.RestlessSleep(2 * windowSize)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))
	timeProvider.RestlessSleep(windowSize / 2)

	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateRepairing, checkResult.State)
}

// TestAnchoredHealthyIfNotAllErrorsSource_GapThenRepairingThenError validates that in a constant stream of errors, the health
// check initially reports repairing and then reports error after the time window.
func TestAnchoredHealthyIfNotAllErrorsSource_GapThenRepairingThenError(t *testing.T) {
	timeProvider := &offsetTimeProvider{}
	anchoredWindow, err := newHealthyIfNotAllErrorsSource(testCheckType, windowSize, windowSize, true, timeProvider)
	assert.NoError(t, err)

	timeProvider.RestlessSleep(2 * windowSize)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))
	timeProvider.RestlessSleep(windowSize / 2)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))

	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateRepairing, checkResult.State)

	timeProvider.RestlessSleep(windowSize / 2)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))

	healthStatus = anchoredWindow.HealthStatus(context.Background())
	checkResult, ok = healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateError, checkResult.State)
}

// TestAnchoredHealthyIfNotAllErrorsSource_GapThenRepairingThenHealthy validates that if a success is submitted during repairing phase,
// the health check recovers.
func TestAnchoredHealthyIfNotAllErrorsSource_GapThenRepairingThenHealthy(t *testing.T) {
	timeProvider := &offsetTimeProvider{}
	anchoredWindow, err := newHealthyIfNotAllErrorsSource(testCheckType, windowSize, windowSize, true, timeProvider)
	assert.NoError(t, err)

	timeProvider.RestlessSleep(2 * windowSize)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))
	timeProvider.RestlessSleep(windowSize / 2)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))

	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateRepairing, checkResult.State)

	timeProvider.RestlessSleep(windowSize / 2)
	anchoredWindow.Submit(nil)

	healthStatus = anchoredWindow.HealthStatus(context.Background())
	checkResult, ok = healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateHealthy, checkResult.State)
}

// TestAnchoredHealthyIfNotAllErrorsSource_RepairingThenGap validates if no more errors happen beyond the repairing phase,
// the health check recovers.
func TestAnchoredHealthyIfNotAllErrorsSource_RepairingThenGap(t *testing.T) {
	timeProvider := &offsetTimeProvider{}
	anchoredWindow, err := newHealthyIfNotAllErrorsSource(testCheckType, windowSize, windowSize, true, timeProvider)
	assert.NoError(t, err)

	timeProvider.RestlessSleep(2 * windowSize)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))
	timeProvider.RestlessSleep(windowSize / 2)
	anchoredWindow.Submit(werror.ErrorWithContextParams(context.Background(), "an error"))

	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateRepairing, checkResult.State)

	timeProvider.RestlessSleep(3 * windowSize / 2)

	healthStatus = anchoredWindow.HealthStatus(context.Background())
	checkResult, ok = healthStatus.Checks[testCheckType]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateHealthy, checkResult.State)
}
