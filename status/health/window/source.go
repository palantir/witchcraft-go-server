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

package window

import (
	"context"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

// EventsToCheckFn is a function that constructs a HealthCheckResult from a set of events.
type EventsToCheckFn func(ctx context.Context, events []Event) health.HealthCheckResult

type healthCheckSource struct {
	slidingWindowManager *TimeWindowedEventStorer
	eventsToCheckFn      EventsToCheckFn
}

// NewHealthCheck creates a HealthCheckSource that polls a TimeWindowedEventStorer and returns a HealthStatus created using an EventsToCheckFn.
func NewHealthCheck(timeWindowedEventStorer *TimeWindowedEventStorer, errorsToCheckFn EventsToCheckFn) status.HealthCheckSource {
	return &healthCheckSource{
		slidingWindowManager: timeWindowedEventStorer,
		eventsToCheckFn:      errorsToCheckFn,
	}
}

func (h *healthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	checkResult := h.eventsToCheckFn(ctx, h.slidingWindowManager.GetEventsInWindow())
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			checkResult.Type: checkResult,
		},
	}
}
