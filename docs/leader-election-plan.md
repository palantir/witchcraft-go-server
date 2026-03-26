# Implementation Plan: Leader Election Integration for witchcraft-go-server

## Summary

Integrate leader election into witchcraft-go-server v3 so services like policymaker can use `WithLeaderElection()` instead of ~50 lines of boilerplate.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| When does initFn run? | After leadership acquired | Simple mental model - one init, runs when ready to work |
| Leadership loss behavior | Server shutdown | K8s restarts pod, clean slate |
| K8s dependency | None - interface only | Consumers provide implementation |
| Separate management port | **Required** | Management server always up, app server only on leadership |
| TaskManager integration | Separate | Keep features independent |

## Prerequisites

**Leader election requires a separate management port.** When `WithLeaderElection()` is used, the server validates that `ManagementPort != Port` in the install config. This simplifies the architecture:

- **Management server** (health/liveness/readiness, pprof, diagnostics) starts immediately
- **Application server** only starts after leadership is acquired
- No dynamic route registration needed on a running server
- Clean separation between always-available management endpoints and leader-only application endpoints

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

## Step 3: Make Health Check Sources a Refreshable

**Problem:** Currently `healthCheckSources` is a `[]healthstatus.HealthCheckSource` slice set once at startup. With leader election, the management server starts before leadership is acquired, but health checks are registered in `initFn` which runs after leadership.

**Solution:** Change `healthCheckSources` to be a refreshable that can be appended to at any time.

**Changes to `witchcraft/witchcraft.go`:**

```go
// Change existing field in Server struct from:
healthCheckSources []healthstatus.HealthCheckSource

// To:
healthCheckSources *refreshable.DefaultRefreshable[[]healthstatus.HealthCheckSource]
```

**Initialize in Start():**

```go
// Early in Start(), initialize the refreshable
s.healthCheckSources = refreshable.NewDefaultRefreshable([]healthstatus.HealthCheckSource{})
```

**Changes to `witchcraft/server_routes.go`:**

Modify `addRoutes()` to wrap the refreshable in a health source that re-evaluates on each call:

```go
// In addRoutes(), change health source creation:
if err := routes.AddHealthRoutes(
    statusResource,
    healthstatus.NewCombinedHealthCheckSource(
        &s.stateManager,
        newRefreshableHealthCheckSource(s.healthCheckSources),
    ),
    healthSharedSecret,
    s.healthStatusChangeHandlers,
); err != nil {
    return werror.Wrap(err, "failed to register health routes")
}
```

**New helper in `witchcraft/server_routes.go`:**

```go
// refreshableHealthCheckSource wraps a refreshable slice of health sources
type refreshableHealthCheckSource struct {
    sources refreshable.Refreshable[[]healthstatus.HealthCheckSource]
}

func newRefreshableHealthCheckSource(sources refreshable.Refreshable[[]healthstatus.HealthCheckSource]) *refreshableHealthCheckSource {
    return &refreshableHealthCheckSource{sources: sources}
}

func (r *refreshableHealthCheckSource) HealthStatus(ctx context.Context) healthstatus.HealthStatus {
    return healthstatus.NewCombinedHealthCheckSource(r.sources.Current()...).HealthStatus(ctx)
}
```

**Adding health checks (in initFn or via WithHealth):**

```go
// Append new sources to the refreshable
current := s.healthCheckSources.Current()
s.healthCheckSources.Update(append(current, newSources...))
```

---

## Step 4: Modify `witchcraft/witchcraft.go` - Change Start() Flow

**Location:** Lines 772-834 in `Start()` method

**Current flow (single server):**
```
Setup → initFn() → addRoutes() → Start HTTP server
```

**New flow with leader election (two servers):**
```
Setup → Start management server → LeaderElector.Run() in goroutine
                                            ↓
                      OnStartedLeading → initFn() → Start application server
                                            ↓
                      OnStoppedLeading → cleanup() → Shutdown both servers
```

**Key insight:** With separate ports required, we can start management and application servers independently:

1. **Management server starts immediately** - serves `/status/health`, `/status/liveness`, `/status/readiness`, `/debug/*`
2. **Application server only starts after leadership acquired** - serves all application routes from `initFn`
3. **No dynamic route registration needed** - app routes registered before app server starts

**Validation in Start():**

