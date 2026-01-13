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

	"github.com/palantir/witchcraft-go-tasks/function"
)

// Job is a simple interface for running an operation periodically.
type Job interface {
	function.NamedRunnable
	// ShouldStartImmediately returns true if the job should run immediately upon starting,
	// rather than waiting for the first interval to elapse.
	ShouldStartImmediately(ctx context.Context) bool
	// GetInterval returns the duration between job executions.
	GetInterval(ctx context.Context) time.Duration
	// LogError is called if the job returns an error during execution.
	LogError(ctx context.Context, err error)
}

// JobRunner is an interface that runs a collection of provided Jobs.
type JobRunner interface {
	// StartJobs begins executing all provided jobs asynchronously.
	// Each job runs in its own goroutine and will continue until the context is cancelled.
	StartJobs(ctx context.Context, jobs []Job)
}
