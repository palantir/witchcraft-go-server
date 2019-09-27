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

// Event is a struct that keeps a generic payload describing an event and a timestamp for when the event happened.
type Event struct {
	// Time is the time the event was submitted.
	Time time.Time
	// Payload is any generic information about this event.
	Payload interface{}
}

// TimeWindowedEventStorer is a thread-safe struct that stores submitted events
// and supports polling for all events submitted within the last windowSize period.
// When any operation is made, all out-of-date events are pruned out of memory.
type TimeWindowedEventStorer struct {
	events      []Event
	eventsMutex sync.Mutex
	windowSize  time.Duration
}

// NewTimeWindowedEventStorer creates a new TimeWindowedEventStorer with the provided windowSize.
func NewTimeWindowedEventStorer(windowSize time.Duration) (*TimeWindowedEventStorer, error) {
	if windowSize <= 0 {
		return nil, werror.Error("attempted to create a sliding window with non positive size")
	}
	return &TimeWindowedEventStorer{
		windowSize: windowSize,
	}, nil
}

func (t *TimeWindowedEventStorer) pruneOldEvents() {
	currentTime := time.Now()
	newStartIndex := 0
	for index, entry := range t.events {
		if currentTime.Sub(entry.Time) <= t.windowSize {
			break
		}
		newStartIndex = index + 1
	}
	t.events = t.events[newStartIndex:]
}

// SubmitEvent prunes all out-of-date events out of memory and then adds a new one.
func (t *TimeWindowedEventStorer) SubmitEvent(payload interface{}) {
	t.eventsMutex.Lock()
	defer t.eventsMutex.Unlock()

	t.pruneOldEvents()
	t.events = append(t.events, Event{
		Time:    time.Now(),
		Payload: payload,
	})
}

// GetEventsInWindow prunes all out-of-date events out of memory and then returns all up-to-date events.
func (t *TimeWindowedEventStorer) GetEventsInWindow() []Event {
	t.eventsMutex.Lock()
	defer t.eventsMutex.Unlock()

	t.pruneOldEvents()
	return t.events
}
