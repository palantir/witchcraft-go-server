// Copyright (c) 2020 Palantir Technologies. All rights reserved.
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

package wdebug

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/palantir/conjure-go-runtime/v3/conjure-go-contract/codecs"
	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft/wdebug"
	"github.com/palantir/witchcraft-go-server/v3/wrouter"
	"github.com/palantir/witchcraft-go-server/v3/wrouter/whttprouter"
	"github.com/stretchr/testify/require"
)

func TestDebugResource(t *testing.T) {
	ctx := context.Background()
	r := wrouter.New(whttprouter.New())
	secret := refreshable.New("secret1")
	err := RegisterRoute(ctx, r, secret, func() []wdebug.DiagnosticHandler { return nil })
	require.NoError(t, err)

	server := httptest.NewServer(r)
	defer server.Close()

	for _, test := range []struct {
		DiagnosticType wdebug.DiagnosticType
		Verify         func(t *testing.T, resp *http.Response)
	}{
		{
			DiagnosticType: DiagnosticTypeGoroutinesV1,
			Verify: func(t *testing.T, resp *http.Response) {
				require.Equal(t, 200, resp.StatusCode)
				require.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
				require.Equal(t, "true", resp.Header.Get("Safe-Loggable"))
				var goroutines string
				require.NoError(t, codecs.Plain.Decode(resp.Body, &goroutines))
				require.NotEmpty(t, goroutines)
				require.Contains(t, goroutines, "github.com/palantir/witchcraft-go-server")
			},
		},
		{
			DiagnosticType: DiagnosticTypeGoroutinesV2,
			Verify: func(t *testing.T, resp *http.Response) {
				require.Equal(t, 200, resp.StatusCode)
				require.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
				require.Equal(t, "true", resp.Header.Get("Safe-Loggable"))
				var body bytes.Buffer
				require.NoError(t, codecs.Binary.Decode(resp.Body, &body))
				require.NotEmpty(t, body.Bytes())
			},
		},
		{
			DiagnosticType: DiagnosticTypeHeapProfileV1,
			Verify: func(t *testing.T, resp *http.Response) {
				require.Equal(t, 200, resp.StatusCode)
				require.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
				require.Equal(t, "true", resp.Header.Get("Safe-Loggable"))
				var body bytes.Buffer
				require.NoError(t, codecs.Binary.Decode(resp.Body, &body))
				require.NotEmpty(t, body.Bytes())
			},
		},
		{
			DiagnosticType: DiagnosticTypeSystemTimeV1,
			Verify: func(t *testing.T, resp *http.Response) {
				require.Equal(t, 200, resp.StatusCode)
				require.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
				require.Equal(t, "true", resp.Header.Get("Safe-Loggable"))
				var systemTime string
				require.NoError(t, codecs.Plain.Decode(resp.Body, &systemTime))
				parsed, err := time.Parse(time.RFC3339Nano, systemTime)
				require.NoError(t, err)
				require.NotEmpty(t, parsed)
			},
		},
	} {
		t.Run(string(test.DiagnosticType), func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/debug/diagnostic/%s", server.URL, test.DiagnosticType), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+secret.Current())
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			test.Verify(t, resp)
		})
	}

	t.Run("400 on unsupported diagnostic type", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/debug/diagnostic/unknown.type.v1", server.URL), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+secret.Current())
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
	t.Run("401 on invalid auth header", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/debug/diagnostic/%s", server.URL, DiagnosticTypeAllocsProfileV1), nil)
		require.NoError(t, err)
		req.Header.Set("Auth", "Bearer "+secret.Current())
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
	t.Run("401 on invalid secret", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/debug/diagnostic/%s", server.URL, DiagnosticTypeAllocsProfileV1), nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer invalid")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

func TestDebugResource_LateRegisteredCustomHandler(t *testing.T) {
	ctx := context.Background()
	r := wrouter.New(whttprouter.New())
	secret := refreshable.New("secret1")
	var handlers atomic.Pointer[[]wdebug.DiagnosticHandler]
	provider := func() []wdebug.DiagnosticHandler {
		if p := handlers.Load(); p != nil {
			return *p
		}
		return nil
	}
	err := RegisterRoute(ctx, r, secret, provider)
	require.NoError(t, err)
	server := httptest.NewServer(r)
	defer server.Close()
	// Before registering custom handler, the type should return 400
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/debug/diagnostic/custom.test.v1", server.URL), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+secret.Current())
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	// Register a custom handler after route setup (simulates async init)
	customHandler := testDiagnosticHandler{
		diagnosticType: "custom.test.v1",
		content:        `{"status":"ok"}`,
	}
	registered := []wdebug.DiagnosticHandler{customHandler}
	handlers.Store(&registered)
	// Now the custom handler should be resolvable
	req, err = http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/debug/diagnostic/custom.test.v1", server.URL), nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+secret.Current())
	resp, err = http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	require.Equal(t, "true", resp.Header.Get("Safe-Loggable"))
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	var parsed map[string]string
	require.NoError(t, json.Unmarshal(body, &parsed))
	require.Equal(t, "ok", parsed["status"])
}

type testDiagnosticHandler struct {
	diagnosticType wdebug.DiagnosticType
	content        string
}

func (h testDiagnosticHandler) Type() wdebug.DiagnosticType { return h.diagnosticType }
func (h testDiagnosticHandler) Documentation() string       { return "test handler" }
func (h testDiagnosticHandler) ContentType() string         { return "application/json" }
func (h testDiagnosticHandler) SafeLoggable() bool          { return true }
func (h testDiagnosticHandler) Extension() string           { return "json" }
func (h testDiagnosticHandler) WriteDiagnostic(_ context.Context, w io.Writer) error {
	_, err := io.WriteString(w, h.content)
	return err
}
