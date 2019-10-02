// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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

package heartbeat

import (
	"context"
	"testing"
	"time"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testCheckType health.CheckType = "TEST_CHECK"
)

func TestHealthCheckSource_NoHeartbeats_Repairing(t *testing.T) {
	source, err := NewHealthCheckSourceWithStartupGracePeriod(testCheckType, 25*time.Millisecond, 25*time.Millisecond)
	require.NoError(t, err)
	status := source.HealthStatus(context.Background())
	require.Equal(t, 1, len(status.Checks))
	check, hasCheck := status.Checks[testCheckType]
	require.True(t, hasCheck)
	assert.Equal(t, health.HealthStateRepairing, check.State)
}

func TestHealthCheckSource_NoHeartbeats_Error(t *testing.T) {
	source, err := NewHealthCheckSourceWithStartupGracePeriod(testCheckType, 25*time.Millisecond, 25*time.Millisecond)
	require.NoError(t, err)
	<-time.After(50 * time.Millisecond)
	status := source.HealthStatus(context.Background())
	require.Equal(t, 1, len(status.Checks))
	check, hasCheck := status.Checks[testCheckType]
	require.True(t, hasCheck)
	assert.Equal(t, health.HealthStateError, check.State)
}

func TestHealthCheckSource_WithHeartbeats_Healthy(t *testing.T) {
	source, err := NewHealthCheckSource(testCheckType, 25*time.Millisecond)
	require.NoError(t, err)
	source.Heartbeat()
	status := source.HealthStatus(context.Background())
	require.Equal(t, 1, len(status.Checks))
	check, hasCheck := status.Checks[testCheckType]
	require.True(t, hasCheck)
	assert.Equal(t, health.HealthStateHealthy, check.State)
}

func TestHealthCheckSource_WithHeartbeats_Error(t *testing.T) {
	source, err := NewHealthCheckSource(testCheckType, 25*time.Millisecond)
	require.NoError(t, err)
	source.Heartbeat()
	<-time.After(50 * time.Millisecond)
	status := source.HealthStatus(context.Background())
	require.Equal(t, 1, len(status.Checks))
	check, hasCheck := status.Checks[testCheckType]
	require.True(t, hasCheck)
	assert.Equal(t, health.HealthStateError, check.State)
}

func TestHealthCheckSource_WithHeartbeats_HealthyThenErrorThenHealthy(t *testing.T) {
	source, err := NewHealthCheckSource(testCheckType, 25*time.Millisecond)
	require.NoError(t, err)
	source.Heartbeat()
	status := source.HealthStatus(context.Background())
	require.Equal(t, 1, len(status.Checks))
	check, hasCheck := status.Checks[testCheckType]
	require.True(t, hasCheck)
	assert.Equal(t, health.HealthStateHealthy, check.State)

	<-time.After(50 * time.Millisecond)
	status = source.HealthStatus(context.Background())
	require.Equal(t, 1, len(status.Checks))
	check, hasCheck = status.Checks[testCheckType]
	require.True(t, hasCheck)
	assert.Equal(t, health.HealthStateError, check.State)

	source.Heartbeat()
	status = source.HealthStatus(context.Background())
	require.Equal(t, 1, len(status.Checks))
	check, hasCheck = status.Checks[testCheckType]
	require.True(t, hasCheck)
	assert.Equal(t, health.HealthStateHealthy, check.State)
}
