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
	whealth "github.com/palantir/witchcraft-go-server/status/health"
)

// UnhealthyIfAtLeastOneError builds an EventsToCheckFn that returns the first non-nil error as an unhealthy check.
// If there are no events, returns healthy.
// Payload is considered to be of type error and panics if it is not.
func UnhealthyIfAtLeastOneError(checkType health.CheckType) EventsToCheckFn {
	return func(ctx context.Context, events []Event) health.HealthCheckResult {
		for _, event := range events {
			if event.Payload != nil {
				return whealth.UnhealthyHealthCheckResult(checkType, event.Payload.(error).Error())
			}
		}
		return whealth.HealthyHealthCheckResult(checkType)
	}
}

// HealthyIfNotAllErrors builds an EventsToCheckFn that returns, if there are only non-nil errors,
// the first non-nil error as an unhealthy check.
// If there are no events, returns healthy.
// Payload is considered to be of type error and panics if it is not.
func HealthyIfNotAllErrors(checkType health.CheckType) EventsToCheckFn {
	return func(ctx context.Context, events []Event) health.HealthCheckResult {
		if len(events) == 0 {
			return whealth.HealthyHealthCheckResult(checkType)
		}
		for _, event := range events {
			if event.Payload == nil {
				return whealth.HealthyHealthCheckResult(checkType)
			}
		}
		return whealth.UnhealthyHealthCheckResult(checkType, events[0].Payload.(error).Error())
	}
}

// KeyErrorPair is a struct that keeps a key as a string and an error.
type KeyErrorPair struct {
	// Key is an identifier for a resource.
	Key string
	// Error is the result of some operation for a resource.
	Error error
}

// MultiKeyUnhealthyIfAtLeastOneError builds an EventsToCheckFn that returns unhealthy
// if there is a non-nil error for at least one key.
// The Params field of the HealthCheckResult is the first error for each key mapped by the key for all keys with errors.
// If there are no events, returns healthy.
// Payload is considered to be of type KeyErrorPair and panics if it is not.
func MultiKeyUnhealthyIfAtLeastOneError(checkType health.CheckType, messageInCaseOfError string) EventsToCheckFn {
	return func(ctx context.Context, events []Event) health.HealthCheckResult {
		params := make(map[string]interface{})
		for _, event := range events {
			keyErrorPair := event.Payload.(KeyErrorPair)
			if keyErrorPair.Error == nil {
				continue
			}
			if _, alreadyHasError := params[keyErrorPair.Key]; !alreadyHasError {
				params[keyErrorPair.Key] = keyErrorPair.Error
			}
		}
		if len(params) > 0 {
			return health.HealthCheckResult{
				Type:    checkType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params:  params,
			}
		}
		return whealth.HealthyHealthCheckResult(checkType)
	}
}

// MultiKeyHealthyIfNotAllErrors builds an EventsToCheckFn that returns unhealthy
// if there is at least one key with only errors.
// The Params field of the HealthCheckResult is the first error for each key mapped by the key for all keys with errors.
// If there are no events, returns healthy.
// Payload is considered to be of type KeyErrorPair and panics if it is not.
func MultiKeyHealthyIfNotAllErrors(checkType health.CheckType, messageInCaseOfError string) EventsToCheckFn {
	return func(ctx context.Context, events []Event) health.HealthCheckResult {
		params := make(map[string]interface{})
		hasSuccess := make(map[string]struct{})
		for _, event := range events {
			keyErrorPair := event.Payload.(KeyErrorPair)
			if _, keyHasSuccess := hasSuccess[keyErrorPair.Key]; keyHasSuccess {
				continue
			}
			if keyErrorPair.Error == nil {
				delete(params, keyErrorPair.Key)
				hasSuccess[keyErrorPair.Key] = struct{}{}
				continue
			}
			if _, alreadyHasError := params[keyErrorPair.Key]; !alreadyHasError {
				params[keyErrorPair.Key] = keyErrorPair.Error
			}
		}
		if len(params) > 0 {
			return health.HealthCheckResult{
				Type:    checkType,
				State:   health.HealthStateError,
				Message: &messageInCaseOfError,
				Params:  params,
			}
		}
		return whealth.HealthyHealthCheckResult(checkType)
	}
}
