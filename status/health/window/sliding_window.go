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
)

type errorEntry struct {
	time time.Time
	err  error
}

// SlidingWindowManager is a thread safe struct where you can submit errors
// and poll for all errors submitted within the last windowSize period.
// It includes nil error submissions.
type SlidingWindowManager struct {
	errors      []errorEntry
	errorsMutex sync.Mutex
	windowSize  time.Duration
}

// NewSlidingWindowManager creates a new SlidingWindowManager with the provided windowSize.
func NewSlidingWindowManager(windowSize time.Duration) SlidingWindowManager {
	return SlidingWindowManager{
		windowSize: windowSize,
	}
}

func (s *SlidingWindowManager) pruneOldErrors() {
	var upToDateStatusEntries []errorEntry
	for _, entry := range s.errors {
		if time.Now().Sub(entry.time) <= s.windowSize {
			upToDateStatusEntries = append(upToDateStatusEntries, entry)
		}
	}
	s.errors = upToDateStatusEntries
}

// SubmitError prunes all out-of-date errors out of memory and adds a new one.
func (s *SlidingWindowManager) SubmitError(err error) {
	s.errorsMutex.Lock()
	defer s.errorsMutex.Unlock()

	s.pruneOldErrors()
	s.errors = append(s.errors, errorEntry{
		time: time.Now(),
		err:  err,
	})
}

// GetErrors prunes all out-of-date errors out of memory and returns all up-to-date errors.
func (s *SlidingWindowManager) GetErrors() []error {
	s.errorsMutex.Lock()
	defer s.errorsMutex.Unlock()

	s.pruneOldErrors()
	var result []error
	for _, entry := range s.errors {
		result = append(result, entry.err)
	}
	return result
}
