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

// KeyedErrorSubmitter allows components whose functionality dictates a portion of health status to only consume this interface.
type KeyedErrorSubmitter interface {
	Submit(key string, err error)
}

// KeyedErrorHealthCheckSource is a health check source with statuses determined by submitted key error pairs.
type KeyedErrorHealthCheckSource interface {
	KeyedErrorSubmitter
	status.HealthCheckSource
}

// multiKeyUnhealthyIfAtLeastOneErrorSource is a HealthCheckSource that keeps the latest errors
// for multiple keys submitted within the last windowSize time frame.
// It returns unhealthy if there is a non-nil error for at least one key within the last windowSize time frame.
// The Params field of the HealthCheckResult is the last error message for each key mapped by the key for all unhealthy keys.
// If there are no items within the last windowSize time frame, returns healthy.
type multiKeyUnhealthyIfAtLeastOneErrorSource struct {
	// multiKeyUnhealthyIfAtLeastOneErrorSource is a multiKeyHealthyIfNotAllErrorsSource that drops all successes.
	source KeyedErrorHealthCheckSource
}

// MustNewMultiKeyUnhealthyIfAtLeastOneErrorSource returns the result of calling NewMultiKeyUnhealthyIfAtLeastOneErrorSource, but panics if it returns an error.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewMultiKeyUnhealthyIfAtLeastOneErrorSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) KeyedErrorHealthCheckSource {
	source, err := NewMultiKeyUnhealthyIfAtLeastOneErrorSource(checkType, messageInCaseOfError, windowSize)
	if err != nil {
		panic(err)
	}
	return source
}

// NewMultiKeyUnhealthyIfAtLeastOneErrorSource creates an multiKeyUnhealthyIfAtLeastOneErrorSource
// with a sliding window of size windowSize and uses the checkType and a message in case of errors.
// windowSize must be a positive value, otherwise returns error.
func NewMultiKeyUnhealthyIfAtLeastOneErrorSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) (KeyedErrorHealthCheckSource, error) {
	source, err := newMultiKeyHealthyIfNotAllErrorsSource(checkType, messageInCaseOfError, windowSize, 0, false, NewOrdinaryTimeProvider())
	if err != nil {
		return nil, err
	}
	return &multiKeyUnhealthyIfAtLeastOneErrorSource{
		source: source,
	}, nil
}

// Submit submits an item as a key error pair.
func (m *multiKeyUnhealthyIfAtLeastOneErrorSource) Submit(key string, err error) {
	if err == nil {
		return
	}
	m.source.Submit(key, err)
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (m *multiKeyUnhealthyIfAtLeastOneErrorSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return m.source.HealthStatus(ctx)
}

// multiKeyUnhealthyIfNoRecentErrorsSource is a HealthCheckSource that keeps the latest errors
// for multiple keys submitted within the last windowSize time frame.
// It returns unhealthy if there is at least one key with a non-nil error within the last windowSize time frame.
// If a nil error is submitted while a non-nil error is still active, the nil error clears the error and restores health.
// The Params field of the HealthCheckResult is the last error message for each key mapped by the key for all unhealthy keys.
// If there are no items within the last windowSize time frame, returns healthy.
type multiKeyHealthyIfNoRecentErrorsSource struct {
	windowSize           time.Duration
	errorStore           TimedKeyStore
	sourceMutex          sync.Mutex
	checkType            health.CheckType
	messageInCaseOfError string
	timeProvider         TimeProvider
}

var _ status.HealthCheckSource = &multiKeyHealthyIfNoRecentErrorsSource{}

// MustNewMultiKeyHealthyIfNoRecentErrorsSource returns the result of calling NewMultiKeyHealthyIfNoRecentErrorsSource, but panics if it returns an error.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewMultiKeyHealthyIfNoRecentErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) KeyedErrorHealthCheckSource {
	source, err := NewMultiKeyHealthyIfNoRecentErrorsSource(checkType, messageInCaseOfError, windowSize)
	if err != nil {
		panic(err)
	}
	return source
}

// NewMultiKeyHealthyIfNoRecentErrorsSource creates an multiKeyUnhealthyIfNoRecentErrorsSource
// with a sliding window of size windowSize and uses the checkType and a message in case of errors.
// windowSize must be a positive value, otherwise returns error.
// Once a non-nil error has been submitted, this will be unhealthy until a nil error is submitted or `windowSize` time
// has passed without a non-nil error. Submitting a non-nil error resets the timer and stays unhealthy
func NewMultiKeyHealthyIfNoRecentErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) (KeyedErrorHealthCheckSource, error) {
	return newMultiKeyHealthyIfNoRecentErrorsSource(checkType, messageInCaseOfError, windowSize, NewOrdinaryTimeProvider())
}

func newMultiKeyHealthyIfNoRecentErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration, timeProvider TimeProvider) (KeyedErrorHealthCheckSource, error) {
	if windowSize <= 0 {
		return nil, werror.Error("windowSize must be positive", werror.SafeParam("windowSize", windowSize.String()))
	}

	return &multiKeyHealthyIfNoRecentErrorsSource{
		windowSize:           windowSize,
		errorStore:           NewTimedKeyStore(timeProvider),
		sourceMutex:          sync.Mutex{},
		checkType:            checkType,
		messageInCaseOfError: messageInCaseOfError,
		timeProvider:         timeProvider,
	}, nil
}

// Submit submits an item as a key error pair.
func (m *multiKeyHealthyIfNoRecentErrorsSource) Submit(key string, err error) {
	m.sourceMutex.Lock()
	defer m.sourceMutex.Unlock()

	m.errorStore.PruneOldKeys(m.windowSize, m.timeProvider)
	m.errorStore.Put(key, err)
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (m *multiKeyHealthyIfNoRecentErrorsSource) HealthStatus(ctx context.Context) health.HealthStatus {
	m.sourceMutex.Lock()
	defer m.sourceMutex.Unlock()

	m.errorStore.PruneOldKeys(m.windowSize, m.timeProvider)

	var healthCheckResult health.HealthCheckResult
	params := make(map[string]interface{})
	for _, item := range m.errorStore.List() {
		// nil error is not error
		if item.Payload != nil {
			params[item.Key] = item.Payload.(error).Error()
		}
	}

	if len(params) > 0 {
		healthCheckResult = health.HealthCheckResult{
			Type:    m.checkType,
			State:   health.New_HealthState(health.HealthState_ERROR),
			Message: &m.messageInCaseOfError,
			Params:  params,
		}
	} else {
		healthCheckResult = whealth.HealthyHealthCheckResult(m.checkType)
	}

	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			m.checkType: healthCheckResult,
		},
	}
}

// multiKeyHealthyIfNotAllErrorsSource is a HealthCheckSource that keeps the latest errors
// for multiple keys submitted within the last windowSize time frame.
// It returns unhealthy if there is at least one key with only non-nil errors within the last windowSize time frame.
// The Params field of the HealthCheckResult is the last error message for each key mapped by the key for all unhealthy keys.
// If there are no items within the last windowSize time frame, returns healthy.
type multiKeyHealthyIfNotAllErrorsSource struct {
	windowSize              time.Duration
	errorStore              TimedKeyStore
	successStore            TimedKeyStore
	gapEndTimeStore         TimedKeyStore
	repairingGracePeriod    time.Duration
	globalRepairingDeadline time.Time
	sourceMutex             sync.Mutex
	checkType               health.CheckType
	messageInCaseOfError    string
	timeProvider            TimeProvider
}

var _ status.HealthCheckSource = &multiKeyHealthyIfNotAllErrorsSource{}

// MustNewMultiKeyHealthyIfNotAllErrorsSource returns the result of calling NewMultiKeyHealthyIfNotAllErrorsSource, but panics if it returns an error.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewMultiKeyHealthyIfNotAllErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) KeyedErrorHealthCheckSource {
	source, err := NewMultiKeyHealthyIfNotAllErrorsSource(checkType, messageInCaseOfError, windowSize)
	if err != nil {
		panic(err)
	}
	return source
}

// NewMultiKeyHealthyIfNotAllErrorsSource creates an multiKeyUnhealthyIfAtLeastOneErrorSource
// with a sliding window of size windowSize and uses the checkType and a message in case of errors.
// windowSize must be a positive value, otherwise returns error.
// Errors submitted in the first time window cause the health check to go to REPAIRING instead of ERROR.
func NewMultiKeyHealthyIfNotAllErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) (KeyedErrorHealthCheckSource, error) {
	return newMultiKeyHealthyIfNotAllErrorsSource(checkType, messageInCaseOfError, windowSize, 0, true, NewOrdinaryTimeProvider())
}

// MustNewAnchoredMultiKeyHealthyIfNotAllErrorsSource returns the result of calling NewAnchoredMultiKeyHealthyIfNotAllErrorsSource, but panics if it returns an error.
// Should only be used in instances where the inputs are statically defined and known to be valid.
func MustNewAnchoredMultiKeyHealthyIfNotAllErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) KeyedErrorHealthCheckSource {
	source, err := NewAnchoredMultiKeyHealthyIfNotAllErrorsSource(checkType, messageInCaseOfError, windowSize)
	if err != nil {
		panic(err)
	}
	return source
}

