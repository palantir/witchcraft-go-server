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

type Source interface {
	Check(ctx context.Context) (state health.HealthState, params map[string]interface{})
}

type healthCheckSource struct {
	// static
	source        Source
	gracePeriod   time.Duration
	retryInterval time.Duration
	checkType     health.CheckType

	// mutable
	mutex           sync.RWMutex
	lastResult      *health.HealthCheckResult
	lastResultTime  time.Time
	lastSuccess     *health.HealthCheckResult
	lastSuccessTime time.Time
}

// NewHealthCheckSource creates a health check source that calls poll every retryInterval in a goroutine. The goroutine
// is cancelled if ctx is cancelled. If gracePeriod elapses without poll returning nil, the returned health check
// source will give a health status of error. checkType is the key to be used in the health result returned by the
// health check source.
func NewHealthCheckSource(ctx context.Context, gracePeriod time.Duration, retryInterval time.Duration, checkType health.CheckType, poll func() error) status.HealthCheckSource {
	return FromHealthCheckSource(ctx, gracePeriod, retryInterval, checkType, &defaultHealthCheckSource{poll: poll})
}

// FromHealthCheckSource creates a health check source that calls source.Check every retryInterval in a goroutine. The goroutine
// is cancelled if ctx is cancelled. If gracePeriod elapses without Check returning HEALTHY, the returned health check
// source will give a health status of error. checkType is the key to be used in the health result returned by the
// health check source.
func FromHealthCheckSource(ctx context.Context, gracePeriod time.Duration, retryInterval time.Duration, checkType health.CheckType, source Source) status.HealthCheckSource {
	checker := &healthCheckSource{
		gracePeriod:   gracePeriod,
		retryInterval: retryInterval,
		checkType:     checkType,
		source:        source,
	}
	go wapp.RunWithRecoveryLogging(ctx, checker.runPoll)
	return checker
}

func (h *healthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	if h.lastResult == nil {
		return toHealthStatus(health.HealthCheckResult{
			Type:    h.checkType,
			State:   health.HealthStateRepairing,
			Message: stringPtr("Check has not yet run"),
		})
	}

	var result health.HealthCheckResult
	switch {
	case time.Since(h.lastSuccessTime) <= h.gracePeriod:
		result = *h.lastSuccess
	case time.Since(h.lastResultTime) <= h.gracePeriod:
		result = *h.lastResult
		result.Message = stringPtr(fmt.Sprintf("No successful checks during %s grace period", h.gracePeriod.String()))
	default:
		result = *h.lastResult
		result.Message = stringPtr(fmt.Sprintf("No completed checks during %s grace period", h.gracePeriod.String()))
		// Mark REPAIRING if we were healthy before expiration.
		if result.State == health.HealthStateHealthy {
			result.State = health.HealthStateRepairing
		}
	}
	return toHealthStatus(result)
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
	// Run check
	state, params := h.source.Check(ctx)
	t := time.Now()
	result := &health.HealthCheckResult{
		Type:   h.checkType,
		State:  state,
		Params: params,
	}
	// Update cached state
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.lastResult = result
	h.lastResultTime = t
	if state == health.HealthStateHealthy {
		h.lastSuccess = result
		h.lastSuccessTime = t
	}
}

func toHealthStatus(result health.HealthCheckResult) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			result.Type: result,
		},
	}
}

type defaultHealthCheckSource struct {
	poll func() error
}

func (d *defaultHealthCheckSource) Check(_ context.Context) (state health.HealthState, params map[string]interface{}) {
	err := d.poll()
	if err != nil {
		return health.HealthStateError, map[string]interface{}{"error": err.Error()}
	}
	return health.HealthStateHealthy, nil
}

func stringPtr(s string) *string {
	return &s
}
