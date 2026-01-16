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

	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-tasks/function"
)

type defaultJob struct {
	name             string
	runnable         function.Runnable
	interval         time.Duration
	logError         func(ctx context.Context, err error)
	startImmediately bool
}

// NewDefaultJob returns a default Job implementation with the provided name and runnable.
//
// The job can be configured using the following options:
//   - WithInterval: sets the duration between job executions (default: 1 minute)
//   - WithStartImmediately: if true, the job runs immediately when started (default: false)
//   - WithErrorLogger: sets a custom error handler (default: logs error with stacktrace via svc1log)
//
// Example:
//
//	job := NewDefaultJob("my-job", myRunnable,
//	    WithInterval(5*time.Minute),
//	    WithStartImmediately(true),
//	)
func NewDefaultJob(name string, runnable function.Runnable, options ...JobOption) Job {
	defaultJob := &defaultJob{
		name:     name,
		runnable: runnable,
		interval: time.Minute,
		logError: func(ctx context.Context, err error) {
			svc1log.FromContext(ctx).Error("Encountered unexpected error during job execution", svc1log.Stacktrace(err))
		},
		startImmediately: false,
	}
	for _, option := range options {
		option.apply(defaultJob)
	}
	return defaultJob
}

func (d *defaultJob) ShouldStartImmediately(ctx context.Context) bool {
	return d.startImmediately
}

func (d *defaultJob) GetInterval(ctx context.Context) time.Duration {
	return d.interval
}

func (d *defaultJob) Name() string {
	return d.name
}

func (d *defaultJob) Run(ctx context.Context) error {
	return d.runnable.Run(ctx)
}

func (d *defaultJob) LogError(ctx context.Context, err error) {
	d.logError(ctx, err)
}
