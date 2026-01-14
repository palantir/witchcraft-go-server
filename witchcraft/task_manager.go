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

	"github.com/palantir/witchcraft-go-tasks/function"
	"github.com/palantir/witchcraft-go-tasks/jobs"
)

type TaskManager interface {
	JobManager
	RunnableManager
}

type defaultTaskManager struct {
	jobManager      JobManager
	runnableManager RunnableManager
}

func NewTaskManager(jobManager JobManager, runnableManager RunnableManager) TaskManager {
	return &defaultTaskManager{
		jobManager:      jobManager,
		runnableManager: runnableManager,
	}
}

func (d *defaultTaskManager) AddJobs(ctx context.Context, jobsArg ...jobs.Job) {
	d.jobManager.AddJobs(ctx, jobsArg...)
}

func (d *defaultTaskManager) AddForeverRunnable(ctx context.Context, namedRunnables ...function.NamedRunnable) {
	d.runnableManager.AddForeverRunnable(ctx, namedRunnables...)
}
