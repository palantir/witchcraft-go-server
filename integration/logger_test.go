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

package integration

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/nmiyake/pkg/dirs"
	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-server/config"
	"github.com/palantir/witchcraft-go-server/rest"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/palantir/witchcraft-go-server/witchcraft/refreshable"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUseLoggerFileWriterProvider tests the behavior of sharing the output writer for a logger before and after a
// server is started. Tests the workflow where an external program creates a logger for a file at a specific path and
// expects a witchcraft.Server that is constructed later to use the same io.Writer for its logger (to ensure that the
// file properly handles writes from different sources).
func TestUseLoggerFileWriterProvider(t *testing.T) {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		err := os.Rename("/.dockerenv", "/.dockerenv-backup")
		require.NoError(t, err, "failed to rename /.dockerenv")
		defer func() {
			err := os.Rename("/.dockerenv-backup", "/.dockerenv")
			require.NoError(t, err, "failed to rename /.dockerenv")
		}()
	}

	testDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	// set working directory
	wd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		err := os.Chdir(wd)
		require.NoError(t, err)
	}()
	err = os.Chdir(testDir)
	require.NoError(t, err)

	cachingWriterProvider := witchcraft.NewCachingFileWriterProvider(witchcraft.DefaultFileWriterProvider())
	svc1Logger := svc1log.New(witchcraft.CreateLogWriter("var/log/service.log", false, os.Stdout, cachingWriterProvider), wlog.DebugLevel)
	svc1Logger.Info("Test output from before server start")

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	managementPort, err := httpserver.AvailablePort()
	require.NoError(t, err)

	server := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, initInfo witchcraft.InitInfo) (func(), error) {
			// register handler that returns "ok"
			err := initInfo.Router.Get("/ok", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rest.WriteJSONResponse(rw, "ok", http.StatusOK)
			}))
			if err != nil {
				return nil, err
			}
			return nil, nil
		}).
		WithInstallConfig(config.Install{
			ProductName: productName,
			Server: config.Server{
				Address:        "localhost",
				Port:           port,
				ManagementPort: managementPort,
				ContextPath:    basePath,
			},
		}).
		WithLoggerFileWriterProvider(cachingWriterProvider).
		WithRuntimeConfigProvider(refreshable.NewDefaultRefreshable([]byte{})).
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithDisableGoRuntimeMetrics().
		WithSelfSignedCertificate()

	serverChan := make(chan error)
	go func() {
		serverChan <- server.Start()
	}()

	success := <-waitForTestServerReady(port, "example/ok", 5*time.Second)
	if !success {
		errMsg := "timed out waiting for server to start"
		select {
		case err := <-serverChan:
			errMsg = fmt.Sprintf("%s: %+v", errMsg, err)
		default:
		}
		require.Fail(t, errMsg)
	}

	svc1Logger.Info("Test output from after server start")

	// verify service log output
	serviceLogBytes, err := ioutil.ReadFile(path.Join("var", "log", "service.log"))
	require.NoError(t, err)

	msgs := getLogFileMessages(t, serviceLogBytes)
	assert.Equal(t, []string{
		"Test output from before server start",
		"Listening to https",
		"Listening to https",
		"Test output from after server start",
	}, msgs)
}
