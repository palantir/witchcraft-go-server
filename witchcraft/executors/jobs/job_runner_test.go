package jobs_test

import (
	"context"
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/stretchr/testify/mock"
	observability_mock "github.palantir.build/deployability/witchcraft-controller-commons/internal/generated/mocks/github.palantir.build/deployability/witchcraft-controller-commons/observability"
	"github.palantir.build/deployability/witchcraft-controller-commons/jobs"
	"github.palantir.build/deployability/witchcraft-controller-commons/jobs/mocks"
	"github.palantir.build/deployability/witchcraft-controller-commons/pkg/testutil"
)

func TestHe(t *testing.T) {
	K8sKeyedErrorHealthCheckSource := new(observability_mock.K8sKeyedErrorHealthCheckSource)
	jobRunner := jobs.NewDefaultJobRunner(K8sKeyedErrorHealthCheckSource, time.Millisecond*200)
	job1 := new(mocks.Job)
	job2 := new(mocks.Job)
	job1.Test(t)
	job2.Test(t)
	jobs := []jobs.Job{
		job1, job2,
	}
	err := werror.Error("bar")
	job1.On("GetName").Return("job1")
	job1.On("Run", mock.Anything).Return(nil).Times(2)
	job2.On("GetName").Return("job2")
	job2.On("Run", mock.Anything).Return(err).Times(2)
	job2.On("LogError", mock.Anything, err).Times(2)
	K8sKeyedErrorHealthCheckSource.On("Submit", mock.Anything, "job1", nil).Times(2)
	K8sKeyedErrorHealthCheckSource.On("Submit", mock.Anything, "job2", err).Times(2)

	ctx, cancel := context.WithCancel(testutil.GetTestContext())
	defer cancel()
	jobRunner.StartJobs(ctx, jobs)
	time.Sleep(time.Millisecond * 500)
}

func TestStartJobsImmediately(t *testing.T) {
	K8sKeyedErrorHealthCheckSource := new(observability_mock.K8sKeyedErrorHealthCheckSource)

	// jobInterval is much longer than the test will run for to explicitly test immediate execution
	jobInterval := time.Second
	jobRunner := jobs.NewDefaultJobRunner(K8sKeyedErrorHealthCheckSource, jobInterval, jobs.WithStartImmediately())

	job1 := new(mocks.Job)
	job2 := new(mocks.Job)
	job1.Test(t)
	job2.Test(t)
	jobs := []jobs.Job{
		job1, job2,
	}
	err := werror.Error("bar")
	job1.On("GetName").Return("job1")
	job1.On("Run", mock.Anything).Return(nil)

	job2.On("GetName").Return("job2")
	job2.On("Run", mock.Anything).Return(err)
	job2.On("LogError", mock.Anything, err)

	K8sKeyedErrorHealthCheckSource.On("Submit", mock.Anything, "job1", nil).Times(1)
	K8sKeyedErrorHealthCheckSource.On("Submit", mock.Anything, "job2", err).Times(1)

	ctx, cancel := context.WithCancel(testutil.GetTestContext())
	defer cancel()
	jobRunner.StartJobs(ctx, jobs)
	time.Sleep(time.Millisecond * 200)
}
