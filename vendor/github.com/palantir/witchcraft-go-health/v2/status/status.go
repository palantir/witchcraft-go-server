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

package status

import (
	"context"
	"net/http"
	"slices"

	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/witchcraft-go-health/v2/conjure/witchcraft/api/health"
)

var (
	healthStateStatusCodes = map[health.HealthState_Value]int{
		health.HealthState_HEALTHY:   http.StatusOK,
		health.HealthState_DEFERRING: 518,
		health.HealthState_SUSPENDED: 519,
		health.HealthState_REPAIRING: 520,
		health.HealthState_WARNING:   521,
		health.HealthState_ERROR:     522,
		health.HealthState_TERMINAL:  523,
	}

	// readyHealthStates defines the health states that indicate a service is ready
	readyHealthStates = []health.HealthState_Value{
		health.HealthState_HEALTHY,
		health.HealthState_DEFERRING,
		health.HealthState_WARNING,
	}
)

// Source provides status that should be sent as a response.
type Source interface {
	Status() (respStatus int, metadata interface{})
}

// HealthCheckSource provides the SLS health status that should be sent as a response.
// Refer to the SLS specification for more information.
type HealthCheckSource interface {
	HealthStatus(ctx context.Context) health.HealthStatus
}

type combinedHealthCheckSource struct {
	healthCheckSources refreshable.Refreshable[[]HealthCheckSource]
}

func NewCombinedHealthCheckSource(healthCheckSources ...HealthCheckSource) HealthCheckSource {
	var healthCheckSourceSlice []HealthCheckSource
	for _, healthCheckSource := range healthCheckSources {
		healthCheckSourceSlice = append(healthCheckSourceSlice, healthCheckSource)
	}
	return &combinedHealthCheckSource{
		healthCheckSources: refreshable.New(healthCheckSourceSlice),
	}
}

func NewCombinedHealthCheckSourceWithRefresh(healthCheckSources refreshable.Refreshable[[]HealthCheckSource]) HealthCheckSource {
	return &combinedHealthCheckSource{
		healthCheckSources: healthCheckSources,
	}
}

func (c *combinedHealthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	result := health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{},
	}
	for _, healthCheckSource := range c.healthCheckSources.Current() {
		if healthCheckSource == nil {
			continue
		}
		for k, v := range healthCheckSource.HealthStatus(ctx).Checks {
			result.Checks[k] = v
		}
	}
	return result
}

// HealthStateStatusCode returns the http status code for the provided health.HealthState_Value or
// http.StatusInternalServerError if the health state value is not recognized.
func HealthStateStatusCode(state health.HealthState_Value) int {
	code, ok := healthStateStatusCodes[state]
	if !ok {
		code = http.StatusInternalServerError
	}
	return code
}

func HealthStatusCode(metadata health.HealthStatus) int {
	worst := http.StatusOK
	for _, result := range metadata.Checks {
		code := HealthStateStatusCode(result.State.Value())
		if worst < code {
			worst = code
		}
	}
	return worst
}

// HealthBasedReadinessSource creates a Source that determines readiness based on the provided
// health check sources. The readiness status is OK if all health check sources report a healthy
// state (HEALTHY, DEFERRING, or WARNING). Otherwise, it returns the status code corresponding to
// the first non-OK health state encountered.
func HealthBasedReadinessSource(healthCheckSources refreshable.Refreshable[[]HealthCheckSource]) Source {
	return &healthBasedReadinessSource{
		healthSource: NewCombinedHealthCheckSourceWithRefresh(healthCheckSources),
	}
}

type healthBasedReadinessSource struct {
	healthSource HealthCheckSource
}

func (h *healthBasedReadinessSource) Status() (int, interface{}) {
	ctx := context.Background()
	healthStatus := h.healthSource.HealthStatus(ctx)
	for _, check := range healthStatus.Checks {
		state := check.State.Value()
		if !slices.Contains(readyHealthStates, state) {
			return HealthStateStatusCode(state), check
		}
	}

	// All checks passed
	return http.StatusOK, nil
}
