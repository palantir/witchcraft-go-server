// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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

package window

import (
	"sync"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
)

type errorEntry struct {
	time time.Time
	err  error
}

// TimeWindowedErrorStorer is a thread-safe struct that stores submitted errors
// and supports polling for all errors submitted within the last windowSize period.
// It includes nil error submissions.
// When any operation is made, all out-of-date errors are pruned out of memory.
type TimeWindowedErrorStorer struct {
	errors      []errorEntry
	errorsMutex sync.Mutex
	windowSize  time.Duration
}

// NewSlidingWindowManager creates a new TimeWindowedErrorStorer with the provided windowSize.
func NewSlidingWindowManager(windowSize time.Duration) (TimeWindowedErrorStorer, error) {
	if windowSize <= 0 {
		return TimeWindowedErrorStorer{}, werror.Error("attempted to create a sliding window with non positive size")
	}
	return TimeWindowedErrorStorer{
		windowSize: windowSize,
	}, nil
}

func (t *TimeWindowedErrorStorer) pruneOldErrors() {
	currentTime := time.Now()
	for i, entry := range t.errors {
		if currentTime.Sub(entry.time) > t.windowSize {
			t.errors = t.errors[i+1:]
		} else {
			break
		}
	}
}

// SubmitError prunes all out-of-date errors out of memory and then adds a new one.
func (t *TimeWindowedErrorStorer) SubmitError(err error) {
	t.errorsMutex.Lock()
	defer t.errorsMutex.Unlock()

	t.pruneOldErrors()
	t.errors = append(t.errors, errorEntry{
		time: time.Now(),
		err:  err,
	})
}

// GetErrors prunes all out-of-date errors out of memory and then returns all up-to-date errors.
func (t *TimeWindowedErrorStorer) GetErrors() []error {
	t.errorsMutex.Lock()
	defer t.errorsMutex.Unlock()

	t.pruneOldErrors()
	var result []error
	for _, entry := range t.errors {
		result = append(result, entry.err)
	}
	return result
}
