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

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
	"github.com/palantir/witchcraft-go-server/status/health/periodic"
	"github.com/palantir/witchcraft-go-server/status/reporter"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/palantir/witchcraft-go-server/witchcraft/refreshable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddHealthCheckSources verifies that custom health check sources report via the health endpoint.
func TestAddHealthCheckSources(t *testing.T) {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, ioutil.Discard, func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server {
		return createTestServer(t, initFn, installCfg, logOutputBuffer).WithHealth(healthCheckWithType{typ: "FOO"}, healthCheckWithType{typ: "BAR"})
	})

	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint))
	require.NoError(t, err)

	bytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResults health.HealthStatus
	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			health.CheckType("FOO"): {
				Type:    health.CheckType("FOO"),
				State:   health.HealthStateHealthy,
				Message: nil,
				Params:  make(map[string]interface{}),
			},
			health.CheckType("BAR"): {
				Type:    health.CheckType("BAR"),
				State:   health.HealthStateHealthy,
				Message: nil,
				Params:  make(map[string]interface{}),
			},
			health.CheckType("CONFIG_RELOAD"): {
				Type:   health.CheckType("CONFIG_RELOAD"),
				State:  health.HealthStateHealthy,
				Params: make(map[string]interface{}),
			},
			health.CheckType("SERVER_STATUS"): {
				Type:    health.CheckType("SERVER_STATUS"),
				State:   health.HealthStateHealthy,
				Message: nil,
				Params:  make(map[string]interface{}),
			},
		},
	}, healthResults)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestHealthReporter verifies the behavior of the reporter package.
// We create 4 health components, flip their health/unhealthy states, and ensure the aggregated states
// returned by the health endpoint reflect what we have set.
func TestHealthReporter(t *testing.T) {
	healthReporter := reporter.NewHealthReporter()

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, ioutil.Discard, func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server {
		return createTestServer(t, initFn, installCfg, logOutputBuffer).WithHealth(healthReporter)
	})

	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	// Initialize health components and set their health
	healthyComponents := []string{"COMPONENT_A", "COMPONENT_B"}
	unhealthyComponents := []string{"COMPONENT_C", "COMPONENT_D"}
	errString := "Something failed"
	var wg sync.WaitGroup
	wg.Add(len(healthyComponents) + len(unhealthyComponents))
	for _, n := range healthyComponents {
		go func(healthReporter reporter.HealthReporter, name string) {
			defer wg.Done()
			component, err := healthReporter.InitializeHealthComponent(name)
			if err != nil {
				panic(fmt.Errorf("failed to initialize %s health reporter: %v", name, err))
			}
			if component.Status() != reporter.StartingState {
				panic(fmt.Errorf("expected reporter to be in REPAIRING before being marked healthy, got %s", component.Status()))
			}
			component.Healthy()
			if component.Status() != reporter.HealthyState {
				panic(fmt.Errorf("expected reporter to be in HEALTHY after being marked healthy, got %s", component.Status()))
			}
		}(healthReporter, n)
	}
	for _, n := range unhealthyComponents {
		go func(healthReporter reporter.HealthReporter, name string) {
			defer wg.Done()
			component, err := healthReporter.InitializeHealthComponent(name)
			if err != nil {
				panic(fmt.Errorf("failed to initialize %s health reporter: %v", name, err))
			}
			if component.Status() != reporter.StartingState {
				panic(fmt.Errorf("expected reporter to be in REPAIRING before being marked healthy, got %s", component.Status()))
			}
			component.Error(errors.New(errString))
			if component.Status() != reporter.ErrorState {
				panic(fmt.Errorf("expected reporter to be in ERROR after being marked with error, got %s", component.Status()))
			}
		}(healthReporter, n)
	}
	wg.Wait()

	// Validate GetHealthComponent
	component, ok := healthReporter.GetHealthComponent(healthyComponents[0])
	assert.True(t, ok)
	assert.Equal(t, reporter.HealthyState, component.Status())

	// Validate health
	resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint))
	require.NoError(t, err)

	bytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResults health.HealthStatus
	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			health.CheckType("COMPONENT_A"): {
				Type:    health.CheckType("COMPONENT_A"),
				State:   reporter.HealthyState,
				Message: nil,
				Params:  make(map[string]interface{}),
			},
			health.CheckType("COMPONENT_B"): {
				Type:    health.CheckType("COMPONENT_B"),
				State:   reporter.HealthyState,
				Message: nil,
				Params:  make(map[string]interface{}),
			},
			health.CheckType("COMPONENT_C"): {
				Type:    health.CheckType("COMPONENT_C"),
				State:   reporter.ErrorState,
				Message: &errString,
				Params:  make(map[string]interface{}),
			},
			health.CheckType("COMPONENT_D"): {
				Type:    health.CheckType("COMPONENT_D"),
				State:   reporter.ErrorState,
				Message: &errString,
				Params:  make(map[string]interface{}),
			},
			health.CheckType("CONFIG_RELOAD"): {
				Type:   health.CheckType("CONFIG_RELOAD"),
				State:  health.HealthStateHealthy,
				Params: make(map[string]interface{}),
			},
			health.CheckType("SERVER_STATUS"): {
				Type:   health.CheckType("SERVER_STATUS"),
				State:  reporter.HealthyState,
				Params: make(map[string]interface{}),
			},
		},
	}, healthResults)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestPeriodicHealthSource tests that basic periodic healthcheck wiring works properly. Unit testing covers the grace
