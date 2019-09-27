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
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
)

// ItemsToCheckFn is a function that constructs a HealthCheckResult from a set of items.
type ItemsToCheckFn func(ctx context.Context, items []ItemWithTimestamp) health.HealthCheckResult

// BaseHealthCheckSource is a HealthCheckSource that polls a TimeWindowedStore.
// It returns a HealthStatus created using an ItemsToCheckFn.
type BaseHealthCheckSource struct {
	timeWindowedStore *TimeWindowedStore
	itemsToCheckFn    ItemsToCheckFn
}

var _ status.HealthCheckSource = &BaseHealthCheckSource{}

// NewBaseHealthCheckSource creates a BaseHealthCheckSource
// with a sliding window of size windowSize and uses the itemsToCheckFn.
// windowSize must be a positive value and itemsToCheckFn must not be nil, otherwise returns error.
func NewBaseHealthCheckSource(windowSize time.Duration, itemsToCheckFn ItemsToCheckFn) (*BaseHealthCheckSource, error) {
	timeWindowedStore, err := NewTimeWindowedStore(windowSize)
	if err != nil {
		return nil, err
	}
	if itemsToCheckFn == nil {
		return nil, werror.Error("itemsToCheckFn cannot be nil")
	}
	return &BaseHealthCheckSource{
		timeWindowedStore: timeWindowedStore,
		itemsToCheckFn:    itemsToCheckFn,
	}, nil
}

// Submit submits an item to the TimeWindowedStore.
func (b *BaseHealthCheckSource) Submit(item interface{}) {
	b.timeWindowedStore.Submit(item)
}

// HealthStatus polls the TimeWindowedStore and creates a HealthStatus using the ItemsToCheckFn.
func (b *BaseHealthCheckSource) HealthStatus(ctx context.Context) health.HealthStatus {
	checkResult := b.itemsToCheckFn(ctx, b.timeWindowedStore.ItemsInWindow())
	return health.HealthStatus{
		Checks: map[health.CheckType]health.HealthCheckResult{
			checkResult.Type: checkResult,
		},
	}
}
