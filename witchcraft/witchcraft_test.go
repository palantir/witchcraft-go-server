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
	"os"
	"strconv"
	"syscall"
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
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
				return nil, werror.ErrorWithContextParams(ctx, "oops", werror.SafeParam("k", "v"))
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
				panic(werror.ErrorWithContextParams(ctx, "oops", werror.SafeParam("k", "v")))
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
				WithMetricsBlacklist(map[string]struct{}{"server.uptime": {}}).
				WithSelfSignedCertificate().
				Start()

			require.Error(t, err)
			test.VerifyLog(t, logOutputBuffer.Bytes())
		})
	}
}

func TestServer_Start_WithPortInUse(t *testing.T) {
	// create a listener to ensure a port is taken
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		_ = ln.Close()
	}()

	// try to start a server on the port that's in use
	host, portStr, err := net.SplitHostPort(ln.Addr().String())
	require.NoError(t, err)
	port, err := strconv.Atoi(portStr)
	require.NoError(t, err)
	server, cleanup := newServer(host, port)
	defer cleanup()
	errc := make(chan error)
	go func() {
		errc <- server.Start()
	}()

	// ensure Start() quickly returns a non-nil error
	select {
	case serr := <-errc:
		assert.NotNil(t, serr, "Start() returned a nil error when the server's port was already in use")
	case <-time.After(2 * time.Second):
		t.Errorf("server stayed up despite its port already being in use")
	}
}

func TestServer_Start_WithStop(t *testing.T) {
	// test the same behavior for both Shutdown() and Close() on a Server
	for name, stop := range map[string]func(*witchcraft.Server, context.Context) error{
		"Shutdown": (*witchcraft.Server).Shutdown,
		"Close": func(server *witchcraft.Server, _ context.Context) error {
			return server.Close()
		},
		"SIGTERM": func(server *witchcraft.Server, _ context.Context) error {
			proc, err := os.FindProcess(os.Getpid())
			require.NoError(t, err)
			return proc.Signal(syscall.SIGTERM)
		},
		"SIGINT": func(server *witchcraft.Server, _ context.Context) error {
			proc, err := os.FindProcess(os.Getpid())
			require.NoError(t, err)
			return proc.Signal(syscall.SIGINT)
		},
	} {
		t.Run(name, func(t *testing.T) {
			testStop(t, stop)
		})
	}
}

func testStop(t *testing.T, stop func(*witchcraft.Server, context.Context) error) {
	// create and start server
	server, cleanup := newServer("127.0.0.1", 0)
	defer cleanup()
	errc := make(chan error)
	go func() {
		errc <- server.Start()
	}()

	// wait for server to come up
	timeout := time.After(2 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		select {
		case <-timeout:
			assert.Fail(t, "timed out waiting for server to start")
		default:
		}
		if server.Running() {
			break
		}
	}

	// stop the server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	assert.NoError(t, stop(server, ctx), "error stopping server")

	// ensure Start() quickly returns a nil error
	select {
	case startErr := <-errc:
		assert.Nil(t, startErr, "Start() incorrectly returned a non-nil error when the server was intentionally stop")
	case <-time.After(2 * time.Second):
		assert.Fail(t, "timed out waiting for Start() to return an error")
	}
}

func newServer(host string, port int) (*witchcraft.Server, func()) {
	server := witchcraft.NewServer().
		WithSelfSignedCertificate().
		WithInstallConfig(config.Install{
			Server: config.Server{
				Address: host,
				Port:    port,
			},
			UseConsoleLog: true,
		}).
		WithRuntimeConfig(config.Runtime{})
	return server, func() { _ = server.Close() }
}
