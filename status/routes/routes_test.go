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

package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/sources"
	healthstatus "github.com/palantir/witchcraft-go-health/status"
	"github.com/palantir/witchcraft-go-server/v2/status"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/wresource"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-server/v2/wrouter/whttprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddStatusRoutes(t *testing.T) {
	type testMetadata struct {
		Message string
		Value   int
	}

	for i, tc := range []struct {
		endpoint  string
		routeFunc func(resource wresource.Resource, source healthstatus.Source) error
		status    int
		metadata  testMetadata
	}{
		{
			endpoint:  status.LivenessEndpoint,
			routeFunc: AddLivenessRoutes,
			status:    http.StatusOK,
			metadata: testMetadata{
				Message: "hello",
				Value:   13,
			},
		},
		{
			endpoint:  status.ReadinessEndpoint,
			routeFunc: AddReadinessRoutes,
			status:    http.StatusServiceUnavailable,
			metadata: testMetadata{
				Message: "goodbye",
				Value:   1,
			},
		},
	} {
		func() {
			r := wrouter.New(whttprouter.New(), nil)
			resource := wresource.New("test", r)
			err := tc.routeFunc(resource, statusFunc(func() (int, interface{}) {
				return tc.status, tc.metadata
			}))
			require.NoError(t, err, "Case %d", i)

			server := httptest.NewServer(r)
			defer server.Close()

			resp, err := http.Get(strings.Join([]string{server.URL, tc.endpoint}, "/"))
			require.NoError(t, err, "Case %d", i)
			assert.Equal(t, tc.status, resp.StatusCode, "Case %d", i)

			bytes, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err, "Case %d", i)
			var gotObj testMetadata
			err = json.Unmarshal(bytes, &gotObj)
			require.NoError(t, err, "Case %d", i)
			assert.Equal(t, tc.metadata, gotObj)
		}()
	}
}

func TestAddHealthRoute(t *testing.T) {
	for _, test := range []struct {
		name           string
		metadata       health.HealthStatus
		sharedSecret   string
		actualSecret   string
		expectedStatus int
	}{
		{
			"zero healthy check",
			health.HealthStatus{
				Checks: make(map[health.CheckType]health.HealthCheckResult),
			},
			"",
			"",
			200,
		},
		{
			"one healthy check",
			health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"SCHEMA_VERSION": {
						Type:   "SCHEMA_VERSION",
						State:  health.New_HealthState(health.HealthState_HEALTHY),
						Params: make(map[string]interface{}),
					},
				},
			},
			"",
			"secret shouldn't matter",
			200,
		},
		{
			"one healthy and one error",
			health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"SCHEMA_VERSION": {
						Type:   "SCHEMA_VERSION",
						State:  health.New_HealthState(health.HealthState_HEALTHY),
						Params: make(map[string]interface{}),
					},
					"SERVER_ERROR": {
						Type:   "SERVER_ERROR",
						State:  health.New_HealthState(health.HealthState_ERROR),
						Params: make(map[string]interface{}),
					},
				},
			},
			"",
			"",
			522,
		},
		{
			"health, terminal, error",
			health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"SCHEMA_VERSION": {
						Type:   "SCHEMA_VERSION",
						State:  health.New_HealthState(health.HealthState_HEALTHY),
						Params: make(map[string]interface{}),
					},
					"SERVER_TERMINAL": {
						Type:   "SERVER_TERMINAL",
						State:  health.New_HealthState(health.HealthState_TERMINAL),
						Params: make(map[string]interface{}),
					},
					"SERVER_ERROR": {
						Type:   "SERVER_ERROR",
						State:  health.New_HealthState(health.HealthState_ERROR),
						Params: make(map[string]interface{}),
					},
				},
			},
			"",
			"",
			523,
		},
		{
			"one healthy check with secret",
			health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"SCHEMA_VERSION": {
						Type:   "SCHEMA_VERSION",
						State:  health.New_HealthState(health.HealthState_HEALTHY),
						Params: make(map[string]interface{}),
					},
				},
			},
			"top-secret",
			"top-secret",
			200,
		},
		{
			"one healthy check with incorrect secret",
			health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"SCHEMA_VERSION": {
						Type:   "SCHEMA_VERSION",
						State:  health.New_HealthState(health.HealthState_HEALTHY),
						Params: make(map[string]interface{}),
					},
				},
			},
			"top-secret",
			"bad-secret",
			401,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := wrouter.New(whttprouter.New())
			resource := wresource.New("test", r)
			err := AddHealthRoutes(resource, healthCheck{value: test.metadata}, refreshable.NewString(refreshable.NewDefaultRefreshable(test.sharedSecret)), nil)
			require.NoError(t, err)

			server := httptest.NewServer(r)
			defer server.Close()

			client := http.DefaultClient
			request, err := http.NewRequest(http.MethodGet, strings.Join([]string{server.URL, status.HealthEndpoint}, "/"), nil)
			require.NoError(t, err)
			request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", test.actualSecret))
			resp, err := client.Do(request)
			require.NoError(t, err)
			assert.Equal(t, test.expectedStatus, resp.StatusCode)
			bytes, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			var gotObj health.HealthStatus
			err = json.Unmarshal(bytes, &gotObj)
			require.NoError(t, err)

			expectedChecks := test.metadata
			// 401 does not return any health check data
			if test.expectedStatus == 401 {
				expectedChecks.Checks = map[health.CheckType]health.HealthCheckResult{
					"HEALTH_STATUS_UNAUTHORIZED": sources.UnhealthyHealthCheckResult("HEALTH_STATUS_UNAUTHORIZED", "unauthorized to access health status; please verify the health-check-shared-secret", map[string]interface{}{}),
				}
			}
			assert.Equal(t, expectedChecks, gotObj)
		})
	}
}

type statusFunc func() (int, interface{})

func (f statusFunc) Status() (int, interface{}) {
	return f()
}

type healthCheck struct {
	value health.HealthStatus
}

func (h healthCheck) HealthStatus(ctx context.Context) health.HealthStatus {
	return h.value
}