```go
if s.leaderElectorProvider != nil {
    // Validate separate management port is configured
    if installCfg.Server.ManagementPort == 0 || installCfg.Server.ManagementPort == installCfg.Server.Port {
        return werror.Error("leader election requires a separate management port: ManagementPort must be set and different from Port")
    }
}
```

**Implementation sketch:**

```go
if s.leaderElectorProvider != nil {
    // Create the LeaderElector using install/runtime config
    leaderElector := s.leaderElectorProvider(fullInstallCfg, refreshableRuntimeCfg)

    // Start management server first (always available)
    mgmtServer := s.createMgmtServer(mgmtRouter)
    go mgmtServer.ListenAndServe()

    // Start leader election - initFn and app server start when we become leader
    var cleanupFn func()
    var appServer *http.Server

    go wapp.RunWithRecoveryLogging(ctx, func(ctx context.Context) {
        err := leaderElector.Run(ctx, LeaderCallbacks{
            OnStartedLeading: func(leaderCtx context.Context) {
                // Now we're leader - run initialization and start app server
                var err error
                cleanupFn, err = s.initFn(leaderCtx, info)
                if err != nil {
                    s.Shutdown(ctx)
                    return
                }

                // Start application server
                appServer = s.createAppServer(router)
                go appServer.ListenAndServe()
            },
            OnStoppedLeading: func() {
                // Lost leadership - cleanup and shutdown
                if appServer != nil {
                    appServer.Shutdown(ctx)
                }
                if cleanupFn != nil {
                    cleanupFn()
                }
                mgmtServer.Shutdown(ctx)
            },
        })
    })

    // Block until shutdown
    <-ctx.Done()
} else {
    // Existing path - no leader election, single server or dual servers
    // ...existing code unchanged...
}
```

---

## Step 5: Create Helper to Start Application Server on Leadership

When leadership is acquired, we need to:
1. Create the application router
2. Run `initFn` (which registers routes and health checks)
3. Start the application HTTP server

```go
func (s *Server[I, R]) startApplicationServerOnLeadership(
    ctx context.Context,
    fullInstallCfg I,
    refreshableRuntimeCfg refreshable.Refreshable[R],
    discovery ConfigurableServiceDiscovery,
) (appServer *http.Server, cleanup func(), err error) {
    // Create application router
    appRouter := createRouter(s.routerImplProvider(), fullInstallCfg.BaseInstallConfig().Server.ContextPath)

    // Add middleware to application router
    s.addMiddleware(appRouter, ...)

    // Setup tracer
    // ...

    // Build InitInfo with the application router
    info := InitInfo[I, R]{
        Router:       appRouter,
        InstallCfg:   fullInstallCfg,
        RuntimeCfg:   refreshableRuntimeCfg,
        ShutdownFn:   s.Shutdown,
        Discovery:    discovery,
        // ... other fields
    }

    // Run initFn - this registers application routes and health checks
    cleanup, err = s.initFn(ctx, info)
    if err != nil {
        return nil, nil, err
    }

    // Create and start the application HTTP server
    appServer = &http.Server{
        Addr:    fmt.Sprintf(":%d", fullInstallCfg.BaseInstallConfig().Server.Port),
        Handler: appRouter,
        // ... TLS config etc
    }

    go func() {
        if err := appServer.ListenAndServe(); err != http.ErrServerClosed {
            // log error
        }
    }()

    return appServer, cleanup, nil
}
```

Note: Health checks registered via `WithHealth()` in `initFn` will append to `s.healthCheckSources` (the refreshable), making them immediately visible on the management server's `/status/health` endpoint.

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
| `witchcraft/leader_election.go` | Create | LeaderElector interface, LeaderCallbacks type, LeaderElectorProvider type |
| `witchcraft/witchcraft.go` | Modify | Add `leaderElectorProvider` field, `WithLeaderElection()` method, modify `Start()` to handle two-server architecture |
| `witchcraft/server_routes.go` | Modify | Add `refreshableHealthCheckSource` type, modify `addRoutes()` to use refreshable health sources |
| `witchcraft/leader_election_test.go` | Create | Tests for leader election behavior |

---

## Behavior Summary

| Scenario | Without Leader Election | With Leader Election |
|----------|------------------------|---------------------|
| Prerequisite | None | Separate management port required |
| initFn runs | Immediately on Start() | After OnStartedLeading |
| Management server | Starts immediately | Starts immediately |
| Application server | Starts immediately | Starts only when leader |
| Health checks | All registered at startup | SERVER_STATUS only until initFn adds more |
| Leadership loss | N/A | Cleanup → App server shutdown → Mgmt server shutdown |
| Non-leader state | N/A | Only management server running, no app server |

