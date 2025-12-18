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
	"sync/atomic"
	"testing"
	"time"

	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/witchcraft-go-server/v3/config"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func newServerWithLeaderElection(t *testing.T, host string, elector witchcraft.LeaderElector) *witchcraft.Server[config.Install, config.Runtime] {
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	mgmtPort, err := httpserver.AvailablePort()
	require.NoError(t, err)

	return witchcraft.NewServer[config.Install, config.Runtime]().
		WithSelfSignedCertificate().
		WithDisableGoRuntimeMetrics().
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
}

func TestLeaderElection_RequiresSeparateManagementPort(t *testing.T) {
	elector := &immediateLeaderElector{}
	server := witchcraft.NewServer[config.Install, config.Runtime]().
		WithSelfSignedCertificate().
		WithDisableGoRuntimeMetrics().
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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
	<-serverErrCh
}
