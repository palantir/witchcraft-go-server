# Implementation Plan: Leader Election Integration for witchcraft-go-server

## Summary

Integrate leader election into witchcraft-go-server v3 so services like policymaker can use `WithLeaderElection()` instead of ~50 lines of boilerplate.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| When does initFn run? | After leadership acquired | Simple mental model - one init, runs when ready to work |
| Leadership loss behavior | Server shutdown | K8s restarts pod, clean slate |
| K8s dependency | None - interface only | Consumers provide implementation |
| Non-leader routes | None (only base mgmt) | initFn doesn't run, so no app routes |
| TaskManager integration | Separate | Keep features independent |

---

## Step 1: Create `witchcraft/leader_election.go`

**New file** with interface and callback types:

```go
package witchcraft

import "context"

// LeaderElector is the interface that leader election implementations must satisfy.
// The Run method blocks until leadership is lost or context is cancelled.
// Implementations exist in separate packages (e.g., leader-election-k8s).
type LeaderElector interface {
    // Run starts the leader election loop.
    // - Calls OnStartedLeading (in a goroutine) when leadership is acquired
    // - Calls OnStoppedLeading (synchronously) when leadership is lost
    // - Blocks until ctx is cancelled, then returns
    Run(ctx context.Context, callbacks LeaderCallbacks) error
}

// LeaderCallbacks defines the callbacks for leadership state changes.
type LeaderCallbacks struct {
    // OnStartedLeading is called (in a goroutine) when this instance becomes leader.
    // The provided context is cancelled when leadership is lost.
    OnStartedLeading func(ctx context.Context)

    // OnStoppedLeading is called synchronously when leadership is lost.
    OnStoppedLeading func()
}
```

---

## Step 2: Modify `witchcraft/witchcraft.go` - Add Field and Method

**Add field to Server struct** (after line 149, near `leaderElection InitFunc`):

```go
// leaderElectorProvider, if set, is called with install/runtime config to create a LeaderElector.
// The returned LeaderElector gates initFn execution on leadership acquisition.
// When leadership is lost, the server shuts down gracefully.
leaderElectorProvider func(installCfg I, runtimeCfg refreshable.Refreshable[R]) LeaderElector
```

**Add WithLeaderElection method** (after line 303-306):

```go
// LeaderElectorProvider is a function that creates a LeaderElector given the server's configuration.
// This allows the LeaderElector implementation to use install/runtime config values
// (e.g., pod name, namespace, election name from config).
type LeaderElectorProvider[I config.BaseInstallConfig, R config.BaseRuntimeConfig] func(
    installCfg I,
    runtimeCfg refreshable.Refreshable[R],
) LeaderElector

// WithLeaderElection configures the server to use leader election.
// The provider function is called with install and runtime config, allowing the
// LeaderElector to be constructed with config values (e.g., pod identity, namespace).
// When configured, WithInitFunc is deferred until leadership is acquired.
// When leadership is lost, the server shuts down gracefully.
func (s *Server[I, R]) WithLeaderElection(provider LeaderElectorProvider[I, R]) *Server[I, R] {
    s.leaderElectorProvider = provider
    return s
}
```

**Example usage:**

```go
server := witchcraft.NewServer[MyInstall, MyRuntime]().
    WithLeaderElection(func(install MyInstall, runtime refreshable.Refreshable[MyRuntime]) witchcraft.LeaderElector {
        return leaderelectionk8s.NewWitchcraftLeaderElector(
            leaderelection.Config{
                Enabled:      true,
                Identity:     install.PodName,
                ElectionName: install.ProductName,
                Namespace:    install.LeaderElectionNamespace,
            },
            k8sClient,
        )
    }).
    WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo[MyInstall, MyRuntime]) (func(), error) {
        // This runs only after leadership is acquired
        return nil, nil
    })
```

---

## Step 3: Make Health Check Sources Dynamic

**Problem:** Currently health check sources are set once at startup. For leader election, we need to add health checks after `initFn` runs (when leadership is acquired).

**Solution:** Use a refreshable to hold health check sources, allowing them to be added dynamically after routes are registered.

**Changes to `witchcraft/witchcraft.go`:**

```go
// Add new field to Server struct
type Server[I, R] struct {
    // ...existing fields...

    // dynamicHealthSources holds health check sources that can be added after startup.
    // Used with leader election where initFn (which registers health checks) runs
    // after the server starts.
    dynamicHealthSources refreshable.Refreshable[[]healthstatus.HealthCheckSource]
}
```

**Changes to `witchcraft/server_routes.go`:**

