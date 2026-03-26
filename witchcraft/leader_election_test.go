// Copyright (c) 2024 Palantir Technologies. All rights reserved.
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

package witchcraft_test

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/witchcraft-go-server/v3/config"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetMetrics resets the global metrics registry to ensure test isolation.
// This is necessary because witchcraft-go-server uses metrics.DefaultMetricsRegistry
// and metrics accumulate across tests.
func resetMetrics() {
	metrics.DefaultMetricsRegistry = metrics.NewRootMetricsRegistry()
}

// mockLeaderElector is a test implementation of LeaderElector that allows
// controlling when leadership is acquired and lost.
type mockLeaderElector struct {
	// acquireLeadership is called to signal that leadership should be acquired.
	// Close this channel to trigger OnStartedLeading.
	acquireLeadership chan struct{}
	// loseLeadership is called to signal that leadership should be lost.
	// Close this channel to trigger OnStoppedLeading.
	loseLeadership chan struct{}
	// runStarted is closed when Run() is called
	runStarted chan struct{}
}

func newMockLeaderElector() *mockLeaderElector {
	return &mockLeaderElector{
		acquireLeadership: make(chan struct{}),
		loseLeadership:    make(chan struct{}),
		runStarted:        make(chan struct{}),
	}
}

func (m *mockLeaderElector) Run(ctx context.Context, callbacks witchcraft.LeaderCallbacks) error {
	close(m.runStarted)
	// Wait for either context cancellation or leadership acquisition
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-m.acquireLeadership:
	}
	// We became leader
	leaderCtx, cancelLeader := context.WithCancel(ctx)
	go callbacks.OnStartedLeading(leaderCtx)
	// Wait for either context cancellation or leadership loss
	select {
	case <-ctx.Done():
		cancelLeader()
		return ctx.Err()
	case <-m.loseLeadership:
		cancelLeader()
		callbacks.OnStoppedLeading()
		// Return after losing leadership - OnStoppedLeading will have triggered shutdown
		return nil
	}
}

// immediateLeaderElector immediately grants leadership and never loses it.
type immediateLeaderElector struct{}

func (i *immediateLeaderElector) Run(ctx context.Context, callbacks witchcraft.LeaderCallbacks) error {
	go callbacks.OnStartedLeading(ctx)
	<-ctx.Done()
	return ctx.Err()
}

type serverWithPorts struct {
	server   *witchcraft.Server[config.Install, config.Runtime]
	port     int
	mgmtPort int
}

func newServerWithLeaderElection(t *testing.T, host string, elector witchcraft.LeaderElector) *witchcraft.Server[config.Install, config.Runtime] {
	return newServerWithLeaderElectionAndPorts(t, host, elector).server
}

func newServerWithLeaderElectionAndPorts(t *testing.T, host string, elector witchcraft.LeaderElector) serverWithPorts {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	mgmtPort, err := httpserver.AvailablePort()
	require.NoError(t, err)

	server := witchcraft.NewServer[config.Install, config.Runtime]().
		WithSelfSignedCertificate().
		WithDisableGoRuntimeMetrics().
		WithLoggerStdoutWriter(io.Discard).
		WithInstallConfig(config.Install{
			Server: config.Server{
				Address:        host,
				Port:           port,
				ManagementPort: mgmtPort,
			},
			UseConsoleLog: true,
		}).
		WithRuntimeConfig(config.Runtime{}).
		WithLeaderElection(func(_ context.Context, _ config.Install, _ refreshable.Refreshable[config.Runtime]) (witchcraft.LeaderElector, error) {
			return elector, nil
		})
	return serverWithPorts{server: server, port: port, mgmtPort: mgmtPort}
}

func testHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
	}
}

func TestLeaderElection_RequiresSeparateManagementPort(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := &immediateLeaderElector{}
	server := witchcraft.NewServer[config.Install, config.Runtime]().
		WithSelfSignedCertificate().
		WithDisableGoRuntimeMetrics().
		WithLoggerStdoutWriter(io.Discard).
		WithInstallConfig(config.Install{
			Server: config.Server{
				Address: "127.0.0.1",
				Port:    0,
				// No management port set - should fail
			},
			UseConsoleLog: true,
		}).
		WithRuntimeConfig(config.Runtime{}).
		WithLeaderElection(func(_ context.Context, _ config.Install, _ refreshable.Refreshable[config.Runtime]) (witchcraft.LeaderElector, error) {
			return elector, nil
		})

	err := server.Start()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "leader election requires a separate management port")
}

