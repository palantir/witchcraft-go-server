// Copyright (c) 2020 Palantir Technologies. All rights reserved.
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
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type componentTestData struct {
	status   int
	metadata interface{}
}

func TestReadinessComponentOperations(t *testing.T) {
	ctx := context.TODO()
	name := ComponentName("TEST_CHECK")
	reporter := NewReadinessReporter()
	component, err := reporter.InitializeReadinessComponent(ctx, name)
	require.NoError(t, err)

	retrievedComponent, exists := reporter.GetReadinessComponent(name)
	assert.True(t, exists)
	assert.Equal(t, component, retrievedComponent)

	success := reporter.UnregisterReadinessComponent(ctx, name)
	assert.True(t, success)
}

func TestStatusBehavior(t *testing.T) {
	for _, tc := range []struct {
		name               string
		components         map[ComponentName]componentTestData
		expectedRespStatus int
		expectedMetadata   interface{}
	}{
		{
			name:               "reporter with no components defaults to ready",
			components:         map[ComponentName]componentTestData{},
			expectedRespStatus: http.StatusOK,
			expectedMetadata:   map[ComponentName]interface{}{},
		},
		{
			name: "component defaults to unready",
			components: map[ComponentName]componentTestData{
				"test-component": {},
			},
			expectedRespStatus: http.StatusInternalServerError,
			expectedMetadata: map[ComponentName]interface{}{
				"test-component": nil,
			},
		},
		{
			name: "reporter with single components",
			components: map[ComponentName]componentTestData{
				"test-component": {
					status:   http.StatusCreated,
					metadata: "metadata",
				},
			},
			expectedRespStatus: http.StatusOK,
			expectedMetadata: map[ComponentName]interface{}{
				"test-component": "metadata",
			},
		},
		{
			name: "reporter with ready and unready components",
			components: map[ComponentName]componentTestData{
				"test-component": {
					status:   http.StatusCreated,
					metadata: "metadata",
				},
				"unready-component": {
					status:   http.StatusInternalServerError,
					metadata: "unready-metadata",
				},
			},
			expectedRespStatus: http.StatusInternalServerError,
			expectedMetadata: map[ComponentName]interface{}{
				"test-component":    "metadata",
				"unready-component": "unready-metadata",
			},
		},
		{
			name: "reporter with multiple unready components returns highest status",
			components: map[ComponentName]componentTestData{
				"internal-server-error-component": {
					status:   http.StatusInternalServerError,
					metadata: "internal-server-error",
				},
				"bad-gateway-component": {
					status:   http.StatusBadGateway,
					metadata: "bad-gateway",
				},
			},
			expectedRespStatus: http.StatusBadGateway,
			expectedMetadata: map[ComponentName]interface{}{
				"internal-server-error-component": "internal-server-error",
				"bad-gateway-component":           "bad-gateway",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.TODO()
			reporter := NewReadinessReporter()
			for name, componentData := range tc.components {
				component, err := reporter.InitializeReadinessComponent(ctx, name)
				require.NoError(t, err)
				if componentData.status != 0 {
					component.SetStatus(componentData.status, componentData.metadata)
				}
			}
			respStatus, metadata := reporter.Status()
			assert.Equal(t, tc.expectedRespStatus, respStatus)
			assert.Equal(t, tc.expectedMetadata, metadata)
		})
	}
}

func TestGetNonExistentComponent(t *testing.T) {
	reporter := NewReadinessReporter()
	retrievedComponent, exists := reporter.GetReadinessComponent("DOESN'T_EXIST")
	assert.Nil(t, retrievedComponent)
	assert.False(t, exists)
}

func TestUnregisterNonExistentComponent(t *testing.T) {
	reporter := NewReadinessReporter()
	success := reporter.UnregisterReadinessComponent(context.TODO(), "DOESN'T_EXIST")
	assert.False(t, success)
}

func TestInitializeExistingComponent(t *testing.T) {
	ctx := context.TODO()
	name := ComponentName("TEST_CHECK")
	reporter := NewReadinessReporter()
	_, err := reporter.InitializeReadinessComponent(ctx, name)
	require.NoError(t, err)

	component, err := reporter.InitializeReadinessComponent(ctx, name)
	assert.Nil(t, component)
	assert.Error(t, err)
}
