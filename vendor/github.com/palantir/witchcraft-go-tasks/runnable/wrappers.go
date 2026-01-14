// Copyright (c) 2025 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package runnable

import (
	"context"
	"time"

	"github.com/palantir/witchcraft-go-health/v2/sources/window"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"github.com/palantir/witchcraft-go-tasks/function"
)

// ErrorFn is a callback function for handling errors from runnable execution.
// It receives the context and the error that occurred.
type ErrorFn func(ctx context.Context, err error)

// Wrapper is a function that wraps a NamedRunnable with additional behavior,
// returning a new NamedRunnable. Wrappers can be composed to add logging,
// error handling, timeouts, and other cross-cutting concerns.
type Wrapper func(runnable function.NamedRunnable) function.NamedRunnable

// WithWrappers combines multiple wrappers into a single wrapper.
// Wrappers are applied in order, so the first wrapper in the list becomes the outermost layer.
// For example, WithWrappers(A, B)(runnable) results in A(B(runnable)).
func WithWrappers(wrappers ...Wrapper) Wrapper {
	return func(runnable function.NamedRunnable) function.NamedRunnable {
		for _, wrapper := range wrappers {
			runnable = wrapper(runnable)
		}
		return runnable
	}
}

// WithFatalLogging wraps a runnable to recover from panics and log them as fatal errors.
// If the runnable panics, the panic is recovered, logged with a stacktrace, and returned as an error.
func WithFatalLogging() Wrapper {
	return func(runnable function.NamedRunnable) function.NamedRunnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			return wapp.RunWithFatalLogging(ctx, runnable.Run)
		})
	}
}

// WithServiceLogging adds the runnable name to the context and
// adds log lines at the start and at the end of the execution.
func WithServiceLogging() Wrapper {
	return func(runnable function.NamedRunnable) function.NamedRunnable {
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
	return func(runnable function.NamedRunnable) function.NamedRunnable {
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
	return func(runnable function.NamedRunnable) function.NamedRunnable {
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
	return func(runnable function.NamedRunnable) function.NamedRunnable {
		return New(runnable.Name(), func(ctx context.Context) error {
			err := runnable.Run(ctx)
			telemetry.Submit(ctx, runnable.Name(), err)
			return err
		})
	}
}

// WithTimeout returns a runnable wrapper that runs the runnable with the given timeout.
// If the runnable exceeds the timeout Context.DeadlineExceeded error will be returned.
func WithTimeout(timeout time.Duration) Wrapper {
	return func(runnable function.NamedRunnable) function.NamedRunnable {
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
	return func(runnable function.NamedRunnable) function.NamedRunnable {
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
	return func(runnable function.NamedRunnable) function.NamedRunnable {
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