func TestLeaderElection_InitFnDeferredUntilLeadership(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := newMockLeaderElector()
	var initFnCalled atomic.Bool

	server := newServerWithLeaderElection(t, "127.0.0.1", elector).
		WithInitFunc(func(_ context.Context, _ witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			initFnCalled.Store(true)
			return nil, nil
		})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	// Wait for leader election to start
	select {
	case <-elector.runStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for leader election to start")
	}

	// Server should be running but initFn should not have been called yet
	assert.Eventually(t, server.Running, 2*time.Second, 100*time.Millisecond, "server should be running")
	assert.False(t, initFnCalled.Load(), "initFn should not be called before leadership is acquired")

	// Acquire leadership
	close(elector.acquireLeadership)

	// Now initFn should be called
	assert.Eventually(t, initFnCalled.Load, 2*time.Second, 100*time.Millisecond, "initFn should be called after leadership is acquired")

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	<-serverErrCh
}

func TestLeaderElection_ShutdownOnLeadershipLoss(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := newMockLeaderElector()

	server := newServerWithLeaderElection(t, "127.0.0.1", elector).
		WithInitFunc(func(_ context.Context, _ witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			return nil, nil
		})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	// Wait for leader election to start
	select {
	case <-elector.runStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for leader election to start")
	}

	// Acquire leadership
	close(elector.acquireLeadership)

	// Wait for server to be fully running
	assert.Eventually(t, server.Running, 2*time.Second, 100*time.Millisecond, "server should be running")

	// Lose leadership - this should trigger shutdown
	close(elector.loseLeadership)

	// Server should shut down
	select {
	case <-serverErrCh:
		// Server shut down as expected
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down after losing leadership")
	}
}

func TestLeaderElection_CleanupCalledOnLeadershipLoss(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := newMockLeaderElector()
	var cleanupCalled atomic.Bool
	initFnDone := make(chan struct{})

	server := newServerWithLeaderElection(t, "127.0.0.1", elector).
		WithInitFunc(func(_ context.Context, _ witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			close(initFnDone)
			return func() {
				cleanupCalled.Store(true)
			}, nil
		})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	// Wait for leader election to start
	select {
	case <-elector.runStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for leader election to start")
	}

	// Acquire leadership
	close(elector.acquireLeadership)

	// Wait for initFn to complete (so cleanup function is set)
	select {
	case <-initFnDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initFn to complete")
	}

	// Lose leadership
	close(elector.loseLeadership)

	// Cleanup should be called
	assert.Eventually(t, cleanupCalled.Load, 2*time.Second, 100*time.Millisecond, "cleanup should be called when leadership is lost")

	<-serverErrCh
}

func TestLeaderElection_ImmediateLeader(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := &immediateLeaderElector{}
	var initFnCalled atomic.Bool

	server := newServerWithLeaderElection(t, "127.0.0.1", elector).
		WithInitFunc(func(_ context.Context, _ witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			initFnCalled.Store(true)
			return nil, nil
		})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- server.Start()
	}()

	// With immediate leader, initFn should be called right away
	assert.Eventually(t, initFnCalled.Load, 5*time.Second, 100*time.Millisecond, "initFn should be called")
	assert.Eventually(t, server.Running, 2*time.Second, 100*time.Millisecond, "server should be running")

	// Cleanup
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	<-serverErrCh
}

func TestLeaderElection_StatusEndpointsAvailableBeforeLeadership(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := newMockLeaderElector()
	svrWithPorts := newServerWithLeaderElectionAndPorts(t, "127.0.0.1", elector)

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- svrWithPorts.server.Start()
	}()

	// Wait for leader election to start (server is running but not leader)
	select {
	case <-elector.runStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for leader election to start")
	}

	// Wait for server to be running
	assert.Eventually(t, svrWithPorts.server.Running, 2*time.Second, 100*time.Millisecond, "server should be running")

	client := testHTTPClient()

	// Status endpoints on management port should be available before leadership
	livenessURL := fmt.Sprintf("https://127.0.0.1:%d/status/liveness", svrWithPorts.mgmtPort)
	resp, err := client.Get(livenessURL)
	require.NoError(t, err, "liveness endpoint should be reachable before leadership")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "liveness should return 200")
	resp.Body.Close()

	readinessURL := fmt.Sprintf("https://127.0.0.1:%d/status/readiness", svrWithPorts.mgmtPort)
	resp, err = client.Get(readinessURL)
	require.NoError(t, err, "readiness endpoint should be reachable before leadership")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "readiness should return 200")
	resp.Body.Close()

	healthURL := fmt.Sprintf("https://127.0.0.1:%d/status/health", svrWithPorts.mgmtPort)
	resp, err = client.Get(healthURL)
	require.NoError(t, err, "health endpoint should be reachable before leadership")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "health should return 200")
	resp.Body.Close()

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = svrWithPorts.server.Shutdown(ctx)
	<-serverErrCh
}

