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

	conjurehealth "github.com/palantir/witchcraft-go-server/conjure/sls/spec/health"
	"github.com/palantir/witchcraft-go-server/status"
	"github.com/palantir/witchcraft-go-server/status/health"
)

type healthCheckSource struct {
	gracePeriodReset         time.Time
	numRunsDuringGracePeriod int
	lastErr                  error
	poll                     func() error
	gracePeriod              time.Duration
	retryInterval            time.Duration
	checkType                conjurehealth.CheckType
	mutex                    sync.RWMutex
}

// NewHealthCheckSource creates a health check source that calls poll every retryInterval in a goroutine. The goroutine
// is cancelled if ctx is cancelled. If gracePeriod elapses without poll returning nil, the returned health check
// source will give a health status of error. checkType is the key to be used in the health result returned by the
// health check source.
func NewHealthCheckSource(ctx context.Context, gracePeriod time.Duration, retryInterval time.Duration, checkType conjurehealth.CheckType, poll func() error) status.HealthCheckSource {
	checker := &healthCheckSource{
		gracePeriodReset: time.Now(),
		lastErr:          nil,
		gracePeriod:      gracePeriod,
		retryInterval:    retryInterval,
		checkType:        checkType,
		poll:             poll,
	}
	go checker.runPoll(ctx)
	return checker
}

func (h *healthCheckSource) HealthStatus(ctx context.Context) conjurehealth.HealthStatus {
	h.mutex.RLock()
	defer func() {
		h.mutex.RUnlock()
	}()
	if time.Now().Sub(h.gracePeriodReset) > h.gracePeriod {
		if h.numRunsDuringGracePeriod > 0 {
			return toHealthStatus(health.UnhealthyHealthCheckResult(h.checkType, fmt.Sprintf("have not seen a success during grace period, grace period reset: %s, grace period duration: %s, last error: %s", h.gracePeriodReset.String(), h.gracePeriod.String(), h.lastError())))
		}
		return toHealthStatus(health.UnhealthyHealthCheckResult(h.checkType, fmt.Sprintf("have not completed a poll during grace period, grace period reset: %s, grace period duration: %s", h.gracePeriodReset.String(), h.gracePeriod.String())))
	}
	return toHealthStatus(health.HealthyHealthCheckResult(h.checkType))
}

func (h *healthCheckSource) runPoll(ctx context.Context) {
	ticker := time.NewTicker(h.retryInterval)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.doPoll()
		}
	}
}

func (h *healthCheckSource) doPoll() {
	err := h.poll()
	h.mutex.Lock()
	defer func() {
		h.mutex.Unlock()
	}()
	h.lastErr = err
	if err == nil {
		h.gracePeriodReset = time.Now()
		h.numRunsDuringGracePeriod = 0
	}
	h.numRunsDuringGracePeriod++
}

func (h *healthCheckSource) lastError() string {
	if h.lastErr == nil {
		return ""
	}
	return h.lastErr.Error()
}

func toHealthStatus(result conjurehealth.HealthCheckResult) conjurehealth.HealthStatus {
	return conjurehealth.HealthStatus{
		Checks: map[conjurehealth.CheckType]conjurehealth.HealthCheckResult{
			result.Type: result,
		},
	}
}
