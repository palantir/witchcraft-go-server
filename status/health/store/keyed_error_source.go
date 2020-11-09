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

package store

import (
	"context"
	"sync"

	"github.com/palantir/witchcraft-go-server/v2/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/v2/status"
)

// KeyedErrorSubmitter is not intended to be implemented separately - it simply
// provides consumers the ability to scope a method argument, such as a constructor, to provide
// only the Submit functionality of the KeyedErrorHealthCheckSource.
type KeyedErrorSubmitter interface {
	// Submit stores non-nil errors by the provided key in a map; keys of submitted nil errors are
	// deleted from the map.
	Submit(key string, err error)
}

// KeyedErrorHealthCheckSource tracks errors by key to compute health status. Only entries with non-nil
// errors are stored. When computing health status, the KeyedErrorHealthCheckSource will return a
// health status with state HealthStateHealthy if it has no error entries. If it has any error entries, it will return
// a health status with the state set to HealthStateError and params including all errors by their keys.
type KeyedErrorHealthCheckSource interface {
	KeyedErrorSubmitter
	status.HealthCheckSource
}

type keyedErrorHealthCheckSource struct {
	lock         sync.Mutex
	keyedErrors  map[string]error
	checkType    health.CheckType
	checkMessage string
}

// NewKeyedErrorHealthCheckSource creates a health messenger that tracks errors by keys to compute health status.
func NewKeyedErrorHealthCheckSource(checkType health.CheckType, checkMessage string) KeyedErrorHealthCheckSource {
	return &keyedErrorHealthCheckSource{
		checkType:    checkType,
		checkMessage: checkMessage,
		keyedErrors:  make(map[string]error),
	}
}

func (k *keyedErrorHealthCheckSource) Submit(key string, err error) {
	k.lock.Lock()
	defer k.lock.Unlock()
	if err == nil {
		delete(k.keyedErrors, key)
	} else {
		k.keyedErrors[key] = err
	}
}

func (k *keyedErrorHealthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	k.lock.Lock()
	defer k.lock.Unlock()
	if len(k.keyedErrors) == 0 {
		return health.HealthStatus{
			Checks: map[health.CheckType]health.HealthCheckResult{
				k.checkType: {
					Message: &k.checkMessage,
					Type:    k.checkType,
					State:   health.New_HealthState(health.HealthState_HEALTHY),
				},
			},
		}
	}
	params := make(map[string]interface{}, len(k.keyedErrors))
	for key, err := range k.keyedErrors {
		params[key] = err.Error()
	}
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			k.checkType: {
				Message: &k.checkMessage,
				Params:  params,
				Type:    k.checkType,
				State:   health.New_HealthState(health.HealthState_ERROR),
			},
		},
	}
}
