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

// ErrorSubmitter allows components whose functionality dictates a portion of health status to only consume this interface.
type ErrorSubmitter interface {
	Submit(error)
}

// ErrorHealthCheckSource is a health check source with statuses determined by submitted errors.
type ErrorHealthCheckSource interface {
	ErrorSubmitter
	status.HealthCheckSource
}

// unhealthyIfAtLeastOneErrorSource is a HealthCheckSource that polls a TimeWindowedStore.
// It returns the first non-nil error as an unhealthy check.
// If there are no items, returns healthy.
type unhealthyIfAtLeastOneErrorSource struct {
	timeWindowedStore *TimeWindowedStore
	checkType         health.CheckType
}

// MustNewUnhealthyIfAtLeastOneErrorSource returns the result of calling NewUnhealthyIfAtLeastOneErrorSource, but panics if it returns an error.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewUnhealthyIfAtLeastOneErrorSource(checkType health.CheckType, windowSize time.Duration) ErrorHealthCheckSource {
	source, err := NewUnhealthyIfAtLeastOneErrorSource(checkType, windowSize)
	if err != nil {
		panic(err)
	}
	return source
}

// NewUnhealthyIfAtLeastOneErrorSource creates an unhealthyIfAtLeastOneErrorSource
// with a sliding window of size windowSize and uses the checkType.
// windowSize must be a positive value, otherwise returns error.
func NewUnhealthyIfAtLeastOneErrorSource(checkType health.CheckType, windowSize time.Duration) (ErrorHealthCheckSource, error) {
	timeWindowedStore, err := NewTimeWindowedStore(windowSize)
	if err != nil {
		return nil, err
	}
	return &unhealthyIfAtLeastOneErrorSource{
		timeWindowedStore: timeWindowedStore,
		checkType:         checkType,
	}, nil
}

// Submit submits an error.
func (u *unhealthyIfAtLeastOneErrorSource) Submit(err error) {
	u.timeWindowedStore.Submit(err)
}

func (u *unhealthyIfAtLeastOneErrorSource) itemsToCheck() health.HealthCheckResult {
	items := u.timeWindowedStore.ItemsInWindow()
	for _, item := range items {
		if item.Item != nil {
			return whealth.UnhealthyHealthCheckResult(u.checkType, item.Item.(error).Error())
		}
	}
	return whealth.HealthyHealthCheckResult(u.checkType)
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (u *unhealthyIfAtLeastOneErrorSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			u.checkType: u.itemsToCheck(),
		},
	}
}

// healthyIfNotAllErrorsSource is a HealthCheckSource that polls a TimeWindowedStore.
// It returns, if there are only non-nil errors, the first non-nil error as an unhealthy check.
// If there are no items, returns healthy.
type healthyIfNotAllErrorsSource struct {
	timeWindowedStore *TimeWindowedStore
	checkType         health.CheckType
}

// MustNewHealthyIfNotAllErrorsSource returns the result of calling NewHealthyIfNotAllErrorsSource, but panics if it returns an error.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) ErrorHealthCheckSource {
	source, err := NewHealthyIfNotAllErrorsSource(checkType, windowSize)
	if err != nil {
		panic(err)
	}
	return source
}

// NewHealthyIfNotAllErrorsSource creates an healthyIfNotAllErrorsSource
// with a sliding window of size windowSize and uses the checkType.
// windowSize must be a positive value, otherwise returns error.
func NewHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) (ErrorHealthCheckSource, error) {
	timeWindowedStore, err := NewTimeWindowedStore(windowSize)
	if err != nil {
		return nil, err
	}
	return &healthyIfNotAllErrorsSource{
		timeWindowedStore: timeWindowedStore,
		checkType:         checkType,
	}, nil
}

// Submit submits an error.
func (h *healthyIfNotAllErrorsSource) Submit(err error) {
	h.timeWindowedStore.Submit(err)
}

func (h *healthyIfNotAllErrorsSource) itemsToCheck() health.HealthCheckResult {
	items := h.timeWindowedStore.ItemsInWindow()
	if len(items) == 0 {
		return whealth.HealthyHealthCheckResult(h.checkType)
	}
	for _, item := range items {
		if item.Item == nil {
			return whealth.HealthyHealthCheckResult(h.checkType)
		}
	}
	return whealth.UnhealthyHealthCheckResult(h.checkType, items[0].Item.(error).Error())
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (h *healthyIfNotAllErrorsSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			h.checkType: h.itemsToCheck(),
		},
	}
}
