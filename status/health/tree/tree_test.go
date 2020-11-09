// Copyright (c) 2020 Palantir Technologies. All rights reserved.
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

package tree_test

import (
	"context"
	"testing"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
	"github.com/palantir/witchcraft-go-server/status/health/tree"
	"github.com/stretchr/testify/assert"
)

type healthStatusFn func(ctx context.Context) health.HealthStatus

func (fn healthStatusFn) HealthStatus(ctx context.Context) health.HealthStatus {
	return fn(ctx)
}

func TestHealthCheckSourceTree(t *testing.T) {
	someChildCheck := healthStatusFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"CHILD": {},
			},
		}
	})
	alwaysErrParentCheck := healthStatusFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"PARENT": {
					State: health.New_HealthState(health.HealthState_ERROR),
				},
			},
		}
	})
	treeSource := tree.NewHealthCheckSourceTree(alwaysErrParentCheck, []status.HealthCheckSource{someChildCheck})
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"PARENT": {
				State: health.New_HealthState(health.HealthState_ERROR),
			},
		},
	}, treeSource.HealthStatus(context.Background()))

	healthyParentCheck := healthStatusFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"PARENT": {
					State: health.New_HealthState(health.HealthState_HEALTHY),
				},
			},
		}
	})
	treeSource = tree.NewHealthCheckSourceTree(healthyParentCheck, []status.HealthCheckSource{someChildCheck})
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"PARENT": {
				State: health.New_HealthState(health.HealthState_HEALTHY),
			},
			"CHILD": {},
		},
	}, treeSource.HealthStatus(context.Background()))
}

func TestHealthCheckSourceTreeWithTraverseForHealthState(t *testing.T) {
	dummyChildCheck := healthStatusFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"CHILD_CHECK": {
					State: health.New_HealthState(health.HealthState_REPAIRING),
				},
			},
		}
	})
	unhealthyParentCheck := healthStatusFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"TEST_CHECK": {
					State: health.New_HealthState(health.HealthState_REPAIRING),
				},
			},
		}
	})
	healthCheckSourceTree := tree.NewHealthCheckSourceTree(unhealthyParentCheck, []status.HealthCheckSource{
		dummyChildCheck,
	}, tree.WithTraverseForHealthStatus(func(healthStatus health.HealthStatus) bool {
		return true
	}))
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"TEST_CHECK": {
				State: health.New_HealthState(health.HealthState_REPAIRING),
			},
			"CHILD_CHECK": {
				State: health.New_HealthState(health.HealthState_REPAIRING),
			},
		},
	}, healthCheckSourceTree.HealthStatus(context.Background()))

	healthCheckSourceTree = tree.NewHealthCheckSourceTree(unhealthyParentCheck, []status.HealthCheckSource{
		dummyChildCheck,
	}, tree.WithTraverseForHealthStatus(func(healthStatus health.HealthStatus) bool {
		return false
	}))

	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"TEST_CHECK": {
				State: health.New_HealthState(health.HealthState_REPAIRING),
			},
		},
	}, healthCheckSourceTree.HealthStatus(context.Background()))
}
