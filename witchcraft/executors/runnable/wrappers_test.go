package runnable

import (
	"context"
	"errors"
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/stretchr/testify/assert"
)

func TestWithWrappers(t *testing.T) {
	logs := make([]string, 0, 5)
	wrapper1 := func(runnable Runnable) Runnable {
		return New("", func(ctx context.Context) error {
			logs = append(logs, "pre-1")
			err := runnable.Run(ctx)
			logs = append(logs, "post-1")
			return err
		})
	}
	wrapper2 := func(runnable Runnable) Runnable {
		return New("", func(ctx context.Context) error {
			logs = append(logs, "pre-2")
			err := runnable.Run(ctx)
			logs = append(logs, "post-2")
			return err
		})
	}
	baseRunnable := New("", func(ctx context.Context) error {
		logs = append(logs, "go")
		return nil
	})

	runnable := WithWrappers(wrapper1, wrapper2)(baseRunnable)
	err := runnable.Run(context.Background())
	assert.NoError(t, err)

	assert.Equal(t, []string{"pre-2", "pre-1", "go", "post-1", "post-2"}, logs)
}

func TestWithFatalLogging(t *testing.T) {
	runnable := New("", func(ctx context.Context) error {
		panic(werror.Error("error message"))
	}, WithFatalLogging())
	err := runnable.Run(context.Background())
	assert.Error(t, err)
}

func TestWithErrorHandler(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		runnableErr  error
		expectedCall bool
	}{
		{
			name:         "success",
			runnableErr:  nil,
			expectedCall: false,
		},
		{
			name:         "error",
			runnableErr:  werror.Error("error message"),
			expectedCall: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			calledHandler := false
			errorHandler := func(ctx context.Context, err error) {
				assert.Error(t, err)
				calledHandler = true
			}
			runnable := New("", func(ctx context.Context) error {
				return testCase.runnableErr
			}, WithErrorHandler(errorHandler))

			err := runnable.Run(context.Background())
			if testCase.expectedCall {
				assert.Error(t, err)
				assert.True(t, calledHandler)
			} else {
				assert.NoError(t, err)
				assert.False(t, calledHandler)
			}
		})
	}
}

func TestWhileChanNotClosed(t *testing.T) {
	counter := 5
	stopChan := make(chan struct{}, counter)
	runnable := New("", func(ctx context.Context) error {
		counter--
		if counter == 0 {
			close(stopChan)
		}
		return nil
	}, WhileChanNotClosed(stopChan))
	err := runnable.Run(context.Background())
	assert.NoError(t, err)
	assert.Zero(t, counter)
}

func TestWithTimeout(t *testing.T) {
	t.Run("returns nil when runFn completes within timeout", func(t *testing.T) {
		ctx := context.Background()
		timeout := 50 * time.Millisecond

		runFn := func(ctx context.Context) error {
			return nil
		}

		err := New("", runFn, WithTimeout(timeout)).Run(ctx)
		assert.NoError(t, err)
	})

	t.Run("returns an error when runFn exceeds the timeout", func(t *testing.T) {
		ctx := context.Background()
		timeout := 50 * time.Millisecond

		runFn := func(ctx context.Context) error {
			time.Sleep(1000 * time.Millisecond)
			return nil
		}

		err := New("", runFn, WithTimeout(timeout)).Run(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("returns the error from runFn", func(t *testing.T) {
		ctx := context.Background()
		timeout := 50 * time.Millisecond

		expectedErr := errors.New("something went wrong")
		runFn := func(ctx context.Context) error {
			return expectedErr
		}

		err := New("", runFn, WithTimeout(timeout)).Run(ctx)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("cancelled context gets passed correctly", func(t *testing.T) {
		ctx := context.Background()
		timeout := 100 * time.Millisecond

		var err error
		runFn := func(ctx context.Context) error {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				return nil
			}
		}

		_ = New("", runFn, WithTimeout(timeout)).Run(ctx)
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestPeriodically(t *testing.T) {
	t.Run("stops running when the context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		count := 0
		runFn := func(ctx context.Context) error {
			count++
			return nil
		}
		interval := time.Millisecond * 100

		go func() {
			time.Sleep(time.Millisecond * 500)
			cancel()
		}()

		err := New("", runFn, Periodically(interval, nil)).Run(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Greater(t, count, 4)
		assert.LessOrEqual(t, count, 6)
	})

	t.Run("stops running when the channel is closed", func(t *testing.T) {
		ctx := context.Background()
		count := 0
		runFn := func(ctx context.Context) error {
			count++
			return nil
		}
		interval := time.Millisecond * 100
		stopChan := make(chan struct{})

		go func() {
			time.Sleep(time.Millisecond * 500)
			close(stopChan)
		}()

		err := New("", runFn, Periodically(interval, stopChan)).Run(ctx)
		assert.NoError(t, err)
		assert.Greater(t, count, 4)
		assert.LessOrEqual(t, count, 6)
	})

	t.Run("returns error", func(t *testing.T) {
		ctx := context.Background()
		count := 0
		runFn := func(ctx context.Context) error {
			count++
			if count == 3 {
				return errors.New("something went wrong")
			}
			return nil
		}
		interval := time.Millisecond * 100

		err := New("", runFn, Periodically(interval, nil)).Run(ctx)
		assert.ErrorContains(t, err, "something went wrong")
		assert.Equal(t, 3, count)
	})
}

func TestDisableErrorPropagation(t *testing.T) {
	t.Run("stops error from bubbling up", func(t *testing.T) {
		ctx := context.Background()
		runFn := func(ctx context.Context) error {
			return errors.New("something went wrong")
		}

		err := New("", runFn, DisableErrorPropagation()).Run(ctx)
		assert.NoError(t, err)
	})
}
