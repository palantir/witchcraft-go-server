// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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

package witchcraft

import (
	"context"

	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-tasks/function"
	"github.com/palantir/witchcraft-go-tasks/runnable"
)

// RunnableManager manages the lifecycle of long-running background tasks within a witchcraft server.
// It provides mechanisms to register and supervise runnables that are expected to run for the
// lifetime of the server. If any registered runnable terminates (either with an error or unexpectedly),
// the manager will initiate a server shutdown.
type RunnableManager interface {
	// AddForeverRunnable registers one or more NamedRunnables that are expected to run indefinitely
	// for the lifetime of the server. Each runnable is started in its own goroutine and wrapped with
	// service logging and fatal error handling. If any runnable returns (with or without an error),
	// the server will be shut down, as this indicates an unexpected termination of a critical
	// background task. The provided context should be the server's context, which will be used
	// for cancellation propagation and logging.
	AddForeverRunnable(ctx context.Context, namedRunnables ...function.NamedRunnable)
}

type defaultRunnableManager struct {
	serverShutdown func(ctx context.Context)
}

func NewRunnableManager(serverShutdown func(ctx context.Context)) RunnableManager {
	return &defaultRunnableManager{
		serverShutdown: serverShutdown,
	}
}

func (d *defaultRunnableManager) AddForeverRunnable(ctx context.Context, namedRunnables ...function.NamedRunnable) {
	for _, runnable := range namedRunnables {
		d.startRunnable(ctx, runnable)
	}
}

func (d *defaultRunnableManager) startRunnable(ctx context.Context, runnableArg function.NamedRunnable) {
	finalRunnable := runnable.WithWrappers(
		runnable.WithServiceLogging(),
		runnable.WithFatalLogging(),
	)(runnableArg)
	go func() {
		err := finalRunnable.Run(ctx)
		if err != nil {
			svc1log.FromContext(ctx).Error("Terminal runnable returned an error; shutting down server", svc1log.Stacktrace(err))
			d.serverShutdown(ctx)
			return
		}
		svc1log.FromContext(ctx).Error("Terminal runnable unexpectedly terminated, shutting down server")
		d.serverShutdown(ctx)
		return
	}()

}
