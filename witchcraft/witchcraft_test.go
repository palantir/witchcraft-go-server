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
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFatalErrorLogging verifies that the server logs errors and panics before returning.
func TestFatalErrorLogging(t *testing.T) {
	for _, test := range []struct {
		Name      string
		InitFn    witchcraft.InitFunc[config.Install, config.Runtime]
		VerifyLog func(t *testing.T, logOutput []byte)
	}{
		{
			Name: "error returned by init function",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				return nil, werror.Error("oops", werror.SafeParam("k", "v"))
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				svc1LogLines := getLogMessagesOfType(t, "service.1", logOutput)
				require.Equal(t, 1, len(svc1LogLines), "Expected exactly 1 service log line to be output")
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_ERROR), log.Level)
				assert.Equal(t, "oops", log.Message)
				assert.Equal(t, "v", log.Params["k"], "safe param not preserved")
				assert.NotEmpty(t, log.Stacktrace)
			},
		},
		{
			Name: "panic init function with error",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				panic(werror.Error("oops", werror.SafeParam("k", "v")))
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				svc1LogLines := getLogMessagesOfType(t, "service.1", logOutput)
				require.Equal(t, 1, len(svc1LogLines), "Expected exactly 1 service log line to be output")
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_ERROR), log.Level)
				assert.Equal(t, "panic recovered", log.Message)
				assert.Equal(t, "v", log.Params["k"], "safe param not preserved")
				assert.NotEmpty(t, log.Stacktrace)
			},
		},
		{
			Name: "panic init function with object",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				panic(map[string]interface{}{"k": "v"})
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				svc1LogLines := getLogMessagesOfType(t, "service.1", logOutput)
				require.Equal(t, 1, len(svc1LogLines), "Expected exactly 1 service log line to be output")
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_ERROR), log.Level)
				assert.Equal(t, "panic recovered", log.Message)
				assert.Equal(t, map[string]interface{}{"k": "v"}, log.UnsafeParams["recovered"])
				assert.NotEmpty(t, log.Stacktrace)
			},
		},
		{
			Name: "fails with shutdown server inside init func",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
				return nil, info.ShutdownServer(ctx)
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				svc1LogLines := getLogMessagesOfType(t, "service.1", logOutput)
				require.Len(t, svc1LogLines, 2, "Expected exactly 2 service log line to be output")
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_INFO), log.Level)
				assert.Equal(t, "Shutting down server", log.Message)
				require.NoError(t, json.Unmarshal(svc1LogLines[1], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_ERROR), log.Level)
				assert.Equal(t, "server was shut down before it could start", log.Message)
			},
		},
		{
			Name: "fails with async shutdown server inside init func",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
				go func() {
					_ = info.ShutdownServer(ctx)
				}()
				return nil, nil
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				svc1LogLines := getLogMessagesOfType(t, "service.1", logOutput)
				require.Len(t, svc1LogLines, 2, "Expected exactly 2 service log line to be output")
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_INFO), log.Level)
				assert.Equal(t, "Shutting down server", log.Message)
				require.NoError(t, json.Unmarshal(svc1LogLines[1], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_ERROR), log.Level)
				assert.Equal(t, "server was shut down before it could start", log.Message)
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			logOutputBuffer := &bytes.Buffer{}
			err := witchcraft.NewServer[config.Install, config.Runtime]().
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

// TestServer_StartFailsBeforeMetricRegistryInitialized creates a Server with no install config, which will cause the Start function to fail,
// before the metrics registry has been initialized. Logging should not panic in this case.
func TestServer_StartFailsBeforeMetricRegistryInitialized(t *testing.T) {
	// Missing install config
	err := witchcraft.NewServer[config.Install, config.Runtime]().Start()
	require.Error(t, err)
	require.Contains(t, err.Error(), "Failed to load install configuration bytes")
}

