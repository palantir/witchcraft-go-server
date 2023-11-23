// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

package wapp

import (
	"context"
	"runtime/debug"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

// RunWithRecoveryLogging wraps a callback, logging any panics recovered as errors.
// Useful as a "catch all" for applications so that they can log fatal events, perhaps before exiting.
func RunWithRecoveryLogging(ctx context.Context, runFn func(ctx context.Context)) {
	defer func() {
		if r := recover(); r != nil {
			_ = handleRecovered(ctx, r, debug.Stack())
		}
	}()
	runFn(ctx)
}

// RunWithFatalLogging wraps a callback, logging errors and panics it returns.
// Useful as a "catch all" for applications so that they can log fatal events, perhaps before exiting.
func RunWithFatalLogging(ctx context.Context, runFn func(ctx context.Context) error) (retErr error) {
	defer func() {
		if retErr != nil {
			svc1log.FromContext(ctx).Error("error", svc1log.Stacktrace(retErr))
		}
		if r := recover(); r != nil {
			if recovered := handleRecovered(ctx, r, debug.Stack()); retErr == nil {
				retErr = recovered
			}
		}
	}()
	return runFn(ctx)
}

// RunWithRecoveryLoggingWithError is identical to RunWithFatalLogging however it only emits logs on panics, not if runFn a normal error
// This can be useful if you want to special case the logging of this error but still want a centralized place to handle panics
func RunWithRecoveryLoggingWithError(ctx context.Context, runFn func(ctx context.Context) error) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			if recovered := handleRecovered(ctx, r, debug.Stack()); retErr == nil {
				retErr = recovered
			}
		}
	}()
	return runFn(ctx)
}

func handleRecovered(ctx context.Context, r interface{}, stack []byte) (retErr error) {
	stacktrace := diag1log.ThreadDumpV1FromGoroutines(stack)
	if err, ok := r.(error); ok {
		svc1log.FromContext(ctx).Error("panic recovered",
			svc1log.SafeParam("stacktrace", stacktrace),
			svc1log.Stacktrace(err))
		retErr = werror.WrapWithContextParams(ctx, err, "panic recovered",
			werror.SafeParam("stacktrace", stacktrace))
	} else {
		svc1log.FromContext(ctx).Error("panic recovered",
			svc1log.SafeParam("stacktrace", stacktrace),
			svc1log.UnsafeParam("recovered", r))
		retErr = werror.ErrorWithContextParams(ctx, "panic recovered",
			werror.SafeParam("stacktrace", stacktrace),
			werror.UnsafeParam("recovered", r))
	}
	evt2log.FromContext(ctx).Event("wapp.panic_recovered",
		evt2log.Value("stacktrace", stacktrace),
		evt2log.UnsafeParam("recovered", r))
	return retErr
}
