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

package integration

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nmiyake/pkg/dirs"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/pkg/tlsconfig"
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestServerStarts ensures that a Witchcraft server starts and is able to serve requests.
// It also verifies the service log output of starting two routers (application and management).
func TestServerStarts(t *testing.T) {
	logOutputBuffer := &bytes.Buffer{}
	server, _, _, _, cleanup := createAndRunTestServer(t, nil, logOutputBuffer)
	defer func() {
		_ = server.Close()
	}()
	defer cleanup()

	// verify service log output
	msgs := getLogFileMessages(t, logOutputBuffer.Bytes())
	assert.Equal(t, []string{"Listening to https", "Listening to https"}, msgs)
}

// TestServerShutdown verifies the behavior when shutting down a Witchcraft server. There are two variants, graceful and abrupt.
// Graceful: server.Shutdown, which allows in-flight requests to complete. We test this using a route which sleeps one second.
// Abrupt: server.Close, which terminates the server immediately and sends EOF on active connections.
func TestServerShutdown(t *testing.T) {
	runTest := func(t *testing.T, graceful bool) {
		logOutputBuffer := &bytes.Buffer{}
		calledC := make(chan bool, 1)
		doneC := make(chan bool, 1)
		initFn := func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			return nil, info.Router.Get("/wait", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				calledC <- true
				// just wait for 1 second to hold the connection open.
				time.Sleep(1 * time.Second)
				doneC <- true
			}))
		}
		server, port, _, serverErr, cleanup := createAndRunTestServer(t, initFn, logOutputBuffer)
		defer func() {
			_ = server.Close()
		}()
		defer cleanup()

		go func() {
			resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/example/wait", port))
			if graceful {
				if err != nil {
					panic(fmt.Errorf("graceful shutdown request failed: %v", err))
				}
				if resp == nil {
					panic(fmt.Errorf("returned response was nil"))
				}
				if resp.StatusCode != 200 {
					panic(fmt.Errorf("server did not respond with 200 OK. Got %s", resp.Status))
				}
			} else {
				if err == nil {
					panic(fmt.Errorf("request allowed to finish successfully"))
				}
			}
		}()

		timeoutCtx, timeoutCancel := context.WithTimeout(context.Background(), time.Second)
		defer timeoutCancel()

		select {
		case <-calledC:
		case <-timeoutCtx.Done():
			require.NoError(t, timeoutCtx.Err(), "timed out waiting for called")
		}

		select {
		case done := <-doneC:
			require.False(t, done, "Connection was already closed!")
		default:
		}

		if graceful {
			// Gracefully shut down server. This should block until the handler has completed.
			require.NoError(t, server.Shutdown(context.Background()))
		} else {
			// Abruptly close server. This will send EOF on open connections.
			require.NoError(t, server.Close())
		}

		var done bool
		select {
		case done = <-doneC:
		default:
		}

		if graceful {
			require.True(t, done, "Handler didn't execute the whole way!")

			// verify service log output
			msgs := getLogFileMessages(t, logOutputBuffer.Bytes())
			assert.Equal(t, []string{"Listening to https", "Listening to https", "Shutting down server", "example was closed", "example-management was closed"}, msgs)
		} else {
			require.False(t, done, "Handler allowed to execute the whole way")
		}

		select {
		case err := <-serverErr:
			require.NoError(t, err)
		default:
		}
	}

	t.Run("graceful shutdown", func(t *testing.T) {
		runTest(t, true)
	})
	t.Run("abrupt shutdown", func(t *testing.T) {
		runTest(t, false)
	})
}

// TestEmptyPathHandler verifies that a route registered at the default path ("/") is served correctly.
func TestEmptyPathHandler(t *testing.T) {
	logOutputBuffer := &bytes.Buffer{}
	var called bool
	server, port, _, serverErr, cleanup := createAndRunTestServer(t, func(ctx context.Context, info witchcraft.InitInfo) (deferFn func(), rErr error) {
		return nil, info.Router.Get("/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			called = true
		}))
	}, logOutputBuffer)
	defer func() {
		_ = server.Close()
	}()
	defer cleanup()

	resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s", port, basePath))
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, "200 OK", resp.Status)
	assert.True(t, called, "called boolean was not set to true (http handler did not execute)")

	// verify service log output
	msgs := getLogFileMessages(t, logOutputBuffer.Bytes())
	assert.Equal(t, []string{"Listening to https", "Listening to https"}, msgs)

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestManagementRoutes verifies the behavior of /liveness, /readiness, and /health endpoints in their default configuration.
// There are two variants - one with a dedicated management port/router and one using the application port/router.
func TestManagementRoutes(t *testing.T) {
	runTests := func(t *testing.T, mgmtPort int) {
		t.Run("Liveness", func(t *testing.T) {
			resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/example/%s", mgmtPort, status.LivenessEndpoint))
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, "200 OK", resp.Status)
		})
		t.Run("Readiness", func(t *testing.T) {
			resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/example/%s", mgmtPort, status.ReadinessEndpoint))
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, "200 OK", resp.Status)
		})
		t.Run("Health", func(t *testing.T) {
			resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/example/%s", mgmtPort, status.HealthEndpoint))
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Equal(t, "200 OK", resp.Status)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
			var healthResp health.HealthStatus
			require.NoError(t, json.Unmarshal(body, &healthResp))
			require.Equal(t, health.HealthStatus{Checks: map[health.CheckType]health.HealthCheckResult{
				"CONFIG_RELOAD": {Type: "CONFIG_RELOAD", State: health.HealthStateHealthy, Params: map[string]interface{}{}},
				"SERVER_STATUS": {Type: "SERVER_STATUS", State: health.HealthStateHealthy, Params: map[string]interface{}{}},
			}}, healthResp)
		})
	}

	t.Run("dedicated port", func(t *testing.T) {
		server, _, managementPort, serverErr, cleanup := createAndRunTestServer(t, nil, ioutil.Discard)
		defer func() {
			_ = server.Close()
		}()
		defer cleanup()

		runTests(t, managementPort)

		select {
		case err := <-serverErr:
			require.NoError(t, err)
		default:
		}
	})

	t.Run("same port", func(t *testing.T) {
		port, err := httpserver.AvailablePort()
		require.NoError(t, err)
		server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, ioutil.Discard, createTestServer)

		defer func() {
			_ = server.Close()
		}()
		defer cleanup()

		runTests(t, port)

		select {
		case err := <-serverErr:
			require.NoError(t, err)
		default:
		}
	})
}

