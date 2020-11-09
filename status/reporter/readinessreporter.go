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

package reporter

import (
	"context"
	"net/http"
	"sync"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/v2/status"
)

// Reporter allows for the creation and aggregation of custom readiness checks.
type Reporter interface {
	status.Source
	InitializeReadinessComponent(ctx context.Context, name ComponentName) (Component, error)
	GetReadinessComponent(name ComponentName) (Component, bool)
	UnregisterReadinessComponent(ctx context.Context, name ComponentName) bool
}

type readinessReporter struct {
	// mutex protects access to `readinessComponents`.
	mutex               sync.RWMutex
	readinessComponents map[ComponentName]Component
}

func NewReadinessReporter() Reporter {
	return &readinessReporter{
		readinessComponents: make(map[ComponentName]Component),
	}
}

func (r *readinessReporter) Status() (respStatus int, metadata interface{}) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Attempt to return the highest "unready" response code. If none exist, return ready.
	highestUnreadyRespStatus := 0
	aggregatedMetadata := make(map[ComponentName]interface{})
	for name, component := range r.readinessComponents {
		respStatus, metadata := component.Status()
		// Response codes within [200, 399] are considered ready.
		//   Refer to readiness section of the SLS specification.
		if (respStatus < 200 || respStatus >= 400) && respStatus > highestUnreadyRespStatus {
			highestUnreadyRespStatus = respStatus
		}
		aggregatedMetadata[name] = metadata
	}
	if highestUnreadyRespStatus == 0 {
		return http.StatusOK, aggregatedMetadata
	}
	return highestUnreadyRespStatus, aggregatedMetadata
}

func (r *readinessReporter) InitializeReadinessComponent(ctx context.Context, name ComponentName) (Component, error) {
	ctx = svc1log.WithLoggerParams(ctx, svc1log.SafeParam("readinessComponent", name))
	var component Component
	component = &readinessComponent{
		name: name,
		// Initialize to not ready, mirroring how health components are initialized to REPAIRING.
		status: http.StatusInternalServerError,
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, exists := r.readinessComponents[name]; exists {
		return nil, werror.ErrorWithContextParams(ctx, "readiness component already exists")
	}

	r.readinessComponents[name] = component
	svc1log.FromContext(ctx).Info("Registered new readiness component.")
	return component, nil
}

func (r *readinessReporter) GetReadinessComponent(name ComponentName) (Component, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	component, exists := r.readinessComponents[name]
	return component, exists
}

func (r *readinessReporter) UnregisterReadinessComponent(ctx context.Context, name ComponentName) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, exists := r.readinessComponents[name]; !exists {
		return false
	}
	delete(r.readinessComponents, name)
	svc1log.FromContext(ctx).Info("Unregistered readiness component.", svc1log.SafeParam("readinessComponent", name))
	return true
}
