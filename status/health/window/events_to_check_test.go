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
)

const (
	testCheckType health.CheckType = "TEST_CHECK"
)

func toEvents(time time.Time, payloads []interface{}) []Event {
	var events []Event
	for _, payload := range payloads {
		events = append(events, Event{
			Time:    time,
			Payload: payload,
		})
	}
	return events
}

func TestUnhealthyIfAtLeastOneError(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		payloads      []interface{}
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no events",
			payloads:      nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when there are only nil events",
			payloads: []interface{}{
				nil,
				nil,
				nil,
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when there is at least one error",
			payloads: []interface{}{
				nil,
				werror.Error("error #1"),
				nil,
				werror.Error("error #2"),
				nil,
			},
			expectedCheck: whealth.UnhealthyHealthCheckResult(testCheckType, "error #1"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			events := toEvents(time.Now(), testCase.payloads)
			actualCheck := UnhealthyIfAtLeastOneError(testCheckType)(context.Background(), events)
			assert.Equal(t, testCase.expectedCheck, actualCheck)
		})
	}
}

func TestHealthyIfNotAllErrors(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		payloads      []interface{}
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no events",
			payloads:      nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when there are only nil events",
			payloads: []interface{}{
				nil,
				nil,
				nil,
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when there is at least one non nil error",
			payloads: []interface{}{
				nil,
				werror.Error("error #1"),
				nil,
				werror.Error("error #2"),
				nil,
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when there are only non nil events",
			payloads: []interface{}{
				werror.Error("error #1"),
				werror.Error("error #2"),
			},
			expectedCheck: whealth.UnhealthyHealthCheckResult(testCheckType, "error #1"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			events := toEvents(time.Now(), testCase.payloads)
			actualCheck := HealthyIfNotAllErrors(testCheckType)(context.Background(), events)
			assert.Equal(t, testCase.expectedCheck, actualCheck)
		})
	}
}

func TestMultiKeyUnhealthyIfAtLeastOneError(t *testing.T) {
	messageInCaseOfError := "message in case of error"
	for _, testCase := range []struct {
		name          string
		payloads      []interface{}
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no events",
			payloads:      nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when all keys are completely healthy",
			payloads: []interface{}{
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "2"},
				KeyErrorPair{Key: "3"},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when some keys are partially healthy",
			payloads: []interface{}{
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "1", Error: werror.Error("error #1 for key 1")},
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "2", Error: werror.Error("error #1 for key 2")},
				KeyErrorPair{Key: "2"},
				KeyErrorPair{Key: "3"},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("error #1 for key 1"),
					"2": werror.Error("error #1 for key 2"),
				},
			},
		},
		{
			name: "unhealthy when all keys are completely unhealthy",
			payloads: []interface{}{
				KeyErrorPair{Key: "1", Error: werror.Error("error #1 for key 1")},
				KeyErrorPair{Key: "2", Error: werror.Error("error #1 for key 2")},
				KeyErrorPair{Key: "2", Error: werror.Error("error #2 for key 2")},
				KeyErrorPair{Key: "3", Error: werror.Error("error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("error #1 for key 1"),
					"2": werror.Error("error #1 for key 2"),
					"3": werror.Error("error #1 for key 3"),
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			events := toEvents(time.Now(), testCase.payloads)
			actualCheck := MultiKeyUnhealthyIfAtLeastOneError(testCheckType, messageInCaseOfError)(context.Background(), events)
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

func TestMultiKeyHealthyIfNotAllErrors(t *testing.T) {
	messageInCaseOfError := "message in case of error"
	for _, testCase := range []struct {
		name          string
		payloads      []interface{}
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no events",
			payloads:      nil,
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when all keys are completely healthy",
			payloads: []interface{}{
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "2"},
				KeyErrorPair{Key: "3"},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "healthy when all keys are partially healthy",
			payloads: []interface{}{
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "1", Error: werror.Error("error #1 for key 1")},
				KeyErrorPair{Key: "1"},
				KeyErrorPair{Key: "2", Error: werror.Error("error #1 for key 2")},
				KeyErrorPair{Key: "2"},
				KeyErrorPair{Key: "3"},
				KeyErrorPair{Key: "3", Error: werror.Error("error #1 for key 3")},
				KeyErrorPair{Key: "3", Error: werror.Error("error #2 for key 3")},
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when some keys are completely unhealthy",
			payloads: []interface{}{
				KeyErrorPair{Key: "1", Error: werror.Error("error #1 for key 1")},
				KeyErrorPair{Key: "2", Error: werror.Error("error #1 for key 2")},
				KeyErrorPair{Key: "2", Error: werror.Error("error #2 for key 2")},
				KeyErrorPair{Key: "3"},
				KeyErrorPair{Key: "3", Error: werror.Error("error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("error #1 for key 1"),
					"2": werror.Error("error #1 for key 2"),
				},
			},
		},
		{
			name: "unhealthy when all keys are completely unhealthy",
			payloads: []interface{}{
				KeyErrorPair{Key: "1", Error: werror.Error("error #1 for key 1")},
				KeyErrorPair{Key: "2", Error: werror.Error("error #1 for key 2")},
				KeyErrorPair{Key: "2", Error: werror.Error("error #2 for key 2")},
				KeyErrorPair{Key: "3", Error: werror.Error("error #1 for key 3")},
			},
			expectedCheck: health.HealthCheckResult{
				Type:    testCheckType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params: map[string]interface{}{
					"1": werror.Error("error #1 for key 1"),
					"2": werror.Error("error #1 for key 2"),
					"3": werror.Error("error #1 for key 3"),
				},
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			events := toEvents(time.Now(), testCase.payloads)
			actualCheck := MultiKeyHealthyIfNotAllErrors(testCheckType, messageInCaseOfError)(context.Background(), events)
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
