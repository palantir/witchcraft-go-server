package runnable

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSequentialRunnable(t *testing.T) {
	ctx := context.Background()

	called1 := false
	runnable1 := New("runnable-1", func(ctx context.Context) error {
		called1 = true
		return nil
	})
	called2 := false
	runnable2 := New("runnable-2", func(ctx context.Context) error {
		called2 = true
		return nil
	})

	runnable := NewSequential("", []Runnable{runnable1, runnable2})
	err := runnable.Run(ctx)
	require.NoError(t, err)
	assert.True(t, called1)
	assert.True(t, called2)
}

func TestSequentialRunnable_Error(t *testing.T) {
	ctx := context.Background()

	called1 := false
	runnable1 := New("runnable-1", func(ctx context.Context) error {
		called1 = true
		return fmt.Errorf("error-1")
	})
	called2 := false
	runnable2 := New("runnable-2", func(ctx context.Context) error {
		called2 = true
		return fmt.Errorf("error-2")
	})

	runnable := NewSequential("", []Runnable{runnable1, runnable2})
	err := runnable.Run(ctx)
	require.Error(t, err)
	assert.Equal(t, fmt.Errorf("error-1"), err)
	assert.True(t, called1)
	assert.False(t, called2)
}
