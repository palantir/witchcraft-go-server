// Copyright (c) 2024 Palantir Technologies. All rights reserved.
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
	"context"

	"github.com/palantir/pkg/refreshable/v2"
)

// LeaderElector is the interface that leader election implementations must satisfy.
// The Run method blocks until the context is cancelled.
// Implementations exist in separate packages (e.g., leader-election-k8s).
type LeaderElector interface {
	// Run starts the leader election loop.
	// - Calls OnStartedLeading (in a goroutine) when leadership is acquired.
	//   The context passed to OnStartedLeading is cancelled when leadership is lost.
	// - Calls OnStoppedLeading (synchronously) when leadership is lost.
	// - Blocks until ctx is cancelled, then returns.
	Run(ctx context.Context, callbacks LeaderCallbacks) error
}

// LeaderElectorProvider is a function that creates a LeaderElector given access to the
// server's install and runtime configuration. This allows leader election implementations
// to read configuration values like election name, namespace, and identity from the
// server's configuration.
type LeaderElectorProvider[I any, R any] func(ctx context.Context, install I, runtime refreshable.Refreshable[R]) (LeaderElector, error)

// LeaderCallbacks defines the callbacks for leadership state changes.
type LeaderCallbacks struct {
	// OnStartedLeading is called (in a goroutine) when this instance becomes leader.
	// The provided context is cancelled when leadership is lost.
	OnStartedLeading func(ctx context.Context)

	// OnStoppedLeading is called synchronously when leadership is lost.
	OnStoppedLeading func()
}
