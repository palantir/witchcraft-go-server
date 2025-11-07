package runnable

import (
	"context"
)

// RunFn is the function type for a runnable.
type RunFn func(ctx context.Context) error

// Runnable is a named object that can be run.
type Runnable interface {
	Run(ctx context.Context) error
	Name() string
}

type genericRunnable struct {
	name  string
	runFn RunFn
}

// New creates a new runnable from the name and function provided.
// The optional Wrappers are stacked in order.
// This is useful to convert RunFn's into runnable objects in place. Example:
//
//	runnable := New(name, runFn)
//
// Or:
//
//	runnable := New(name, func(ctx context.Context) error {
//		/* stuff to be run */
//	})
func New(name string, runFn RunFn, wrappers ...Wrapper) Runnable {
	runnable := &genericRunnable{
		name:  name,
		runFn: runFn,
	}
	return WithWrappers(wrappers...)(runnable)
}

func (r *genericRunnable) Name() string {
	return r.name
}

func (r *genericRunnable) Run(ctx context.Context) error {
	return r.runFn(ctx)
}
