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
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFatalErrorLogging verifies that the server logs errors and panics before returning.
func TestFatalErrorLogging(t *testing.T) {
	for _, test := range []struct {
		Name      string
		InitFn    witchcraft.InitFunc
		VerifyLog func(t *testing.T, logOutput []byte)
	}{
		{
			Name: "error returned by init function",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo) (cleanup func(), rErr error) {
				return nil, werror.Error("oops", werror.SafeParam("k", "v"))
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(logOutput, &log))
				assert.Equal(t, logging.LogLevelError, log.Level)
				assert.Equal(t, "oops", log.Message)
				assert.Equal(t, "v", log.Params["k"], "safe param not preserved")
				assert.NotEmpty(t, log.Stacktrace)
			},
		},
		{
			Name: "panic init function with error",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo) (cleanup func(), rErr error) {
				panic(werror.Error("oops", werror.SafeParam("k", "v")))
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(logOutput, &log))
				assert.Equal(t, logging.LogLevelError, log.Level)
				assert.Equal(t, "panic recovered", log.Message)
				assert.Equal(t, "v", log.Params["k"], "safe param not preserved")
				assert.NotEmpty(t, log.Stacktrace)
			},
		},
		{
			Name: "panic init function with object",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo) (cleanup func(), rErr error) {
				panic(map[string]interface{}{"k": "v"})
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(logOutput, &log))
				assert.Equal(t, logging.LogLevelError, log.Level)
				assert.Equal(t, "panic recovered", log.Message)
				assert.Equal(t, map[string]interface{}{"k": "v"}, log.UnsafeParams["recovered"])
				assert.NotEmpty(t, log.Stacktrace)
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			logOutputBuffer := &bytes.Buffer{}
			err := witchcraft.NewServer().
				WithInitFunc(test.InitFn).
				WithInstallConfig(config.Install{UseConsoleLog: true}).
				WithRuntimeConfig(config.Runtime{}).
				WithLoggerStdoutWriter(logOutputBuffer).
				WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
				WithDisableGoRuntimeMetrics().
				WithSelfSignedCertificate().
				Start()

			require.Error(t, err)
			test.VerifyLog(t, logOutputBuffer.Bytes())
		})
	}
}

func TestWhenPortInUse(t *testing.T) {

	// create a listener to ensure a port is taken
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = ln.Close()
	}()

	// try and start a Server on that port, expecting it to fail quickly with a non-nil error
	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)

	server := witchcraft.NewServer().WithSelfSignedCertificate().WithInstallConfig(config.Install{
		Server: config.Server{
			Address: host,
			Port: port,
		},
	}).WithRuntimeConfig(config.Install{})
	errc := make(chan error)
	go func() {
		errc <- server.Start()
	}()
	select {
	case serr := <-errc:
		assert.NotNil(t, serr, "start returned a nil error when its port was already in use")
	case <- time.Tick(2 * time.Second):
		t.Errorf("server stayed up despite its port already being in use")
	}
}
