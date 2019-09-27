// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

// keyErrorPair is a struct that keeps a key as a string and an err.
type keyErrorPair struct {
	// key is an identifier for a resource.
	key string
	// err is the result of some operation for a resource.
	err error
}

// MultiKeyUnhealthyIfAtLeastOneErrorSource is a HealthCheckSource that polls a TimeWindowedEventStorer.
// It returns unhealthy if there is a non-nil error for at least one key.
// The Params field of the HealthCheckResult is the first error for each key mapped by the key for all unhealthy keys.
// If there are no events, returns healthy.
type MultiKeyUnhealthyIfAtLeastOneErrorSource struct {
	timeWindowedEventStorer *TimeWindowedEventStorer
	checkType               health.CheckType
	messageInCaseOfError    string
}

var _ status.HealthCheckSource = &MultiKeyUnhealthyIfAtLeastOneErrorSource{}

// NewMultiKeyUnhealthyIfAtLeastOneErrorSource creates an MultiKeyUnhealthyIfAtLeastOneErrorSource
// with a sliding window of size windowSize and uses the checkType and a message in case of errors.
func NewMultiKeyUnhealthyIfAtLeastOneErrorSource(windowSize time.Duration, checkType health.CheckType, messageInCaseOfError string) (*MultiKeyUnhealthyIfAtLeastOneErrorSource, error) {
	timeWindowedEventStorer, err := NewTimeWindowedEventStorer(windowSize)
	if err != nil {
		return nil, err
	}
	return &MultiKeyUnhealthyIfAtLeastOneErrorSource{
		timeWindowedEventStorer: timeWindowedEventStorer,
		checkType:               checkType,
		messageInCaseOfError:    messageInCaseOfError,
	}, nil
}

// SubmitError submits an event to the TimeWindowedEventStorer.
func (m *MultiKeyUnhealthyIfAtLeastOneErrorSource) SubmitError(key string, err error) {
	m.timeWindowedEventStorer.SubmitEvent(keyErrorPair{
		key: key,
		err: err,
	})
}

func (m *MultiKeyUnhealthyIfAtLeastOneErrorSource) eventsToCheck() health.HealthCheckResult {
	events := m.timeWindowedEventStorer.GetEventsInWindow()
	params := make(map[string]interface{})
	for _, event := range events {
		keyErrorPair := event.Payload.(keyErrorPair)
		if keyErrorPair.err == nil {
			continue
		}
		if _, alreadyHasError := params[keyErrorPair.key]; !alreadyHasError {
			params[keyErrorPair.key] = keyErrorPair.err
		}
	}
	if len(params) > 0 {
		return health.HealthCheckResult{
			Type:    m.checkType,
			State:   health.HealthStateError,
			Message: &m.messageInCaseOfError,
			Params:  params,
		}
	}
	return whealth.HealthyHealthCheckResult(m.checkType)
}

// HealthStatus polls the TimeWindowedEventStorer and creates the HealthStatus.
func (m *MultiKeyUnhealthyIfAtLeastOneErrorSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			m.checkType: m.eventsToCheck(),
		},
	}
}

// MultiKeyHealthyIfNotAllErrorsSource is a HealthCheckSource that polls a TimeWindowedEventStorer.
// It returns unhealthy if there is at least one key with only non-nil errors.
// The Params field of the HealthCheckResult is the first error for each key mapped by the key for all unhealthy keys.
// If there are no events, returns healthy.
type MultiKeyHealthyIfNotAllErrorsSource struct {
	timeWindowedEventStorer *TimeWindowedEventStorer
	checkType               health.CheckType
	messageInCaseOfError    string
}

var _ status.HealthCheckSource = &MultiKeyHealthyIfNotAllErrorsSource{}

// NewMultiKeyHealthyIfNotAllErrorsSource creates an MultiKeyUnhealthyIfAtLeastOneErrorSource
// with a sliding window of size windowSize and uses the checkType and a message in case of errors.
func NewMultiKeyHealthyIfNotAllErrorsSource(windowSize time.Duration, checkType health.CheckType, messageInCaseOfError string) (*MultiKeyHealthyIfNotAllErrorsSource, error) {
	timeWindowedEventStorer, err := NewTimeWindowedEventStorer(windowSize)
	if err != nil {
		return nil, err
	}
	return &MultiKeyHealthyIfNotAllErrorsSource{
		timeWindowedEventStorer: timeWindowedEventStorer,
		checkType:               checkType,
		messageInCaseOfError:    messageInCaseOfError,
	}, nil
}

// SubmitError submits an event to the TimeWindowedEventStorer.
func (m *MultiKeyHealthyIfNotAllErrorsSource) SubmitError(key string, err error) {
	m.timeWindowedEventStorer.SubmitEvent(keyErrorPair{
		key: key,
		err: err,
	})
}

func (m *MultiKeyHealthyIfNotAllErrorsSource) eventsToCheck() health.HealthCheckResult {
	events := m.timeWindowedEventStorer.GetEventsInWindow()
	params := make(map[string]interface{})
	hasSuccess := make(map[string]struct{})
	for _, event := range events {
		keyErrorPair := event.Payload.(keyErrorPair)
		if _, keyHasSuccess := hasSuccess[keyErrorPair.key]; keyHasSuccess {
			continue
		}
		if keyErrorPair.err == nil {
			delete(params, keyErrorPair.key)
			hasSuccess[keyErrorPair.key] = struct{}{}
			continue
		}
		if _, alreadyHasError := params[keyErrorPair.key]; !alreadyHasError {
			params[keyErrorPair.key] = keyErrorPair.err
		}
	}
	if len(params) > 0 {
		return health.HealthCheckResult{
			Type:    m.checkType,
			State:   health.HealthStateError,
			Message: &m.messageInCaseOfError,
			Params:  params,
		}
	}
	return whealth.HealthyHealthCheckResult(m.checkType)
}

// HealthStatus polls the TimeWindowedEventStorer and creates the HealthStatus.
func (m *MultiKeyHealthyIfNotAllErrorsSource) HealthStatus(ctx context.Context) health.HealthStatus {
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			m.checkType: m.eventsToCheck(),
		},
	}
}
