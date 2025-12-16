package witchcraft

import "github.com/palantir/witchcraft-go-tasks/jobs"

type TaskManager interface {
	TaskManagerAdder
	TaskManagerGetter
}

type TaskManagerAdder interface {
	AddJobs(job ...jobs.Job)
}

type TaskManagerGetter interface {
}

type defaultTaskManager struct {
	jobs []jobs.Job
}

func NewTaskManager() TaskManager {
	return &defaultTaskManager{}
}

func (d *defaultTaskManager) AddJobs(job ...jobs.Job) {
	for _, job := range job {
		d.jobs = append(d.jobs, job)
	}
}
