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
	"math"
	"testing"

	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/stretchr/testify/assert"
)

func TestSamplerForRate(t *testing.T) {
	for _, test := range []struct {
		rate     float64
		expected [16]bool
	}{
		{
			rate:     -1,
			expected: [16]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate:     0,
			expected: [16]bool{false, false, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate:     0.123,
			expected: [16]bool{true, true, false, false, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate:     0.25,
			expected: [16]bool{true, true, true, true, false, false, false, false, false, false, false, false, false, false, false, false},
		},
		{
			rate:     0.9,
			expected: [16]bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, false, false},
		},
		{
			rate:     1,
			expected: [16]bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
		},
		{
			rate:     2,
			expected: [16]bool{true, true, true, true, true, true, true, true, true, true, true, true, true, true, true, true},
		},
	} {
		t.Run(fmt.Sprint(test.rate), func(t *testing.T) {
			sampler := traceSamplerFromSampleRate(test.rate)
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

func TestGetTracingOptions(t *testing.T) {
	install := config.Install{
		ProductName: "product",
	}
	port := 10
	for _, test := range []struct {
		name       string
		sampler    wtracing.Sampler
		fallBack   wtracing.Sampler
		sampleRate *float64
		idsToTry   map[uint64]bool
	}{
		{
			name:    "sample set",
			sampler: func(id uint64) bool { return id == 50 },
			idsToTry: map[uint64]bool{
				1:              false,
				50:             true,
				math.MaxUint64: false,
			},
		},
		{
			name:     "fall back set",
			fallBack: func(id uint64) bool { return id == 50 },
			idsToTry: map[uint64]bool{
				1:              false,
				50:             true,
				math.MaxUint64: false,
			},
		},
		{
			name: "rate set",
			idsToTry: map[uint64]bool{
				1:                  true,
				scaleMaxUint64(.3): true,
				scaleMaxUint64(.7): false,
				math.MaxUint64:     false,
			},
			sampleRate: asFloat(.5),
		},
	} {
		t.Run(fmt.Sprint(test.name), func(t *testing.T) {
			impl := wtracing.FromTracerOptions(getTracingOptions(test.sampler, install, test.fallBack, port, test.sampleRate)...)
			assert.Equal(t, wtracing.Endpoint{
				ServiceName: "product",
				Port:        10,
			}, *impl.LocalEndpoint)
			for id, expected := range test.idsToTry {
				assert.Equal(t, impl.Sampler(id), expected)
			}
		})
	}
}

func scaleMaxUint64(f float64) uint64 {
	return uint64(math.MaxUint64 * f)
}

func asFloat(f float64) *float64 {
	return &f
}
