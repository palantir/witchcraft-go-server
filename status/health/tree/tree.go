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

package tree

import (
	"context"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

type (
	// TraverseForHealthStatus accepts the health status result of the current health check source node
	// and returns whether the current health check source tree node should traverse to child health check
	// source nodes or not.
	TraverseForHealthStatus func(status health.HealthStatus) bool
)

type healthCheckSourceTreeNode struct {
	healthCheckSource                 status.HealthCheckSource
	combinedChildrenHealthCheckSource status.HealthCheckSource
	traverseForHealthState            TraverseForHealthStatus
}

type CheckSourceTreeParam interface {
	apply(*healthCheckSourceTreeNode)
}

type healthCheckSourceTreeParamFn func(*healthCheckSourceTreeNode)

func (fn healthCheckSourceTreeParamFn) apply(node *healthCheckSourceTreeNode) {
	fn(node)
}

// WithTraverseForHealthStatus allows consumer-provided logic to determine when the child health checks of the node
// should be invoked based on the current node's returned health status.
func WithTraverseForHealthStatus(fn TraverseForHealthStatus) CheckSourceTreeParam {
	return healthCheckSourceTreeParamFn(func(node *healthCheckSourceTreeNode) {
		node.traverseForHealthState = fn
	})
}

// NewHealthCheckSourceTree returns a new health check source tree node that uses the result of its own health check
// to determine if it should invoke the provided child health checks. The default behavior is to only traverse to child
// checks if the most severe health state of the current node's health status is health.HealthStateHealthy.
func NewHealthCheckSourceTree(
	ownHealthCheckSource status.HealthCheckSource,
	childrenHealthCheckSources []status.HealthCheckSource,
	params ...CheckSourceTreeParam,
) status.HealthCheckSource {
	n := &healthCheckSourceTreeNode{
		healthCheckSource:                 ownHealthCheckSource,
		combinedChildrenHealthCheckSource: status.NewCombinedHealthCheckSource(childrenHealthCheckSources...),
		traverseForHealthState:            defaultTraverseForHealthStatus,
	}
	for _, param := range params {
		param.apply(n)
	}
	return n
}

func (n *healthCheckSourceTreeNode) HealthStatus(ctx context.Context) health.HealthStatus {
	ownHealthStatus := n.healthCheckSource.HealthStatus(ctx)
	if !n.traverseForHealthState(ownHealthStatus) {
		return ownHealthStatus
	}
	healthStatusFromChildSources := n.combinedChildrenHealthCheckSource.HealthStatus(ctx)
	for checkType, checkResult := range healthStatusFromChildSources.Checks {
		ownHealthStatus.Checks[checkType] = checkResult
	}
	return ownHealthStatus
}

func healthStateFromChecks(checks map[health.CheckType]health.HealthCheckResult) health.HealthState {
	healthState := health.HealthStateHealthy
	for _, checkResult := range checks {
		if status.HealthStateStatusCodes[checkResult.State] > status.HealthStateStatusCodes[healthState] {
			healthState = checkResult.State
		}
	}
	return healthState
}

func defaultTraverseForHealthStatus(healthStatus health.HealthStatus) bool {
	return healthStateFromChecks(healthStatus.Checks) == health.HealthStateHealthy
}
