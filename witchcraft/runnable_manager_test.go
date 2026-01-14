// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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

package witchcraft

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-tasks/runnable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunnableManager_AddForeverRunnable_StartsRunnable(t *testing.T) {
	var shutdownCalled atomic.Bool
	serverShutdown := func(ctx context.Context) {
		shutdownCalled.Store(true)
	}
	manager := NewRunnableManager(serverShutdown)
	ctx := context.Background()
	var runnableRan atomic.Bool
	testRunnable := runnable.New("test-runnable", func(ctx context.Context) error {
		runnableRan.Store(true)
		return nil
	})
	manager.AddForeverRunnable(ctx, testRunnable)
	require.Eventually(t, func() bool {
		return runnableRan.Load()
	}, time.Second, 10*time.Millisecond, "runnable should have run")
	require.Eventually(t, func() bool {
		return shutdownCalled.Load()
	}, time.Second, 10*time.Millisecond, "serverShutdown should be called when runnable completes")
}

func TestRunnableManager_AddForeverRunnable_MultipleRunnables(t *testing.T) {
	var shutdownCount atomic.Int32
	serverShutdown := func(ctx context.Context) {
		shutdownCount.Add(1)
	}
	manager := NewRunnableManager(serverShutdown)
	ctx := context.Background()
	var runnable1Ran, runnable2Ran atomic.Bool
	testRunnable1 := runnable.New("test-runnable-1", func(ctx context.Context) error {
		runnable1Ran.Store(true)
		return nil
	})
	testRunnable2 := runnable.New("test-runnable-2", func(ctx context.Context) error {
		runnable2Ran.Store(true)
		return nil
	})
	manager.AddForeverRunnable(ctx, testRunnable1, testRunnable2)
	require.Eventually(t, func() bool {
		return runnable1Ran.Load() && runnable2Ran.Load()
	}, time.Second, 10*time.Millisecond, "both runnables should have run")
	require.Eventually(t, func() bool {
		return shutdownCount.Load() == 2
	}, time.Second, 10*time.Millisecond, "serverShutdown should be called for each runnable")
}

func TestRunnableManager_AddForeverRunnable_RunnableReturnsError_CallsShutdown(t *testing.T) {
	var shutdownCalled atomic.Bool
	serverShutdown := func(ctx context.Context) {
		shutdownCalled.Store(true)
	}
	manager := NewRunnableManager(serverShutdown)
	ctx := context.Background()
	testRunnable := runnable.New("error-runnable", func(ctx context.Context) error {
		return werror.Error("runnable error")
	})
	manager.AddForeverRunnable(ctx, testRunnable)
	require.Eventually(t, func() bool {
		return shutdownCalled.Load()
	}, time.Second, 10*time.Millisecond, "serverShutdown should be called when runnable returns error")
}

func TestRunnableManager_AddForeverRunnable_EmptyRunnables(t *testing.T) {
	var shutdownCalled atomic.Bool
	serverShutdown := func(ctx context.Context) {
		shutdownCalled.Store(true)
	}
	manager := NewRunnableManager(serverShutdown)
	ctx := context.Background()
	manager.AddForeverRunnable(ctx)
	assert.False(t, shutdownCalled.Load(), "serverShutdown should not be called when no runnables are added")
}
