package witchcraft

import (
	"github.com/palantir/witchcraft-go-tasks/function"
	"github.com/palantir/witchcraft-go-tasks/jobs"
)

type TaskManager interface {
	TaskManagerAdder
	TaskManagerGetter
}

type TaskManagerAdder interface {
	AddJobs(job ...jobs.Job)
	AddForeverRunnable(namedRunnable ...function.NamedRunnable)
}

type TaskManagerGetter interface {
	GetJobs() []jobs.Job
}

type defaultTaskManager struct {
	jobs             []jobs.Job
	foreverRunnables []function.NamedRunnable
}

func (d *defaultTaskManager) AddForeverRunnable(namedRunnables ...function.NamedRunnable) {
	for _, namedRunnable := range namedRunnables {
		d.foreverRunnables = append(d.foreverRunnables, namedRunnable)
	}
}

func NewTaskManager() TaskManager {
	return &defaultTaskManager{}
}

func (d *defaultTaskManager) AddJobs(job ...jobs.Job) {
	for _, job := range job {
		d.jobs = append(d.jobs, job)
	}
}

func (d *defaultTaskManager) GetJobs() []jobs.Job {
	return d.jobs
}