---

## How Application Routes Work with Leader Election

**Without leader election:**
```
Start() → initFn() → addRoutes() → Start HTTP server(s) → All routes available
```

**With leader election (two servers, separate ports):**
```
Start() → addRoutes() (mgmt) → Start management server → LeaderElector.Run()
                                                               ↓
                                      (waiting for lease - management server only)
                                                               ↓
                                      OnStartedLeading → initFn() → Start application server
                                                               ↓
                                      (both servers running, all routes available)
```

**Key simplification:** With separate ports required, there's no need for dynamic route registration. The application server is created and started fresh when leadership is acquired. Routes are registered in `initFn` before the app server starts.

**Non-leader behavior:**
- Management server running on management port: `/status/health`, `/status/liveness`, `/status/readiness`, `/debug/*`
- Application port has no server listening (connection refused)
- When leadership is acquired, application server starts and serves traffic

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
│  2. Validate ManagementPort != Port (required for leader election)           │
│  3. Initialize metrics registry                                               │
│  4. Initialize loggers (svc1log, evt2log, etc.)                              │
│  5. Load runtime config (refreshable)                                         │
│  6. Create management router                                                  │
│  7. Add middleware to management router                                       │
│  8. Setup signal handlers (SIGQUIT, SIGTERM, SIGINT)                         │
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
        │  Run initFn()         │           │  addRoutes() (mgmt)   │
        │  addRoutes()          │           │  Start MGMT SERVER    │
        │  Start HTTP server(s) │           │  (port: ManagementPort)│
        │  READY TO SERVE       │           └───────────────────────┘
        └───────────────────────┘                       │
                                                        ▼
                                            ┌───────────────────────┐
                                            │  LeaderElector.Run()  │
                                            │  (goroutine, blocks)  │
                                            └───────────────────────┘
                                                        │
                                                        ▼
                                            ┌───────────────────────┐
                                            │  Waiting for lease... │
                                            │                       │
                                            │  Mgmt server running: │
                                            │  - /status/health ✓   │
                                            │  - /status/liveness ✓ │
                                            │  - /status/readiness ✓│
                                            │                       │
                                            │  App port: NO SERVER  │
                                            │  (connection refused) │
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
                                            │  1. Create app router │
                                            │  2. Run initFn()      │
                                            │     (registers routes,│
                                            │      health checks)   │
                                            │  3. Start APP SERVER  │
                                            │     (port: Port)      │
                                            │                       │
                                            │  BOTH SERVERS RUNNING │
                                            └───────────────────────┘
```

### Runtime: Instance Loses Leadership

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                    Instance is LEADER, both servers running                   │
│                                                                              │
│  Management Server (ManagementPort): /status/*, /debug/*                     │
│  Application Server (Port): Application routes from initFn                   │
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
│  2. Shutdown APPLICATION SERVER (graceful)                                   │
│  3. Call cleanup function (returned by initFn)                               │
│  4. Shutdown MANAGEMENT SERVER (graceful)                                    │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                          Graceful Shutdown                                    │
│  - Stop accepting new connections on both servers                            │
│  - Wait for in-flight requests to complete                                   │
│  - Close both HTTP servers                                                   │
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
│                    (same setup as "Becomes Leader" above)                    │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                addRoutes() (mgmt) → Start MANAGEMENT SERVER                  │
└──────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌──────────────────────────────────────────────────────────────────────────────┐
│                         LeaderElector.Run()                                   │
│                      Waiting for lease forever...                            │
│                                                                              │
│  Management Server (ManagementPort):                                         │
│  - /status/liveness: 200 OK                                                  │
│  - /status/readiness: 200 OK                                                 │
│  - /status/health: 200 OK (SERVER_STATUS: HEALTHY only)                      │
│  - /debug/*: Available                                                       │
│                                                                              │
│  Application Port:                                                           │
│  - NO SERVER LISTENING                                                       │
│  - Connection refused (not 404!)                                             │
│  - initFn never called, no routes registered                                 │
└──────────────────────────────────────────────────────────────────────────────┘
```

---

## Future Work (Not in Scope)

1. **Adapter in leader-election-k8s** - Implement witchcraft.LeaderElector interface
2. **Policymaker migration** - Use WithLeaderElection instead of manual loop
3. **Health check for leadership status** - Optional LEADER_ELECTION health source
