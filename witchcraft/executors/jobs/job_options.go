package jobs

import (
	"context"
	"time"
)

// JobOption is an option that can be used to configure a job runner.
type JobOption interface {
	apply(*defaultJob)
}

type jobOptionFunc func(*defaultJob)

func (f jobOptionFunc) apply(job *defaultJob) {
	f(job)
}

func WithInterval(interval time.Duration) JobOption {
	return jobOptionFunc(func(job *defaultJob) {
		job.interval = interval
	})
}

func WithErrorLogger(errorLogger func(ctx context.Context, err error)) JobOption {
	return jobOptionFunc(func(job *defaultJob) {
		job.logError = errorLogger
	})
}

func WithStartImmediately(startImmediately bool) JobOption {
	return jobOptionFunc(func(job *defaultJob) {
		job.startImmediately = startImmediately
	})
}
