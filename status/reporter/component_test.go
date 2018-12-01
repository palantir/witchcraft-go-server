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
	"errors"
	"testing"

	"github.com/palantir/witchcraft-go-server/conjure/sls/spec/health"
	"github.com/stretchr/testify/assert"
)

const (
	validComponent   = "TEST_COMPONENT"
	invalidComponent = "test-invalid"
)

func setup(t *testing.T) (HealthComponent, HealthReporter) {
	healthReporter := NewHealthReporter()
	healthComponent, err := healthReporter.InitializeHealthComponent(validComponent)
	assert.NoError(t, err)
	return healthComponent, healthReporter
}

func TestProperInitializing(t *testing.T) {
	component, _ := setup(t)
	assert.Equal(t, StartingState, component.Status())
}

func TestHealthy(t *testing.T) {
	component, _ := setup(t)
	component.Healthy()
	assert.Equal(t, HealthyState, component.Status())
}

func TestWarningSetting(t *testing.T) {
	component, healthReporter := setup(t)
	component.Warning("warning message")
	assert.Equal(t, WarningState, component.Status())
	status, found := healthReporter.getHealthCheck(validComponent)
	assert.True(t, found)
	assert.Equal(t, "warning message", *status.Message)
}

func TestErrorSetting(t *testing.T) {
	component, healthReporter := setup(t)
	component.Error(errors.New("err"))
	assert.Equal(t, ErrorState, component.Status())
	status, found := healthReporter.getHealthCheck(validComponent)
	assert.True(t, found)
	assert.Equal(t, "err", *status.Message)
}

func TestSetHealthAndGetHealthResult(t *testing.T) {
	message := "err"
	component, _ := setup(t)
	component.SetHealth(
		health.HealthStateTerminal,
		&message,
		map[string]interface{}{"stack": "trace", "other": errors.New("err2")},
	)
	assert.Equal(t, health.HealthStateTerminal, component.Status())
	result := component.GetHealthCheck()
	assert.Equal(t, "err", *result.Message)
	assert.Equal(t, map[string]interface{}{"stack": "trace", "other": errors.New("err2")}, result.Params)
}

func TestNonCompliantName(t *testing.T) {
	healthReporter := NewHealthReporter()
	_, err := healthReporter.InitializeHealthComponent(invalidComponent)
	assert.Error(t, err)
}