// NewAnchoredMultiKeyHealthyIfNotAllErrorsSource creates an multiKeyUnhealthyIfAtLeastOneErrorSource
// with a sliding window of size windowSize and uses the checkType and a message in case of errors.
// Each key has a repairing deadline that is one window size after a moment of idleness.
// If all errors happen before their respective repairing deadline, the health check returns REPAIRING instead of ERROR.
// windowSize must be a positive value, otherwise returns error.
// Errors submitted in the first time window cause the health check to go to REPAIRING instead of ERROR.
func NewAnchoredMultiKeyHealthyIfNotAllErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) (KeyedErrorHealthCheckSource, error) {
	return newMultiKeyHealthyIfNotAllErrorsSource(checkType, messageInCaseOfError, windowSize, windowSize, true, NewOrdinaryTimeProvider())
}

func newMultiKeyHealthyIfNotAllErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize, repairingGracePeriod time.Duration, requireFirstFullWindow bool, timeProvider TimeProvider) (KeyedErrorHealthCheckSource, error) {
	if windowSize <= 0 {
		return nil, werror.Error("windowSize must be positive", werror.SafeParam("windowSize", windowSize.String()))
	}
	if repairingGracePeriod < 0 {
		return nil, werror.Error("repairingGracePeriod must be non negative", werror.SafeParam("repairingGracePeriod", repairingGracePeriod.String()))
	}

	source := &multiKeyHealthyIfNotAllErrorsSource{
		windowSize:              windowSize,
		errorStore:              NewTimedKeyStore(timeProvider),
		successStore:            NewTimedKeyStore(timeProvider),
		gapEndTimeStore:         NewTimedKeyStore(timeProvider),
		repairingGracePeriod:    repairingGracePeriod,
		globalRepairingDeadline: timeProvider.Now(),
		checkType:               checkType,
		messageInCaseOfError:    messageInCaseOfError,
		timeProvider:            timeProvider,
	}

	if requireFirstFullWindow {
		source.globalRepairingDeadline = timeProvider.Now().Add(windowSize)
	}

	return source, nil
}

// Submit submits an item as a key error pair.
func (m *multiKeyHealthyIfNotAllErrorsSource) Submit(key string, err error) {
	m.sourceMutex.Lock()
	defer m.sourceMutex.Unlock()

	m.errorStore.PruneOldKeys(m.windowSize, m.timeProvider)
	m.successStore.PruneOldKeys(m.windowSize, m.timeProvider)
	m.gapEndTimeStore.PruneOldKeys(m.repairingGracePeriod+m.windowSize, m.timeProvider)

	_, hasError := m.errorStore.Get(key)
	_, hasSuccess := m.successStore.Get(key)
	if !hasError && !hasSuccess {
		m.gapEndTimeStore.Put(key, nil)
	}

	if err == nil {
		m.successStore.Put(key, nil)
	} else {
		m.errorStore.Put(key, err)
	}
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (m *multiKeyHealthyIfNotAllErrorsSource) HealthStatus(ctx context.Context) health.HealthStatus {
	m.sourceMutex.Lock()
	defer m.sourceMutex.Unlock()

	var healthCheckResult health.HealthCheckResult

	m.errorStore.PruneOldKeys(m.windowSize, m.timeProvider)
	m.successStore.PruneOldKeys(m.windowSize, m.timeProvider)
	m.gapEndTimeStore.PruneOldKeys(m.repairingGracePeriod+m.windowSize, m.timeProvider)

	params := make(map[string]interface{})
	shouldError := false
	for _, item := range m.errorStore.List() {
		if _, hasSuccess := m.successStore.Get(item.Key); hasSuccess {
			continue
		}

		if gapEndTime, hasRepairingDeadline := m.gapEndTimeStore.Get(item.Key); hasRepairingDeadline {
			repairingDeadline := gapEndTime.Time.Add(m.repairingGracePeriod)
			if m.globalRepairingDeadline.After(repairingDeadline) {
				repairingDeadline = m.globalRepairingDeadline
			}
			if item.Time.After(repairingDeadline) || item.Time.Equal(repairingDeadline) {
				shouldError = true
			}
		} else {
			shouldError = true
		}

		params[item.Key] = item.Payload.(error).Error()
	}

	if len(params) > 0 {
		healthCheckResult = health.HealthCheckResult{
			Type:    m.checkType,
			State:   health.New_HealthState(health.HealthState_REPAIRING),
			Message: &m.messageInCaseOfError,
			Params:  params,
		}
		if shouldError {
			healthCheckResult.State = health.New_HealthState(health.HealthState_ERROR)
		}
	} else {
		healthCheckResult = whealth.HealthyHealthCheckResult(m.checkType)
	}

	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			m.checkType: healthCheckResult,
		},
	}
}
