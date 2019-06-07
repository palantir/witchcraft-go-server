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

type CheckFunc func(ctx context.Context) *health.HealthCheckResult

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
		source:        source,
		gracePeriod:   gracePeriod,
		retryInterval: retryInterval,
		checkStates:   map[health.CheckType]*checkState{},
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
			result.Message = stringPtr(wrap(result.Message, fmt.Sprintf("No successful checks during %s grace period", h.gracePeriod.String())))
		default:
			result = *checkState.lastResult
			result.Message = stringPtr(wrap(result.Message, fmt.Sprintf("No completed checks during %s grace period", h.gracePeriod.String())))
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
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// ensure that doPoll is not called if context is cancelled (without this, if ctx.Done() and ticker.C fire
			// at the same time and the ticker.C case is selected at the top-level, doPoll may be called even though the
			// context is done).
			select {
			case <-ctx.Done():
				return
			default:
			}
			h.doPoll(ctx)
		}
	}
}

func (h *healthCheckSource) doPoll(ctx context.Context) {
	type resultWithTime struct {
		result *health.HealthCheckResult
		time   time.Time
	}

	// Run checks
	resultsWithTimes := make([]resultWithTime, 0, len(h.source.Checks))
	for _, check := range h.source.Checks {
		resultsWithTimes = append(resultsWithTimes, resultWithTime{
			time:   time.Now(),
			result: check(ctx),
		})
	}

	// Update cached state
	h.mutex.Lock()
	defer h.mutex.Unlock()
	for _, resultWithTime := range resultsWithTimes {
		newState := &checkState{
			lastResult:     resultWithTime.result,
			lastResultTime: resultWithTime.time,
		}
		// populate last success state from previous state (if present)
		if previousState, ok := h.checkStates[resultWithTime.result.Type]; ok {
			newState.lastSuccess = previousState.lastSuccess
			newState.lastSuccessTime = previousState.lastSuccessTime
		}
		// if current result is successful, update success state
		if resultWithTime.result.State == health.HealthStateHealthy {
			newState.lastSuccess = resultWithTime.result
			newState.lastSuccessTime = resultWithTime.time
		}
		h.checkStates[resultWithTime.result.Type] = newState
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
			checkType: func(ctx context.Context) *health.HealthCheckResult {
				err := poll()
				if err != nil {
					return &health.HealthCheckResult{
						Type:    checkType,
						State:   health.HealthStateError,
						Message: stringPtr(err.Error()),
					}
				}
				return &health.HealthCheckResult{
					Type:  checkType,
					State: health.HealthStateHealthy,
				}
			},
		},
	}
}

func wrap(baseStringPtr *string, prependStr string) string {
	if baseStringPtr == nil {
		return prependStr
	}
	return prependStr + ": " + *baseStringPtr
}

func stringPtr(s string) *string {
	return &s
}
