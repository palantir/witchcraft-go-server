package leaderelection

import (
	"context"
	"time"
)

type Config struct {
	Enabled bool `yaml:"enabled"`
	// Identity is a unique key used to distinguish who holds the lock
	Identity string `yaml:"identity"`
	// Namespace defines the space within each ElectionName and must be unique.
	Namespace string `yaml:"namespace"`
	// ElectionName must be unique within a namespace.
	ElectionName string `yaml:"election-name"`
	// TTL is max amount of time that leadership is held without it being renewed (which resets the timer).
	TTL time.Duration `yaml:"ttl"`
}

type LeaderCallbacks struct {
	// OnStartedLeading is called when a LeaderElector client starts leading and will be run asynchronously.
	// The context will be canceled when leadership is lost.
	// Clients should handle the context cancelation and gracefully shut down.
	OnStartedLeading func(ctx context.Context)

	// OnStoppedLeading is called when a LeaderElector client stops leading and will be run synchronously.
	OnStoppedLeading func()

	// OnNewLeader is called when the client observes a leader that is not the previously observed leader.
	OnNewLeader func(identity string)
}

type LeaderElector interface {
	// Run starts the leader election loop and returns when leadership is lost.
	Run(ctx context.Context) error
}
