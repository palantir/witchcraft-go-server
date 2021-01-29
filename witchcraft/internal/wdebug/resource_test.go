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
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/codecs"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-server/v2/wrouter/whttprouter"
	"github.com/stretchr/testify/require"
)

func TestDebugResource(t *testing.T) {
	ctx := context.Background()
	r := wrouter.New(whttprouter.New())
	secret := refreshable.NewDefaultRefreshable("secret1")
	err := RegisterRoute(r, refreshable.NewString(secret))
	require.NoError(t, err)

	server := httptest.NewServer(r)
	defer server.Close()

	for _, test := range []struct {
		DiagnosticType DiagnosticType
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
	} {
		t.Run(string(test.DiagnosticType), func(t *testing.T) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/debug/diagnostic/%s", server.URL, test.DiagnosticType), nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+secret.Current().(string))
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			test.Verify(t, resp)
		})
	}
}
