// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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

package ratelimit

import (
	"net/http"
	"sync/atomic"

	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status/reporter"
	"github.com/palantir/witchcraft-go-server/wrouter"
)

type MatchFunc func(req *http.Request, vals wrouter.RequestVals) bool

var MatchReadonly MatchFunc = func(req *http.Request, vals wrouter.RequestVals) bool {
	return req.Method == http.MethodGet || req.Method == http.MethodHead || req.Method == http.MethodOptions
}

var MatchMutating MatchFunc = func(req *http.Request, vals wrouter.RequestVals) bool {
	return req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodDelete || req.Method == http.MethodPatch
}

// NewInflightLimitMiddleware returns a middleware which counts and limits the number of
// inflight requests that match a the provided MatchFunc filter. If health is non-nil,
// it will be set to REPAIRING when the middleware returns StatusTooManyRequests (429),
//
// TODO: We should set the Retry-After header based on how many requests we're rejecting.
//		 Maybe enqueue requests in a channel for a few seconds in case other requests return quickly?
func NewInflightLimitMiddleware(limit int64, matches MatchFunc, healthcheck reporter.HealthComponent) wrouter.RouteHandlerMiddleware {
	l := &limiter{
		Limit:   limit,
		Matches: matches,
		Health:  healthcheck,
	}
	healthcheck.Healthy()
	return l.ServeHTTP
}

type limiter struct {
	Limit   int64
	Matches MatchFunc
	Health  reporter.HealthComponent

	current int64
}

func (l *limiter) ServeHTTP(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
	if l.Matches(req, reqVals) {
		current := atomic.AddInt64(&l.current, 1)
		defer atomic.AddInt64(&l.current, -1)

		if current > l.Limit {
			msg := "Throttling due to too many inflight requests"
			if l.Health != nil && l.Health.Status() != health.HealthStateRepairing {
				l.Health.SetHealth(health.HealthStateRepairing, &msg, nil)
			}
			svc1log.FromContext(req.Context()).Warn(msg,
				svc1log.SafeParam("current", current),
				svc1log.SafeParam("limit", l.Limit))

			// Return early, triggering failover or exponential backoff.
			http.Error(rw, msg, http.StatusTooManyRequests)
			return
		}

		// Mark ourselves healthy!
		if l.Health != nil && l.Health.Status() != health.HealthStateHealthy {
			l.Health.Healthy()
		}
	}
	next(rw, req, reqVals)
}