// period logic - this test covers the plumbing.
func TestPeriodicHealthSource(t *testing.T) {
	inputSource := periodic.Source{
		Checks: map[health.CheckType]periodic.CheckFunc{
			"HEALTHY_CHECK": func(ctx context.Context) *health.HealthCheckResult {
				return &health.HealthCheckResult{
					Type:  "HEALTHY_CHECK",
					State: health.HealthStateHealthy,
				}
			},
			"ERROR_CHECK": func(ctx context.Context) *health.HealthCheckResult {
				return &health.HealthCheckResult{
					Type:    "ERROR_CHECK",
					State:   health.HealthStateError,
					Message: stringPtr("something went wrong"),
					Params:  map[string]interface{}{"foo": "bar"},
				}
			},
		},
	}
	expectedStatus := health.HealthStatus{Checks: map[health.CheckType]health.HealthCheckResult{
		"HEALTHY_CHECK": {
			Type:    "HEALTHY_CHECK",
			State:   health.HealthStateHealthy,
			Message: nil,
			Params:  make(map[string]interface{}),
		},
		"ERROR_CHECK": {
			Type:    "ERROR_CHECK",
			State:   health.HealthStateError,
			Message: stringPtr("No successful checks during 1m0s grace period: something went wrong"),
			Params:  map[string]interface{}{"foo": "bar"},
		},
		health.CheckType("CONFIG_RELOAD"): {
			Type:   health.CheckType("CONFIG_RELOAD"),
			State:  health.HealthStateHealthy,
			Params: make(map[string]interface{}),
		},
		health.CheckType("SERVER_STATUS"): {
			Type:    health.CheckType("SERVER_STATUS"),
			State:   health.HealthStateHealthy,
			Message: nil,
			Params:  make(map[string]interface{}),
		},
	}}
	periodicHealthCheckSource := periodic.FromHealthCheckSource(context.Background(), time.Second*60, time.Millisecond*1, inputSource)

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, ioutil.Discard, func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server {
		return createTestServer(t, initFn, installCfg, logOutputBuffer).WithHealth(periodicHealthCheckSource)
	})

	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	// Wait for checks to run at least once
	time.Sleep(5 * time.Millisecond)

	resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint))
	require.NoError(t, err)

	bytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResults health.HealthStatus
	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, expectedStatus, healthResults)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestHealthSharedSecret verifies that a non-empty health check shared secret is required by the endpoint when configured.
// If the secret is not provided or is incorrect, the endpoint returns 401 Unauthorized.
func TestHealthSharedSecret(t *testing.T) {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, ioutil.Discard, func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server {
		return createTestServer(t, initFn, installCfg, logOutputBuffer).
			WithHealth(emptyHealthCheckSource{}).
			WithDisableGoRuntimeMetrics().
			WithRuntimeConfig(config.Runtime{
				HealthChecks: config.HealthChecksConfig{
					SharedSecret: "top-secret",
				},
			})
	})

	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	client := testServerClient()
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint), nil)
	require.NoError(t, err)
	request.Header.Set("Authorization", "Bearer top-secret")
	resp, err := client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	bytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResults health.HealthStatus
	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			health.CheckType("CONFIG_RELOAD"): {
				Type:   health.CheckType("CONFIG_RELOAD"),
				State:  health.HealthStateHealthy,
				Params: make(map[string]interface{}),
			},
			health.CheckType("SERVER_STATUS"): {
				Type:   health.CheckType("SERVER_STATUS"),
				State:  reporter.HealthyState,
				Params: make(map[string]interface{}),
			},
		},
	}, healthResults)

	// bad header should return 401
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", "bad-secret"))
	resp, err = client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestRuntimeConfigReloadHealth verifies that runtime configuration that is invalid when strict unmarshal mode is true