func TestLeaderElection_AppRoutesOnlyAfterLeadership(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := newMockLeaderElector()
	svrWithPorts := newServerWithLeaderElectionAndPorts(t, "127.0.0.1", elector)
	initFnDone := make(chan struct{})

	svrWithPorts.server.WithInitFunc(func(_ context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
		// Register an application route
		err := info.Router.Get("/app/test", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
		}))
		if err != nil {
			return nil, err
		}
		close(initFnDone)
		return nil, nil
	})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- svrWithPorts.server.Start()
	}()

	// Wait for leader election to start
	select {
	case <-elector.runStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for leader election to start")
	}

	// Wait for server to be running
	assert.Eventually(t, svrWithPorts.server.Running, 2*time.Second, 100*time.Millisecond, "server should be running")

	client := testHTTPClient()

	// Application route should NOT be available before leadership (returns 404)
	appURL := fmt.Sprintf("https://127.0.0.1:%d/app/test", svrWithPorts.port)
	resp, err := client.Get(appURL)
	require.NoError(t, err, "app endpoint request should complete")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "app route should return 404 before leadership")
	resp.Body.Close()

	// Status endpoints should still work on management port
	livenessURL := fmt.Sprintf("https://127.0.0.1:%d/status/liveness", svrWithPorts.mgmtPort)
	resp, err = client.Get(livenessURL)
	require.NoError(t, err, "liveness should be reachable")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "liveness should return 200")
	resp.Body.Close()

	// Acquire leadership
	close(elector.acquireLeadership)

	// Wait for initFn to complete
	select {
	case <-initFnDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initFn to complete")
	}

	// Now application route should be available
	resp, err = client.Get(appURL)
	require.NoError(t, err, "app endpoint should be reachable after leadership")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "app route should return 200 after leadership")
	resp.Body.Close()

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = svrWithPorts.server.Shutdown(ctx)
	<-serverErrCh
}

func TestLeaderElection_StatusEndpointsOnBothPorts(t *testing.T) {
	resetMetrics()
	t.Cleanup(resetMetrics)
	elector := newMockLeaderElector()
	svrWithPorts := newServerWithLeaderElectionAndPorts(t, "127.0.0.1", elector)
	initFnDone := make(chan struct{})

	svrWithPorts.server.WithInitFunc(func(_ context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
		// Register a route on main router to verify it's separate from management
		err := info.Router.Get("/app/data", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		if err != nil {
			return nil, err
		}
		close(initFnDone)
		return nil, nil
	})

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- svrWithPorts.server.Start()
	}()

	// Wait for leader election to start
	select {
	case <-elector.runStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for leader election to start")
	}

	// Acquire leadership immediately
	close(elector.acquireLeadership)

	// Wait for initFn to complete
	select {
	case <-initFnDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for initFn to complete")
	}

	client := testHTTPClient()

	// Status endpoints should be available on management port
	mgmtLivenessURL := fmt.Sprintf("https://127.0.0.1:%d/status/liveness", svrWithPorts.mgmtPort)
	resp, err := client.Get(mgmtLivenessURL)
	require.NoError(t, err, "liveness on mgmt port should be reachable")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "liveness on mgmt port should return 200")
	resp.Body.Close()

	// Application routes should only be on main port, not management port
	mgmtAppURL := fmt.Sprintf("https://127.0.0.1:%d/app/data", svrWithPorts.mgmtPort)
	resp, err = client.Get(mgmtAppURL)
	require.NoError(t, err, "app route on mgmt port request should complete")
	assert.Equal(t, http.StatusNotFound, resp.StatusCode, "app route should NOT be on management port")
	resp.Body.Close()

	// Application routes should be on main port
	mainAppURL := fmt.Sprintf("https://127.0.0.1:%d/app/data", svrWithPorts.port)
	resp, err = client.Get(mainAppURL)
	require.NoError(t, err, "app route on main port should be reachable")
	assert.Equal(t, http.StatusOK, resp.StatusCode, "app route should be on main port")
	resp.Body.Close()

	// Cleanup
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = svrWithPorts.server.Shutdown(ctx)
	<-serverErrCh
}
