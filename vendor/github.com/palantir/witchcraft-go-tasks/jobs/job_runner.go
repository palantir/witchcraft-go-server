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

package jobs

import (
	"context"
	"time"

	"github.com/palantir/witchcraft-go-health/v2/sources/window"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
)

type defaultJobRunner struct {
	keyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource
}

// NewDefaultJobRunner returns the default implementation of the JobRunner.
//
// The runner executes each job in its own goroutine, respecting the job's interval and
// start-immediately settings. It integrates with the provided KeyedErrorHealthCheckSource
// to report job execution health status, submitting success or error after each job run.
//
// Features:
//   - Runs each job asynchronously in a separate goroutine
//   - Recovers from panics during job execution and logs them
//   - Adds tracing spans for each job execution
//   - Logs job lifecycle events (start, errors) via svc1log
//   - Stops all jobs gracefully when the context is cancelled
//
// Example:
//
//	healthCheckSource := window.MustNewKeyedErrorHealthCheckSource(
//	    health.CheckType("jobs"),
//	    window.UnhealthyIfAtLeastOneError,
//	)
//	runner := NewDefaultJobRunner(healthCheckSource)
//	runner.StartJobs(ctx, []Job{job1, job2})
func NewDefaultJobRunner(keyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource) JobRunner {
	return &defaultJobRunner{
		keyedErrorHealthCheckSource: keyedErrorHealthCheckSource,
	}
}

// StartJobs starts all of the provided jobs. Creates a new goroutine for each job.
func (d defaultJobRunner) StartJobs(ctx context.Context, jobs []Job) {
	for _, job := range jobs {
		d.startJobAsync(ctx, job)
	}
}

func (d defaultJobRunner) startJobAsync(ctx context.Context, job Job) {
	go d.startJob(ctx, job)
}

func (d defaultJobRunner) startJob(ctx context.Context, job Job) {
	ctx = svc1log.WithLoggerParams(ctx, svc1log.SafeParam("jobName", job.Name()))
	if job.ShouldStartImmediately(ctx) {
		d.runJobNoError(ctx, job)
	}
	svc1log.FromContext(ctx).Info("Starting ticker for job")
	for {
		select {
		case <-time.After(job.GetInterval(ctx)):
			d.runJobNoError(ctx, job)
		case <-ctx.Done():
			return
		}
	}
}

func (d defaultJobRunner) runJobNoError(ctx context.Context, job Job) {
	fnSpan, ctx := wtracing.StartSpanFromTracerInContext(ctx, "defaultJobRunner.runJobNoError")
	defer fnSpan.Finish()
	svc1log.FromContext(ctx).Debug("Starting single iteration of the job")
	err := wapp.RunWithRecoveryLoggingWithError(ctx, job.Run)
	d.keyedErrorHealthCheckSource.Submit(ctx, job.Name(), err)
	if err != nil {
		job.LogError(ctx, err)
	}
}
