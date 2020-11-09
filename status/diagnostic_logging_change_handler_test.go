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

package status_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
	"github.com/stretchr/testify/assert"
)

func TestDiagnosticLoggingChangeHandler(t *testing.T) {
	handler := status.NewDiagnosticLoggingChangeHandler()
	for _, testCase := range []struct {
		name            string
		prev, curr      health.HealthStatus
		expectLogOutput bool
	}{
		{
			name: "no log if health status code repairing or less",
			prev: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST_CHECK": {State: health.New_HealthState(health.HealthState_HEALTHY)},
				},
			},
			curr: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST_CHECK": {State: health.New_HealthState(health.HealthState_REPAIRING)},
				},
			},
		},
		{
			name: "log if health status code worse than repairing",
			prev: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST_CHECK": {State: health.New_HealthState(health.HealthState_HEALTHY)},
				},
			},
			curr: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST_CHECK": {State: health.New_HealthState(health.HealthState_WARNING)},
				},
			},
			expectLogOutput: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := diag1log.WithLogger(context.Background(), diag1log.New(&buf))
			handler.HandleHealthStatusChange(ctx, testCase.prev, testCase.curr)
			if testCase.expectLogOutput {
				assert.True(t, len(buf.Bytes()) > 0)
			} else {
				assert.Empty(t, buf.Bytes())
			}
		})
	}
}
