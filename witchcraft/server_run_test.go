// Copyright (c) 2022 Palantir Technologies. All rights reserved.
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

package witchcraft

import (
	"os"
	"testing"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/stretchr/testify/assert"
)

func TestIsLoopback(t *testing.T) {
	svcLogger := svc1log.New(os.Stdout, wlog.DebugLevel)
	for _, test := range []struct {
		name             string
		addr             string
		expectedLoopback bool
	}{
		{
			name:             "localhost is loopback",
			addr:             "localhost",
			expectedLoopback: true,
		},
		{
			name:             "127.0.0.1 is loopback",
			addr:             "127.0.0.1",
			expectedLoopback: true,
		},
		{
			name:             "google.com is not loopback",
			addr:             "google.com",
			expectedLoopback: false,
		},
		{
			name:             "8.8.8.8 is not loopback",
			addr:             "8.8.8.8",
			expectedLoopback: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ok := isLoopback(test.addr, svcLogger)
			assert.Equal(t, test.expectedLoopback, ok)
		})
	}
}
