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

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	whealth "github.com/palantir/witchcraft-go-server/status/health"
)

// UnhealthyIfAtLeastOneError builds an ErrorsToCheckFn that returns the first non-nil error as an unhealthy check.
// If there are no non-nil errors, returns healthy.
func UnhealthyIfAtLeastOneError(checkType health.CheckType) ErrorsToCheckFn {
	return func(ctx context.Context, errors []error) health.HealthCheckResult {
		for _, err := range errors {
			if err != nil {
				return whealth.UnhealthyHealthCheckResult(checkType, err.Error())
			}
		}
		return whealth.HealthyHealthCheckResult(checkType)
	}
}

// HealthyIfNotAllErrors builds an ErrorsToCheckFn that returns, if there are only non-nil errors, the first non-nil error as an unhealthy check.
// If there are no errors, returns healthy.
func HealthyIfNotAllErrors(checkType health.CheckType) ErrorsToCheckFn {
	return func(ctx context.Context, errors []error) health.HealthCheckResult {
		if len(errors) == 0 {
			return whealth.HealthyHealthCheckResult(checkType)
		}
		for _, err := range errors {
			if err == nil {
				return whealth.HealthyHealthCheckResult(checkType)
			}
		}
		return whealth.UnhealthyHealthCheckResult(checkType, errors[0].Error())
	}
}
