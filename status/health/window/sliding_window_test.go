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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeWindowedEventStorer_ErrorOnCreate(t *testing.T) {
	_, err := NewTimeWindowedEventStorer(0)
	assert.Error(t, err)
}

func TestTimeWindowedEventStorer_NoEvents(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(time.Millisecond)
	assert.NoError(t, err)
	errors := manager.GetEventsInWindow()
	assert.Nil(t, errors)
}

func TestTimeWindowedEventStorer_AllEventsUpToDate(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(time.Second)
	assert.NoError(t, err)
	manager.SubmitEvent("payload #1")
	manager.SubmitEvent("payload #2")
	manager.SubmitEvent("payload #3")
	events := manager.GetEventsInWindow()
	assert.EqualValues(t, len(events), 3)
	assert.EqualValues(t, "payload #1", events[0].Payload)
	assert.EqualValues(t, "payload #2", events[1].Payload)
	assert.EqualValues(t, "payload #3", events[2].Payload)
}

func TestTimeWindowedEventStorer_AllEventsOutOfDate(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(50 * time.Millisecond)
	assert.NoError(t, err)
	manager.SubmitEvent("payload #1")
	manager.SubmitEvent("payload #2")
	manager.SubmitEvent("payload #3")
	<-time.After(100 * time.Millisecond)
	events := manager.GetEventsInWindow()
	assert.Empty(t, events)
}

func TestTimeWindowedEventStorer_SomeEventsOutOfDate(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(500 * time.Millisecond)
	assert.NoError(t, err)
	manager.SubmitEvent("payload #1")
	manager.SubmitEvent("payload #2")
	<-time.After(time.Second)
	manager.SubmitEvent("payload #3")
	manager.SubmitEvent("payload #4")
	events := manager.GetEventsInWindow()
	assert.EqualValues(t, len(events), 2)
	assert.EqualValues(t, "payload #3", events[0].Payload)
	assert.EqualValues(t, "payload #4", events[1].Payload)
}
