package jobs

import (
	"context"
	"time"

	"github.com/palantir/witchcraft-go-server/v2/witchcraft/executors/api"
)

// Job is a simple interface for running an operation
type Job interface {
	api.NamedExecutor
	ShouldStartImmediately(ctx context.Context) bool
	GetInterval(ctx context.Context) time.Duration
	// LogError is the called if the job returns an error
	LogError(ctx context.Context, err error)
}

// JobRunner is an interface for safely running all the specified jobs discussed above
type JobRunner interface {
	StartJobs(ctx context.Context, jobs []Job)
}
