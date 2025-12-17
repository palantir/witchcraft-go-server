// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package middleware

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/palantir/witchcraft-go-health/v2/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
)

const (
	endpointFiveHundredsCheckType              = "ENDPOINT_FIVE_HUNDREDS"
	endpointFiveHundredsWindowCount            = 5
	endpointFiveHundredsWindowSize             = 2 * time.Minute
	endpointFiveHundredsMessage                = "There have been HTTP 500s returned by endpoints in every 2 minutes rolling period in the last 10 minutes. This indicates a non-transient error."
	endpointFiveHundredsBrokenEndpointsMessage = "At least one endpoint is consistently failing since startup"
)

type EndpointFiveHundredsHealthCheck struct {
	endpoints     sync.Map // map[string]*endpointFiveHundredsStatus
	alwaysHealthy bool
}

// NewEndpointFiveHundredsHealthCheck creates a new EndpointFiveHundredsHealthCheck and starts its internal goroutine.
// If alwaysHealthy is true, the health check will always return HEALTHY but any failing/broken endpoints will still be
// included in the results params map as information.
func NewEndpointFiveHundredsHealthCheck(ctx context.Context, alwaysHealthy bool) *EndpointFiveHundredsHealthCheck {
	e := &EndpointFiveHundredsHealthCheck{alwaysHealthy: alwaysHealthy}
	ticker := time.Tick(endpointFiveHundredsWindowSize)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker:
				e.shiftWindow()
			}
		}
	}()
	return e
}

func (e *EndpointFiveHundredsHealthCheck) MarkResponse(route wrouter.RouteSpec, code int) {
	endpoint := fmt.Sprintf("%s %s", route.Method, route.PathTemplate)
	statusI, _ := e.endpoints.LoadOrStore(endpoint, &endpointFiveHundredsStatus{})
	status := statusI.(*endpointFiveHundredsStatus)

	if code == http.StatusInternalServerError {
		status.markError()
	} else if code < 400 {
		status.markSuccess()
	}
}

func (e *EndpointFiveHundredsHealthCheck) HealthStatus(context.Context) health.HealthStatus {
	status := e.currentStatus()
	if status == nil {
		return health.HealthStatus{Checks: map[health.CheckType]health.HealthCheckResult{}}
	}
	return health.HealthStatus{Checks: map[health.CheckType]health.HealthCheckResult{status.Type: *status}}
}

func (e *EndpointFiveHundredsHealthCheck) currentStatus() *health.HealthCheckResult {
	hasEndpoints := false
	failingEndpoints := make([]string, 0)
	brokenEndpoints := make([]string, 0)
	e.endpoints.Range(func(key, value interface{}) bool {
		hasEndpoints = true
		endpoint := key.(string)
		status := value.(*endpointFiveHundredsStatus)
		if status.isError() {
			if status.hadSuccessEver {
				failingEndpoints = append(failingEndpoints, endpoint)
			} else {
				brokenEndpoints = append(brokenEndpoints, endpoint)
			}
		}
		return true
	})
	slices.Sort(brokenEndpoints)
	slices.Sort(failingEndpoints)

	// If no endpoints have been called, return an empty status.
	if !hasEndpoints {
		return nil
	}
	result := health.HealthCheckResult{
		Type:  endpointFiveHundredsCheckType,
		State: health.New_HealthState(health.HealthState_HEALTHY),
	}
	if len(failingEndpoints) > 0 || len(brokenEndpoints) > 0 {
		message := endpointFiveHundredsMessage
		if len(brokenEndpoints) > 0 {
			message += "\n" + endpointFiveHundredsBrokenEndpointsMessage
		}
		result.Message = &message
		result.Params = map[string]interface{}{
			"brokenEndpoints":  brokenEndpoints,  // endpoints that have never returned a non-500 status code
			"failingEndpoints": failingEndpoints, // endpoints that have returned 500 status codes in every window
		}
		if !e.alwaysHealthy {
			result.State = health.New_HealthState(health.HealthState_WARNING)
		}
	}
	return &result
}

func (e *EndpointFiveHundredsHealthCheck) shiftWindow() {
	e.endpoints.Range(func(key, value interface{}) bool {
		value.(*endpointFiveHundredsStatus).shift()
		return true
	})
}

// endpointFiveHundredsStatus is a struct that keeps track of the 500 status codes for a single endpoint.
type endpointFiveHundredsStatus struct {
	mutex sync.RWMutex
	// windows[0] is the current window, windows[1] is the previous window, etc.
	// Only windows[1:] are used to determine if the endpoint is unhealthy.
	// true indicates a 500 occurred in that window.
	windows [1 + endpointFiveHundredsWindowCount]bool
	// hadSuccessEver is true if the endpoint has ever returned a non-500 status code.
	hadSuccessEver bool
}

func (e *endpointFiveHundredsStatus) markError() {
	e.mutex.Lock()
	e.windows[0] = true
	e.mutex.Unlock()
}

func (e *endpointFiveHundredsStatus) markSuccess() {
	e.mutex.Lock()
	e.windows[0] = false
	e.hadSuccessEver = true
	e.mutex.Unlock()
}

func (e *endpointFiveHundredsStatus) isError() bool {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	// If any windows are false, the endpoint is healthy.
	for i := 1; i < len(e.windows); i++ {
		if !e.windows[i] {
			return false
		}
	}
	return true
}

func (e *endpointFiveHundredsStatus) shift() {
	e.mutex.Lock()
	// shift each window to the right, starting from the end
	for i := len(e.windows) - 1; i > 0; i-- {
		e.windows[i] = e.windows[i-1]
	}
	// reset the current window
	e.windows[0] = false
	e.mutex.Unlock()
}
