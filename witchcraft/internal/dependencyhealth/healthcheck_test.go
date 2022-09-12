// Copyright (c) 2022 Palantir Technologies. All rights reserved.
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

package dependencyhealth

import (
	"context"
	"testing"

	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/stretchr/testify/assert"
)

func TestServiceDependencyHealthCheck(t *testing.T) {
	for _, test := range []struct {
		Name     string
		Services map[ServiceName]map[HostPort]HostStatus
		Expected health.HealthStatus
	}{
		{
			Name:     "empty",
			Services: map[ServiceName]map[HostPort]HostStatus{},
			Expected: health.HealthStatus{},
		},
		{
			Name: "no errors",
			Services: map[ServiceName]map[HostPort]HostStatus{
				"serviceA": {
					"hostA:443": mockHostMetrics{active: true, healthy: true},
					"hostB:443": mockHostMetrics{active: true, healthy: true},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					serviceDependencyCheckType: {
						Type:    serviceDependencyCheckType,
						State:   health.New_HealthState(health.HealthState_HEALTHY),
						Message: &serviceDependencyMsgHealthy,
						Params:  map[string]interface{}{},
					},
				},
			},
		},
		{
			Name: "one impaired",
			Services: map[ServiceName]map[HostPort]HostStatus{
				"serviceA": {
					"hostA:443": mockHostMetrics{active: true, healthy: true},
					"hostB:443": mockHostMetrics{active: true, healthy: false},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					serviceDependencyCheckType: {
						Type:    serviceDependencyCheckType,
						State:   health.New_HealthState(health.HealthState_HEALTHY),
						Message: &serviceDependencyMsgServiceImpaired,
						Params: map[string]interface{}{
							"serviceA": []string{"hostB:443"},
						},
					},
				},
			},
		},
		{
			Name: "all impaired",
			Services: map[ServiceName]map[HostPort]HostStatus{
				"serviceA": {
					"hostA:443": mockHostMetrics{active: true, healthy: false},
					"hostB:443": mockHostMetrics{active: true, healthy: false},
				},
				"serviceB": {
					"hostA:443": mockHostMetrics{active: true, healthy: true},
					"hostB:443": mockHostMetrics{active: true, healthy: false},
				},
			},
			Expected: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					serviceDependencyCheckType: {
						Type:    serviceDependencyCheckType,
						State:   health.New_HealthState(health.HealthState_WARNING),
						Message: &serviceDependencyMsgServiceFailed,
						Params: map[string]interface{}{
							"serviceA": []string{"hostA:443", "hostB:443"},
							"serviceB": []string{"hostB:443"},
						},
					},
				},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			hc := &ServiceDependencyHealthCheck{hosts: mockHostMetricsRegistry{services: test.Services}}
			result := hc.HealthStatus(context.Background())
			assert.Equal(t, test.Expected, result)
		})
	}
}

type mockHostMetricsRegistry struct {
	services map[ServiceName]map[HostPort]HostStatus
}

func (m mockHostMetricsRegistry) HostMetrics(string, string, string) HostStatus {
	panic("implement me")
}

func (m mockHostMetricsRegistry) AllServices() map[ServiceName]map[HostPort]HostStatus {
	return m.services
}

type mockHostMetrics struct {
	HostStatus
	healthy bool
	active  bool
}

func (m mockHostMetrics) IsActive() bool {
	return m.active
}

func (m mockHostMetrics) IsHealthy() bool {
	return m.healthy
}
