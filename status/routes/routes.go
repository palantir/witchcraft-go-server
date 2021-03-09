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

package routes

import (
	"net/http"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	healthstatus "github.com/palantir/witchcraft-go-health/status"
	"github.com/palantir/witchcraft-go-server/v2/status"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/wresource"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

func AddLivenessRoutes(resource wresource.Resource, source healthstatus.Source) error {
	return resource.Get("liveness", status.LivenessEndpoint, handler(source), wrouter.DisableTelemetry())
}

func AddReadinessRoutes(resource wresource.Resource, source healthstatus.Source) error {
	return resource.Get("readiness", status.ReadinessEndpoint, handler(source), wrouter.DisableTelemetry())
}

func AddHealthRoutes(resource wresource.Resource, source healthstatus.HealthCheckSource, sharedSecret refreshable.String, healthStatusChangeHandlers []status.HealthStatusChangeHandler) error {
	return resource.Get("health", status.HealthEndpoint, status.NewHealthCheckHandler(source, sharedSecret, healthStatusChangeHandlers), wrouter.DisableTelemetry())
}

// handler returns an HTTP handler that writes a response based on the provided source. The status code of the response
// is determined based on the status reported by the source and the status metadata returned by the source is written as
// JSON in the response body.
func handler(source healthstatus.Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		respCode, metadata := source.Status()

		// if metadata is nil, create an empty json object instead of returning 'null' which http-remoting rejects.
		if metadata == nil {
			metadata = struct{}{}
		}

		httpserver.WriteJSONResponse(w, metadata, respCode)
	})
}
