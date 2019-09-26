// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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
)

func TestMultiKeyUnhealthyIfAtLeastOneErrorSource(t *testing.T) {
	messageInCaseOfError := "message in case of err"
	for _, testCase := range []struct {
		name          string
		keyErrorPairs []keyErrorPair
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no events",
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
				{key: "1", err: werror.Error("err #1 for key 1")},
				{key: "1"},
				{key: "2", err: werror.Error("err #1 for key 2")},
				{key: "2"},
				{key: "3"},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("err #1 for key 1"),
					"2": werror.Error("err #1 for key 2"),
				},
			},
		},
		{
			name: "unhealthy when all keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("err #1 for key 1")},
				{key: "2", err: werror.Error("err #1 for key 2")},
				{key: "2", err: werror.Error("err #2 for key 2")},
				{key: "3", err: werror.Error("err #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("err #1 for key 1"),
					"2": werror.Error("err #1 for key 2"),
					"3": werror.Error("err #1 for key 3"),
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source, err := NewMultiKeyUnhealthyIfAtLeastOneErrorSource(time.Hour, testCheckType, messageInCaseOfError)
			assert.NoError(t, err)
			for _, keyErrorPair := range testCase.keyErrorPairs {
				source.SubmitError(keyErrorPair.key, keyErrorPair.err)
			}
			actualStatus := source.HealthStatus(context.Background())
			assert.EqualValues(t, 1, len(actualStatus.Checks))
			actualCheck, checkTypeOk := actualStatus.Checks[testCheckType]
			assert.True(t, checkTypeOk)
			assert.Equal(t, testCase.expectedCheck.Type, actualCheck.Type)
			assert.Equal(t, testCase.expectedCheck.State, actualCheck.State)
			assert.Equal(t, testCase.expectedCheck.Message, actualCheck.Message)
			assert.Equal(t, len(testCase.expectedCheck.Params), len(actualCheck.Params))
			for key, expectedValue := range testCase.expectedCheck.Params {
				actualValue, hasKey := actualCheck.Params[key]
				assert.True(t, hasKey)
				assert.Equal(t, expectedValue.(error).Error(), actualValue.(error).Error())
			}
		})
	}
}

func TestMultiKeyHealthyIfNotAllErrorsSource(t *testing.T) {
	messageInCaseOfError := "message in case of err"
	for _, testCase := range []struct {
		name          string
		keyErrorPairs []keyErrorPair
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no events",
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
				{key: "1", err: werror.Error("err #1 for key 1")},
				{key: "1"},
				{key: "2", err: werror.Error("err #1 for key 2")},
				{key: "2"},
				{key: "3"},
				{key: "3", err: werror.Error("err #1 for key 3")},
				{key: "3", err: werror.Error("err #2 for key 3")},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when some keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("err #1 for key 1")},
				{key: "2", err: werror.Error("err #1 for key 2")},
				{key: "2", err: werror.Error("err #2 for key 2")},
				{key: "3"},
				{key: "3", err: werror.Error("err #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("err #1 for key 1"),
					"2": werror.Error("err #1 for key 2"),
				},
			},
		},
		{
			name: "unhealthy when all keys are completely unhealthy",
			keyErrorPairs: []keyErrorPair{
				{key: "1", err: werror.Error("err #1 for key 1")},
				{key: "2", err: werror.Error("err #1 for key 2")},
				{key: "2", err: werror.Error("err #2 for key 2")},
				{key: "3", err: werror.Error("err #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("err #1 for key 1"),
					"2": werror.Error("err #1 for key 2"),
					"3": werror.Error("err #1 for key 3"),
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source, err := NewMultiKeyHealthyIfNotAllErrorsSource(time.Hour, testCheckType, messageInCaseOfError)
			assert.NoError(t, err)
			for _, keyErrorPair := range testCase.keyErrorPairs {
				source.SubmitError(keyErrorPair.key, keyErrorPair.err)
			}
			actualStatus := source.HealthStatus(context.Background())
			assert.EqualValues(t, 1, len(actualStatus.Checks))
			actualCheck, checkTypeOk := actualStatus.Checks[testCheckType]
			assert.True(t, checkTypeOk)
			assert.Equal(t, testCase.expectedCheck.Type, actualCheck.Type)
			assert.Equal(t, testCase.expectedCheck.State, actualCheck.State)
			assert.Equal(t, testCase.expectedCheck.Message, actualCheck.Message)
			assert.Equal(t, len(testCase.expectedCheck.Params), len(actualCheck.Params))
			for key, expectedValue := range testCase.expectedCheck.Params {
				actualValue, hasKey := actualCheck.Params[key]
				assert.True(t, hasKey)
				assert.Equal(t, expectedValue.(error).Error(), actualValue.(error).Error())
			}
		})
	}
}
