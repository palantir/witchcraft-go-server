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

	"github.com/palantir/witchcraft-go-tasks/function"
	"github.com/palantir/witchcraft-go-tasks/parallel"
)

type foreverRunnable struct {
	name      string
	runnables []function.NamedRunnable
	wrappers  []Wrapper
}

// NewForever combines multiple runnables intended to run indefinitely, such as controllers, to run as a single runnable in parallel.
// The optional wrappers are stacked on top of each underneath runnable in order.
// On panic or return of a runnable, the context is cancelled and NewForever returns.
func NewForever(name string, runnables []function.NamedRunnable, wrappers ...Wrapper) function.NamedRunnable {
	return &foreverRunnable{
		name:      name,
		runnables: runnables,
		wrappers:  wrappers,
	}
}

func (p *foreverRunnable) Run(ctx context.Context) error {
	numRunnables := uint(len(p.runnables))
	parallel.Forever(ctx, numRunnables, func(ctx context.Context, idx uint) error {
		return WithWrappers(p.wrappers...)(p.runnables[idx]).Run(ctx)
	})
	return nil
}

func (p *foreverRunnable) Name() string {
	return p.name
}
