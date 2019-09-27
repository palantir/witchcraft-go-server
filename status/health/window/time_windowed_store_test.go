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
	"github.com/stretchr/testify/require"
)

func TestTimeWindowedStore_ErrorOnCreate(t *testing.T) {
	_, err := NewTimeWindowedStore(0)
	// This should error as 0 is not a valid windowSize.
	assert.Error(t, err)
}

func TestTimeWindowedStore_NoItems(t *testing.T) {
	store, err := NewTimeWindowedStore(time.Millisecond)
	require.NoError(t, err)
	items := store.ItemsInWindow()
	assert.Nil(t, items)
}

func TestTimeWindowedStore_AllItemsUpToDate(t *testing.T) {
	store, err := NewTimeWindowedStore(time.Second)
	require.NoError(t, err)
	store.Submit("item #1")
	store.Submit("item #2")
	store.Submit("item #3")
	items := store.ItemsInWindow()
	assert.Equal(t, len(items), 3)
	assert.Equal(t, "item #1", items[0].Item)
	assert.Equal(t, "item #2", items[1].Item)
	assert.Equal(t, "item #3", items[2].Item)
}

func TestTimeWindowedStore_AllItemsOutOfDate(t *testing.T) {
	store, err := NewTimeWindowedStore(50 * time.Millisecond)
	require.NoError(t, err)
	store.Submit("item #1")
	store.Submit("item #2")
	store.Submit("item #3")
	<-time.After(100 * time.Millisecond)
	items := store.ItemsInWindow()
	assert.Empty(t, items)
}

func TestTimeWindowedStore_SomeItemsOutOfDate(t *testing.T) {
	store, err := NewTimeWindowedStore(50 * time.Millisecond)
	require.NoError(t, err)
	store.Submit("item #1")
	store.Submit("item #2")
	<-time.After(100 * time.Millisecond)
	store.Submit("item #3")
	store.Submit("item #4")
	items := store.ItemsInWindow()
	assert.Equal(t, len(items), 2)
	assert.Equal(t, "item #3", items[0].Item)
	assert.Equal(t, "item #4", items[1].Item)
}