func TestServer_WithOriginFromCallLine(t *testing.T) {
	productName, productVersion := "productName", "1.0.0"
	for _, test := range []struct {
		Name      string
		Install   config.Install
		InitFn    witchcraft.InitFunc[config.Install, config.Runtime]
		VerifyLog func(t *testing.T, logOutput []byte)
	}{
		{
			Name:    "svc log in init function",
			Install: config.Install{UseConsoleLog: true},
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				svc1log.FromContext(ctx).Info("a message", svc1log.SafeParam("k", "v"))
				return nil, werror.Error("oops", werror.SafeParam("k", "v"))
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				_, file, line, _ := runtime.Caller(0)
				file = path.Base(file) // janky way to trim gopath
				line = line - 4        // janky way to refer to log line above
				originSuffix := fmt.Sprintf("%s:%d", file, line)

				svc1LogLines := getLogMessagesOfType(t, "service.1", logOutput)
				require.Equal(t, 2, len(svc1LogLines), "Expected exactly 2 service log line to be output")

				var log1 logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log1))
				assert.Equal(t, "a message", log1.Message)
				assert.True(t, strings.HasSuffix(*log1.Origin, originSuffix), "Expected origin %s to have suffix %s", *log1.Origin, originSuffix)
				assert.Equal(t, "v", log1.Params["k"], "safe param not preserved")

				var log2 logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[1], &log2))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_ERROR), log2.Level)
				assert.Equal(t, "oops", log2.Message)
				assert.Equal(t, "v", log2.Params["k"], "safe param not preserved")
				assert.NotEmpty(t, log2.Stacktrace)
				assert.Equal(t, "witchcraft.go", path.Base(strings.Split(*log2.Origin, ":")[0]), "Unexpected origin %s", *log2.Origin)
			},
		},
		{
			Name:    "wrapped svc log in init function",
			Install: config.Install{ProductName: productName, ProductVersion: productVersion, UseWrappedLogs: true, UseConsoleLog: true},
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				svc1log.FromContext(ctx).Info("a message", svc1log.SafeParam("k", "v"))
				return nil, werror.Error("oops", werror.SafeParam("k", "v"))
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				_, file, line, _ := runtime.Caller(0)
				file = path.Base(file) // janky way to trim gopath
				line = line - 4        // janky way to refer to log line above
				originSuffix := fmt.Sprintf("%s:%d", file, line)

				svc1LogLines := getWrappedLogMessagesOfType(t, productName, productVersion, "service.1", logOutput)
				require.Equal(t, 2, len(svc1LogLines), "Expected exactly 2 service log line to be output")

				var log1 logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log1))
				assert.Equal(t, "a message", log1.Message)
				assert.True(t, strings.HasSuffix(*log1.Origin, originSuffix), "Expected origin %s to have suffix %s", *log1.Origin, originSuffix)
				assert.Equal(t, "v", log1.Params["k"], "safe param not preserved")

				var log2 logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[1], &log2))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_ERROR), log2.Level)
				assert.Equal(t, "oops", log2.Message)
				assert.Equal(t, "v", log2.Params["k"], "safe param not preserved")
				assert.NotEmpty(t, log2.Stacktrace)
				assert.Equal(t, "witchcraft.go", path.Base(strings.Split(*log2.Origin, ":")[0]), "Unexpected origin %s", *log2.Origin)
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			logOutputBuffer := &bytes.Buffer{}
			err := witchcraft.NewServer[config.Install, config.Runtime]().
				WithOriginFromCallLine().
				WithInitFunc(test.InitFn).
				WithInstallConfig(test.Install).
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

// BenchmarkServer_Loggers benchmarks the time for Server loggers to log a fixed number of lines.
func BenchmarkServer_Loggers(b *testing.B) {
	for _, test := range []struct {
		Name        string
		LoggingBody func(context.Context)
	}{
		{
			Name: "svc1log",
			LoggingBody: func(ctx context.Context) {
				svc1log.FromContext(ctx).Info("info!")
			},
		},
		{
			Name: "evt2log",
			LoggingBody: func(ctx context.Context) {
				evt2log.FromContext(ctx).Event("event!")
			},
		},
		{
			Name: "metric1log",
			LoggingBody: func(ctx context.Context) {
				metric1log.FromContext(ctx).Metric("metric!", metric1log.MetricTypeKey)
			},
		},
		{
			Name: "trc1log",
			LoggingBody: func(ctx context.Context) {
				trc1log.FromContext(ctx).Log(wtracing.SpanModel{})
			},
		},
		{
			Name: "audit2log",
			LoggingBody: func(ctx context.Context) {
				audit2log.FromContext(ctx).Audit("audit!", audit2log.AuditResultSuccess)
			},
		},
	} {
		b.Run(test.Name, func(b *testing.B) {
			err := witchcraft.NewServer[config.Install, config.Runtime]().
				WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						test.LoggingBody(ctx)
					}
					b.StopTimer()
					return nil, werror.ErrorWithContextParams(ctx, "must error to get Start to return!")
				}).
				WithInstallConfig(config.Install{UseConsoleLog: true}).
				WithRuntimeConfig(config.Runtime{}).
				WithLoggerStdoutWriter(io.Discard).
				WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
				WithDisableGoRuntimeMetrics().
				WithSelfSignedCertificate().
				Start()

			// Requires an error so that `Start` will return
			require.Error(b, err)
		})
	}
}

