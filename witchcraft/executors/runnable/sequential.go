package runnable

import (
	"context"
)

type sequentialRunnable struct {
	name      string
	runnables []Runnable
	wrappers  []Wrapper
}

// NewSequential combines multiple runnables that run in sequence into a single runnable.
// The optional wrappers are stacked on top of each underneath runnable in order.
// Stops execution at the first runnable that returns a non nil error and returns such error.
// Returns nil at the end if all runnables succeed.
func NewSequential(name string, runnables []Runnable, wrappers ...Wrapper) Runnable {
	return &sequentialRunnable{
		name:      name,
		runnables: runnables,
		wrappers:  wrappers,
	}
}

func (p *sequentialRunnable) Run(ctx context.Context) error {
	for _, runnable := range p.runnables {
		if err := WithWrappers(p.wrappers...)(runnable).Run(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *sequentialRunnable) Name() string {
	return p.name
}
