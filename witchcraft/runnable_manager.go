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

type RunnableManager interface {
	AddNamedRunnable(ctx context.Context, namedRunnables ...function.NamedRunnable)
}

type defaultRunnableManager struct {
	serverShutdown func(ctx context.Context)
}

func NewRunnableManager(serverShutdown func(ctx context.Context)) RunnableManager {
	return &defaultRunnableManager{
		serverShutdown: serverShutdown,
	}
}

func (d *defaultRunnableManager) AddNamedRunnable(ctx context.Context, namedRunnables ...function.NamedRunnable) {
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
