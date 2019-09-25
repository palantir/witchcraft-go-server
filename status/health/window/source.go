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
	"github.com/palantir/witchcraft-go-server/status"
)

// ErrorsToCheckFn is a function that constructs a HealthCheckResult from a set of errors.
type ErrorsToCheckFn func(ctx context.Context, errors []error) health.HealthCheckResult

type healchCheckSource struct {
	slidingWindowManager *SlidingWindowManager
	errorsToCheckFn      ErrorsToCheckFn
}

// NewHealthCheck creates a HealthCheckSource that polls a SlidingWindowManager and returns a HealthStatus created using an ErrorsToCheckFn.
func NewHealthCheck(slidingWindowManager *SlidingWindowManager, errorsToCheckFn ErrorsToCheckFn) status.HealthCheckSource {
	return &healchCheckSource{
		slidingWindowManager: slidingWindowManager,
		errorsToCheckFn:      errorsToCheckFn,
	}
}

func (h *healchCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	checkResult := h.errorsToCheckFn(ctx, h.slidingWindowManager.GetErrors())
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			checkResult.Type: checkResult,
		},
	}
}
