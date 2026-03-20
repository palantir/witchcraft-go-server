// Copyright (c) 2023 Palantir Technologies. All rights reserved.
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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/witchcraft-go-server/v3/config"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft/wdebug"
	"github.com/stretchr/testify/require"
)

const (
	testDiagnosticType = "go.goroutines.v1"
	testSecret         = "secretForTest"
	tmpDir             = "/tmp"
	tmpFilePath        = "/tmp/tmpFile"
)

type testDiagnosticHandler struct {
	diagnosticType wdebug.DiagnosticType
}

func (h testDiagnosticHandler) Type() wdebug.DiagnosticType                          { return h.diagnosticType }
func (h testDiagnosticHandler) Documentation() string                                 { return "test diagnostic" }
func (h testDiagnosticHandler) ContentType() string                                   { return "text/plain" }
func (h testDiagnosticHandler) SafeLoggable() bool                                    { return true }
func (h testDiagnosticHandler) Extension() string                                     { return "txt" }
func (h testDiagnosticHandler) WriteDiagnostic(_ context.Context, w io.Writer) error {
	_, err := w.Write([]byte("ok"))
	return err
}

func TestServer_DiagnosticsSharedSecret(t *testing.T) {
	tests := []struct {
		name          string
		secret        string
		runtimeConfig config.Runtime
		prepare       func()
		cleanup       func()
	}{
		{
			name:          "no secret specified in runtime config",
			secret:        "any secret should work",
			runtimeConfig: config.Runtime{},
		},
		{
			name:   "debug-shared-secret is specified in runtime config",
			secret: testSecret,
			runtimeConfig: config.Runtime{
				DiagnosticsConfig: config.DiagnosticsConfig{DebugSharedSecret: testSecret},
			},
		},
		{
			name:   "debug-shared-secret-file is specified in runtime config",
			secret: testSecret,
			runtimeConfig: config.Runtime{
				DiagnosticsConfig: config.DiagnosticsConfig{DebugSharedSecretFile: tmpFilePath},
			},
			prepare: func() {
				err := os.MkdirAll(tmpDir, 0755)
				require.NoError(t, err)
				err = os.WriteFile(tmpFilePath, []byte(testSecret), 0644)
				require.NoError(t, err)
			},
			cleanup: func() {
				err := os.Remove(tmpFilePath)
				require.NoError(t, err)
			},
		},
		{
			name:   "both debug-shared-secret and debug-shared-secret-file is specified in runtime config",
			secret: testSecret,
			runtimeConfig: config.Runtime{
				DiagnosticsConfig: config.DiagnosticsConfig{
					DebugSharedSecret: testSecret, DebugSharedSecretFile: filepath.Join(tmpDir, "fileDoesntExist"),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.prepare != nil {
				tt.prepare()
				defer tt.cleanup()
			}

			port, err := httpserver.AvailablePort()
			require.NoError(t, err)
			server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, nil, io.Discard,
				func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
					return createTestServer(t, initFn, installCfg, logOutputBuffer).
						WithRuntimeConfig(tt.runtimeConfig)
				})

			defer func() {
				require.NoError(t, server.Close())
			}()
			defer cleanup()
			client := testServerClient()

			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/debug/diagnostic/%s", port, basePath, testDiagnosticType), nil)
			require.NoError(t, err)
			request.Header.Set("Authorization", "Bearer "+tt.secret)
			resp, err := client.Do(request)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			select {
			case err := <-serverErr:
				require.NoError(t, err)
			default:
			}
		})
	}
}

// TestServer_DiagnosticRegisteredInInitGoroutineNotFound proves that diagnostic handlers
// registered via InitInfo.Router.WithCustomDiagnosticHandlers inside a goroutine (simulating
// an async initialization) are not available on the server. The handler registered on the
// builder before Start is served, but the one registered asynchronously in the init function
// returns 400 because addRoutes snapshots the handlers before the goroutine runs.
func TestServer_DiagnosticRegisteredInInitGoroutineNotFound(t *testing.T) {
	builderHandler := testDiagnosticHandler{diagnosticType: "custom.builder.v1"}
	initInfoHandler := testDiagnosticHandler{diagnosticType: "custom.initinfo.v1"}
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port,
		func(_ context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
			go func() {
				time.Sleep(1 * time.Second)
				info.Router.WithCustomDiagnosticHandlers(initInfoHandler)
			}()
			return nil, nil
		},
		io.Discard,
		func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
			return createTestServer(t, initFn, installCfg, logOutputBuffer).
				WithCustomDiagnosticHandlers(builderHandler)
		},
	)
	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()
	// Wait for the goroutine to have registered its handler
	time.Sleep(2 * time.Second)
	client := testServerClient()
	// Builder-registered diagnostic should be served
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/debug/diagnostic/%s", port, basePath, "custom.builder.v1"), nil)
	require.NoError(t, err)
	resp, err := client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode, "builder-registered diagnostic should return 200")
	// InitInfo goroutine-registered diagnostic should NOT be served (proves the bug)
	req, err = http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/debug/diagnostic/%s", port, basePath, "custom.initinfo.v1"), nil)
	require.NoError(t, err)
	resp, err = client.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode, "initInfo goroutine-registered diagnostic should return 400 because it was registered after addRoutes snapshot")
	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}
