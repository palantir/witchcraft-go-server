// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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

package tcpjson

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsyncWriter(t *testing.T) {
	out := &bytes.Buffer{}
	w := StartAsyncWriter(out, metrics.DefaultMetricsRegistry)
	for i := 0; i < 5; i++ {
		str := strconv.Itoa(i)
		go func() {
			_, _ = w.Write([]byte(str))
		}()
	}
	time.Sleep(time.Millisecond)

	written := out.String()
	t.Log(written)
	assert.Len(t, written, 5)
	for i := 0; i < 5; i++ {
		assert.Contains(t, written, strconv.Itoa(i))
	}

	t.Run("fails when closed", func(t *testing.T) {
		require.NoError(t, w.Close())
		_, err := w.Write([]byte("will fail!"))
		require.EqualError(t, err, "write to closed asyncWriter")
	})
}
