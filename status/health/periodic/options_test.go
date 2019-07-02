package periodic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/stretchr/testify/assert"
)

func TestWithInitialPoll(t *testing.T) {
	var pollAlwaysErr = func() error {
		return fmt.Errorf("error")
	}
	periodicCheckWithInitialPoll := NewHealthCheckSource(
		context.Background(),
		time.Minute,
		time.Second,
		"CHECK_TYPE",
		pollAlwaysErr,
		WithInitialPoll())
	<-time.After(time.Second)
	healthStatus := periodicCheckWithInitialPoll.HealthStatus(context.Background())
	check, ok := healthStatus.Checks["CHECK_TYPE"]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateError, check.State)
}
