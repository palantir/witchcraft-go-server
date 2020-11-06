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
	"testing"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/stretchr/testify/assert"
)

const testCheckType = health.CheckType("TEST")

func TestGetHealthy(t *testing.T) {
	reporter := newHealthReporter()
	reporter.healthComponents = map[health.CheckType]HealthComponent{
		testCheckType: &healthComponent{
			name:  testCheckType,
			state: health.New_HealthState(health.HealthState_HEALTHY),
		},
	}

	status, found := reporter.getHealthCheck(testCheckType)
	assert.True(t, found)
	assert.Equal(t, status.State.Value(), health.HealthState_HEALTHY)
}

func TestGetError(t *testing.T) {
	reporter := newHealthReporter()
	reporter.healthComponents = map[health.CheckType]HealthComponent{
		testCheckType: &healthComponent{
			name:  testCheckType,
			state: health.New_HealthState(health.HealthState_ERROR),
		},
	}

	status, found := reporter.getHealthCheck(testCheckType)
	assert.True(t, found)
	assert.Equal(t, status.State.Value(), health.HealthState_ERROR)
}

func TestGetNotFound(t *testing.T) {
	reporter := NewHealthReporter()
	status := reporter.HealthStatus(context.TODO())
	_, found := status.Checks[testCheckType]
	assert.False(t, found)
}

func TestInitializeCollision(t *testing.T) {
	reporter := NewHealthReporter()
	component, err := reporter.InitializeHealthComponent(validComponent)
	assert.NotNil(t, component)
	assert.NoError(t, err)

	componentCollision, err := reporter.InitializeHealthComponent(validComponent)
	assert.Nil(t, componentCollision)
	assert.Error(t, err)
}

func TestInitializeThenGet(t *testing.T) {
	reporter := NewHealthReporter()
	component, err := reporter.InitializeHealthComponent(validComponent)
	assert.NotNil(t, component)
	assert.NoError(t, err)

	fromGet, ok := reporter.GetHealthComponent(validComponent)
	assert.True(t, ok)
	assert.Equal(t, fromGet, component)
}

func TestUnregisterThenGet(t *testing.T) {
	reporter := newHealthReporter()
	component, err := reporter.InitializeHealthComponent(validComponent)
	assert.NotNil(t, component)
	assert.NoError(t, err)

	assert.True(t, reporter.UnregisterHealthComponent(validComponent))

	_, present := reporter.GetHealthComponent(validComponent)
	assert.False(t, present)
}

func TestUnregisterThenGetComponentStatus(t *testing.T) {
	reporter := newHealthReporter()
	component, err := reporter.InitializeHealthComponent(validComponent)
	assert.NotNil(t, component)
	assert.NoError(t, err)

	assert.True(t, reporter.UnregisterHealthComponent(validComponent))

	_, present := reporter.GetHealthComponent(validComponent)
	assert.False(t, present)

	assert.NotPanics(t, func() { component.Status() })
}

func TestUnregisterOnUninitialized(t *testing.T) {
	reporter := newHealthReporter()
	assert.False(t, reporter.UnregisterHealthComponent(validComponent))
}