// TestClientTLS verifies that a Witchcraft server configured to require client TLS authentication enforces that config.
func TestClientTLS(t *testing.T) {
	testDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()

	err = os.Chdir(testDir)
	require.NoError(t, err)

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	managementPort, err := httpserver.AvailablePort()
	require.NoError(t, err)

	server := witchcraft.NewServer().
		WithClientAuth(tls.RequireAndVerifyClientCert).
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithRuntimeConfig(struct{}{}).
		WithInstallConfig(config.Install{
			ProductName:   productName,
			UseConsoleLog: true,
			Server: config.Server{
				Address:        "localhost",
				Port:           port,
				ManagementPort: managementPort,
				ContextPath:    basePath,
				CertFile:       path.Join(wd, "testdata/server-cert.pem"),
				KeyFile:        path.Join(wd, "testdata/server-key.pem"),
				ClientCAFiles:  []string{path.Join(wd, "testdata/ca-cert.pem")},
			},
		})

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()

	select {
	case err := <-serverChan:
		require.NoError(t, err)
	default:
	}

	// Management port should not require client certs.
	ready := <-waitForTestServerReady(managementPort, path.Join(basePath, status.LivenessEndpoint), 5*time.Second)
	if !ready {
		errMsg := "timed out waiting for server to start. This could mean management server incorrectly requested client certs."
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		}
		require.Fail(t, errMsg)
	}
	require.True(t, ready)

	defer func() {
		require.NoError(t, server.Close())
	}()

	// Assert regular client receives error
	_, err = testServerClient().Get(fmt.Sprintf("https://localhost:%d/example", port))
	require.Error(t, err, "Client allowed to make request without cert")
	assert.Contains(t, err.Error(), "tls: bad certificate")

	// Assert client w/ certs does not receive error
	tlsConf, err := tlsconfig.NewClientConfig(tlsconfig.ClientRootCAFiles(path.Join(wd, "testdata/ca-cert.pem")), tlsconfig.ClientKeyPairFiles(path.Join(wd, "testdata/client-cert.pem"), path.Join(wd, "testdata/client-key.pem")))
	require.NoError(t, err)
	_, err = (&http.Client{Transport: &http.Transport{TLSClientConfig: tlsConf}}).Get(fmt.Sprintf("https://localhost:%d/example", port))
	require.NoError(t, err)
}

func TestDefaultNotFoundHandler(t *testing.T) {
	logOutputBuffer := &bytes.Buffer{}
	server, port, _, serverErr, cleanup := createAndRunTestServer(t, func(ctx context.Context, info witchcraft.InitInfo) (deferFn func(), rErr error) {
		return nil, info.Router.Get("/foo", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(200)
		}))
	}, logOutputBuffer)
	defer func() {
		_ = server.Close()
	}()
	defer cleanup()

	const testTraceID = "1000000000000001"
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("https://localhost:%d/TestDefaultNotFoundHandler", port), nil)
	require.NoError(t, err)
	req.Header.Set("X-B3-TraceId", testTraceID)

	resp, err := testServerClient().Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "404 Not Found", resp.Status)

	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	cerr, err := errors.UnmarshalError(body)
	if assert.NoError(t, err) {
		assert.Equal(t, errors.NotFound, cerr.Code())
	}

	t.Run("request log", func(t *testing.T) {
		// find request log for 404, assert trace ID matches request
		reqlogs := getLogMessagesOfType(t, "request.2", logOutputBuffer.Bytes())
		var notFoundReqLogs []map[string]interface{}
		for _, reqlog := range reqlogs {
			if reqlog["traceId"] == testTraceID {
				notFoundReqLogs = append(notFoundReqLogs, reqlog)
			}
		}
		if assert.Len(t, notFoundReqLogs, 1, "expected exactly one request log with trace id") {
			reqlog := notFoundReqLogs[0]
			assert.Equal(t, 404.0, reqlog["status"])
			assert.Equal(t, "POST", reqlog["method"])
			assert.Equal(t, "/*", reqlog["path"])
		}
	})

	t.Run("service log", func(t *testing.T) {
		// find service log for 404, assert trace ID matches request
		svclogs := getLogMessagesOfType(t, "service.1", logOutputBuffer.Bytes())
		var notFoundSvcLogs []map[string]interface{}
		for _, svclog := range svclogs {
			if svclog["traceId"] == testTraceID {
				notFoundSvcLogs = append(notFoundSvcLogs, svclog)
			}
		}
		if assert.Len(t, notFoundSvcLogs, 1, "expected exactly one service log with trace id") {
			svclog := notFoundSvcLogs[0]
			assert.Equal(t, "INFO", svclog["level"])
			assert.Equal(t, fmt.Sprintf("error handling request: %v", cerr.Error()), svclog["message"])
			assert.Equal(t, map[string]interface{}{"errorInstanceId": cerr.InstanceID().String()}, svclog["params"])
		}
	})

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}
