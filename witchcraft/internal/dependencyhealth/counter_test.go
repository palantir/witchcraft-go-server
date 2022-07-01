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
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimeBucket(t *testing.T) {
	bucket := &timeBucket{
		bucket: 10 * time.Second,
	}
	frozenTime := time.Now()
	setTime := func(seconds int) {
		bucket.now = func() time.Time { return frozenTime.Add(time.Duration(seconds) * time.Second) }
	}
	setTime(0)
	require.Equal(t, 0, bucket.Count())

	bucket.Mark()
	require.Equal(t, 1, bucket.Count())

	setTime(1)
	bucket.Mark()
	require.Equal(t, 2, bucket.Count())

	setTime(2)
	bucket.Mark()
	require.Equal(t, 3, bucket.Count())

	setTime(3)
	bucket.Mark()
	require.Equal(t, 4, bucket.Count())

	setTime(11)
	require.Equal(t, 2, bucket.Count())

	setTime(15)
	require.Equal(t, 0, bucket.Count())

	bucket.Mark()
	require.Equal(t, 1, bucket.Count())

	for i := 100; i < 200; i++ {
		setTime(i)
		bucket.Mark()
	}
	require.Equal(t, 10, bucket.Count())
	require.Less(t, cap(bucket.slice), 25, "slice grew very large within loop - will memory leak in the future.")
}
