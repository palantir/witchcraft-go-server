package witchcraft

import (
	"context"

	"github.com/palantir/witchcraft-go-tasks/function"
	"github.com/palantir/witchcraft-go-tasks/jobs"
)

type TaskManager interface {
	JobManager
}

type defaultTaskManager struct {
	jobs             []jobs.Job
	foreverRunnables []function.NamedRunnable
	jobManager       JobManager
}

func (d *defaultTaskManager) AddForeverRunnable(namedRunnables ...function.NamedRunnable) {
	for _, namedRunnable := range namedRunnables {
		d.foreverRunnables = append(d.foreverRunnables, namedRunnable)
	}
}

func NewTaskManager(jobManager JobManager) TaskManager {
	return &defaultTaskManager{
		jobManager: jobManager,
	}
}

func (d *defaultTaskManager) AddJobs(ctx context.Context, job ...jobs.Job) {
	d.jobManager.AddJobs(ctx, job...)
}
