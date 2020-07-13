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

package status

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	whealth "github.com/palantir/witchcraft-go-server/status/health"
	"github.com/stretchr/testify/assert"
)

type expectedLog struct {
	level, msg string
}

func (l *expectedLog) matches(logOutput string) bool {
	return strings.Contains(logOutput, l.level) && strings.Contains(logOutput, l.msg)
}

func TestLoggingChangeHandler(t *testing.T) {
	for _, testCase := range []struct {
		name       string
		prev, curr health.HealthStatus
		expected   *expectedLog
	}{
		{
			name: "log error when new status code is greater than 200",
			prev: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST": whealth.HealthyHealthCheckResult("TEST"),
				},
			},
			curr: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST": whealth.UnhealthyHealthCheckResult("TEST", "message"),
				},
			},
			expected: &expectedLog{
				level: "ERROR",
				msg:   "Health status code changed.",
			},
		},
		{
			name: "log info when new status code is 200",
			prev: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST": whealth.UnhealthyHealthCheckResult("TEST", "message"),
				},
			},
			curr: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST": whealth.HealthyHealthCheckResult("TEST"),
				},
			},
			expected: &expectedLog{
				level: "INFO",
				msg:   "Health status code changed.",
			},
		},
		{
			name: "log when checks differ",
			prev: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST": whealth.HealthyHealthCheckResult("TEST"),
				},
			},
			curr: health.HealthStatus{
				Checks: map[health.CheckType]health.HealthCheckResult{
					"TEST_2": whealth.HealthyHealthCheckResult("TEST_2"),
				},
			},
			expected: &expectedLog{
				level: "INFO",
				msg:   "Health checks content changed without status change.",
			},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := svc1log.WithLogger(context.Background(), svc1log.New(&buf, wlog.DebugLevel))
			loggingHealthStatusChangeHandler().HandleHealthStatusChange(ctx, testCase.prev, testCase.curr)
			if testCase.expected == nil {
				assert.Empty(t, buf.String())
			} else {
				assert.True(t, testCase.expected.matches(buf.String()))
			}
		})
	}
}

func init() {
	wlog.SetDefaultLoggerProvider(wlog.NewJSONMarshalLoggerProvider())
}
