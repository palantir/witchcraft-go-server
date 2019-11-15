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

package refreshable

import (
	"context"
	"sync/atomic"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	whealth "github.com/palantir/witchcraft-go-server/status/health"
)

type ValidatingRefreshable struct {
	validatedRefreshable Refreshable
	lastValidateErr      *atomic.Value
	healthCheckType      health.CheckType
}

func (v *ValidatingRefreshable) HealthStatus(ctx context.Context) health.HealthStatus {
	healthCheckResult := whealth.HealthyHealthCheckResult(v.healthCheckType)

	err := v.lastValidateErr.Load()
	if err != nil {
		healthCheckResult = whealth.UnhealthyHealthCheckResult(v.healthCheckType, err.(error).Error())
	}

	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			v.healthCheckType: healthCheckResult,
		},
	}
}

func (v *ValidatingRefreshable) Current() interface{} {
	return v.validatedRefreshable.Current()
}

func (v *ValidatingRefreshable) Subscribe(consumer func(interface{})) (unsubscribe func()) {
	return v.validatedRefreshable.Subscribe(consumer)
}

func (v *ValidatingRefreshable) Map(mapFn func(interface{}) interface{}) Refreshable {
	return v.validatedRefreshable.Map(mapFn)
}

// NewValidatingRefreshable returns a new Refreshable whose current value is the latest value that passes the provided
// validatingFn successfully. This refreshable is also a HealthCheckSource that will be unhealthy whenever there are
// updates that have failed validation.
func NewValidatingRefreshable(origRefreshable Refreshable, healthCheckType health.CheckType, validatingFn func(interface{}) error) (*ValidatingRefreshable, error) {
	currentVal := origRefreshable.Current()
	if err := validatingFn(currentVal); err != nil {
		return nil, werror.Wrap(err, "failed to create validating Refreshable because initial value could not be validated")
	}

	validatedRefreshable := NewDefaultRefreshable(currentVal)

	var lastValidateErr atomic.Value
	v := ValidatingRefreshable{
		healthCheckType:      healthCheckType,
		validatedRefreshable: validatedRefreshable,
		lastValidateErr:      &lastValidateErr,
	}

	_ = origRefreshable.Subscribe(func(i interface{}) {
		if err := validatingFn(i); err != nil {
			v.lastValidateErr.Store(err)
			return
		}

		if err := validatedRefreshable.Update(i); err != nil {
			v.lastValidateErr.Store(err)
			return
		}

		v.lastValidateErr.Store(nil)
	})
	return &v, nil
}
