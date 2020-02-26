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
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/stretchr/testify/assert"
)

const (
	checkName        = "TEST_CHECK"
	windowSize       = 100 * time.Millisecond
	halfwindowSize   = windowSize / 2
	doubleWindowSize = windowSize * 2
)

// TestErrorInInitialWindow validates that the anchor prevents a single error
// in the first window from causing the health status to become unhealthy
func TestErrorInInitialWindow(t *testing.T) {
	anchoredWindow := MustNewAnchoredHealthyIfNotAllErrorsSource(checkName, windowSize)
	anchoredWindow.Submit(werror.Error("an error"))
	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[checkName]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateHealthy, checkResult.State)
}

// TestErrorBeforeAndAfterGap validates that errors in the anchor period
// will cause health status change as the window slides past the anchored
// window without new healthy statuses to keep state healthy
func TestErrorBeforeAndAfterGap(t *testing.T) {
	anchoredWindow := MustNewAnchoredHealthyIfNotAllErrorsSource(checkName, windowSize)
	time.Sleep(halfwindowSize)
	anchoredWindow.Submit(werror.Error("an error"))
	time.Sleep(halfwindowSize)
	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[checkName]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateError, checkResult.State)
}

// TestHealthyInGapBeforeAnchor validates that a healthy status
// anchor is applied after a gap period and prevents a single error from
// changing status in new period after gap
func TestHealthyInGapBeforeAnchor(t *testing.T) {
	anchoredWindow := MustNewAnchoredHealthyIfNotAllErrorsSource(checkName, windowSize)

	time.Sleep(halfwindowSize)
	anchoredWindow.Submit(werror.Error("an error"))
	time.Sleep(halfwindowSize)

	healthStatus := anchoredWindow.HealthStatus(context.Background())
	checkResult, ok := healthStatus.Checks[checkName]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateError, checkResult.State)
	time.Sleep(doubleWindowSize)

	anchoredWindow.Submit(werror.Error("an error"))
	healthStatus = anchoredWindow.HealthStatus(context.Background())
	checkResult, ok = healthStatus.Checks[checkName]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateHealthy, checkResult.State)
}