Modify `addRoutes()` to use a combined health source that includes both:
1. Internal sources (CONFIG_RELOAD, SERVICE_DEPENDENCY, ENDPOINT_FIVE_HUNDREDS)
2. Dynamic sources from the refreshable (populated when initFn runs)

```go
// In addRoutes(), change health source creation:
combinedSource := healthstatus.NewCombinedHealthCheckSource(
    &s.stateManager,
    // Wrap dynamic sources in a source that re-evaluates on each call
    newDynamicHealthCheckSource(s.dynamicHealthSources),
)
```

**New helper in `witchcraft/server_routes.go`:**

```go
// dynamicHealthCheckSource wraps a refreshable of health sources
type dynamicHealthCheckSource struct {
    sources refreshable.Refreshable[[]healthstatus.HealthCheckSource]
}

func (d *dynamicHealthCheckSource) HealthStatus(ctx context.Context) healthstatus.HealthStatus {
    combined := healthstatus.NewCombinedHealthCheckSource(d.sources.Current()...)
    return combined.HealthStatus(ctx)
}
```

---

## Step 4: Modify `witchcraft/witchcraft.go` - Change Start() Flow

**Location:** Lines 772-834 in `Start()` method

**Current flow:**
```
Setup → initFn() → addRoutes() → Start HTTP server
```

**New flow with leader election:**
```
Setup → addRoutes() → Start HTTP server → [LeaderElector.Run in goroutine]
                                                        ↓
                              OnStartedLeading → initFn() → update dynamicHealthSources
                                                        ↓
                              OnStoppedLeading → cleanup() → Shutdown()
```

**Key changes:**

1. All routes (including /status/health) registered before server starts
2. Health endpoint returns empty/minimal health until initFn adds sources via refreshable
3. When leadership acquired, initFn runs and updates `dynamicHealthSources`
4. Health endpoint now includes user-registered health checks
5. On leadership loss, cleanup runs and server shuts down

**Implementation sketch:**

```go
if s.leaderElectorProvider != nil {
    // Create the LeaderElector using install/runtime config
    leaderElector := s.leaderElectorProvider(fullInstallCfg, refreshableRuntimeCfg)

    // Start leader election - initFn will be called when we become leader
    go wapp.RunWithRecoveryLogging(ctx, func(ctx context.Context) {
        err := leaderElector.Run(ctx, LeaderCallbacks{
            OnStartedLeading: func(leaderCtx context.Context) {
                // Now we're leader - run initialization
                cleanupFn, err := s.runInitialization(leaderCtx, router, mgmtRouter, ...)
                if err != nil {
                    s.Shutdown(ctx)
                    return
                }
                // Store cleanup for OnStoppedLeading
            },
            OnStoppedLeading: func() {
                // Lost leadership - cleanup and shutdown
                if cleanupFn != nil {
                    cleanupFn()
                }
                s.Shutdown(ctx)
            },
        })
    })
} else {
    // Existing path - no leader election
    cleanupFn, err := s.initFn(ctx, info)
    // ...existing code...
}
```

---

## Step 5: Extract initFn Logic to Helper Method

To avoid code duplication, extract the initFn execution into a helper:

```go
func (s *Server[I, R]) runLeaderInitialization(
    ctx context.Context,
    router, mgmtRouter wrouter.Router,
    fullInstallCfg I,
    refreshableRuntimeCfg refreshable.Refreshable[R],
    discovery ConfigurableServiceDiscovery,
    internalHealthCheckSources []healthstatus.HealthCheckSource,
) (cleanup func(), err error) {
    // Tracer setup (lines 774-779)
    // initFn call (lines 797-810)
    // Update dynamicHealthSources refreshable with user + internal sources
    s.dynamicHealthSources.Update(append(s.healthCheckSources, internalHealthCheckSources...))
    return cleanupFn, nil
}
```

---

## Step 6: Add Tests

**New file:** `witchcraft/leader_election_test.go`

Test cases:
1. `TestWithoutLeaderElection` - existing behavior unchanged
2. `TestWithLeaderElection_InitFnDeferred` - initFn not called until OnStartedLeading
3. `TestWithLeaderElection_ShutdownOnLeadershipLoss` - server shuts down when OnStoppedLeading called
4. `TestWithLeaderElection_CleanupCalled` - cleanup function from initFn is called on leadership loss
5. `TestWithLeaderElection_InitFnError` - server shuts down if initFn returns error after gaining leadership

---

## Files Changed Summary

