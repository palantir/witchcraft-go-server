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

package reporter

import (
	"context"
	"regexp"
	"sync"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/status"
)

const slsHealthNameRegex = "^[A-Z_]+$"

var _ HealthReporter = &healthReporter{}

type HealthReporter interface {
	status.HealthCheckSource
	InitializeHealthComponent(name string) (HealthComponent, error)
	GetHealthComponent(name string) (HealthComponent, bool)
	UnregisterHealthComponent(name string) bool
}

type healthReporter struct {
	mutex            sync.RWMutex
	healthComponents map[health.CheckType]HealthComponent
}

// NewHealthReporter - creates a new HealthReporter; an implementation of status.HealthCheckSource
// which initializes HealthComponents to report on the health of each individual health.CheckType.
func NewHealthReporter() HealthReporter {
	return newHealthReporter()
}

func newHealthReporter() *healthReporter {
	return &healthReporter{
		healthComponents: make(map[health.CheckType]HealthComponent),
	}
}

// InitializeHealthComponent - Creates a health component for the given name where the component should be stated as
// initializing until a future call modifies the initializing status. The created health component is stored in the
// HealthReporter and can be fetched later by name via GetHealthComponent.
// Returns health.HealthState_ERROR if the component name is non-SLS compliant, or the name is already in use
func (r *healthReporter) InitializeHealthComponent(name string) (HealthComponent, error) {
	isSLSCompliant := regexp.MustCompile(slsHealthNameRegex).MatchString
	if !isSLSCompliant(name) {
		return nil, werror.Error("component name is not a valid SLS health component name",
			werror.SafeParam("name", name),
			werror.SafeParam("validPattern", slsHealthNameRegex))
	}
	componentName := health.CheckType(name)
	healthComponent := &healthComponent{
		name:  componentName,
		state: health.New_HealthState(health.HealthState_REPAIRING),
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, ok := r.healthComponents[componentName]; ok {
		return nil, werror.Error("Health component name already exists", werror.SafeParam("name", name))
	}

	r.healthComponents[componentName] = healthComponent
	return healthComponent, nil
}

// GetHealthComponent - Gets an initialized health component by name.
func (r *healthReporter) GetHealthComponent(name string) (HealthComponent, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	c, ok := r.healthComponents[health.CheckType(name)]
	return c, ok
}

// UnregisterHealthComponent - Removes a health component by name if already initialized.
func (r *healthReporter) UnregisterHealthComponent(name string) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var changesMade bool
	componentName := health.CheckType(name)

	if _, present := r.healthComponents[componentName]; present {
		delete(r.healthComponents, componentName)
		changesMade = true
	}

	return changesMade
}

// HealthStatus returns a copy of the current HealthStatus, and cannot be used to modify the current state
func (r *healthReporter) HealthStatus(ctx context.Context) health.HealthStatus {
	checks := make(map[health.CheckType]health.HealthCheckResult, len(r.healthComponents))
	for checkType, component := range r.healthComponents {
		checks[checkType] = component.GetHealthCheck()
	}
	return health.HealthStatus{Checks: checks}
}

func (r *healthReporter) getHealthCheck(check health.CheckType) (health.HealthCheckResult, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	component, found := r.healthComponents[check]
	if !found {
		return health.HealthCheckResult{}, false
	}
	return component.GetHealthCheck(), found
}
