package jobs

import (
	"context"
	"time"

	"github.com/palantir/witchcraft-go-health/sources/window"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
)

type defaultJobRunner struct {
	k8sKeyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource
}

// NewDefaultJobRunner returns the default implementation of the JobRunner
func NewDefaultJobRunner(k8sKeyedErrorHealthCheckSource window.KeyedErrorHealthCheckSource) JobRunner {
	return &defaultJobRunner{
		k8sKeyedErrorHealthCheckSource: k8sKeyedErrorHealthCheckSource,
	}
}

func (d defaultJobRunner) StartJobs(ctx context.Context, jobs []Job) {
	for _, job := range jobs {
		d.startJobAsync(ctx, job)
	}
}

func (d defaultJobRunner) startJobAsync(ctx context.Context, job Job) {
	go d.startJob(ctx, job)
}

func (d defaultJobRunner) startJob(ctx context.Context, job Job) {
	ctx = svc1log.WithLoggerParams(ctx, svc1log.SafeParam("jobName", job.GetName()))
	if job.ShouldStartImmediately(ctx) {
		d.runJobNoError(ctx, job)
	}
	svc1log.FromContext(ctx).Info("Starting ticker for job")
	for {
		select {
		case <-time.After(job.GetInterval(ctx)):
			d.runJobNoError(ctx, job)
		case <-ctx.Done():
			return
		}
	}
}

func (d defaultJobRunner) runJobNoError(ctx context.Context, job Job) {
	fnSpan, ctx := wtracing.StartSpanFromTracerInContext(ctx, "defaultJobRunner.runJobNoError")
	defer fnSpan.Finish()
	svc1log.FromContext(ctx).Debug("Starting single iteration of the job")
	err := wapp.RunWithRecoveryLoggingWithError(ctx, job.Run)
	// TODO Fix CTX
	d.k8sKeyedErrorHealthCheckSource.Submit(job.GetName(), err)
	if err != nil {
		job.LogError(ctx, err)
	}
}
