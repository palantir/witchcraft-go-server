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
	"context"
	"net/http"
	"sync/atomic"

	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/reporter"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

type MatchFunc func(req *http.Request, vals wrouter.RequestVals) bool

var MatchReadOnly MatchFunc = func(req *http.Request, vals wrouter.RequestVals) bool {
	return req.Method == http.MethodGet || req.Method == http.MethodHead || req.Method == http.MethodOptions
}

var MatchMutating MatchFunc = func(req *http.Request, vals wrouter.RequestVals) bool {
	return req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodDelete || req.Method == http.MethodPatch
}

// NewInFlightRequestLimitMiddleware returns a middleware which counts and limits the number of
// inflight requests that match the provided MatchFunc filter. If MatchFunc is nil, it will
// match all requests. When the number of active matched requests exceeds the limit, the middleware
// returns StatusTooManyRequests (429).
//
// If healthcheck is non-nil, it will be set to REPAIRING when the middleware is throttling
// and HEALTHY when the current counter falls below the limit. It is initialized to HEALTHY.
//
// If limit is ever negative it will be treated as a 0, i.e. all requests will be throttled.
//
// TODO: We should set the Retry-After header based on how many requests we're rejecting.
//
//	Maybe enqueue requests in a channel for a few seconds in case other requests return quickly?
func NewInFlightRequestLimitMiddleware(limit func() int, matches MatchFunc, healthcheck reporter.HealthComponent) wrouter.RouteHandlerMiddleware {
	l := &limiter{
		Limit:   limit,
		Matches: matches,
		Health:  healthcheck,
	}
	if healthcheck != nil {
		healthcheck.Healthy()
	}
	return l.ServeHTTP
}

type limiter struct {
	Limit   func() int
	Matches MatchFunc
	Health  reporter.HealthComponent

	current int64
}

const inFlightThrottledMessage = "Throttling due to too many in-flight requests"

func (l *limiter) ServeHTTP(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
	if l.Matches == nil || l.Matches(req, reqVals) {
		throttled := l.increment(req.Context())
		defer l.decrement(req.Context())

		if throttled {
			// Return early, triggering failover or exponential backoff.
			http.Error(rw, inFlightThrottledMessage, http.StatusTooManyRequests)
			return
		}
	}
	next(rw, req, reqVals)
}

// increment adds 1 to the current counter. If the new value is over the Limit,
// increment returns 'true' to indicate the request should be rejected/throttled and
// l.Health is set to REPAIRING (if not already in that state).
func (l *limiter) increment(ctx context.Context) (throttled bool) {
	current := atomic.AddInt64(&l.current, 1)
	limit := l.limit()
	if current <= limit {
		return false
	}
	if l.Health != nil && l.Health.Status() != health.HealthState_REPAIRING {
		msg := inFlightThrottledMessage
		l.Health.SetHealth(health.HealthState_REPAIRING, &msg, nil)
	}
	svc1log.FromContext(ctx).Warn(inFlightThrottledMessage,
		svc1log.SafeParam("current", current),
		svc1log.SafeParam("limit", limit))

	return true
}

// increment subtracts 1 from the current counter. If the new value is under the Limit,
// l.Health is set to HEALTHY (if not already in that state).
func (l *limiter) decrement(ctx context.Context) {
	current := atomic.AddInt64(&l.current, -1)
	if current < l.limit() && l.Health != nil && l.Health.Status() != health.HealthState_HEALTHY {
		l.Health.Healthy()
	}
}

// limit returns the current value of l.limit, or zero if the limit is negative.
func (l *limiter) limit() int64 {
	current := l.Limit()
	if current < 0 {
		current = 0
	}
	return int64(current)
}
