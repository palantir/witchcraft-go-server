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

type parallelRunnable struct {
	name               string
	maxParallelWorkers uint
	runnables          []function.NamedRunnable
	wrappers           []Wrapper
}

// NewParallel combines multiple runnables that run in parallel into a single runnable.
// The optional wrappers are stacked on top of each underneath runnable in order.
// Waits for all runnables to finish and returns the first error.
// Returns nil at the end if all runnables succeed.
// A value of zero maxParallelWorkers means unbounded parallel workers.
func NewParallel(name string, maxParallelWorkers uint, runnables []function.NamedRunnable, wrappers ...Wrapper) function.NamedRunnable {
	return &parallelRunnable{
		name:               name,
		maxParallelWorkers: maxParallelWorkers,
		runnables:          runnables,
		wrappers:           wrappers,
	}
}

func (p *parallelRunnable) Run(ctx context.Context) error {
	numRunnables := uint(len(p.runnables))
	parallelWorkers := numRunnables
	if p.maxParallelWorkers > 0 && parallelWorkers > p.maxParallelWorkers {
		parallelWorkers = p.maxParallelWorkers
	}
	return parallel.For(ctx, parallelWorkers, numRunnables, func(ctx context.Context, idx uint) error {
		return WithWrappers(p.wrappers...)(p.runnables[idx]).Run(ctx)
	})
}

func (p *parallelRunnable) Name() string {
	return p.name
}
