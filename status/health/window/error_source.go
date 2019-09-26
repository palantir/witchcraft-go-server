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
	whealth "github.com/palantir/witchcraft-go-server/status/health"
)

// UnhealthyIfAtLeastOneErrorSource is a HealthCheckSource that polls a TimeWindowedEventStorer.
// It returns the first non-nil error as an unhealthy check.
// If there are no events, returns healthy.
type UnhealthyIfAtLeastOneErrorSource struct {
	timeWindowedEventStorer *TimeWindowedEventStorer
	checkType               health.CheckType
}

var _ status.HealthCheckSource = &UnhealthyIfAtLeastOneErrorSource{}

// NewUnhealthyIfAtLeastOneErrorSource creates an UnhealthyIfAtLeastOneErrorSource
// with a sliding window of size windowSize and uses the checkType.
func NewUnhealthyIfAtLeastOneErrorSource(windowSize time.Duration, checkType health.CheckType) (*UnhealthyIfAtLeastOneErrorSource, error) {
	timeWindowedEventStorer, err := NewTimeWindowedEventStorer(windowSize)
	if err != nil {
		return nil, err
	}
	return &UnhealthyIfAtLeastOneErrorSource{
		timeWindowedEventStorer: &timeWindowedEventStorer,
		checkType:               checkType,
	}, nil
}

// SubmitError submits an event to the TimeWindowedEventStorer.
func (u *UnhealthyIfAtLeastOneErrorSource) SubmitError(err error) {
	u.timeWindowedEventStorer.SubmitEvent(err)
}

func (u *UnhealthyIfAtLeastOneErrorSource) eventsToCheck() health.HealthCheckResult {
	events := u.timeWindowedEventStorer.GetEventsInWindow()
	for _, event := range events {
		if event.Payload != nil {
			return whealth.UnhealthyHealthCheckResult(u.checkType, event.Payload.(error).Error())
		}
	}
	return whealth.HealthyHealthCheckResult(u.checkType)
}

// HealthStatus polls the TimeWindowedEventStorer and creates the HealthStatus.
func (u *UnhealthyIfAtLeastOneErrorSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			u.checkType: u.eventsToCheck(),
		},
	}
}

// HealthyIfNotAllErrorsSource is a HealthCheckSource that polls a TimeWindowedEventStorer.
// It returns, if there are only non-nil errors, the first non-nil error as an unhealthy check.
// If there are no events, returns healthy.
type HealthyIfNotAllErrorsSource struct {
	timeWindowedEventStorer *TimeWindowedEventStorer
	checkType               health.CheckType
}

var _ status.HealthCheckSource = &HealthyIfNotAllErrorsSource{}

// NewHealthyIfNotAllErrorsSource creates an HealthyIfNotAllErrorsSource
// with a sliding window of size windowSize and uses the checkType.
func NewHealthyIfNotAllErrorsSource(windowSize time.Duration, checkType health.CheckType) (*HealthyIfNotAllErrorsSource, error) {
	timeWindowedEventStorer, err := NewTimeWindowedEventStorer(windowSize)
	if err != nil {
		return nil, err
	}
	return &HealthyIfNotAllErrorsSource{
		timeWindowedEventStorer: &timeWindowedEventStorer,
		checkType:               checkType,
	}, nil
}

// SubmitError submits an event to the TimeWindowedEventStorer.
func (h *HealthyIfNotAllErrorsSource) SubmitError(err error) {
	h.timeWindowedEventStorer.SubmitEvent(err)
}

func (h *HealthyIfNotAllErrorsSource) eventsToCheck() health.HealthCheckResult {
	events := h.timeWindowedEventStorer.GetEventsInWindow()
	if len(events) == 0 {
		return whealth.HealthyHealthCheckResult(h.checkType)
	}
	for _, event := range events {
		if event.Payload == nil {
			return whealth.HealthyHealthCheckResult(h.checkType)
		}
	}
	return whealth.UnhealthyHealthCheckResult(h.checkType, events[0].Payload.(error).Error())
}

// HealthStatus polls the TimeWindowedEventStorer and creates the HealthStatus.
func (h *HealthyIfNotAllErrorsSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			h.checkType: h.eventsToCheck(),
		},
	}
}
