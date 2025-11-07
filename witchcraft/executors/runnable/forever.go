package runnable

import (
	"context"
)

type foreverRunnable struct {
	name      string
	runnables []Runnable
	wrappers  []Wrapper
}

// NewForever combines multiple runnables intended to run indefinitely, such as controllers, to run as a single runnable in parallel.
// The optional wrappers are stacked on top of each underneath runnable in order.
// On panic or return of a runnable, the context is cancelled and NewForever returns.
func NewForever(name string, runnables []Runnable, wrappers ...Wrapper) Runnable {
	return &foreverRunnable{
		name:      name,
		runnables: runnables,
		wrappers:  wrappers,
	}
}

func (p *foreverRunnable) Run(ctx context.Context) error {
	numRunnables := uint(len(p.runnables))
	Forever(ctx, numRunnables, func(ctx context.Context, idx uint) error {
		return WithWrappers(p.wrappers...)(p.runnables[idx]).Run(ctx)
	})
	return nil
}

func (p *foreverRunnable) Name() string {
	return p.name
}
