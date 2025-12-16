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

// Supplier is a generic interface for a function that produces a value of type R
// without requiring any input arguments.
//
// The type parameter R represents the output type that the supplier produces.
// The Get method takes a context for cancellation/timeout support and returns
// both the result and an error if the operation fails.
type Supplier[R any] interface {
	Get(ctx context.Context) (R, error)
}

// SupplierFunc is a named type for a function that implements the Supplier interface.
type SupplierFunc[R any] func(ctx context.Context) (R, error)

// Get calls the underlying function with the provided context.
func (f SupplierFunc[R]) Get(ctx context.Context) (R, error) {
	return f(ctx)
}

// NewSupplierFromFunc returns a Supplier[R] by using type conversion to convert the provided function to a SupplierFunc.
func NewSupplierFromFunc[R any](funcToCall func(ctx context.Context) (R, error)) Supplier[R] {
	return SupplierFunc[R](funcToCall)
}

// NewSupplierFromFunction returns a Supplier[R] created from the provided Function[T, R] and argument. The returned Supplier's Get function calls the Apply function of the provided Function with the provided argument, ignores the R value returned by the function, and returns the error returned by the function.
func NewSupplierFromFunction[T any, R any](arg T, function Function[T, R]) Supplier[R] {
	return NewSupplierFromFunc(func(ctx context.Context) (R, error) {
		return function.Apply(ctx, arg)
	})
}

// NewSupplierFromValue returns a Supplier[R] that implements the Get function by returning the provided value of type R.
func NewSupplierFromValue[R any](value R) Supplier[R] {
	return NewSupplierFromFunc(func(_ context.Context) (R, error) {
		return value, nil
	})
}
