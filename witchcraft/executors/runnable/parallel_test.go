package runnable

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParallelRunnable(t *testing.T) {
	ctx := context.Background()
	started1 := make(chan struct{})
	started2 := make(chan struct{})
	done := make(chan struct{})

	runnable1 := New("runnable-1", func(ctx context.Context) error {
		close(started1)
		<-done
		return nil
	})
	runnable2 := New("runnable-2", func(ctx context.Context) error {
		close(started2)
		<-done
		return nil
	})
	runnable := NewParallel("", 0, []Runnable{runnable1, runnable2})

	runnableExited := make(chan struct{})
	defer func() {
		<-runnableExited
	}()
	go func() {
		err := runnable.Run(ctx)
		assert.NoError(t, err)
		close(runnableExited)
	}()
	<-started1
	<-started2
	close(done)
}
