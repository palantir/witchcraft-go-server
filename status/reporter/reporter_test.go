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
	"testing"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/spec/health"
	"github.com/stretchr/testify/assert"
)

const testCheckType = health.CheckType("TEST")

func TestSetHealthy(t *testing.T) {
	reporter := newHealthReporter()
	reporter.setHealthCheck(testCheckType, health.HealthCheckResult{
		Type:  testCheckType,
		State: health.HealthStateHealthy,
	})

	assert.Equal(t, reporter.currentStatus.Checks[testCheckType].State, health.HealthStateHealthy)
}

func TestSetError(t *testing.T) {
	reporter := newHealthReporter()
	reporter.setHealthCheck(testCheckType, health.HealthCheckResult{
		Type:  testCheckType,
		State: health.HealthStateError,
	})

	assert.Equal(t, reporter.currentStatus.Checks[testCheckType].State, health.HealthStateError)
}

func TestGetHealthy(t *testing.T) {
	reporter := newHealthReporter()
	reporter.currentStatus = health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:  testCheckType,
				State: health.HealthStateHealthy,
			},
		},
	}

	status, found := reporter.getHealthCheck(testCheckType)
	assert.True(t, found)
	assert.Equal(t, status.State, health.HealthStateHealthy)
}

func TestGetError(t *testing.T) {
	reporter := newHealthReporter()
	reporter.currentStatus = health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			testCheckType: {
				Type:  testCheckType,
				State: health.HealthStateError,
			},
		},
	}

	status, found := reporter.getHealthCheck(testCheckType)
	assert.True(t, found)
	assert.Equal(t, status.State, health.HealthStateError)
}

func TestGetNotFound(t *testing.T) {
	reporter := NewHealthReporter()
	_, found := reporter.getHealthCheck(testCheckType)
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

// Test that setHealthCheckAndComponentIfAbsent is idempotent
func TestSetIfNotExist(t *testing.T) {
	reporter := newHealthReporter()
	iterations := 10
	testNames := []string{"a", "b", "c", "d", "e", "f", "g"}
	returnChan := make(chan bool, iterations*len(testNames))
	for _, n := range testNames {
		checkType := health.CheckType(n)
		status := health.HealthCheckResult{
			Type:  checkType,
			State: StartingState,
		}
		component := &healthComponent{
			checkType,
			reporter,
		}
		for i := 0; i < iterations; i++ {
			go func() {
				returnChan <- reporter.setHealthCheckAndComponentIfAbsent(checkType, component, status)
			}()
		}
	}
	sets := 0
	total := 0
	for r := range returnChan {
		if r {
			sets++
		}
		total++
		if total == iterations*len(testNames) {
			assert.Equal(t, len(testNames), sets)
			for _, n := range testNames {
				status, found := reporter.getHealthCheck(health.CheckType(n))
				assert.True(t, found)
				assert.Equal(t, StartingState, status.State)
			}
			return
		}
	}
	assert.Fail(t, "Did get the correct number of total calls", total)
}
