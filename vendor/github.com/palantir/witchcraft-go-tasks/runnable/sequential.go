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

type sequentialRunnable struct {
	name      string
	runnables []function.NamedRunnable
	wrappers  []Wrapper
}

// NewSequential combines multiple runnables that run in sequence into a single runnable.
// The optional wrappers are stacked on top of each underneath runnable in order.
// Stops execution at the first runnable that returns a non nil error and returns such error.
// Returns nil at the end if all runnables succeed.
func NewSequential(name string, runnables []function.NamedRunnable, wrappers ...Wrapper) function.NamedRunnable {
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
