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

package heartbeat

import (
	"context"
	"sync"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/v2/status"
)

// HealthCheckSource is a thread-safe HealthCheckSource based on heartbeats.
// This is used to monitor if some process is continuously running by receiving heartbeats (pings) with timeouts.
// Heartbeats are submitted manually using the Heartbeat or the HeartbeatIfSuccess functions.
// If no heartbeats are observed within the last heartbeatTimeout time frame, returns unhealthy. Otherwise, returns healthy.
// A startup grace period can also be specified, where the check will return repairing if no heartbeats
// were observed but the source was created within the last startupTimeout time frame.
type HealthCheckSource struct {
	lastHeartbeatTime time.Time
	heartbeatTimeout  time.Duration
	heartbeatMutex    sync.RWMutex

	sourceStartupTime time.Time
	startupTimeout    time.Duration

	checkType health.CheckType
}

var _ status.HealthCheckSource = &HealthCheckSource{}

// MustNewHealthCheckSourceWithStartupGracePeriod creates a HealthCheckSource with the specified
// heartbeatTimeout and startupTimeout. The returning HealthCheckResult is of type checkType.
// Panics if heartbeatTimeout is non-positive or if startupTimeout is negative.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewHealthCheckSourceWithStartupGracePeriod(checkType health.CheckType, heartbeatTimeout, startupTimeout time.Duration) *HealthCheckSource {
	healthCheckSource, err := NewHealthCheckSourceWithStartupGracePeriod(checkType, heartbeatTimeout, startupTimeout)
	if err != nil {
		panic(err)
	}
	return healthCheckSource
}

// NewHealthCheckSourceWithStartupGracePeriod creates a HealthCheckSource with the specified
// heartbeatTimeout and startupTimeout. The returning HealthCheckResult is of type checkType.
// Returns an error if heartbeatTimeout is non-positive or if startupTimeout is negative.
func NewHealthCheckSourceWithStartupGracePeriod(checkType health.CheckType, heartbeatTimeout, startupTimeout time.Duration) (*HealthCheckSource, error) {
	if heartbeatTimeout <= 0 {
		return nil, werror.Error("heartbeatTimeout must be positive")
	}
	if startupTimeout < 0 {
		return nil, werror.Error("startupTimeout cannot be negative")
	}
	return &HealthCheckSource{
		heartbeatTimeout:  heartbeatTimeout,
		sourceStartupTime: time.Now(),
		startupTimeout:    startupTimeout,
		checkType:         checkType,
	}, nil
}

// MustNewHealthCheckSource creates a HealthCheckSource with the specified
// heartbeatTimeout and zero as startupTimeout. The returning HealthCheckResult is of type checkType.
// Panics if heartbeatTimeout is non-positive.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewHealthCheckSource(checkType health.CheckType, heartbeatTimeout time.Duration) *HealthCheckSource {
	healthCheckSource, err := NewHealthCheckSource(checkType, heartbeatTimeout)
	if err != nil {
		panic(err)
	}
	return healthCheckSource
}

// NewHealthCheckSource creates a HealthCheckSource with the specified
// heartbeatTimeout and zero as startupTimeout. The returning HealthCheckResult is of type checkType.
// Returns an error if heartbeatTimeout is non-positive.
func NewHealthCheckSource(checkType health.CheckType, heartbeatTimeout time.Duration) (*HealthCheckSource, error) {
	return NewHealthCheckSourceWithStartupGracePeriod(checkType, heartbeatTimeout, 0)
}

// HealthStatus constructs a HealthStatus object based on the submitted heartbeats.
// First checks if there were any heartbeats submitted.
// If there were any heartbeats submitted, checks if the latest one was submitted within the last heartbeatTimeout time frame.
// If it was, returns healthy. If not, returns unhealthy.
// If there were no heartbeats submitted, checks if the source started within the last startupTimeout time frame.
// If it was, returns repairing. If not, returns unhealthy.
func (h *HealthCheckSource) HealthStatus(_ context.Context) health.HealthStatus {
	h.heartbeatMutex.RLock()
	defer h.heartbeatMutex.RUnlock()
	curTime := time.Now()

	if h.lastHeartbeatTime.IsZero() {
		params := map[string]interface{}{
			"sourceStartupTime": h.sourceStartupTime.String(),
			"startupTimeout":    h.startupTimeout.String(),
		}
		if curTime.Sub(h.sourceStartupTime) < h.startupTimeout {
			message := "Waiting for initial heartbeat"
			return health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					h.checkType: {
						Type:    h.checkType,
						State:   health.New_HealthState(health.HealthState_REPAIRING),
						Message: &message,
						Params:  params,
					},
				},
			}
		}

		message := "No heartbeats since startup"
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				h.checkType: {
					Type:    h.checkType,
					State:   health.New_HealthState(health.HealthState_ERROR),
					Message: &message,
					Params:  params,
				},
			},
		}
	}

	params := map[string]interface{}{
		"lastHeartbeatTime": h.lastHeartbeatTime.String(),
		"heartbeatTimeout":  h.heartbeatTimeout.String(),
	}
	if curTime.Sub(h.lastHeartbeatTime) < h.heartbeatTimeout {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				h.checkType: {
					Type:   h.checkType,
					State:  health.New_HealthState(health.HealthState_HEALTHY),
					Params: params,
				},
			},
		}
	}

	message := "Last heartbeat was too long ago"
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			h.checkType: {
				Type:    h.checkType,
				State:   health.New_HealthState(health.HealthState_ERROR),
				Message: &message,
				Params:  params,
			},
		},
	}
}

// Heartbeat submits a heartbeat.
func (h *HealthCheckSource) Heartbeat() {
	h.heartbeatMutex.Lock()
	defer h.heartbeatMutex.Unlock()
	h.lastHeartbeatTime = time.Now()
}

// HeartbeatIfSuccess submits a heartbeat if err is nil.
func (h *HealthCheckSource) HeartbeatIfSuccess(err error) {
	if err != nil {
		return
	}
	h.Heartbeat()
}
