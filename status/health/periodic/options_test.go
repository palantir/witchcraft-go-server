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

package periodic

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/stretchr/testify/assert"
)

func TestWithInitialPoll(t *testing.T) {
	var pollAlwaysErr = func() error {
		return fmt.Errorf("error")
	}
	periodicCheckWithInitialPoll := NewHealthCheckSource(
		context.Background(),
		time.Minute,
		time.Second,
		"CHECK_TYPE",
		pollAlwaysErr,
		WithInitialPoll())
	<-time.After(time.Second)
	healthStatus := periodicCheckWithInitialPoll.HealthStatus(context.Background())
	check, ok := healthStatus.Checks["CHECK_TYPE"]
	assert.True(t, ok)
	assert.Equal(t, health.HealthStateError, check.State)
}
