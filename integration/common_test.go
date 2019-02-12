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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/nmiyake/pkg/dirs"
	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/rest"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/palantir/witchcraft-go-server/witchcraft/refreshable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const (
	basePath    = "/example"
	productName = "example"
	installYML  = "var/conf/install.yml"
	runtimeYML  = "var/conf/runtime.yml"
)

// createAndRunTestServer returns a running witchcraft.Server that is initialized with simple default configuration in a
// temporary directory. Returns the server, the path to the temporary directory, a channel that returns the error
// returned by the server when it stops and a cleanup function that will remove the temporary directory.
func createAndRunTestServer(t *testing.T, initFn witchcraft.InitFunc, logOutputBuffer io.Writer) (server *witchcraft.Server, port int, managementPort int, serverErr <-chan error, cleanup func()) {
	var err error
	port, err = httpserver.AvailablePort()
	require.NoError(t, err)
	managementPort, err = httpserver.AvailablePort()
	require.NoError(t, err)

	server, serverErr, cleanup = createAndRunCustomTestServer(t, port, managementPort, initFn, logOutputBuffer, createTestServer)
	return server, port, managementPort, serverErr, cleanup
}

type serverCreatorFn func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server

func createAndRunCustomTestServer(t *testing.T, port, managementPort int, initFn witchcraft.InitFunc, logOutputBuffer io.Writer, createServer serverCreatorFn) (server *witchcraft.Server, serverErr <-chan error, cleanup func()) {
	installCfg := config.Install{
		ProductName:   productName,
		UseConsoleLog: true,
		Server: config.Server{
			Address:        "localhost",
			Port:           port,
			ManagementPort: managementPort,
			ContextPath:    basePath,
		},
	}

	var err error
	var dir string
	dir, cleanup, err = dirs.TempDir("", "test-server")
	success := false
	defer func() {
		if !success {
			// invoke cleanup function only if this function exits unsuccessfully. Otherwise, the cleanup
			// function is returned and the caller is expected to defer it.
			cleanup()
		}
	}()
	require.NoError(t, err)

	// restore working directory at the end of the function
	restoreWd, err := dirs.SetwdWithRestorer(dir)
	require.NoError(t, err)
	defer restoreWd()

	err = os.MkdirAll("var/conf", 0755)
	require.NoError(t, err)
	installCfgYML, err := yaml.Marshal(installCfg)
	require.NoError(t, err)
	err = ioutil.WriteFile(installYML, installCfgYML, 0644)
	require.NoError(t, err)
	err = ioutil.WriteFile(runtimeYML, []byte(`number: 1`), 0644)
	require.NoError(t, err)

	server = createServer(t, initFn, installCfg, logOutputBuffer)

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()
	serverErr = serverChan
	success = <-waitForTestServerReady(port, "example/ok", 5*time.Second)
	if !success {
		errMsg := "timed out waiting for server to start"
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		default:
		}
		require.Fail(t, errMsg)
	}
	return server, serverErr, cleanup
}

// createTestServer creates a test *witchcraft.Server that has been constructed but not started. The server has the
// context path "/example" and has a handler for an "/ok" method that returns the JSON "ok" on GET calls. Returns the
// server and the port that the server will use when started.
func createTestServer(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) (server *witchcraft.Server) {
	server = witchcraft.
		NewServer().
		WithInitFunc(func(ctx context.Context, initInfo witchcraft.InitInfo) (func(), error) {
			// register handler that returns "ok"
			err := initInfo.Router.Get("/ok", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rest.WriteJSONResponse(rw, "ok", http.StatusOK)
			}))
			if err != nil {
				return nil, err
			}
			if initFn != nil {
				return initFn(ctx, initInfo)
			}
			return nil, nil
		}).
		WithInstallConfig(installCfg).
		WithRuntimeConfigProvider(refreshable.NewDefaultRefreshable([]byte{})).
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithDisableGoRuntimeMetrics().
		WithSelfSignedCertificate()
	if logOutputBuffer != nil {
		server.WithLoggerStdoutWriter(logOutputBuffer)
	}
	return server
}

// waitForTestServerReady returns a channel that returns true when a test server is ready on the provided port. Returns
// false if the server is not ready within the provided timeout duration.
func waitForTestServerReady(port int, path string, timeout time.Duration) <-chan bool {
	return httpserver.Ready(func() (*http.Response, error) {
		resp, err := testServerClient().Get(fmt.Sprintf("https://localhost:%d/%s", port, path))
		return resp, err
	},
		httpserver.WaitTimeoutParam(timeout),
	)
}

func testServerClient() *http.Client {
	return &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
}

// getLogFileMessages returns a slice of the content of all of the "msg" fields in the log entries in the given log
// file.
func getLogFileMessages(t *testing.T, logOutput []byte) []string {
	lines := bytes.Split(logOutput, []byte("\n"))
	var messages []string
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var currEntry map[string]interface{}
		assert.NoError(t, json.Unmarshal(line, &currEntry), "failed to parse json line %q", string(line))
		if msg, ok := currEntry["message"]; ok {
			messages = append(messages, msg.(string))
		}
	}
	return messages
}
