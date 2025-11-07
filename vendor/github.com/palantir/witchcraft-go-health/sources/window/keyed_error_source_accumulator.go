// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-health/status"
)

// KeyedErrorSourceAccumulator extends the KeyedErrorHealthCheckSource interface
// It includes an additional method AddHealthCheckSource which allows health sources to be added into the base health sources
// These sources are queried in conjunction with the given KeyedErrorHealthCheckSource
// This can be useful if you are working in legacy code bases and are in the process of converting health check, but need to add existing checks into a KeyedErrorHealthCheckSource
type KeyedErrorSourceAccumulator interface {
	AddHealthCheckSource(healthCheckSource status.HealthCheckSource)
	KeyedErrorHealthCheckSource
}

type defaultKeyedErrorSourceAccumulator struct {
	mutex                       sync.Mutex
	keyedErrorHealthCheckSource KeyedErrorHealthCheckSource
	allSources                  []status.HealthCheckSource
}

// NewDefaultKeyedErrorSourceAccumulator is the default implementation of KeyedErrorSourceAccumulator
func NewDefaultKeyedErrorSourceAccumulator(keyedErrorHealthCheckSource KeyedErrorHealthCheckSource) KeyedErrorSourceAccumulator {
	return &defaultKeyedErrorSourceAccumulator{
		keyedErrorHealthCheckSource: keyedErrorHealthCheckSource,
		allSources:                  []status.HealthCheckSource{keyedErrorHealthCheckSource},
	}
}

func (n *defaultKeyedErrorSourceAccumulator) AddHealthCheckSource(healthCheckSource status.HealthCheckSource) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	n.allSources = append(n.allSources, healthCheckSource)
}

func (n *defaultKeyedErrorSourceAccumulator) Submit(key string, err error) {
	n.keyedErrorHealthCheckSource.Submit(key, err)
}

func (n *defaultKeyedErrorSourceAccumulator) HealthStatus(ctx context.Context) health.HealthStatus {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	result := health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{},
	}
	for _, healthCheckSource := range n.allSources {
		for k, v := range healthCheckSource.HealthStatus(ctx).Checks {
			result.Checks[k] = v
		}
	}
	return result

}
