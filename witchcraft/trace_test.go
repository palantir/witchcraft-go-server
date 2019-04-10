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
