// Copyright (c) 2020 Palantir Technologies. All rights reserved.
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
	"github.com/palantir/witchcraft-go-server/v2/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/v2/status"
	whealth "github.com/palantir/witchcraft-go-server/v2/status/health"
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
// It returns the latest non-nil error as an unhealthy check.
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
	underlyingSource, err := newHealthyIfNotAllErrorsSource(checkType, windowSize, 0, false, NewOrdinaryTimeProvider())
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
// It returns, if there are only non-nil errors, the latest non-nil error as an unhealthy check.
// If there are no items, returns healthy.
type healthyIfNotAllErrorsSource struct {
	timeProvider         TimeProvider
	windowSize           time.Duration
	lastErrorTime        time.Time
	lastError            error
	lastSuccessTime      time.Time
	sourceMutex          sync.RWMutex
	checkType            health.CheckType
	repairingGracePeriod time.Duration
	repairingDeadline    time.Time
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
// Errors submitted in the first time window cause the health check to go to REPAIRING instead of ERROR.
func NewHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) (ErrorHealthCheckSource, error) {
	return newHealthyIfNotAllErrorsSource(checkType, windowSize, 0, true, NewOrdinaryTimeProvider())
}

// MustNewAnchoredHealthyIfNotAllErrorsSource returns the result of calling
// NewAnchoredHealthyIfNotAllErrorsSource but panics if that call returns an error
// Should only be used in instances where the inputs are statically defined and known to be valid.
// Care should be taken in considering health submission rate and window size when using anchored
// windows. Windows too close to service emission frequency may cause errors to not surface.
func MustNewAnchoredHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) ErrorHealthCheckSource {
	source, err := NewAnchoredHealthyIfNotAllErrorsSource(checkType, windowSize)
	if err != nil {
		panic(err)
	}
	return source
}

// NewAnchoredHealthyIfNotAllErrorsSource creates an healthyIfNotAllErrorsSource
// with supplied checkType, using sliding window of size windowSize, which will
// anchor (force the window to be at least the grace period) by defining a repairing deadline
// at the end of the initial window or one window size after the end of a gap.
// If all errors happen before the repairing deadline, the health check returns REPAIRING instead of ERROR.
// windowSize must be a positive value, otherwise returns error.
// Care should be taken in considering health submission rate and window size when using anchored
// windows. Windows too close to service emission frequency may cause errors to not surface.
func NewAnchoredHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize time.Duration) (ErrorHealthCheckSource, error) {
	return newHealthyIfNotAllErrorsSource(checkType, windowSize, windowSize, true, NewOrdinaryTimeProvider())
}

func newHealthyIfNotAllErrorsSource(checkType health.CheckType, windowSize, repairingGracePeriod time.Duration, requireFirstFullWindow bool, timeProvider TimeProvider) (ErrorHealthCheckSource, error) {
	if windowSize <= 0 {
		return nil, werror.Error("windowSize must be positive", werror.SafeParam("windowSize", windowSize.String()))
	}
	if repairingGracePeriod < 0 {
		return nil, werror.Error("repairingGracePeriod must be non negative", werror.SafeParam("repairingGracePeriod", repairingGracePeriod.String()))
	}

	source := &healthyIfNotAllErrorsSource{
		timeProvider:         timeProvider,
		windowSize:           windowSize,
		checkType:            checkType,
		repairingGracePeriod: repairingGracePeriod,
		repairingDeadline:    timeProvider.Now(),
	}

	// If requireFirstFullWindow, extend the repairing deadline to one windowSize from now.
	if requireFirstFullWindow {
		source.repairingDeadline = timeProvider.Now().Add(windowSize)
	}

	return source, nil
}

// Submit submits an error.
func (h *healthyIfNotAllErrorsSource) Submit(err error) {
	h.sourceMutex.Lock()
	defer h.sourceMutex.Unlock()

	// If using anchored windows when last submit is greater than the window
	// it will re-anchor the next window with a new repairing deadline.
	if !h.hasSuccessInWindow() && !h.hasErrorInWindow() {
		newRepairingDeadline := h.timeProvider.Now().Add(h.repairingGracePeriod)
		if newRepairingDeadline.After(h.repairingDeadline) {
			h.repairingDeadline = newRepairingDeadline
		}
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
	if h.hasSuccessInWindow() {
		healthCheckResult = whealth.HealthyHealthCheckResult(h.checkType)
	} else if h.hasErrorInWindow() {
		if h.lastErrorTime.Before(h.repairingDeadline) {
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

func (h *healthyIfNotAllErrorsSource) hasSuccessInWindow() bool {
	return !h.lastSuccessTime.IsZero() && h.timeProvider.Now().Sub(h.lastSuccessTime) <= h.windowSize
}

func (h *healthyIfNotAllErrorsSource) hasErrorInWindow() bool {
	return !h.lastErrorTime.IsZero() && h.timeProvider.Now().Sub(h.lastErrorTime) <= h.windowSize
}