// does not produces an error health check if strict unmarshal mode is not specified (since default value is false).
func TestRuntimeConfigReloadHealthWithStrictUnmarshalFalse(t *testing.T) {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	validCfgYML := `logging:
  level: info
`
	invalidCfgYML := `
invalid-key: invalid-value
`
	runtimeConfigRefreshable := refreshable.NewDefaultRefreshable([]byte(validCfgYML))
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, ioutil.Discard, func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server {
		return createTestServer(t, initFn, installCfg, logOutputBuffer).
			WithRuntimeConfigProvider(runtimeConfigRefreshable).
			WithDisableGoRuntimeMetrics()
	})

	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	client := testServerClient()
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint), nil)
	require.NoError(t, err)
	resp, err := client.Do(request)
	require.NoError(t, err)

	bytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResults health.HealthStatus
	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			health.CheckType("CONFIG_RELOAD"): {
				Type:   health.CheckType("CONFIG_RELOAD"),
				State:  health.HealthStateHealthy,
				Params: make(map[string]interface{}),
			},
			health.CheckType("SERVER_STATUS"): {
				Type:   health.CheckType("SERVER_STATUS"),
				State:  reporter.HealthyState,
				Params: make(map[string]interface{}),
			},
		},
	}, healthResults)

	// write invalid runtime config and observe health check go unhealthy
	err = runtimeConfigRefreshable.Update([]byte(invalidCfgYML))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	request, err = http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint), nil)
	require.NoError(t, err)
	resp, err = client.Do(request)
	require.NoError(t, err)

	bytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			health.CheckType("CONFIG_RELOAD"): {
				Type:   health.CheckType("CONFIG_RELOAD"),
				State:  health.HealthStateHealthy,
				Params: make(map[string]interface{}),
			},
			health.CheckType("SERVER_STATUS"): {
				Type:   health.CheckType("SERVER_STATUS"),
				State:  reporter.HealthyState,
				Params: make(map[string]interface{}),
			},
		},
	}, healthResults)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestRuntimeConfigReloadHealth verifies that runtime configuration that is invalid when strict unmarshal mode is true
// produces an error health check.
func TestRuntimeConfigReloadHealthWithStrictUnmarshalTrue(t *testing.T) {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	validCfgYML := `logging:
  level: info
`
	invalidCfgYML := `
invalid-key: invalid-value
`
	runtimeConfigRefreshable := refreshable.NewDefaultRefreshable([]byte(validCfgYML))
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, ioutil.Discard, func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server {
		return createTestServer(t, initFn, installCfg, logOutputBuffer).
			WithRuntimeConfigProvider(runtimeConfigRefreshable).
			WithDisableGoRuntimeMetrics().
			WithStrictUnmarshalConfig()
	})

	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	client := testServerClient()
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint), nil)
	require.NoError(t, err)
	resp, err := client.Do(request)
	require.NoError(t, err)

	bytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var healthResults health.HealthStatus
	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			health.CheckType("CONFIG_RELOAD"): {
				Type:   health.CheckType("CONFIG_RELOAD"),
				State:  health.HealthStateHealthy,
				Params: make(map[string]interface{}),
			},
			health.CheckType("SERVER_STATUS"): {
				Type:   health.CheckType("SERVER_STATUS"),
				State:  reporter.HealthyState,
				Params: make(map[string]interface{}),
			},
		},
	}, healthResults)

	// write invalid runtime config and observe health check go unhealthy
	err = runtimeConfigRefreshable.Update([]byte(invalidCfgYML))
	require.NoError(t, err)
	time.Sleep(500 * time.Millisecond)

	request, err = http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.HealthEndpoint), nil)
	require.NoError(t, err)
	resp, err = client.Do(request)
	require.NoError(t, err)

	bytes, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	err = json.Unmarshal(bytes, &healthResults)
	require.NoError(t, err)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			health.CheckType("CONFIG_RELOAD"): {
				Type:    health.CheckType("CONFIG_RELOAD"),
				State:   health.HealthStateError,
				Params:  make(map[string]interface{}),
				Message: stringPtr("Refreshable validation failed, please look at service logs for more information."),
			},
			health.CheckType("SERVER_STATUS"): {
				Type:   health.CheckType("SERVER_STATUS"),
				State:  reporter.HealthyState,
				Params: make(map[string]interface{}),
			},
		},
	}, healthResults)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

type emptyHealthCheckSource struct{}

func (emptyHealthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return health.HealthStatus{}
}

type healthCheckWithType struct {
	typ health.CheckType
}

func (cwt healthCheckWithType) HealthStatus(_ context.Context) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			cwt.typ: {
				Type:    cwt.typ,
				State:   health.HealthStateHealthy,
				Message: nil,
				Params:  make(map[string]interface{}),
			},
		},
	}
}

func stringPtr(s string) *string {
	return &s
}
