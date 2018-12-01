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
	"github.com/palantir/witchcraft-go-server/conjure/sls/spec/health"
)

var _ HealthComponent = &healthComponent{}

// HealthComponent is an extensible component that represents one part of the whole health picture for a service.
type HealthComponent interface {
	Healthy()
	Warning(message string)
	Error(err error)
	SetHealth(healthState health.HealthState, message *string, params map[string]interface{})
	Status() health.HealthState
	GetHealthCheck() health.HealthCheckResult
}

const (
	StartingState = health.HealthStateRepairing
	HealthyState  = health.HealthStateHealthy
	WarningState  = health.HealthStateWarning
	ErrorState    = health.HealthStateError
)

type healthComponent struct {
	name     health.CheckType
	reporter HealthReporter
}

func (r *healthComponent) Healthy() {
	r.SetHealth(HealthyState, nil, nil)
}

func (r *healthComponent) Warning(warningMsg string) {
	r.SetHealth(WarningState, &warningMsg, nil)
}

func (r *healthComponent) Error(err error) {
	errorString := err.Error()
	r.SetHealth(ErrorState, &errorString, nil)
}

func (r *healthComponent) SetHealth(healthState health.HealthState, message *string, params map[string]interface{}) {
	r.reporter.setHealthCheck(r.name, health.HealthCheckResult{
		Type:    r.name,
		State:   healthState,
		Message: message,
		Params:  params,
	})
}

// Returns the health status for the health component
func (r *healthComponent) Status() health.HealthState {
	return r.GetHealthCheck().State
}

// Returns the entire HealthCheckResult for the component
func (r *healthComponent) GetHealthCheck() health.HealthCheckResult {
	result, found := r.reporter.getHealthCheck(r.name)
	if !found {
		// This should never happen so long as the reporter controls the instantiation of components
		// and registers them into the Checks map during instantiation.
		panic("This component was never registered with its HealthReporter")
	}
	return result
}
