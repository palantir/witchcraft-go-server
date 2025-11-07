package runnable

import (
	"context"
	"time"

	"github.com/palantir/witchcraft-go-health/sources/window"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
)

// ErrorFn is an error handler.
type ErrorFn func(ctx context.Context, err error)

// Wrapper is a method that envolves a runnable with some logic generating another runnable.
type Wrapper func(runnable Runnable) Runnable

// WithWrappers stacks an array of wrappers in order.
func WithWrappers(wrappers ...Wrapper) Wrapper {
	return func(runnable Runnable) Runnable {
		for _, wrapper := range wrappers {
			runnable = wrapper(runnable)
		}
		return runnable
	}
}

// WithFatalLogging envolves a runnable with fatal logging.
func WithFatalLogging() Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			return wapp.RunWithFatalLogging(ctx, runnable.Run)
		})
	}
}

// WithServiceLogging adds the runnable name to the context and
// adds log lines at the start and at the end of the execution.
func WithServiceLogging() Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			ctx = svc1log.WithLoggerParams(ctx, svc1log.SafeParam("runnableName", runnable.Name()))
			svc1log.FromContext(ctx).Info("Starting runnable")
			if err := runnable.Run(ctx); err != nil {
				svc1log.FromContext(ctx).Error("Runnable errored", svc1log.Stacktrace(err))
				return err
			}
			svc1log.FromContext(ctx).Info("Runnable finished")
			return nil
		})
	}
}

// WithErrorHandler adds an error handler that is invoked when the runnable returns a non nil error.
// Does nothing if the error handler is nil.
func WithErrorHandler(errorFn ErrorFn) Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			if err := runnable.Run(ctx); err != nil {
				if errorFn != nil {
					errorFn(ctx, err)
				}
				return err
			}
			return nil
		})
	}
}

// DisableErrorPropagation returns a runnable wrapper that logs the error returned by the runnable instead of returning it.
func DisableErrorPropagation() Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			if err := runnable.Run(ctx); err != nil {
				svc1log.FromContext(ctx).Error("Error while running runnable", svc1log.Stacktrace(err))
			}
			return nil
		})
	}
}

// WithTelemetry returns a runnable wrapper that submits the error returned by the runnable to the given KeyedErrorHealthCheckSource.
func WithTelemetry(telemetry window.KeyedErrorHealthCheckSource) Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			err := runnable.Run(ctx)
			telemetry.Submit(runnable.Name(), err)
			return err
		})
	}
}

// WithTimeout returns a runnable wrapper that runs the runnable with the given timeout.
// If the runnable exceeds the timeout Context.DeadlineExceeded error will be returned.
func WithTimeout(timeout time.Duration) Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			errChan := make(chan error, 1)
			go func() {
				errChan <- runnable.Run(ctx)
			}()

			select {
			case err := <-errChan:
				return err
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}
}

// WhileChanNotClosed restarts the runnable when it finishes until it returns an error or until the channel is closed.
func WhileChanNotClosed(stopChan <-chan struct{}) Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			for {
				select {
				case <-stopChan:
					return nil
				default:
				}
				if err := runnable.Run(ctx); err != nil {
					return err
				}
			}
		})
	}
}

// Periodically returns a runnable wrapper that runs the runnable periodically with the given interval
// until it returns an error or until the channel is closed.
func Periodically(interval time.Duration, stopChan <-chan struct{}) Wrapper {
	return func(runnable Runnable) Runnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				if err := runnable.Run(ctx); err != nil {
					return err
				}

				select {
				case <-ticker.C:
					continue
				case <-stopChan:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		})
	}
}
