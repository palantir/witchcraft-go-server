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
	"time"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

// EventsToCheckFn is a function that constructs a HealthCheckResult from a set of events.
type EventsToCheckFn func(ctx context.Context, events []Event) health.HealthCheckResult

// BaseHealthCheckSource is a HealthCheckSource that polls a TimeWindowedEventStorer.
// It returns a HealthStatus created using an EventsToCheckFn.
type BaseHealthCheckSource struct {
	timeWindowedEventStorer *TimeWindowedEventStorer
	eventsToCheckFn         EventsToCheckFn
}

var _ status.HealthCheckSource = &BaseHealthCheckSource{}

// NewBaseHealthCheckSource creates a BaseHealthCheckSource
// with a sliding window of size windowSize and uses the errorsToCheckFn.
func NewBaseHealthCheckSource(windowSize time.Duration, errorsToCheckFn EventsToCheckFn) (*BaseHealthCheckSource, error) {
	timeWindowedEventStorer, err := NewTimeWindowedEventStorer(windowSize)
	if err != nil {
		return nil, err
	}
	return &BaseHealthCheckSource{
		timeWindowedEventStorer: timeWindowedEventStorer,
		eventsToCheckFn:         errorsToCheckFn,
	}, nil
}

// SubmitEvent submits an event to the TimeWindowedEventStorer.
func (b *BaseHealthCheckSource) SubmitEvent(payload interface{}) {
	b.timeWindowedEventStorer.SubmitEvent(payload)
}

// HealthStatus polls the TimeWindowedEventStorer and creates a HealthStatus using the EventsToCheckFn.
func (b *BaseHealthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	checkResult := b.eventsToCheckFn(ctx, b.timeWindowedEventStorer.GetEventsInWindow())
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			checkResult.Type: checkResult,
		},
	}
}
