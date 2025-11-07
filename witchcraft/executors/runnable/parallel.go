package runnable

import (
	"context"
)

type parallelRunnable struct {
	name               string
	maxParallelWorkers uint
	runnables          []Runnable
	wrappers           []Wrapper
}

// NewParallel combines multiple runnables that run in parallel into a single runnable.
// The optional wrappers are stacked on top of each underneath runnable in order.
// Waits for all runnables to finish and returns the first error.
// Returns nil at the end if all runnables succeed.
// A value of zero maxParallelWorkers means unbounded parallel workers.
func NewParallel(name string, maxParallelWorkers uint, runnables []Runnable, wrappers ...Wrapper) Runnable {
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
	return For(ctx, parallelWorkers, numRunnables, func(ctx context.Context, idx uint) error {
		return WithWrappers(p.wrappers...)(p.runnables[idx]).Run(ctx)
	})
}

func (p *parallelRunnable) Name() string {
	return p.name
}
