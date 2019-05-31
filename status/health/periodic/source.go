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

package periodic

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

type CheckFunc func(ctx context.Context) (state health.HealthState, params map[string]interface{})

type Source struct {
	Checks map[health.CheckType]CheckFunc
}

type checkState struct {
	lastResult      *health.HealthCheckResult
	lastResultTime  time.Time
	lastSuccess     *health.HealthCheckResult
	lastSuccessTime time.Time
}

type healthCheckSource struct {
	// static
	source        Source
	gracePeriod   time.Duration
	retryInterval time.Duration

	// mutable
	mutex       sync.RWMutex
	checkStates map[health.CheckType]*checkState
}

// NewHealthCheckSource creates a health check source that calls poll every retryInterval in a goroutine. The goroutine
// is cancelled if ctx is cancelled. If gracePeriod elapses without poll returning nil, the returned health check
// source will give a health status of error. checkType is the key to be used in the health result returned by the
// health check source.
func NewHealthCheckSource(ctx context.Context, gracePeriod time.Duration, retryInterval time.Duration, checkType health.CheckType, poll func() error) status.HealthCheckSource {
	return FromHealthCheckSource(ctx, gracePeriod, retryInterval, newDefaultHealthCheckSource(checkType, poll))
}

// FromHealthCheckSource creates a health check source that calls the the provided Source.Checks functions every
// retryInterval in a goroutine. The goroutine is cancelled if ctx is cancelled. For each check, if gracePeriod elapses
// without CheckFunc returning HEALTHY, the returned health check source's HealthStatus will return a HealthCheckResult
// of error.
func FromHealthCheckSource(ctx context.Context, gracePeriod time.Duration, retryInterval time.Duration, source Source) status.HealthCheckSource {
	checker := &healthCheckSource{
		gracePeriod:   gracePeriod,
		retryInterval: retryInterval,
		source:        source,
	}
	go wapp.RunWithRecoveryLogging(ctx, checker.runPoll)
	return checker
}

func (h *healthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	results := make([]health.HealthCheckResult, 0, len(h.source.Checks))
	for checkType := range h.source.Checks {
		checkState, ok := h.checkStates[checkType]
		if !ok {
			results = append(results, health.HealthCheckResult{
				Type:    checkType,
				State:   health.HealthStateRepairing,
				Message: stringPtr("Check has not yet run"),
			})
			continue
		}
		var result health.HealthCheckResult
		switch {
		case time.Since(checkState.lastSuccessTime) <= h.gracePeriod:
			result = *checkState.lastSuccess
		case time.Since(checkState.lastResultTime) <= h.gracePeriod:
			result = *checkState.lastResult
			result.Message = stringPtr(fmt.Sprintf("No successful checks during %s grace period", h.gracePeriod.String()))
		default:
			result = *checkState.lastResult
			result.Message = stringPtr(fmt.Sprintf("No completed checks during %s grace period", h.gracePeriod.String()))
			// Mark REPAIRING if we were healthy before expiration.
			if result.State == health.HealthStateHealthy {
				result.State = health.HealthStateRepairing
			}
		}
		results = append(results, result)
	}

	return toHealthStatus(results)
}

func (h *healthCheckSource) runPoll(ctx context.Context) {
	ticker := time.NewTicker(h.retryInterval)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.doPoll(ctx)
		}
	}
}

func (h *healthCheckSource) doPoll(ctx context.Context) {
	type resultWithTime struct {
		time   time.Time
		result *health.HealthCheckResult
	}

	// Run checks
	resultsWithTimes := make([]resultWithTime, 0, len(h.source.Checks))
	for checkType, check := range h.source.Checks {
		state, params := check(ctx)
		resultsWithTimes = append(resultsWithTimes, resultWithTime{
			time: time.Now(),
			result: &health.HealthCheckResult{
				Type:   checkType,
				State:  state,
				Params: params,
			},
		})
	}

	// Update cached state
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for _, resultWithTime := range resultsWithTimes {
		state := h.checkStates[resultWithTime.result.Type]
		state.lastResult = resultWithTime.result
		state.lastResultTime = resultWithTime.time
		if resultWithTime.result.State == health.HealthStateHealthy {
			state.lastSuccess = resultWithTime.result
			state.lastSuccessTime = resultWithTime.time
		}
	}
}

func toHealthStatus(results []health.HealthCheckResult) health.HealthStatus {
	checks := make(map[health.CheckType]health.HealthCheckResult, len(results))
	for _, result := range results {
		checks[result.Type] = result
	}
	return health.HealthStatus{
		Checks: checks,
	}
}

func newDefaultHealthCheckSource(checkType health.CheckType, poll func() error) Source {
	return Source{
		Checks: map[health.CheckType]CheckFunc{
			checkType: func(ctx context.Context) (state health.HealthState, params map[string]interface{}) {
				err := poll()
				if err != nil {
					return health.HealthStateError, map[string]interface{}{"error": err.Error()}
				}
				return health.HealthStateHealthy, nil
			},
		},
	}
}

func stringPtr(s string) *string {
	return &s
}