| File | Action | Description |
|------|--------|-------------|
| `witchcraft/leader_election.go` | Create | LeaderElector interface, LeaderCallbacks type |
| `witchcraft/witchcraft.go` | Modify | Add fields, WithLeaderElection method, modify Start() |
| `witchcraft/server_routes.go` | Modify | Use refreshable for dynamic health sources |
| `witchcraft/leader_election_test.go` | Create | Tests for leader election behavior |

---

## Behavior Summary

| Scenario | Without Leader Election | With Leader Election |
|----------|------------------------|---------------------|
| initFn runs | Immediately on Start() | After OnStartedLeading |
| App routes exist | Always | Only when leader |
| Liveness/Readiness | Always available | Always available |
| Health endpoint | Always available | Always available (but minimal until leader) |
| Health checks | All registered at startup | SERVER_STATUS only until initFn adds more |
| Leadership loss | N/A | Cleanup → Shutdown |
| Non-leader state | N/A | HTTP server running, all mgmt routes work, no app routes |

---

## Lifecycle Diagrams

### Startup: Instance Becomes Leader

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Server.Start()                                   │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│  1. Load install config                                                       │
│  2. Initialize metrics registry                                               │
│  3. Initialize loggers (svc1log, evt2log, etc.)                              │
│  4. Load runtime config (refreshable)                                         │
│  5. Create routers (main + management)                                        │
│  6. Add middleware (telemetry, user middleware, 404 handler)                  │
│  7. Setup signal handlers (SIGQUIT, SIGTERM, SIGINT)                         │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                    ┌─────────────────┴─────────────────┐
                    │                                   │
                    ▼                                   ▼
        ┌───────────────────────┐           ┌───────────────────────┐
        │  NO Leader Election   │           │  WITH Leader Election │
        └───────────────────────┘           └───────────────────────┘
                    │                                   │
                    ▼                                   ▼
        ┌───────────────────────┐           ┌───────────────────────┐
        │  Run initFn()         │           │  addRoutes()          │
        │  addRoutes()          │           │  (all routes, health  │
        │  Start HTTP server    │           │   uses refreshable)   │
        │  READY TO SERVE       │           │  Start HTTP server    │
        └───────────────────────┘           └───────────────────────┘
                                                        │
                                                        ▼
                                            ┌───────────────────────┐
                                            │  LeaderElector.Run()  │
                                            │  (goroutine, blocks)  │
                                            └───────────────────────┘
                                                        │
                                                        ▼
                                            ┌───────────────────────┐
                                            │  Waiting for lease... │
                                            │  /health returns only │
                                            │  SERVER_STATUS        │
                                            └───────────────────────┘
                                                        │
                                                        │ Lease acquired!
                                                        ▼
                                            ┌───────────────────────┐
                                            │  OnStartedLeading()   │
                                            │  (called in goroutine)│
                                            └───────────────────────┘
                                                        │
                                                        ▼
                                            ┌───────────────────────┐
                                            │  1. Setup tracer      │
                                            │  2. Run initFn()      │
                                            │  3. Update refreshable│
                                            │     with health checks│
                                            │  FULLY READY          │
                                            └───────────────────────┘
```

### Runtime: Instance Loses Leadership

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                         Instance is LEADER, serving traffic                   │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ Lease renewal fails / another pod wins
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          OnStoppedLeading()                                   │
│                        (called synchronously)                                 │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│  1. Cancel leader context (signals initFn goroutines to stop)                │
│  2. Call cleanup function (returned by initFn)                               │
│  3. Call server.Shutdown(ctx)                                                │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Graceful Shutdown                                    │
│  - Stop accepting new connections                                            │
│  - Wait for in-flight requests to complete                                   │
│  - Close HTTP server                                                         │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Start() returns                                      │
│                     (K8s will restart the pod)                               │
└──────────────────────────────────────────────────────────────────────────────┘
```

### Startup: Instance Never Becomes Leader

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              Server.Start()                                   │
│                         (same setup as above)                                │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                            addRoutes()                                        │
│              (all routes including health with refreshable)                  │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Start HTTP server                                    │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                         LeaderElector.Run()                                   │
│                      Waiting for lease forever...                            │
│                                                                              │
│  State while waiting:                                                        │
│  - /status/liveness: 200 OK                                                  │
│  - /status/readiness: 200 OK                                                 │
│  - /status/health: 200 OK (SERVER_STATUS: HEALTHY only)                      │
│  - No app routes registered (initFn not called)                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Future Work (Not in Scope)

1. **Adapter in leader-election-k8s** - Implement witchcraft.LeaderElector interface
2. **Policymaker migration** - Use WithLeaderElection instead of manual loop
3. **Health check for leadership status** - Optional LEADER_ELECTION health source
