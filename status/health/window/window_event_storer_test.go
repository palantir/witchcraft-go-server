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

	"github.com/stretchr/testify/require"
)

func TestTimeWindowedEventStorer_ErrorOnCreate(t *testing.T) {
	_, err := NewTimeWindowedEventStorer(0)
	// This should error as 0 is not a valid windowSize.
	require.Error(t, err)
}

func TestTimeWindowedEventStorer_NoEvents(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(time.Millisecond)
	require.NoError(t, err)
	items := manager.GetItemsInWindow()
	require.Nil(t, items)
}

func TestTimeWindowedEventStorer_AllEventsUpToDate(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(time.Second)
	require.NoError(t, err)
	manager.Submit("payload #1")
	manager.Submit("payload #2")
	manager.Submit("payload #3")
	items := manager.GetItemsInWindow()
	require.Equal(t, len(items), 3)
	require.Equal(t, "payload #1", items[0].Payload)
	require.Equal(t, "payload #2", items[1].Payload)
	require.Equal(t, "payload #3", items[2].Payload)
}

func TestTimeWindowedEventStorer_AllEventsOutOfDate(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(50 * time.Millisecond)
	require.NoError(t, err)
	manager.Submit("payload #1")
	manager.Submit("payload #2")
	manager.Submit("payload #3")
	<-time.After(100 * time.Millisecond)
	items := manager.GetItemsInWindow()
	require.Empty(t, items)
}

func TestTimeWindowedEventStorer_SomeEventsOutOfDate(t *testing.T) {
	manager, err := NewTimeWindowedEventStorer(500 * time.Millisecond)
	require.NoError(t, err)
	manager.Submit("payload #1")
	manager.Submit("payload #2")
	<-time.After(time.Second)
	manager.Submit("payload #3")
	manager.Submit("payload #4")
	items := manager.GetItemsInWindow()
	require.Equal(t, len(items), 2)
	require.Equal(t, "payload #3", items[0].Payload)
	require.Equal(t, "payload #4", items[1].Payload)
}
