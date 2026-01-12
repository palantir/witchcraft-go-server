package witchcraft

import (
	"context"
	"sync/atomic"

	"github.com/palantir/witchcraft-go-health/v2/sources/window"
	healthstatus "github.com/palantir/witchcraft-go-health/v2/status"
	"github.com/palantir/witchcraft-go-tasks/jobs"
)

type JobManager interface {
	AddJobs(ctx context.Context, job ...jobs.Job)
}

type defaultJobManager struct {
	hasHealthBeenAdded          atomic.Bool
	jobRunner                   jobs.JobRunner
	keyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource
	registerHealth              func(healthSources ...healthstatus.HealthCheckSource)
}

func NewJobManager(keyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource) JobManager {
	jobRunner := jobs.NewDefaultJobRunner(keyedErrorHealthCheckSource)
	return &defaultJobManager{
		jobRunner:                   jobRunner,
		keyedErrorHealthCheckSource: keyedErrorHealthCheckSource,
	}
}

func (d *defaultJobManager) AddJobs(ctx context.Context, job ...jobs.Job) {
	if len(job) == 0 {
		return
	}
	if !d.hasHealthBeenAdded.Load() {
		d.hasHealthBeenAdded.Store(true)
		d.registerHealth(d.keyedErrorHealthCheckSource)
	}
	d.jobRunner.StartJobs(ctx, job)
}
