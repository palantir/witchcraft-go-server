package runnable

import (
	"context"
	"testing"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/stretchr/testify/assert"
)

func TestParallelFor_Success(t *testing.T) {
	numIndexes := uint(50)
	numWorkers := uint(5)
	expectedVisited := make([]bool, numIndexes)
	actualVisited := make([]bool, numIndexes)
	for idx := uint(0); idx < numIndexes; idx++ {
		expectedVisited[idx] = true
	}
	err := For(context.Background(), numWorkers, numIndexes, func(ctx context.Context, idx uint) error {
		actualVisited[idx] = true
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, expectedVisited, actualVisited)
}

func TestParallelFor_Error(t *testing.T) {
	numIndexes := uint(50)
	numWorkers := uint(5)
	expectedVisited := make([]bool, numIndexes)
	actualVisited := make([]bool, numIndexes)
	for idx := uint(0); idx < numIndexes; idx++ {
		expectedVisited[idx] = true
	}
	err := For(context.Background(), numWorkers, numIndexes, func(ctx context.Context, idx uint) error {
		actualVisited[idx] = true
		return werror.Error("error message")
	})
	assert.Error(t, err)
	assert.Equal(t, expectedVisited, actualVisited)
}

func TestParallelForever_Panic(t *testing.T) {
	firstChan := make(chan int)
	secondChan := make(chan int)
	Forever(context.Background(), 2, func(ctx context.Context, idx uint) error {
		if idx == 0 {
			<-firstChan
			panic("oops!")
		}
		firstChan <- 1
		secondChan <- 2
		// never finishes
		secondChan <- 3
		return nil
	})
	assert.Equal(t, <-secondChan, 2)
	assert.Equal(t, len(firstChan), 0)
}
