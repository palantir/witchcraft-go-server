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

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

// KeyedErrorSubmitter is a separate interface to allow method signatures to only accept the Submit functionality of
// a KeyedErrorHealthCheckSource
type KeyedErrorSubmitter interface {
	Submit(key string, err error)
}

// KeyedErrorHealthCheckSource tracks errors by key to compute health status
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

// NewKeyedErrorHealthCheckSource creates a health messenger that tracks errors by keys to compute health status
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
					State:   health.HealthStateHealthy,
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
				State:   health.HealthStateError,
			},
		},
	}
}
