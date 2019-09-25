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

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	health_util "github.com/palantir/witchcraft-go-server/status/health"
	"github.com/stretchr/testify/assert"
)

const (
	checkType = "TEST_CHECK"
)

func TestUnhealthyIfAtLeastOneError(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		errors        []error
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no errors",
			errors:        nil,
			expectedCheck: health_util.HealthyHealthCheckResult(checkType),
		},
		{
			name: "healthy when there are only nil errors",
			errors: []error{
				nil,
				nil,
				nil,
			},
			expectedCheck: health_util.HealthyHealthCheckResult(checkType),
		},
		{
			name: "unhealthy when there is at least one error",
			errors: []error{
				nil,
				werror.Error("error #1"),
				nil,
				werror.Error("error #2"),
				nil,
			},
			expectedCheck: health_util.UnhealthyHealthCheckResult(checkType, "error #1"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actualCheck := UnhealthyIfAtLeastOneError(checkType)(context.Background(), testCase.errors)
			assert.Equal(t, testCase.expectedCheck, actualCheck)
		})
	}
}

func TestHealthyIfNotAllErrors(t *testing.T) {
	for _, testCase := range []struct {
		name          string
		errors        []error
		expectedCheck health.HealthCheckResult
	}{
		{
			name:          "healthy when there are no errors",
			errors:        nil,
			expectedCheck: health_util.HealthyHealthCheckResult(checkType),
		},
		{
			name: "healthy when there are only nil errors",
			errors: []error{
				nil,
				nil,
				nil,
			},
			expectedCheck: health_util.HealthyHealthCheckResult(checkType),
		},
		{
			name: "healthy when there is at least one non nil error",
			errors: []error{
				nil,
				werror.Error("error #1"),
				nil,
				werror.Error("error #2"),
				nil,
			},
			expectedCheck: health_util.HealthyHealthCheckResult(checkType),
		},
		{
			name: "unhealthy when there are only non nil errors",
			errors: []error{
				werror.Error("error #1"),
				werror.Error("error #2"),
			},
			expectedCheck: health_util.UnhealthyHealthCheckResult(checkType, "error #1"),
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			actualCheck := HealthyIfNotAllErrors(checkType)(context.Background(), testCase.errors)
			assert.Equal(t, testCase.expectedCheck, actualCheck)
		})
	}
}
