package registry

import (
	"context"
	"testing"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
	whealth "github.com/palantir/witchcraft-go-server/status/health"
	"github.com/stretchr/testify/assert"
)

func TestHealthCheckRegistry_RegisterNewChecks(t *testing.T) {
	ctx := context.Background()
	registry := NewHealthCheckRegistry()

	healthStatus := registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{},
	}, healthStatus)

	registry.Register("source-1", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-1": whealth.HealthyHealthCheckResult("check-1"),
			},
		}
	}))

	healthStatus = registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-1": whealth.HealthyHealthCheckResult("check-1"),
		},
	}, healthStatus)

	registry.Register("source-2", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-2": whealth.UnhealthyHealthCheckResult("check-2", "error message"),
			},
		}
	}))

	healthStatus = registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-1": whealth.HealthyHealthCheckResult("check-1"),
			"check-2": whealth.UnhealthyHealthCheckResult("check-2", "error message"),
		},
	}, healthStatus)
}

func TestHealthCheckRegistry_UnregisterOldChecks(t *testing.T) {
	ctx := context.Background()
	registry := NewHealthCheckRegistry()

	registry.Register("source-1", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-1": whealth.HealthyHealthCheckResult("check-1"),
			},
		}
	}))

	registry.Register("source-2", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-2": whealth.UnhealthyHealthCheckResult("check-2", "error message"),
			},
		}
	}))

	healthStatus := registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-1": whealth.HealthyHealthCheckResult("check-1"),
			"check-2": whealth.UnhealthyHealthCheckResult("check-2", "error message"),
		},
	}, healthStatus)

	registry.Unregister("source-1")

	healthStatus = registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-2": whealth.UnhealthyHealthCheckResult("check-2", "error message"),
		},
	}, healthStatus)

	registry.Unregister("source-2")

	healthStatus = registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{},
	}, healthStatus)
}

func TestHealthCheckRegistry_ReplaceOldChecks(t *testing.T) {
	ctx := context.Background()
	registry := NewHealthCheckRegistry()

	registry.Register("source-1", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-1": whealth.HealthyHealthCheckResult("check-1"),
			},
		}
	}))

	healthStatus := registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-1": whealth.HealthyHealthCheckResult("check-1"),
		},
	}, healthStatus)

	registry.Register("source-1", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-2": whealth.UnhealthyHealthCheckResult("check-2", "error message"),
			},
		}
	}))

	healthStatus = registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-2": whealth.UnhealthyHealthCheckResult("check-2", "error message"),
		},
	}, healthStatus)
}

func TestHealthCheckRegistry_CollidingCheckTypes(t *testing.T) {
	ctx := context.Background()
	registry := NewHealthCheckRegistry()

	registry.Register("source-1", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-1": whealth.HealthyHealthCheckResult("check-1"),
			},
		}
	}))

	healthStatus := registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-1": whealth.HealthyHealthCheckResult("check-1"),
		},
	}, healthStatus)

	registry.Register("source-2", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-1": whealth.UnhealthyHealthCheckResult("check-1", "error message"),
			},
		}
	}))

	healthStatus = registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-1": whealth.UnhealthyHealthCheckResult("check-1", "error message"),
		},
	}, healthStatus)

	registry.Register("source-3", status.HealthCheckSourceFn(func(ctx context.Context) health.HealthStatus {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				"check-1": whealth.RepairingHealthCheckResult("check-1", "repairing message"),
			},
		}
	}))

	healthStatus = registry.HealthStatus(ctx)
	assert.Equal(t, health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			"check-1": whealth.RepairingHealthCheckResult("check-1", "repairing message"),
		},
	}, healthStatus)
}
