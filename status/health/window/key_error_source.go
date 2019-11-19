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
	source, err := NewMultiKeyHealthyIfNotAllErrorsSource(checkType, messageInCaseOfError, windowSize)
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

// multiKeyHealthyIfNotAllErrorsSource is a HealthCheckSource that keeps the latest errors
// for multiple keys submitted within the last windowSize time frame.
// It returns unhealthy if there is at least one key with only non-nil errors within the last windowSize time frame.
// The Params field of the HealthCheckResult is the last error message for each key mapped by the key for all unhealthy keys.
// If there are no items within the last windowSize time frame, returns healthy.
type multiKeyHealthyIfNotAllErrorsSource struct {
	windowSize           time.Duration
	errorStore           TimedKeyStore
	successStore         TimedKeyStore
	lastError            map[string]error
	sourceMutex          sync.Mutex
	checkType            health.CheckType
	messageInCaseOfError string
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
func NewMultiKeyHealthyIfNotAllErrorsSource(checkType health.CheckType, messageInCaseOfError string, windowSize time.Duration) (KeyedErrorHealthCheckSource, error) {
	if windowSize <= 0 {
		return nil, werror.Error("windowSize must be positive", werror.SafeParam("windowSize", windowSize))
	}
	return &multiKeyHealthyIfNotAllErrorsSource{
		windowSize:           windowSize,
		errorStore:           NewTimedKeyStore(),
		successStore:         NewTimedKeyStore(),
		lastError:            make(map[string]error),
		checkType:            checkType,
		messageInCaseOfError: messageInCaseOfError,
	}, nil
}

// Submit submits an item as a key error pair.
func (m *multiKeyHealthyIfNotAllErrorsSource) Submit(key string, err error) {
	m.sourceMutex.Lock()
	defer m.sourceMutex.Unlock()

	m.pruneOldKeys(m.errorStore, m.lastError)
	m.pruneOldKeys(m.successStore, nil)

	if err == nil {
		m.successStore.Put(key)
		delete(m.lastError, key)
		m.errorStore.Delete(key)
	} else {
		m.lastError[key] = err
		m.errorStore.Put(key)
	}
}

// HealthStatus polls the items inside the window and creates the HealthStatus.
func (m *multiKeyHealthyIfNotAllErrorsSource) HealthStatus(ctx context.Context) health.HealthStatus {
	m.sourceMutex.Lock()
	defer m.sourceMutex.Unlock()

	m.pruneOldKeys(m.errorStore, m.lastError)
	m.pruneOldKeys(m.successStore, nil)

	params := make(map[string]interface{})
	for _, key := range m.errorStore.List().Keys() {
		if _, hasSuccess := m.successStore.Get(key); hasSuccess {
			continue
		}
		params[key] = m.lastError[key].Error()
	}

	var healthCheckResult health.HealthCheckResult
	if len(params) > 0 {
		healthCheckResult = health.HealthCheckResult{
			Type:    m.checkType,
			State:   health.HealthStateError,
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

func (m *multiKeyHealthyIfNotAllErrorsSource) pruneOldKeys(store TimedKeyStore, errors map[string]error) {
	curTime := time.Now()
	for {
		oldest, exists := store.Oldest()
		if !exists {
			return
		}

		if curTime.Sub(oldest.Time) < m.windowSize {
			return
		}

		store.Delete(oldest.Key)
		if errors != nil {
			delete(errors, oldest.Key)
		}
	}
}
