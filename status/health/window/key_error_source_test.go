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
	"github.com/palantir/witchcraft-go-server/v2/conjure/witchcraft/api/health"
	whealth "github.com/palantir/witchcraft-go-server/v2/status/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type keyErrorPair struct {
	key string
	err error
}

func TestMultiKeyUnhealthyIfAtLeastOneErrorSource(t *testing.T) {
	messageInCaseOfError := "message in case of error"
	for _, testCase := range []struct {
		name          string
		keyErrorPairs []keyErrorPair
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no items",
			keyErrorPairs: nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when all keys are completely healthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1"},
				{key: "2"},
				{key: "3"},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when some keys are partially healthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "1"},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2"},
				{key: "3"},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #1 for key 1",
					"2": "Error #1 for key 2",
				},
			},
		},
		{
			name: "unhealthy when all keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2", err: werror.Error("Error #2 for key 2")},
				{key: "3", err: werror.Error("Error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #1 for key 1",
					"2": "Error #2 for key 2",
					"3": "Error #1 for key 3",
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source, err := NewMultiKeyUnhealthyIfAtLeastOneErrorSource(testCheckType, messageInCaseOfError, time.Hour)
			require.NoError(t, err)
			for _, keyErrorPair := range testCase.keyErrorPairs {
				source.Submit(keyErrorPair.key, keyErrorPair.err)
			}
			expectedStatus := health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					testCheckType: testCase.expectedCheck,
				},
			}
			actualStatus := source.HealthStatus(context.Background())
			assert.Equal(t, expectedStatus, actualStatus)
		})
	}
}

func TestMultiKeyHealthyIfNotAllErrorsSource_OutsideStartWindow(t *testing.T) {
	messageInCaseOfError := "message in case of error"
	for _, testCase := range []struct {
		name          string
		keyErrorPairs []keyErrorPair
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no items",
			keyErrorPairs: nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when all keys are completely healthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1"},
				{key: "2"},
				{key: "3"},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when all keys are partially healthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "1"},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2"},
				{key: "3"},
				{key: "3", err: werror.Error("Error #1 for key 3")},
				{key: "3", err: werror.Error("Error #2 for key 3")},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when some keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2", err: werror.Error("Error #2 for key 2")},
				{key: "3"},
				{key: "3", err: werror.Error("Error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #1 for key 1",
					"2": "Error #2 for key 2",
				},
			},
		},
		{
			name: "unhealthy when all keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2", err: werror.Error("Error #2 for key 2")},
				{key: "3", err: werror.Error("Error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #1 for key 1",
					"2": "Error #2 for key 2",
					"3": "Error #1 for key 3",
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			timeProvider := &offsetTimeProvider{}
			source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, time.Hour, 0, true, timeProvider)
			require.NoError(t, err)

			// sleep puts all tests outside the required healthy start window
			timeProvider.RestlessSleep(time.Hour)

			for _, keyErrorPair := range testCase.keyErrorPairs {
				source.Submit(keyErrorPair.key, keyErrorPair.err)
			}
			expectedStatus := health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					testCheckType: testCase.expectedCheck,
				},
			}
			actualStatus := source.HealthStatus(context.Background())
			assert.Equal(t, expectedStatus, actualStatus)
		})
	}
}

func TestMultiKeyHealthyIfNotAllErrorsSource_InsideStartWindow(t *testing.T) {
	messageInCaseOfError := "message in case of error"
	for _, testCase := range []struct {
		name          string
		keyErrorPairs []keyErrorPair
		expectedCheck health.HealthCheckResult
	}{
		{
			name: "healthy when all keys are partially healthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "1"},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2"},
				{key: "3"},
				{key: "3", err: werror.Error("Error #1 for key 3")},
				{key: "3", err: werror.Error("Error #2 for key 3")},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when some keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2", err: werror.Error("Error #2 for key 2")},
				{key: "3"},
				{key: "3", err: werror.Error("Error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #1 for key 1",
					"2": "Error #2 for key 2",
				},
			},
		},
		{
			name: "healthy when all keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2", err: werror.Error("Error #2 for key 2")},
				{key: "3", err: werror.Error("Error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #1 for key 1",
					"2": "Error #2 for key 2",
					"3": "Error #1 for key 3",
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			timeProvider := &offsetTimeProvider{}
			source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, time.Hour, 0, true, timeProvider)
			require.NoError(t, err)

			for _, keyErrorPair := range testCase.keyErrorPairs {
				source.Submit(keyErrorPair.key, keyErrorPair.err)
			}
			expectedStatus := health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					testCheckType: testCase.expectedCheck,
				},
			}
			actualStatus := source.HealthStatus(context.Background())
			assert.Equal(t, expectedStatus, actualStatus)
		})
	}
}

func TestMultiKeyHealthyIfNotAllErrorsSource_InitialWindowErrorsReturnRepairing(t *testing.T) {
	ctx := context.Background()
	messageInCaseOfError := "message in case of error"
	const timeWindow = time.Minute

	timeProvider := &offsetTimeProvider{}
	source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, timeWindow, 0, true, timeProvider)
	require.NoError(t, err)

	// move partially into the initial health check window
	timeProvider.RestlessSleep(3 * timeWindow / 4)
	source.Submit("1", werror.ErrorWithContextParams(ctx, "error for key: 1"))

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}, source.HealthStatus(ctx))

	// move out of the initial health check window but keep error inside sliding window
	timeProvider.RestlessSleep(timeWindow / 2)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}, source.HealthStatus(ctx))
}

