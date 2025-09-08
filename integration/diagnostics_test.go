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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/stretchr/testify/require"
)

const (
	testDiagnosticType = "go.goroutines.v1"
	testSecret         = "secretForTest"
	tmpDir             = "/tmp"
	tmpFilePath        = "/tmp/tmpFile"
)

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
