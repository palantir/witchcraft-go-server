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
				werror.Error("Error #1"),
				nil,
				werror.Error("Error #2"),
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
				werror.Error("Error #1"),
				nil,
				werror.Error("Error #2"),
				nil,
			},
			expectedCheck: whealth.HealthyHealthCheckResult(testCheckType),
		},
		{
			name: "unhealthy when there are only non nil items",
			errors: []error{
				werror.Error("Error #1"),
				werror.Error("Error #2"),
			},
			expectedCheck: whealth.UnhealthyHealthCheckResult(testCheckType, "Error #2"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			source, err := NewHealthyIfNotAllErrorsSource(testCheckType, time.Hour)
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
