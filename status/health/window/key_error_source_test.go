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
				State:   health.HealthStateError,
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
				State:   health.HealthStateError,
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

func TestMultiKeyHealthyIfNotAllErrorsSourceOutsideStartWindow(t *testing.T) {
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
				State:   health.HealthStateError,
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
				State:   health.HealthStateError,
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
			source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, time.Hour, true, timeProvider)
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

func TestMultiKeyHealthyIfNotAllErrorsSourceInsideStartWindow(t *testing.T) {
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
				State:   health.HealthStateRepairing,
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
				State:   health.HealthStateRepairing,
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
			source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, time.Hour, true, timeProvider)
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

func TestMultiKeyHealthyIfNotAllErrorsSourceStartOnlyErrorWithWindowTransition(t *testing.T) {
	ctx := context.Background()
	messageInCaseOfError := "message in case of error"

	repairStatus := health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.HealthStateRepairing,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}

	timeProvider := &offsetTimeProvider{}
	source, err := newMultiKeyHealthyIfNotAllErrorsSource(testCheckType, messageInCaseOfError, time.Hour, true, timeProvider)
	require.NoError(t, err)

	// move partially into the initial health check window
	timeProvider.RestlessSleep(20 * time.Minute)
	source.Submit("1", werror.ErrorWithContextParams(ctx, "error for key: 1"))

	// still inside the initial window so error should be suppressed
	actualStatus := source.HealthStatus(context.Background())
	assert.Equal(t, repairStatus, actualStatus)

	// move out of the initial health check window but keep error inside sliding window
	timeProvider.RestlessSleep(50 * time.Minute)

	expectedPostWindow := health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": "error for key: 1",
				},
			},
		},
	}

	actualStatus = source.HealthStatus(context.Background())
	assert.Equal(t, expectedPostWindow, actualStatus)

}
