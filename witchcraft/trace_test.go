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
		expected map[uint64]bool
	}{
		{
			rate: 0,
			expected: map[uint64]bool{
				0: false,
				1: false,
				2: false,
			},
		},
		{
			rate: 1,
			expected: map[uint64]bool{
				0: true,
				1: true,
				2: true,
			},
		},
		{
			rate: 0.25,
			expected: map[uint64]bool{
				0: true,
				1: false,
				2: false,
				3: false,
				4: true,
				5: false,
			},
		},
		{
			rate: 0.123,
			expected: map[uint64]bool{
				0: true,
				1: false,
				2: false,
				3: false,
				4: false,
				5: false,
			},
		},
		{
			rate: 2,
			expected: map[uint64]bool{
				0: true,
				1: true,
				2: true,
			},
		},
		{
			rate: -1,
			expected: map[uint64]bool{
				0: false,
				1: false,
				2: false,
			},
		},
	} {
		t.Run(fmt.Sprint(test.rate), func(t *testing.T) {
			sampler := (&Server{}).samplerForRate(test.rate)
			for id, sampled := range test.expected {
				t.Run(fmt.Sprint(id), func(t *testing.T) {
					assert.Equal(t, sampled, sampler(id))
				})
			}
		})
	}
}
