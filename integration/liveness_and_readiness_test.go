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
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/palantir/pkg/httpserver"
	healthstatus "github.com/palantir/witchcraft-go-health/v2/status"
	"github.com/palantir/witchcraft-go-server/v3/config"
	"github.com/palantir/witchcraft-go-server/v3/status"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAddLivenessSource verifies that custom liveness sources can be configured both via
// WithLiveness on the server and via info.Router.WithLiveness in the init function,
// and that the later call overrides the earlier one.
func TestAddLivenessSource(t *testing.T) {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port,
		func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			return nil, nil
		},
		io.Discard, func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
			return createTestServer(t, initFn, installCfg, logOutputBuffer).
				WithLiveness(customLivenessSource{statusCode: http.StatusOK, metadata: map[string]string{"source": "server"}})
		})
	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.LivenessEndpoint))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestAddReadinessSource verifies that custom readiness sources can be configured both via
// WithReadiness on the server and via info.Router.WithReadiness in the init function,
// and that the later call overrides the earlier one.
func TestAddReadinessSource(t *testing.T) {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port,
		func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			return nil, nil
		},
		io.Discard, func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
			return createTestServer(t, initFn, installCfg, logOutputBuffer).
				WithReadiness(customReadinessSource{statusCode: http.StatusOK, metadata: map[string]string{"source": "server"}})
		})
	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.ReadinessEndpoint))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestLivenessAndReadinessNotReady verifies that custom liveness and readiness sources
// can report non-OK status codes.
func TestLivenessAndReadinessNotReady(t *testing.T) {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port,
		func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			info.Router.WithLiveness(customLivenessSource{statusCode: http.StatusServiceUnavailable, metadata: map[string]string{"reason": "not live"}})
			info.Router.WithReadiness(customReadinessSource{statusCode: http.StatusServiceUnavailable, metadata: map[string]string{"reason": "not ready"}})
			return nil, nil
		},
		io.Discard, createTestServer)
	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	livenessResp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.LivenessEndpoint))
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, livenessResp.StatusCode)

	readinessResp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s/%s", port, basePath, status.ReadinessEndpoint))
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, readinessResp.StatusCode)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

type customLivenessSource struct {
	statusCode int
	metadata   map[string]string
}

func (c customLivenessSource) Status() (respStatus int, metadata any) {
	return c.statusCode, c.metadata
}

var _ healthstatus.Source = customLivenessSource{}

type customReadinessSource struct {
	statusCode int
	metadata   map[string]string
}

func (c customReadinessSource) Status() (respStatus int, metadata any) {
	return c.statusCode, c.metadata
}

var _ healthstatus.Source = customReadinessSource{}
