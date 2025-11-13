// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package middleware

import (
	"context"
	"testing"

	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
	"github.com/stretchr/testify/assert"
)

func TestEndpointFiveHundredsHealthCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	endpoint := wrouter.RouteSpec{Method: "GET", PathTemplate: "/test"}
	endpoint2 := wrouter.RouteSpec{Method: "POST", PathTemplate: "/test"}
	endpoint3 := wrouter.RouteSpec{Method: "DELETE", PathTemplate: "/test"}

	for _, tt := range []struct {
		Name           string
		AlwaysHealthy  bool
		EarlierSuccess bool
		Responses      [6]map[wrouter.RouteSpec][]int
		Expected       *health.HealthCheckResult
	}{
		{
			Name:      "No data",
			Responses: [6]map[wrouter.RouteSpec][]int{},
			Expected:  nil,
		},
		{
			Name: "200 in window 0",
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {200}},
			},
			Expected: &health.HealthCheckResult{
				Type:  endpointFiveHundredsCheckType,
				State: health.New_HealthState(health.HealthState_HEALTHY),
			},
		},
		{
			Name: "500 in window 0",
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {500}},
			},
			Expected: &health.HealthCheckResult{
				Type:  endpointFiveHundredsCheckType,
				State: health.New_HealthState(health.HealthState_HEALTHY),
			},
		},
		{
			Name: "All 200s",
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {200}},
				{endpoint: {200}},
				{endpoint: {200}},
				{endpoint: {200}},
				{endpoint: {200}},
				{endpoint: {200}},
			},
			Expected: &health.HealthCheckResult{
				Type:  endpointFiveHundredsCheckType,
				State: health.New_HealthState(health.HealthState_HEALTHY),
			},
		},
		{
			Name: "All 500s",
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
			},
			Expected: &health.HealthCheckResult{
				Type:    endpointFiveHundredsCheckType,
				Message: ptrTo(endpointFiveHundredsMessage + "\n" + endpointFiveHundredsBrokenEndpointsMessage),
				State:   health.New_HealthState(health.HealthState_WARNING),
				Params: map[string]interface{}{
					"brokenEndpoints":  []string{"GET /test"},
					"failingEndpoints": []string{},
				},
			},
		},
		{
			Name:           "All 500s with earlier success",
			EarlierSuccess: true,
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
			},
			Expected: &health.HealthCheckResult{
				Type:    endpointFiveHundredsCheckType,
				Message: ptrTo(endpointFiveHundredsMessage),
				State:   health.New_HealthState(health.HealthState_WARNING),
				Params: map[string]interface{}{
					"brokenEndpoints":  []string{},
					"failingEndpoints": []string{"GET /test"},
				},
			},
		},
		{
			Name: "Mix 200s 500s",
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {500}},
				{endpoint: {200}},
				{endpoint: {500}},
				{endpoint: {200}},
				{endpoint: {500}},
				{endpoint: {200}},
			},
			Expected: &health.HealthCheckResult{
				Type:  endpointFiveHundredsCheckType,
				State: health.New_HealthState(health.HealthState_HEALTHY),
			},
		},
		{
			Name: "Mix 200s 500s same window",
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {200, 500}},
				{endpoint: {200, 500}},
				{endpoint: {200, 500}},
				{endpoint: {400, 500}},
				{endpoint: {400, 500}},
				{endpoint: {400, 500}},
			},
			Expected: &health.HealthCheckResult{
				Type:    endpointFiveHundredsCheckType,
				Message: ptrTo(endpointFiveHundredsMessage),
				State:   health.New_HealthState(health.HealthState_WARNING),
				Params: map[string]interface{}{
					"brokenEndpoints":  []string{},
					"failingEndpoints": []string{"GET /test"},
				},
			},
		},
		{
			Name:          "Always healthy",
			AlwaysHealthy: true,
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
				{endpoint: {500}},
			},
			Expected: &health.HealthCheckResult{
				Type:    endpointFiveHundredsCheckType,
				State:   health.New_HealthState(health.HealthState_HEALTHY),
				Message: ptrTo(endpointFiveHundredsMessage + "\n" + endpointFiveHundredsBrokenEndpointsMessage),
				Params: map[string]interface{}{
					"brokenEndpoints":  []string{"GET /test"},
					"failingEndpoints": []string{},
				},
			},
		},
		{
			Name: "multiple endpoints",
			Responses: [6]map[wrouter.RouteSpec][]int{
				{endpoint: {500}, endpoint2: {200}, endpoint3: {200, 500}},
				{endpoint: {500}, endpoint2: {200}, endpoint3: {200, 500}},
				{endpoint: {500}, endpoint2: {200}, endpoint3: {200, 500}},
				{endpoint: {500}, endpoint2: {200}, endpoint3: {200, 500}},
				{endpoint: {500}, endpoint2: {200}, endpoint3: {200, 500}},
				{endpoint: {500}, endpoint2: {200}, endpoint3: {200, 500}},
			},
			Expected: &health.HealthCheckResult{
				Type:    endpointFiveHundredsCheckType,
				Message: ptrTo(endpointFiveHundredsMessage + "\n" + endpointFiveHundredsBrokenEndpointsMessage),
				State:   health.New_HealthState(health.HealthState_WARNING),
				Params: map[string]interface{}{
					"brokenEndpoints":  []string{"GET /test"},
					"failingEndpoints": []string{"DELETE /test"},
				},
			},
		},
	} {
		t.Run(tt.Name, func(t *testing.T) {
			healthcheck := NewEndpointFiveHundredsHealthCheck(ctx, tt.AlwaysHealthy)

			if tt.EarlierSuccess {
				healthcheck.MarkResponse(endpoint, 200)
				healthcheck.shiftWindow()
			}

			for _, window := range tt.Responses {
				for routeSpec, codes := range window {
					for _, code := range codes {
						healthcheck.MarkResponse(routeSpec, code)
					}
				}
				healthcheck.shiftWindow()
			}

			assert.Equal(t, tt.Expected, healthcheck.currentStatus())
		})
	}

}

func ptrTo[T any](v T) *T {
	return &v
}
