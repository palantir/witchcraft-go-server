// Copyright (c) 2022 Palantir Technologies. All rights reserved.
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

package dependencyhealth

import (
	"sync"
	"time"
)

// timeBucket measure the number of times Mark is called within the bucket interval.
// This is necessary over using gometrics.Meter because it only updates every 5 seconds
// but our results must be read-after-write consistent.
type timeBucket struct {
	mux    sync.Mutex
	bucket time.Duration
	now    func() time.Time
	slice  []time.Time
}

func newTimeBucket(bucket time.Duration) *timeBucket {
	return &timeBucket{
		bucket: bucket,
		now:    time.Now,
		slice:  make([]time.Time, 0),
	}
}

// Mark prunes any old bucket entries and appends a new b.now value.
func (b *timeBucket) Mark() {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.prune()
	b.slice = append(b.slice, b.now())
}

// Count prunes any old bucket entries and returns the number of entries within the window.
func (b *timeBucket) Count() int {
	b.mux.Lock()
	defer b.mux.Unlock()
	b.prune()
	return len(b.slice)
}

func (b *timeBucket) prune() {
	horizon := b.now().Add(-1 * b.bucket)
	// short circuit when slice does not need pruning
	if len(b.slice) == 0 || b.slice[0].After(horizon) {
		return
	}

	trimIdx := -1
	for i, t := range b.slice {
		if t.After(horizon) {
			trimIdx = i
			break
		}
	}
	if trimIdx == -1 {
		// no entries within window
		b.slice = nil
	} else {
		newSlice := make([]time.Time, len(b.slice)-trimIdx)
		copy(newSlice, b.slice[trimIdx:])
		b.slice = newSlice
	}
}
