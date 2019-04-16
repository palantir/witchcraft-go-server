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
	"testing"
	"time"

	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-server/status/reporter"
	"github.com/palantir/witchcraft-go-server/witchcraft"
	"github.com/palantir/witchcraft-go-server/witchcraft/ratelimit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewInflightLimitMiddleware(t *testing.T) {
	healthReporter := reporter.NewHealthReporter()
	healthComponent, err := healthReporter.InitializeHealthComponent("INFLIGHT_MUTATING_REQUESTS")
	require.NoError(t, err)
	requireHealthy := func(msg string) {
		require.Equal(t, string(health.HealthStateHealthy), string(healthComponent.Status()), msg)
	}

	limiter := ratelimit.NewInFlightRequestLimitMiddleware(2, ratelimit.MatchMutating, healthComponent)

	wait, closeWait := context.WithCancel(context.Background())
	defer closeWait()

	initFn := func(ctx context.Context, info witchcraft.InitInfo) (cleanup func(), rErr error) {
		info.Router.RootRouter().AddRouteHandlerMiddleware(limiter)
		if err := info.Router.Get("/get", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
		})); err != nil {
			return nil, err
		}
		if err := info.Router.Post("/post", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			<-wait.Done()
			rw.WriteHeader(http.StatusOK)
		})); err != nil {
			return nil, err
		}

		return nil, nil
	}

	port, err := httpserver.AvailablePort()
	require.NoError(t, err)
	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, initFn, ioutil.Discard, createTestServer)
	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	client := testServerClient()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	doGet := func() *http.Response {
		getURL := fmt.Sprintf("https://localhost:%d/%s/get", port, basePath)
		request, err := http.NewRequest(http.MethodGet, getURL, nil)
		if err != nil {
			assert.NoError(t, err)
		}
		resp, err := client.Do(request.WithContext(ctx))
		if err != nil {
			assert.NoError(t, err)
		}
		return resp
	}
	doPost := func() *http.Response {
		postURL := fmt.Sprintf("https://localhost:%d/%s/post", port, basePath)
		request, err := http.NewRequest(http.MethodPost, postURL, nil)
		if err != nil {
			assert.NoError(t, err)
		}
		resp, err := client.Do(request.WithContext(ctx))
		if err != nil {
			assert.NoError(t, err)
		}
		return resp
	}

	// Fill up rate limit
	resp1c := make(chan *http.Response)
	requireHealthy("expected healthy before the first request")
	go func() { resp1c <- doPost() }()
	resp2c := make(chan *http.Response)
	requireHealthy("expected healthy before the second request")
	go func() { resp2c <- doPost() }()

	time.Sleep(time.Millisecond) // let things settle
	requireHealthy("expected healthy before the third request")

	resp3 := doPost()
	require.Equal(t, http.StatusTooManyRequests, resp3.StatusCode, "expected third POST request to be rate limited")

	require.Equal(t, http.StatusOK, doGet().StatusCode, "expected get request to be successful")
	require.Equal(t, http.StatusOK, doGet().StatusCode, "expected get request to be successful")

	// free the waiting requests, which should return 200
	closeWait()
	resp1 := <-resp1c
	assert.Equal(t, http.StatusOK, resp1.StatusCode, "expected blocked request 1 to return 200 when unblocked")
	resp2 := <-resp2c
	assert.Equal(t, http.StatusOK, resp2.StatusCode, "expected blocked request 2 to return 200 when unblocked")

	// we should now be unblocked and healthy
	require.Equal(t, http.StatusOK, doPost().StatusCode, "expected fourth POST request to be unblocked")
	requireHealthy("expected healthy after accepted request")

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}
