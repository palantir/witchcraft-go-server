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
)

// JobOption is an option that can be used to configure Jobs created using the NewDefaultJob function.
type JobOption interface {
	apply(*defaultJob)
}

type jobOptionFunc func(*defaultJob)

func (f jobOptionFunc) apply(job *defaultJob) {
	f(job)
}

// WithInterval sets the duration between job executions.
// The interval is measured from the start of one execution to the start of the next.
// Default is 1 minute if not specified.
func WithInterval(interval time.Duration) JobOption {
	return jobOptionFunc(func(job *defaultJob) {
		job.interval = interval
	})
}

// WithErrorLogger sets a custom error handler that is called when the job returns an error.
// The provided function receives the context and the error returned by the job.
// Default behavior logs the error with a stacktrace via svc1log.
func WithErrorLogger(errorLogger func(ctx context.Context, err error)) JobOption {
	return jobOptionFunc(func(job *defaultJob) {
		job.logError = errorLogger
	})
}

// WithStartImmediately controls whether the job executes immediately when started.
// If true, the job runs once immediately before waiting for the first interval.
// If false (default), the job waits for the first interval before its first execution.
func WithStartImmediately(startImmediately bool) JobOption {
	return jobOptionFunc(func(job *defaultJob) {
		job.startImmediately = startImmediately
	})
}
