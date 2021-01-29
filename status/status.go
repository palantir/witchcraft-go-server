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
	"net/http"
	"sync/atomic"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/status"
)

// HealthHandler is responsible for checking the health-check-shared-secret if it is provided and
// invoking a HealthCheckSource if the secret is correct or unset.
type healthHandlerImpl struct {
	healthCheckSharedSecret refreshable.String
	check                   status.HealthCheckSource
	previousHealth          *atomic.Value
	changeHandler           HealthStatusChangeHandler
}

func NewHealthCheckHandler(checkSource status.HealthCheckSource, sharedSecret refreshable.String, healthStatusChangeHandlers []HealthStatusChangeHandler) http.Handler {
	previousHealth := &atomic.Value{}
	previousHealth.Store(health.HealthStatus{})
	allHandlers := []HealthStatusChangeHandler{loggingHealthStatusChangeHandler()}
	if len(healthStatusChangeHandlers) > 0 {
		allHandlers = append(allHandlers, healthStatusChangeHandlers...)
	}
	return &healthHandlerImpl{
		healthCheckSharedSecret: sharedSecret,
		check:                   checkSource,
		previousHealth:          previousHealth,
		changeHandler:           multiHealthStatusChangeHandler(allHandlers),
	}
}

func (h *healthHandlerImpl) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	metadata, newHealthStatusCode := h.computeNewHealthStatus(req)
	previousHealth := h.previousHealth.Load()
	if previousHealth != nil {
		if previousHealthTyped, ok := previousHealth.(health.HealthStatus); ok && checksDiffer(previousHealthTyped.Checks, metadata.Checks) {
			h.changeHandler.HandleHealthStatusChange(req.Context(), previousHealthTyped, metadata)
		}
	}

	h.previousHealth.Store(metadata)

	httpserver.WriteJSONResponse(w, metadata, newHealthStatusCode)
}

func (h *healthHandlerImpl) computeNewHealthStatus(req *http.Request) (health.HealthStatus, int) {
	if sharedSecret := h.healthCheckSharedSecret.CurrentString(); sharedSecret != "" {
		token, err := httpserver.ParseBearerTokenHeader(req)
		if err != nil || sharedSecret != token {
			return health.HealthStatus{}, http.StatusUnauthorized
		}
	}
	metadata := h.check.HealthStatus(req.Context())
	return metadata, status.HealthStatusCode(metadata)
}

func checksDiffer(previousChecks, newChecks map[health.CheckType]health.HealthCheckResult) bool {
	if len(previousChecks) != len(newChecks) {
		return true
	}
	for previousCheckType, previouscheckResult := range previousChecks {
		newCheckResult, checkTypePresent := newChecks[previousCheckType]
		if !checkTypePresent {
			return true
		}
		if previouscheckResult.State != newCheckResult.State {
			return true
		}
	}
	return false
}
