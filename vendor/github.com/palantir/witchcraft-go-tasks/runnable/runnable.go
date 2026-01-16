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
)

// RunFn is the function signature for a runnable operation.
// It takes a context for cancellation/timeout support and returns an error if the operation fails.
type RunFn func(ctx context.Context) error

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
func New(name string, runFn RunFn, wrappers ...Wrapper) function.NamedRunnable {
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
