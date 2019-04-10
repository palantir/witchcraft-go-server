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

package witchcraft

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSamplerForRate(t *testing.T) {
	for _, test := range []struct {
		rate     float64
		expected [16]bool
	}{
		{
			rate: -1,
			expected: [16]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate: 0,
			expected: [16]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate: 0.123,
			expected: [16]bool{true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate: 0.25,
			expected: [16]bool{true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate: 0.9,
			expected: [16]bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
		},
		{
			rate: 1,
			expected: [16]bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
		},
		{
			rate: 2,
			expected: [16]bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
		},
	} {
		t.Run(fmt.Sprint(test.rate), func(t *testing.T) {
			sampler := (&Server{}).samplerForRate(test.rate)
			for i, id := range []uint64{
				0x0000000000000000,
				0x1111111111111111,
				0x2222222222222222,
				0x3333333333333333,
				0x4444444444444444,
				0x5555555555555555,
				0x6666666666666666,
				0x7777777777777777,
				0x8888888888888888,
				0x9999999999999999,
				0xaaaaaaaaaaaaaaaa,
				0xbbbbbbbbbbbbbbbb,
				0xcccccccccccccccc,
				0xdddddddddddddddd,
				0xeeeeeeeeeeeeeeee,
				0xffffffffffffffff,
			} {
				t.Run(fmt.Sprintf("%x", id), func(t *testing.T) {
					assert.Equal(t, test.expected[i], sampler(id))
				})
			}
		})
	}
}
