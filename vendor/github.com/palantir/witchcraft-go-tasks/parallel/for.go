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

package parallel

import (
	"context"

	"github.com/palantir/witchcraft-go-logging/wlog/wapp"
	"golang.org/x/sync/errgroup"
)

// ForFn is a function that will be run inside the parallel for loop.
// If you need a for loop range over a generic type slice, use a lambda function to get the object from the index.
// For example, if you want to do a parallel for over an array of strings arr, do:
//
// var arr []string
// /* add stuff to arr */
//
//	err := parallel.For(ctx, numWorkers, uint(len(arr)), func(ctx context.Context, idx uint) error {
//			str := arr[idx]
//			err := /* do stuff with str */
//			svc1log.FromContext(ctx).Error(/* possibly log err here */)
//			return err
//	})
type ForFn func(ctx context.Context, idx uint) error

// For performs a for loop in parallel using an index channel. This technique is known in Java as a thread pool.
// You can specify the maximum amount of workers and the len of the for loop.
// If any iteration returns an error or panics, the first error or panic to happen is returned by the loop using the wapp.RunWithRecoveryLoggingWithError wrap.
// If you want to log all errors, do so inside the forFn.
func For(ctx context.Context, numWorkers uint, numIndexes uint, forFn ForFn) error {
	if numIndexes == 0 {
		return nil
	}

	indexChan := make(chan uint, numIndexes)
	for idx := range numIndexes {
		indexChan <- idx
	}
	close(indexChan)

	if numWorkers > numIndexes {
		numWorkers = numIndexes
	}
	var errGroup errgroup.Group
	for w := uint(0); w < numWorkers; w++ {
		errGroup.Go(func() error {
			return wapp.RunWithRecoveryLoggingWithError(ctx, func(ctx context.Context) error {
				return worker(ctx, indexChan, forFn)
			})
		})
	}
	return errGroup.Wait()
}

func worker(ctx context.Context, indexChan chan uint, forFn ForFn) error {
	var firstErr error
	for idx := range indexChan {
		// preserve the worker thread in case of a panic
		err := wapp.RunWithFatalLogging(ctx, func(ctx context.Context) error {
			return forFn(ctx, idx)
		})
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Forever runs numWorkers goroutines in parallel. The goroutines are expected to run forever (i.e. controllers).
// If any of the goroutines panic or error, the panic or error will be recovered from and logged by wapp.RunWithFatalLogging.
// Forever returns after the first worker panics or exits. At this moment it cancels the
// context of all other go routines, allowing the calling code to perform necessary shutdown.
func Forever(ctx context.Context, numWorkers uint, forFn ForFn) {
	if numWorkers == 0 {
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for w := range numWorkers {
		go func(idx uint) {
			defer cancel()
			_ = wapp.RunWithFatalLogging(ctx, func(ctx context.Context) error {
				return forFn(ctx, idx)
			})
		}(w)
	}
	<-ctx.Done()
}
