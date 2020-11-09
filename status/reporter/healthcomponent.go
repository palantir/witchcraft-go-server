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
	"sync"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
)

var _ HealthComponent = &healthComponent{}

// HealthComponent is an extensible component that represents one part of the whole health picture for a service.
type HealthComponent interface {
	Healthy()
	Warning(message string)
	Error(err error)
	SetHealth(healthState health.HealthState_Value, message *string, params map[string]interface{})
	Status() health.HealthState_Value
	GetHealthCheck() health.HealthCheckResult
}

type healthComponent struct {
	sync.RWMutex

	name    health.CheckType
	state   health.HealthState
	message *string
	params  map[string]interface{}
}

func (r *healthComponent) Healthy() {
	r.SetHealth(health.HealthState_HEALTHY, nil, nil)
}

func (r *healthComponent) Warning(warningMsg string) {
	r.SetHealth(health.HealthState_WARNING, &warningMsg, nil)
}

func (r *healthComponent) Error(err error) {
	errorString := err.Error()
	r.SetHealth(health.HealthState_ERROR, &errorString, nil)
}

func (r *healthComponent) SetHealth(healthState health.HealthState_Value, message *string, params map[string]interface{}) {
	r.Lock()
	defer r.Unlock()

	r.state = health.New_HealthState(healthState)
	r.message = message
	r.params = params
}

// Returns the health status for the health component
func (r *healthComponent) Status() health.HealthState_Value {
	r.RLock()
	defer r.RUnlock()

	return r.state.Value()
}

// Returns the entire HealthCheckResult for the component
func (r *healthComponent) GetHealthCheck() health.HealthCheckResult {
	r.Lock()
	defer r.Unlock()

	var message *string
	params := make(map[string]interface{}, len(r.params))

	if r.message != nil {
		messageCopy := *r.message
		message = &messageCopy
	}
	for key, value := range r.params {
		params[key] = value
	}

	return health.HealthCheckResult{
		Type:    r.name,
		State:   r.state,
		Message: message,
		Params:  params,
	}
}
