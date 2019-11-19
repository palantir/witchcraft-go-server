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

func assertStoreContent(t *testing.T, store TimedKeyStore, list []string, subtestName string) {
	t.Run(subtestName, func(t *testing.T) {
		if len(list) > 0 {
			assert.Equal(t, list, store.List().Keys())
		} else {
			assert.Nil(t, store.List())
		}

		oldest, nonEmpty := store.Oldest()
		if len(list) > 0 {
			assert.Equal(t, list[0], oldest.Key)
			assert.True(t, nonEmpty)
		} else {
			assert.False(t, nonEmpty)
		}

		newest, nonEmpty := store.Newest()
		if len(list) > 0 {
			assert.Equal(t, list[len(list)-1], newest.Key)
			assert.True(t, nonEmpty)
		} else {
			assert.False(t, nonEmpty)
		}

		var lastTime time.Time
		for _, key := range list {
			timedKey, exists := store.Get(key)
			assert.True(t, exists)
			assert.Equal(t, key, timedKey.Key)
			assert.True(t, lastTime.Before(timedKey.Time))
			lastTime = timedKey.Time
		}
	})
}

func TestTimedKeyStore(t *testing.T) {
	store := NewTimedKeyStore()
	assertStoreContent(t, store, []string{}, "store initially empty")

	store.Delete("a")
	assertStoreContent(t, store, []string{}, "removed unexisting key from empty store")

	store.Put("a")
	assertStoreContent(t, store, []string{"a"}, "added a single key a")

	store.Delete("a")
	assertStoreContent(t, store, []string{}, "removed a single key a")

	store.Put("b")
	assertStoreContent(t, store, []string{"b"}, "added a single key b")

	store.Put("c")
	assertStoreContent(t, store, []string{"b", "c"}, "added a second key c")

	store.Put("b")
	assertStoreContent(t, store, []string{"c", "b"}, "updated key b")

	store.Delete("d")
	assertStoreContent(t, store, []string{"c", "b"}, "removed unexisting key from non empty store")

	store.Put("e")
	assertStoreContent(t, store, []string{"c", "b", "e"}, "added a third key e")

	store.Delete("b")
	assertStoreContent(t, store, []string{"c", "e"}, "removed key b from the middle of the list")

	store.Put("d")
	assertStoreContent(t, store, []string{"c", "e", "d"}, "added a new key d")

	store.Delete("d")
	assertStoreContent(t, store, []string{"c", "e"}, "removed newest key d")

	store.Delete("c")
	assertStoreContent(t, store, []string{"e"}, "removed oldest key c")

	store.Delete("e")
	assertStoreContent(t, store, []string{}, "removed the last key e")
}