func TestAnchoredMultiKeyHealthyIfNotAllErrorsSource_GapThenRepairingThenHealthy(t *testing.T) {
	ctx := context.Background()
	messageInCaseOfError := "message in case of error"
	const timeWindow = time.Minute

	timeProvider := &offsetTimeProvider{}
	source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, timeWindow, timeWindow, true, timeProvider)
	require.NoError(t, err)

	// move out of the initial health check window
	timeProvider.RestlessSleep(2 * timeWindow)

	source.Submit("1", werror.ErrorWithContextParams(ctx, "error for key: 1"))
	timeProvider.RestlessSleep(timeWindow / 2)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}, source.HealthStatus(ctx))

	source.Submit("2", werror.ErrorWithContextParams(ctx, "error for key: 2"))
	timeProvider.RestlessSleep(timeWindow / 4)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
					"2": "error for key: 2",
				},
			},
		},
	}, source.HealthStatus(ctx))

	source.Submit("1", nil)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"2": "error for key: 2",
				},
			},
		},
	}, source.HealthStatus(ctx))

	source.Submit("2", nil)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: whealth.HealthyHealthCheckResult(testCheckType),
		},
	}, source.HealthStatus(ctx))
}

func TestAnchoredMultiKeyHealthyIfNotAllErrorsSource_GapThenRepairingThenGap(t *testing.T) {
	ctx := context.Background()
	messageInCaseOfError := "message in case of error"
	const timeWindow = time.Minute

	timeProvider := &offsetTimeProvider{}
	source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, timeWindow, timeWindow, true, timeProvider)
	require.NoError(t, err)

	// move out of the initial health check window
	timeProvider.RestlessSleep(2 * timeWindow)

	source.Submit("1", werror.ErrorWithContextParams(ctx, "error for key: 1"))
	timeProvider.RestlessSleep(timeWindow / 2)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}, source.HealthStatus(ctx))

	source.Submit("2", werror.ErrorWithContextParams(ctx, "error for key: 2"))

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
					"2": "error for key: 2",
				},
			},
		},
	}, source.HealthStatus(ctx))

	timeProvider.RestlessSleep(3 * timeWindow / 4)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"2": "error for key: 2",
				},
			},
		},
	}, source.HealthStatus(ctx))

	timeProvider.RestlessSleep(timeWindow / 2)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: whealth.HealthyHealthCheckResult(testCheckType),
		},
	}, source.HealthStatus(ctx))
}

func TestAnchoredMultiKeyHealthyIfNotAllErrorsSource_GapThenRepairingThenError(t *testing.T) {
	ctx := context.Background()
	messageInCaseOfError := "message in case of error"
	const timeWindow = time.Minute

	timeProvider := &offsetTimeProvider{}
	source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, timeWindow, timeWindow, true, timeProvider)
	require.NoError(t, err)

	// move out of the initial health check window
	timeProvider.RestlessSleep(2 * timeWindow)

	source.Submit("1", werror.ErrorWithContextParams(ctx, "error for key: 1"))
	timeProvider.RestlessSleep(timeWindow / 2)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}, source.HealthStatus(ctx))

	source.Submit("2", werror.ErrorWithContextParams(ctx, "error for key: 2"))
	timeProvider.RestlessSleep(timeWindow / 4)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_REPAIRING),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
					"2": "error for key: 2",
				},
			},
		},
	}, source.HealthStatus(ctx))

	source.Submit("1", werror.ErrorWithContextParams(ctx, "error for key: 1"))
	timeProvider.RestlessSleep(timeWindow / 2)
	source.Submit("1", werror.ErrorWithContextParams(ctx, "error for key: 1"))

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
					"2": "error for key: 2",
				},
			},
		},
	}, source.HealthStatus(ctx))

	timeProvider.RestlessSleep(timeWindow / 2)

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}, source.HealthStatus(ctx))
}

func TestMultiKeyUnhealthyIfNoRecentErrorsSource(t *testing.T) {
	messageInCaseOfError := "message in case of error"
	for _, testCase := range []struct {
		name                     string
		keyErrorPairs            []keyErrorPair
		expectedCheck            health.HealthCheckResult
		durationAfterSubmissions time.Duration
	}{
		{
			name:          "healthy when there are no items",
			keyErrorPairs: nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when all keys are completely healthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1"},
				{key: "2"},
				{key: "3"},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when some keys are unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "1"},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "3"},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"2": "Error #1 for key 2",
				},
			},
		},
		{
			name: "healthy when outside window",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
			},
			expectedCheck:            whealth.HealthyHealthCheckResult(testCheckType),
			durationAfterSubmissions: time.Hour,
		},
		{
			name: "unhealthy when inside window",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "1", err: werror.Error("Error #2 for key 1")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #2 for key 1",
				},
			},
		},
		{
			name: "healthy when last keys are healthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1"},
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "1"},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2"},
				{key: "3"},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when all keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("Error #1 for key 1")},
				{key: "2", err: werror.Error("Error #1 for key 2")},
				{key: "2", err: werror.Error("Error #2 for key 2")},
				{key: "3", err: werror.Error("Error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "Error #1 for key 1",
					"2": "Error #2 for key 2",
					"3": "Error #1 for key 3",
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			timeProvider := &offsetTimeProvider{}
			source, err := newMultiKeyHealthyIfNoRecentErrorsSource(testCheckType, messageInCaseOfError, time.Minute, timeProvider)
			require.NoError(t, err)
			for _, keyErrorPair := range testCase.keyErrorPairs {
				source.Submit(keyErrorPair.key, keyErrorPair.err)
			}
			timeProvider.RestlessSleep(testCase.durationAfterSubmissions)
			expectedStatus := health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					testCheckType: testCase.expectedCheck,
				},
			}
			actualStatus := source.HealthStatus(context.Background())
			assert.Equal(t, expectedStatus, actualStatus)
		})
	}
}
