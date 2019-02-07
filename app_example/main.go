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

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/rest"
	"github.com/palantir/witchcraft-go-server/status/slo"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/palantir/witchcraft-go-server/witchcraft/refreshable"
	"github.com/palantir/witchcraft-go-server/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
)

type AppInstallConfig struct {
	config.Install `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}

type AppRuntimeConfig struct {
	config.Runtime `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}

func main() {
	slos := slo.NewSLOs()
	if err := witchcraft.
		NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			// register endpoint that uses install configuration
			if err := registerInstallNumEndpoint(slos, info.Router, info.InstallConfig.(AppInstallConfig).MyNum); err != nil {
				return nil, err
			}

			// register endpoint that uses runtime configuration
			myNumRefreshable := refreshable.NewInt(info.RuntimeConfig.Map(func(in interface{}) interface{} {
				return in.(AppRuntimeConfig).MyNum
			}))
			if err := registerRuntimeNumEndpoint(slos, info.Router, myNumRefreshable); err != nil {
				return nil, err
			}

			// long-running background task
			go func() {
				ticker := time.NewTicker(time.Second)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						func() {
							if tracer := wtracing.TracerFromContext(ctx); tracer != nil {
								// creates a context with a new root span so that every invocation is treated as its own
								// span (and has its own trace ID).
								span := tracer.StartSpan("ticker job")
								ctx = wtracing.ContextWithSpan(ctx, span)

								//_ = span // comment out next line and comment in this line to not emit trace logging
								defer span.Finish() // emits trace log based on sampling policy
							}
							svc1log.FromContext(ctx).Info("Tick")
						}()
					case <-ctx.Done():
					}
				}
			}()
			return nil, nil
		},
		).
		WithInstallConfigType(AppInstallConfig{}).
		WithRuntimeConfigType(AppRuntimeConfig{}).
		WithSelfSignedCertificate().
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithHealth(slos).
		Start(); err != nil {
		panic(err)
	}
}

func registerInstallNumEndpoint(slos slo.ServiceLevelObjectives, router wrouter.Router, num int) error {
	handlerFn := slos.Register("installNum", func(rw http.ResponseWriter, req *http.Request) {
		rest.WriteJSONResponse(rw, num, http.StatusOK)
	}, time.Millisecond*100, time.Millisecond*200, time.Millisecond*300)

	return router.Get("/installNum", handlerFn)
}

func registerRuntimeNumEndpoint(slos slo.ServiceLevelObjectives, router wrouter.Router, numProvider refreshable.Int) error {
	handlerFn := slos.Register("runtimeNum", func(rw http.ResponseWriter, req *http.Request) {
		rest.WriteJSONResponse(rw, numProvider.CurrentInt(), http.StatusOK)
	}, time.Millisecond*100, time.Millisecond*200, time.Millisecond*300)
	return router.Get("/runtimeNum", handlerFn)
}
