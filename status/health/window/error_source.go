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
	"sync"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
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
	// unhealthyIfAtLeastOneErrorSource is a healthyIfNotAllErrorsSource that drops all successes.
	source ErrorHealthCheckSource
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
	underlyingSource, err := newHealthyIfNotAllErrorsSource(checkType, windowSize, false, false, NewOrdinaryTimeProvider())
	if err != nil {
		return nil, err
	}

	return &unhealthyIfAtLeastOneErrorSource{
		source: underlyingSource,
	}, nil
}

// Submit submits an error.
func (u *unhealthyIfAtLeastOneErrorSource) Submit(err error) {
	if err == nil {
		return
	}
	u.source.Submit(err)
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (u *unhealthyIfAtLeastOneErrorSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return u.source.HealthStatus(ctx)
}

// healthyIfNotAllErrorsSource is a HealthCheckSource that polls a TimeWindowedStore.
// It returns, if there are only non-nil errors, the first non-nil error as an unhealthy check.
// If there are no items, returns healthy.
type healthyIfNotAllErrorsSource struct {
	timeProvider           TimeProvider
	useAnchoredWindows     bool
	windowSize             time.Duration
	lastErrorTime          time.Time
	lastError              error
	lastSuccessTime        time.Time
	sourceMutex            sync.RWMutex
	checkType              health.CheckType
	requireFirstFullWindow bool
	startTime              time.Time
}

// MustNewHealthyIfNotAllErrorsSource returns the result of calling NewHealthyIfNotAllErrorsSource, but panics if it returns an error.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) ErrorHealthCheckSource {
	source, err := newHealthyIfNotAllErrorsSource(checkType, windowSize, false, true, NewOrdinaryTimeProvider())
	if err != nil {
		panic(err)
	}
	return source
}

// NewHealthyIfNotAllErrorsSource creates an healthyIfNotAllErrorsSource
// with a sliding window of size windowSize and uses the checkType.
// windowSize must be a positive value, otherwise returns error.
func NewHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) (ErrorHealthCheckSource, error) {
	return newHealthyIfNotAllErrorsSource(checkType, windowSize, false, true, NewOrdinaryTimeProvider())
}

// MustNewAnchoredHealthyIfNotAllErrorsSource returns the result of calling
// NewAnchoredHealthyIfNotAllErrorsSource but panics if that call returns an error
// Should only be used in instances where the inputs are statically defined and known to be valid.
// Care should be taken in considering health submission rate and window size when using anchored
// windows. Windows too close to service emission frequency may cause errors to not surface
func MustNewAnchoredHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) ErrorHealthCheckSource {
	source, err := newHealthyIfNotAllErrorsSource(checkType, windowSize, true, true, NewOrdinaryTimeProvider())
	if err != nil {
		panic(err)
	}
	return source
}

// NewAnchoredHealthyIfNotAllErrorsSource creates an healthyIfNotAllErrorsSource
// with supplied checkType, using sliding window of size windowSize, which will
// anchor (force the window to be at least the grace period) by inserting a healthy
// check at the beginning of new the initial window or after gaps greater than windowSize
// windowSize must be a positive value, otherwise returns error. Care should be taken in
// considering health submission rate and window size when using anchored windows.
// Windows too close to service emission frequency may cause errors to not surface
func NewAnchoredHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) (ErrorHealthCheckSource, error) {
	return newHealthyIfNotAllErrorsSource(checkType, windowSize, true, true, NewOrdinaryTimeProvider())
}

func newHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration, useAnchoredWindows bool, requireFirstFullWindow bool, timeProvider TimeProvider) (ErrorHealthCheckSource, error) {
	if windowSize <= 0 {
		return nil, werror.Error("windowSize must be positive", werror.SafeParam("windowSize", windowSize))
	}

	retVal := &healthyIfNotAllErrorsSource{
		timeProvider:           timeProvider,
		useAnchoredWindows:     useAnchoredWindows,
		windowSize:             windowSize,
		checkType:              checkType,
		startTime:              timeProvider.Now(),
		requireFirstFullWindow: requireFirstFullWindow,
	}

	// When anchored treat first initial window as anchored
	if useAnchoredWindows {
		retVal.lastSuccessTime = retVal.timeProvider.Now()
	}

	return retVal, nil
}

// Submit submits an error.
func (h *healthyIfNotAllErrorsSource) Submit(err error) {
	h.sourceMutex.Lock()
	defer h.sourceMutex.Unlock()

	// If using anchored windows when last submit is greater than the window
	// it will re-anchor the next window with a new healthy point.
	if h.useAnchoredWindows && h.timeProvider.Now().Sub(h.lastSuccessTime) > h.windowSize &&
		(h.lastErrorTime.IsZero() || h.timeProvider.Now().Sub(h.lastErrorTime) > h.windowSize) {
		// This check source treats no data as healthy so implicitly
		// are already reporting healthy when doing the re-anchor
		h.lastSuccessTime = h.timeProvider.Now()
	}

	if err != nil {
		h.lastError = err
		h.lastErrorTime = h.timeProvider.Now()
	} else {
		h.lastSuccessTime = h.timeProvider.Now()
	}
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (h *healthyIfNotAllErrorsSource) HealthStatus(ctx context.Context) health.HealthStatus {
	h.sourceMutex.RLock()
	defer h.sourceMutex.RUnlock()

	var healthCheckResult health.HealthCheckResult
	curTime := h.timeProvider.Now()
	if !h.lastSuccessTime.IsZero() && curTime.Sub(h.lastSuccessTime) < h.windowSize {
		healthCheckResult = whealth.HealthyHealthCheckResult(h.checkType)
	} else if !h.lastErrorTime.IsZero() && curTime.Sub(h.lastErrorTime) < h.windowSize {

		if h.requireFirstFullWindow && curTime.Sub(h.startTime) < h.windowSize {
			healthCheckResult = whealth.RepairingHealthCheckResult(h.checkType, h.lastError.Error())
		} else {
			healthCheckResult = whealth.UnhealthyHealthCheckResult(h.checkType, h.lastError.Error())
		}
	} else {
		healthCheckResult = whealth.HealthyHealthCheckResult(h.checkType)
	}

	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			h.checkType: healthCheckResult,
		},
	}
}
