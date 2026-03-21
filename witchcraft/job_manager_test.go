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

	"github.com/palantir/witchcraft-go-health/v2/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/v2/sources/window"
	healthstatus "github.com/palantir/witchcraft-go-health/v2/status"
	"github.com/palantir/witchcraft-go-tasks/function"
	"github.com/palantir/witchcraft-go-tasks/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobManager_AddJobs_EmptyJobs(t *testing.T) {
	healthCheckSource := window.MustNewKeyedErrorHealthCheckSource(
		health.CheckType("test"),
		window.HealthyIfNotAllErrors,
	)
	var healthRegistered bool
	registerHealth := func(healthSource healthstatus.HealthCheckSource) {
		healthRegistered = true
	}
	jobManager := NewJobManager(healthCheckSource, registerHealth)
	jobManager.AddJobs(context.Background())
	assert.False(t, healthRegistered, "health should not be registered when no jobs are added")
}

func TestJobManager_AddJobs_RegistersHealthOnce(t *testing.T) {
	healthCheckSource := window.MustNewKeyedErrorHealthCheckSource(
		health.CheckType("test"),
		window.HealthyIfNotAllErrors,
	)
	var healthRegisterCount int
	registerHealth := func(healthSource healthstatus.HealthCheckSource) {
		healthRegisterCount++
	}
	jobManager := NewJobManager(healthCheckSource, registerHealth)
	ctx := t.Context()
	job1 := jobs.NewDefaultJob("job1", function.NewRunnableFromFunc(func(ctx context.Context) error {
		return nil
	}))
	job2 := jobs.NewDefaultJob("job2", function.NewRunnableFromFunc(func(ctx context.Context) error {
		return nil
	}))
	jobManager.AddJobs(ctx, job1)
	jobManager.AddJobs(ctx, job2)
	assert.Equal(t, 1, healthRegisterCount, "health should be registered exactly once")
}

func TestJobManager_AddJobs_StartsJobs(t *testing.T) {
	healthCheckSource := window.MustNewKeyedErrorHealthCheckSource(
		health.CheckType("test"),
		window.HealthyIfNotAllErrors,
	)
	registerHealth := func(healthSource healthstatus.HealthCheckSource) {}
	jobManager := NewJobManager(healthCheckSource, registerHealth)
	ctx := t.Context()
	var jobRan atomic.Bool
	job := jobs.NewDefaultJob("test-job", function.NewRunnableFromFunc(func(ctx context.Context) error {
		jobRan.Store(true)
		return nil
	}), jobs.WithStartImmediately(true))
	jobManager.AddJobs(ctx, job)
	require.Eventually(t, func() bool {
		return jobRan.Load()
	}, time.Second, 10*time.Millisecond, "job should have run")
}
