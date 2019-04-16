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

	"github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

const slsHealthNameRegex = "^[A-Z_]+$"

var _ HealthReporter = &healthReporter{}

type HealthReporter interface {
	status.HealthCheckSource
	InitializeHealthComponent(name string) (HealthComponent, error)
	GetHealthComponent(name string) (HealthComponent, bool)
	UnregisterHealthComponent(name string) bool

	setHealthCheck(component health.CheckType, status health.HealthCheckResult)
	setHealthCheckAndComponentIfAbsent(componentName health.CheckType, component HealthComponent, status health.HealthCheckResult) bool
	getHealthCheck(component health.CheckType) (health.HealthCheckResult, bool)
}

type healthReporter struct {
	currentStatus    health.HealthStatus
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
		currentStatus: health.HealthStatus{
			Checks: make(map[health.CheckType]health.HealthCheckResult),
		},
		healthComponents: make(map[health.CheckType]HealthComponent),
	}
}

// InitializeHealthComponent - Creates a health component for the given name where the component should be stated as
// initializing until a future call modifies the initializing status. The created health component is stored in the
// HealthReporter and can be fetched later by name via GetHealthComponent.
// Returns ErrorState if the component name is non-SLS compliant, or the name is already in use
func (r *healthReporter) InitializeHealthComponent(name string) (HealthComponent, error) {
	isSLSCompliant := regexp.MustCompile(slsHealthNameRegex).MatchString
	if !isSLSCompliant(name) {
		return nil, werror.Error("component name is not a valid SLS health component name",
			werror.SafeParam("name", name),
			werror.SafeParam("validPattern", slsHealthNameRegex))
	}
	componentName := health.CheckType(name)
	status := health.HealthCheckResult{
		Type:  componentName,
		State: StartingState,
	}
	healthComponent := &healthComponent{
		name:     componentName,
		reporter: r,
	}
	if r.setHealthCheckAndComponentIfAbsent(componentName, healthComponent, status) {
		return healthComponent, nil
	}
	return nil, werror.Error("Health component name already exists", werror.SafeParam("name", name))
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

	if _, present := r.currentStatus.Checks[componentName]; present {
		delete(r.currentStatus.Checks, componentName)
		changesMade = true
	}

	return changesMade
}

// HealthStatus returns a copy of the current HealthStatus, and cannot be used to modify the current state
func (r *healthReporter) HealthStatus(ctx context.Context) health.HealthStatus {
	checks := make(map[health.CheckType]health.HealthCheckResult, len(r.currentStatus.Checks))
	for key, value := range r.currentStatus.Checks {
		checks[key] = value
	}
	return health.HealthStatus{Checks: checks}
}

func (r *healthReporter) setHealthCheckAndComponentIfAbsent(componentName health.CheckType, component HealthComponent, status health.HealthCheckResult) bool {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if _, found := r.currentStatus.Checks[componentName]; found {
		return false
	}
	r.currentStatus.Checks[componentName] = status
	r.healthComponents[componentName] = component
	return true
}

func (r *healthReporter) setHealthCheck(component health.CheckType, status health.HealthCheckResult) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.currentStatus.Checks[component] = status
}

func (r *healthReporter) getHealthCheck(component health.CheckType) (health.HealthCheckResult, bool) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	check, found := r.currentStatus.Checks[component]
	return check, found
}
