// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

package witchcraft_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/conjure/sls/spec/logging"
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFatalErrorLogging verifies that the server logs errors before returning.
func TestFatalErrorLogging(t *testing.T) {
	logOutputBuffer := &bytes.Buffer{}
	err := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (cleanup func(), rErr error) {
			return nil, werror.Error("oops")
		}).
		WithInstallConfig(config.Install{UseConsoleLog: true}).
		WithRuntimeConfig(config.Runtime{}).
		WithLoggerStdoutWriter(logOutputBuffer).
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithDisableGoRuntimeMetrics().
		WithSelfSignedCertificate().
		Start()

	require.EqualError(t, err, "oops")

	var log logging.ServiceLogV1
	require.NoError(t, json.Unmarshal(logOutputBuffer.Bytes(), &log))
	assert.Equal(t, logging.LogLevelError, log.Level)
	assert.Equal(t, "oops", log.Message)
	assert.NotEmpty(t, log.Stacktrace)
}
