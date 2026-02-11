// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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

package whttpservemux_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/palantir/witchcraft-go-server/v3/wrouter"
	"github.com/palantir/witchcraft-go-server/v3/wrouter/whttpservemux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterNotFoundHandler(t *testing.T) {
	notFoundCalled := false
	router := whttpservemux.New()
	router.Register(http.MethodGet, []wrouter.PathSegment{
		{Type: wrouter.LiteralSegment, Value: "exists"},
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	router.RegisterNotFoundHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		notFoundCalled = true
		w.WriteHeader(http.StatusNotFound)
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/missing")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.True(t, notFoundCalled)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// Ensures that PathParams returns the correct value from req.PathValue.
// This would fail if the handler from [http.ServeMux.Handler] was used
// instead of [http.ServeMux.ServeHTTP].
func TestPathParams(t *testing.T) {
	var gotParams map[string]string
	router := whttpservemux.New()
	router.Register(http.MethodGet, []wrouter.PathSegment{
		{Type: wrouter.LiteralSegment, Value: "datasets"},
		{Type: wrouter.PathParamSegment, Value: "rid"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParams = router.PathParams(r, []string{"rid"})
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/datasets/my-dataset-id")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, map[string]string{"rid": "my-dataset-id"}, gotParams)
}

func TestTrailingPathParam(t *testing.T) {
	var gotParams map[string]string
	router := whttpservemux.New()
	router.Register(http.MethodGet, []wrouter.PathSegment{
		{Type: wrouter.LiteralSegment, Value: "file"},
		{Type: wrouter.TrailingPathParamSegment, Value: "path"},
	}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotParams = router.PathParams(r, []string{"path"})
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	resp, err := http.Get(server.URL + "/file/var/data/my-file.txt")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Trailing path param value must not have a leading slash.
	assert.Equal(t, map[string]string{"path": "var/data/my-file.txt"}, gotParams)
}

func TestRootPathExactMatch(t *testing.T) {
	rootCalled := false
	router := whttpservemux.New()
	router.Register(http.MethodGet, []wrouter.PathSegment{
		{Type: wrouter.LiteralSegment, Value: ""},
	}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rootCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	// Root path should match.
	resp, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.True(t, rootCalled)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Non-root path must not match the root route.
	rootCalled = false
	resp, err = http.Get(server.URL + "/other")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.False(t, rootCalled)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
