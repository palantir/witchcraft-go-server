// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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
	"fmt"
	"net/http"
	netpprof "net/http/pprof"
	"runtime/pprof"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/refreshable"
	werror "github.com/palantir/witchcraft-go-error"
	healthstatus "github.com/palantir/witchcraft-go-health/status"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/status/routes"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/internal/middleware"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/internal/wdebug"
	refreshablefile "github.com/palantir/witchcraft-go-server/v2/witchcraft/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/wresource"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
)

func (s *Server) initRouters(installCfg config.Install) (rRouter wrouter.Router, rMgmtRouter wrouter.Router) {
	routerWithContextPath := createRouter(s.routerImplProvider(), installCfg.Server.ContextPath)
	mgmtRouterWithContextPath := routerWithContextPath
	if mgmtPort := installCfg.Server.ManagementPort; mgmtPort != 0 && mgmtPort != installCfg.Server.Port {
		mgmtRouterWithContextPath = createRouter(s.routerImplProvider(), installCfg.Server.ContextPath)
	}
	return routerWithContextPath, mgmtRouterWithContextPath
}

// addRoutes registers /debug/diagnostic/* and /status/*
func (s *Server) addRoutes(ctx context.Context, mgmtRouterWithContextPath wrouter.Router, runtimeCfg config.RefreshableRuntime) error {
	secretRefreshable, err := getSecretRefreshable(ctx, runtimeCfg.DiagnosticsConfig())
	if err != nil {
		return err
	}
	if err := wdebug.RegisterRoute(mgmtRouterWithContextPath, secretRefreshable); err != nil {
		return err
	}

	statusResource := wresource.New("status", mgmtRouterWithContextPath)

	// add health endpoints
	if err := routes.AddHealthRoutes(
		statusResource,
		healthstatus.NewCombinedHealthCheckSource(append(s.healthCheckSources, &s.stateManager)...),
		runtimeCfg.HealthChecks().SharedSecret(),
		s.healthStatusChangeHandlers,
	); err != nil {
		return werror.Wrap(err, "failed to register health routes")
	}

	// add liveness endpoints
	if s.livenessSource == nil {
		s.livenessSource = &s.stateManager
	}
	if err := routes.AddLivenessRoutes(statusResource, s.livenessSource); err != nil {
		return werror.Wrap(err, "failed to register liveness routes")
	}

	// add readiness endpoints
	if s.readinessSource == nil {
		s.readinessSource = &s.stateManager
	}
	if err := routes.AddReadinessRoutes(statusResource, s.readinessSource); err != nil {
		return werror.Wrap(err, "failed to register readiness routes")
	}
	return nil
}

func (s *Server) addMiddleware(rootRouter wrouter.RootRouter, registry metrics.RootRegistry, tracerOptions []wtracing.TracerOption) {
	rootRouter.AddRequestHandlerMiddleware(
		// add middleware that injects loggers into request context, extracts UID, SID, and TokenID
		// into context for loggers, sets a tracer on the context, starts a root span and sets it on the context,
		// and recovers from panics.
		middleware.NewRequestTelemetry(
			s.svcLogger,
			s.evtLogger,
			s.auditLogger,
			s.metricLogger,
			s.diagLogger,
			s.reqLogger,
			s.trcLogger,
			tracerOptions,
			s.idsExtractor,
			registry,
		),
	)

	// add user-provided middleware
	rootRouter.AddRequestHandlerMiddleware(s.handlers...)

	// add route middleware
	rootRouter.AddRouteHandlerMiddleware(middleware.NewRouteTelemetry(registry))

	// add not found handler
	rootRouter.RegisterNotFoundHandler(httpserver.NewJSONHandler(
		func(_ http.ResponseWriter, _ *http.Request) error {
			return werror.Convert(errors.NewNotFound())
		}, httpserver.StatusCodeMapper, httpserver.ErrHandler),
	)
}

func createRouter(routerImpl wrouter.RouterImpl, ctxPath string) wrouter.Router {
	routerHandler := wrouter.New(routerImpl)

	var routerWithContextPath wrouter.Router
	routerWithContextPath = routerHandler
	if ctxPath != "/" {
		// only create subrouter if context path is non-empty
		routerWithContextPath = routerHandler.Subrouter(ctxPath)
	}
	return routerWithContextPath
}

func addPprofRoutes(router wrouter.Router) error {
	debugger := wresource.New("debug", router.Subrouter("/debug"))
	if err := debugger.Get("pprofIndex", "/pprof/", http.HandlerFunc(netpprof.Index)); err != nil {
		return err
	}
	if err := debugger.Get("pprofCmdLine", "/pprof/cmdline", http.HandlerFunc(netpprof.Cmdline)); err != nil {
		return err
	}
	if err := debugger.Get("pprofCpuProfile", "/pprof/profile", http.HandlerFunc(netpprof.Profile)); err != nil {
		return err
	}
	if err := debugger.Get("pprofSymbol", "/pprof/symbol", http.HandlerFunc(netpprof.Symbol)); err != nil {
		return err
	}
	if err := debugger.Get("pprofTrace", "/pprof/trace", http.HandlerFunc(netpprof.Trace)); err != nil {
		return err
	}
	return debugger.Get("pprofHeapProfile", "/pprof/heap", http.HandlerFunc(heap))
}

// heap responds with the pprof-formatted heap profile.
func heap(w http.ResponseWriter, _ *http.Request) {
	// Set Content Type assuming WriteHeapProfile will work,
	// because if it does it starts writing.
	w.Header().Set("Content-Type", "application/octet-stream")
	if err := pprof.WriteHeapProfile(w); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprintf(w, "Could not dump heap: %s\n", err)
		return
	}
}

func getSecretRefreshable(ctx context.Context, diagnosticsConfig config.RefreshableDiagnosticsConfig) (refreshable.String, error) {
	secretFromConfig := diagnosticsConfig.DebugSharedSecret()
	secretFilePath := diagnosticsConfig.DebugSharedSecretFile()
	// if both fields are undefined, then the server doesn't use a secret for the /debug/diagnostics/* route
	if secretFromConfig.CurrentString() == "" && secretFilePath.CurrentString() == "" {
		return refreshable.NewString(refreshable.NewDefaultRefreshable("")), nil
	}

	if secretFromConfig.CurrentString() != "" {
		return secretFromConfig, nil
	}
	fileRefreshable, err := refreshablefile.NewFileRefreshable(ctx, secretFilePath.CurrentString())
	if err != nil {
		return nil, err
	}
	secretStringFromFileRefreshable := refreshable.NewString(fileRefreshable.Map(func(i interface{}) interface{} {
		secretBytes, ok := i.([]byte)
		if !ok {
			return ""
		}
		return string(secretBytes)
	}))
	return secretStringFromFileRefreshable, nil
}
