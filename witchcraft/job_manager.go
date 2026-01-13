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

	"github.com/palantir/witchcraft-go-health/v2/sources/window"
	healthstatus "github.com/palantir/witchcraft-go-health/v2/status"
	"github.com/palantir/witchcraft-go-tasks/jobs"
)

// JobManager manages the lifecycle of background jobs within a witchcraft server.
type JobManager interface {
	// AddJobs registers and starts the provided jobs. Each job runs in its own goroutine
	// and executes periodically based on its configured interval. Jobs that have
	// WithStartImmediately set will run once immediately upon being added.
	//
	// The first call to AddJobs with at least one job will register a health check source
	// with the server. Subsequent calls will not register additional health sources.
	// Job execution results (success or error) are reported to this health check.
	// If no job is every registered, the health check will never be added
	//
	// The provided context is passed to each job and can be used for cancellation.
	// When the context is cancelled, all jobs that were started with that context will stop
	AddJobs(ctx context.Context, job ...jobs.Job)
}

type defaultJobManager struct {
	hasHealthBeenAdded          atomic.Bool
	jobRunner                   jobs.JobRunner
	keyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource
	registerHealth              func(healthSource healthstatus.HealthCheckSource)
}

func NewJobManager(
	keyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource,
	registerHealth func(healthSource healthstatus.HealthCheckSource)) JobManager {
	return &defaultJobManager{
		jobRunner:                   jobs.NewDefaultJobRunner(keyedErrorHealthCheckSource),
		keyedErrorHealthCheckSource: keyedErrorHealthCheckSource,
		registerHealth:              registerHealth,
	}
}

func (d *defaultJobManager) AddJobs(ctx context.Context, jobs ...jobs.Job) {
	if len(jobs) == 0 {
		return
	}
	if d.hasHealthBeenAdded.CompareAndSwap(false, true) {
		d.registerHealth(d.keyedErrorHealthCheckSource)
	}
	d.jobRunner.StartJobs(ctx, jobs)
}
