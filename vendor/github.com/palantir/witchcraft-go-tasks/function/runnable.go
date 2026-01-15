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

package function

import (
	"context"
)

// Runnable is a functional interface for executing an operation that may fail.
//
// The Run method takes a context for cancellation/timeout support and returns
// an error if the operation fails. Runnables are commonly used for background tasks,
// periodic jobs, and composable execution units.
type Runnable interface {
	Run(ctx context.Context) error
}

// NamedRunnable is a Runnable that has a name.
type NamedRunnable interface {
	Runnable
	// Name returns the name for the runnable. A Runnable's name is mostly informational, and generally used to provide a human-readable name for a runnable that can be logged or used for debugging.
	Name() string
}

type defaultNamedRunnable struct {
	name     string
	runnable Runnable
}

func NewNamedRunnable(name string, runnable Runnable) NamedRunnable {
	return &defaultNamedRunnable{
		name:     name,
		runnable: runnable,
	}
}

func (d *defaultNamedRunnable) Run(ctx context.Context) error {
	return d.runnable.Run(ctx)
}

func (d *defaultNamedRunnable) Name() string {
	return d.name
}

// RunnableFunc is a named type for a function that implements the Runnable interface.
type RunnableFunc func(ctx context.Context) error

// Run implements the Runnable interface
func (f RunnableFunc) Run(ctx context.Context) error {
	return f(ctx)
}

// NewRunnableFromFunc creates a Runnable by using type conversion to convert the provided function to a RunnableFunc.
func NewRunnableFromFunc(funcToCall func(ctx context.Context) error) Runnable {
	return RunnableFunc(funcToCall)
}

// NewRunnableFromFunction returns a Runnable created from the provided Function[T, R] and argument. The returned Runnable's Run function calls the Apply function of the provided Function with the provided argument, ignores the R value returned by the function, and returns the error returned by the function.
func NewRunnableFromFunction[T, R any](function Function[T, R], arg T) Runnable {
	return NewRunnableFromConsumer(arg, NewConsumerFromFunction(function))
}

// NewRunnableFromConsumer returns a Runnable based on the provided Consumer[T] and argument. The returned Runnable's Run function calls the Accept function of the provided consumer with the provided argument and returns the error returned by the function.
func NewRunnableFromConsumer[T any](arg T, consumer Consumer[T]) Runnable {
	return NewRunnableFromFunc(func(ctx context.Context) error {
		return consumer.Accept(ctx, arg)
	})
}

// NewRunnableFromSupplier returns a Runnable created from the provided Supplier[R]. The returned Runnable's Run function calls the Get function of the provided Supplier, ignores the R value returned by the function, and returns the error returned by the function.
func NewRunnableFromSupplier[R any](supplier Supplier[R]) Runnable {
	return NewRunnableFromFunc(func(ctx context.Context) error {
		_, err := supplier.Get(ctx)
		return err
	})
}