// TestServer_WithWrappedLoggers creates a server with wrapped loggers and validates wrapped payloads of each log type.
func TestServer_WithWrappedLoggers(t *testing.T) {
	productName, productVersion := "productName", "1.0.0"
	for _, test := range []struct {
		Name      string
		InitFn    witchcraft.InitFunc[config.Install, config.Runtime]
		VerifyLog func(t *testing.T, logOutput []byte)
	}{
		{
			Name: "svc1log",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				svc1log.FromContext(ctx).Info("info!")
				return nil, werror.ErrorWithContextParams(ctx, "must error to get Start to return!")
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				svc1LogLines := getWrappedLogMessagesOfType(t, productName, productVersion, "service.1", logOutput)
				// An extra service log line is output when the server fails to start
				require.Equal(t, 2, len(svc1LogLines), "Expected exactly 2 service log lines to be output")
				var log logging.ServiceLogV1
				require.NoError(t, json.Unmarshal(svc1LogLines[0], &log))
				assert.Equal(t, logging.New_LogLevel(logging.LogLevel_INFO), log.Level)
				assert.Equal(t, "info!", log.Message)
			},
		},
		{
			Name: "evt2log",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				evt2log.FromContext(ctx).Event("info!")
				return nil, werror.ErrorWithContextParams(ctx, "must error to get Start to return!")
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				evt2LogLines := getWrappedLogMessagesOfType(t, productName, productVersion, "event.2", logOutput)
				require.Equal(t, 1, len(evt2LogLines), "Expected exactly 1 event log line to be output")
				var log logging.EventLogV2
				require.NoError(t, json.Unmarshal(evt2LogLines[0], &log))
				assert.Equal(t, "info!", log.EventName)
			},
		},
		{
			Name: "metric1log",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				metric1log.FromContext(ctx).Metric("metric!", metric1log.MetricTypeKey)
				return nil, werror.ErrorWithContextParams(ctx, "must error to get Start to return!")
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				metric1LogLines := getWrappedLogMessagesOfType(t, productName, productVersion, "metric.1", logOutput)
				require.Equal(t, 1, len(metric1LogLines), "Expected exactly 1 metric log line to be output")
				var log logging.MetricLogV1
				require.NoError(t, json.Unmarshal(metric1LogLines[0], &log))
				assert.Equal(t, "metric!", log.MetricName)
				assert.Equal(t, metric1log.MetricTypeKey, log.MetricType)
			},
		},
		{
			Name: "trc1log",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				trc1log.FromContext(ctx).Log(wtracing.SpanModel{Name: "trace!"})
				return nil, werror.ErrorWithContextParams(ctx, "must error to get Start to return!")
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				trc1LogLines := getWrappedLogMessagesOfType(t, productName, productVersion, "trace.1", logOutput)
				require.Equal(t, 1, len(trc1LogLines), "Expected exactly 1 trace log line to be output")
				var log logging.TraceLogV1
				require.NoError(t, json.Unmarshal(trc1LogLines[0], &log))
				assert.Equal(t, "trace!", log.Span.Name)
			},
		},
		{
			Name: "audit2log",
			InitFn: func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (cleanup func(), rErr error) {
				audit2log.FromContext(ctx).Audit("audit!", audit2log.AuditResultSuccess)
				return nil, werror.ErrorWithContextParams(ctx, "must error to get Start to return!")
			},
			VerifyLog: func(t *testing.T, logOutput []byte) {
				audit2LogLines := getWrappedLogMessagesOfType(t, productName, productVersion, "audit.2", logOutput)
				require.Equal(t, 1, len(audit2LogLines), "Expected exactly 1 audit log line to be output")
				var log logging.AuditLogV2
				require.NoError(t, json.Unmarshal(audit2LogLines[0], &log))
				assert.Equal(t, "audit!", log.Name)
				assert.Equal(t, logging.AuditResult_SUCCESS, log.Result.Value())
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			logOutputBuffer := &bytes.Buffer{}
			err := witchcraft.NewServer[config.Install, config.Runtime]().
				WithInitFunc(test.InitFn).
				WithInstallConfig(config.Install{UseConsoleLog: true, UseWrappedLogs: true, ProductName: productName, ProductVersion: productVersion}).
				WithRuntimeConfig(config.Runtime{}).
				WithLoggerStdoutWriter(logOutputBuffer).
				WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
				WithDisableGoRuntimeMetrics().
				WithMetricsBlacklist(map[string]struct{}{"server.uptime": {}, "logging.sls": {}, "logging.sls.length": {}}).
				WithSelfSignedCertificate().
				Start()

			require.Error(t, err)
			test.VerifyLog(t, logOutputBuffer.Bytes())

			// Requires an error so that `Start` will return
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
	for name, stop := range map[string]func(*witchcraft.Server[config.Install, config.Runtime], context.Context) error{
		"Shutdown": (*witchcraft.Server[config.Install, config.Runtime]).Shutdown,
		"Close": func(server *witchcraft.Server[config.Install, config.Runtime], _ context.Context) error {
			return server.Close()
		},
		"SIGTERM": func(server *witchcraft.Server[config.Install, config.Runtime], _ context.Context) error {
			proc, err := os.FindProcess(os.Getpid())
			require.NoError(t, err)
			return proc.Signal(syscall.SIGTERM)
		},
		"SIGINT": func(server *witchcraft.Server[config.Install, config.Runtime], _ context.Context) error {
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

func testStop(t *testing.T, stop func(*witchcraft.Server[config.Install, config.Runtime], context.Context) error) {
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

func newServer(host string, port int) (*witchcraft.Server[config.Install, config.Runtime], func()) {
	server := witchcraft.NewServer[config.Install, config.Runtime]().
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

// getLogMessagesOfType returns a slice of the content of all of the log entries that have a "type" field that match the
// provided value.
func getLogMessagesOfType(t *testing.T, typ string, logOutput []byte) [][]byte {
	lines := bytes.Split(logOutput, []byte("\n"))
	var logLines [][]byte
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var currEntry map[string]interface{}
		assert.NoError(t, json.Unmarshal(line, &currEntry), "failed to parse json line %q", string(line))
		if logLineType, ok := currEntry["type"]; ok && logLineType == typ {
			logLines = append(logLines, line)
		}
	}
	return logLines
}

// getWrappedLogMessagesOfType validates that the logOutput is in wrapped.1 format, has the right entityName and
// entityVersion, and returns a slice of the content of all wrapped payloads of the given "type" that match the
// provided value
func getWrappedLogMessagesOfType(t *testing.T, entityName, entityVersion, typ string, logOutput []byte) [][]byte {
	lines := bytes.Split(logOutput, []byte("\n"))
	var logLines [][]byte
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var currEntry map[string]interface{}
		assert.NoError(t, json.Unmarshal(line, &currEntry), "failed to parse json line %q", string(line))
		assert.Equal(t, "wrapped.1", currEntry["type"])
		assert.Equal(t, entityName, currEntry["entityName"])
		assert.Equal(t, entityVersion, currEntry["entityVersion"])
		if payload, ok := currEntry["payload"].(map[string]interface{}); ok {
			if payloadLog, ok := payload[payload["type"].(string)].(map[string]interface{}); ok {
				if payloadLogLineType, ok := payloadLog["type"]; ok && payloadLogLineType == typ {
					if payloadBytes, err := json.Marshal(payloadLog); err == nil {
						logLines = append(logLines, payloadBytes)
					}
				}
			}
		}
	}
	return logLines
}
