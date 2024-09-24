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

package wrouter_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	// underscore import to use zap implementation
	_ "github.com/palantir/witchcraft-go-logging/wlog-zap"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-server/v2/wrouter/wgorillamux"
	"github.com/palantir/witchcraft-go-server/v2/wrouter/whttprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouterImpls(t *testing.T) {
	for i, tc := range []struct {
		name string
		impl wrouter.RouterImpl
	}{
		{"wgorillamux", wgorillamux.New()},
		{"whttprouter", whttprouter.New()},
	} {
		// create router
		r := wrouter.New(tc.impl, nil)

		// register routes
		matched := make(map[string]bool)
		err := r.Register("GET", "/foo", mustMatchHandler(t, "GET", "/foo", nil, matched))
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		err = r.Register("GET", "/datasets/{rid}", mustMatchHandler(t, "GET", "/datasets/id-500", map[string]string{
			"rid": "id-500",
		}, matched))
		require.NoError(t, err, "Case %d: %s", i, tc.name)

		subrouter := r.Subrouter("/foo")
		err = subrouter.Register("GET", "/{id}", mustMatchHandler(t, "GET", "/foo/13", map[string]string{
			"id": "13",
		}, matched))
		require.NoError(t, err, "Case %d: %s", i, tc.name)

		err = r.Register("GET", "/file/{path*}", mustMatchHandler(t, "GET", "/file/var/data/my-file.txt", map[string]string{
			"path": "var/data/my-file.txt",
		}, matched))
		require.NoError(t, err, "Case %d: %s", i, tc.name)

		wantRoutes := []wrouter.RouteSpec{
			{
				Method:       "GET",
				PathTemplate: "/datasets/{rid}",
			},
			{
				Method:       "GET",
				PathTemplate: "/file/{path*}",
			},
			{
				Method:       "GET",
				PathTemplate: "/foo",
			},
			{
				Method:       "GET",
				PathTemplate: "/foo/{id}",
			},
		}
		assert.Equal(t, wantRoutes, r.RegisteredRoutes(), "Case %d: %s", i, tc.name)

		server := httptest.NewServer(r)
		defer server.Close()

		_, err = http.Get(server.URL + "/datasets/id-500")
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		_, err = http.Get(server.URL + "/file/var/data/my-file.txt")
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		_, err = http.Get(server.URL + "/foo")
		require.NoError(t, err, "Case %d: %s", i, tc.name)
		_, err = http.Get(server.URL + "/foo/13")
		require.NoError(t, err, "Case %d: %s", i, tc.name)

		var sortedKeys []string
		for k := range matched {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)
		for _, k := range sortedKeys {
			assert.True(t, matched[k], "Case %d: %s\nMatcher not called for %s", i, tc.name, k)
		}
	}
}

// Smoke test that tests registering routes, using subrouters, listing routes, etc. on all router implementations.
func TestRouterImplSmoke(t *testing.T) {
	for i, tc := range []struct {
		name string
		impl wrouter.RouterImpl
	}{
		{"wgorillamux", wgorillamux.New()},
		{"whttprouter", whttprouter.New()},
	} {
		func() {
			// create router
			r := wrouter.New(tc.impl, nil)

			// register routes
			matched := make(map[string]bool)
			err := r.Register("GET", "/foo", mustMatchHandler(t, "GET", "/foo", nil, matched))
			require.NoError(t, err, "Case %d: %s", i, tc.name)
			err = r.Register("GET", "/datasets/{rid}", mustMatchHandler(t, "GET", "/datasets/id-500", map[string]string{
				"rid": "id-500",
			}, matched))
			require.NoError(t, err, "Case %d: %s", i, tc.name)

			subrouter := r.Subrouter("/foo")
			err = subrouter.Register("GET", "/{id}", mustMatchHandler(t, "GET", "/foo/13", map[string]string{
				"id": "13",
			}, matched))
			require.NoError(t, err, "Case %d: %s", i, tc.name)

			err = r.Register("GET", "/file/{path*}", mustMatchHandler(t, "GET", "/file/var/data/my-file.txt", map[string]string{
				"path": "var/data/my-file.txt",
			}, matched))
			require.NoError(t, err, "Case %d: %s", i, tc.name)

			wantRoutes := []wrouter.RouteSpec{
				{
					Method:       "GET",
					PathTemplate: "/datasets/{rid}",
				},
				{
					Method:       "GET",
					PathTemplate: "/file/{path*}",
				},
				{
					Method:       "GET",
					PathTemplate: "/foo",
				},
				{
					Method:       "GET",
					PathTemplate: "/foo/{id}",
				},
			}
			assert.Equal(t, wantRoutes, r.RegisteredRoutes(), "Case %d: %s", i, tc.name)

			server := httptest.NewServer(r)
			defer server.Close()

			_, err = http.Get(server.URL + "/datasets/id-500")
			require.NoError(t, err, "Case %d: %s", i, tc.name)
			_, err = http.Get(server.URL + "/file/var/data/my-file.txt")
			require.NoError(t, err, "Case %d: %s", i, tc.name)
			_, err = http.Get(server.URL + "/foo")
			require.NoError(t, err, "Case %d: %s", i, tc.name)
			_, err = http.Get(server.URL + "/foo/13")
			require.NoError(t, err, "Case %d: %s", i, tc.name)

			var sortedKeys []string
			for k := range matched {
				sortedKeys = append(sortedKeys, k)
			}
			sort.Strings(sortedKeys)
			for _, k := range sortedKeys {
				assert.True(t, matched[k], "Case %d: %s\nMatcher not called for %s", i, tc.name, k)
			}
		}()
	}
}

// Tests specific cases for registering routes and calling endpoints that hit them.
func TestRouterImplRouteHandling(t *testing.T) {
	for i, tc := range []struct {
		name   string
		routes []singleRouteTest
	}{
		{
			name: "empty path",
			routes: []singleRouteTest{
				{
					method: "GET",
					path:   "/",
					reqURL: "/",
				},
			},
		},
		{
			name: "paths with common starting variables",
			routes: []singleRouteTest{
				{
					method: "GET",
					path:   "/{entityId}",
					reqURL: "/latest",
					wantPathParams: map[string]string{
						"entityId": "latest",
					},
				},
				{
					method: "GET",
					path:   "/{entityId}/{date}",
					reqURL: "/test-id/test-date",
					wantPathParams: map[string]string{
						"entityId": "test-id",
						"date":     "test-date",
					},
				},
			},
		},
		{
			name: "paths with same path expression with only method differing",
			routes: []singleRouteTest{
				{
					method: "GET",
					path:   "/{getId*}",
					reqURL: "/latest",
					wantPathParams: map[string]string{
						"getId": "latest",
					},
				},
				{
					method: "POST",
					path:   "/{postId*}",
					reqURL: "/test-id",
					wantPathParams: map[string]string{
						"postId": "test-id",
					},
				},
			},
		},
		// Following test does not work for zhttprouter due to https://github.com/julienschmidt/httprouter/issues/183.
		// The preceding test ("paths with common starting variables") demonstrates a work-around for this behavior --
		// one can register a path where the shorter segment shares the path parameter variable and then only execute
		// the handler if the variable value matches the desired literal path (in this case, "latest").
		//{
		//	name: "path with literal and variable at same level",
		//	routes: []singleRouteTest{
		//		{
		//			method: "GET",
		//			path:   "/latest",
		//			reqURL: "/latest",
		//		},
		//		{
		//			method: "GET",
		//			path:   "/{entityId}/{date}",
		//			reqURL: "/test-id/test-date",
		//			wantPathParams: map[string]string{
		//				"entityId": "test-id",
		//				"date": "test-date",
		//			},
		//		},
		//	},
		//},
	} {
		for _, routerImpl := range []struct {
			name string
			impl wrouter.RouterImpl
		}{
			{"wgorillamux", wgorillamux.New()},
			{"whttprouter", whttprouter.New()},
		} {
			func() {
				// create router
				r := wrouter.New(routerImpl.impl, nil)

				// register routes
				matched := make(map[string]bool)
				for _, currRoute := range tc.routes {
					err := r.Register(currRoute.method, currRoute.path, mustMatchHandler(t, currRoute.method, currRoute.reqURL, currRoute.wantPathParams, matched))
					require.NoError(t, err, "Case %d: %s %s", i, tc.name, routerImpl.name)
				}

				// start server
				server := httptest.NewServer(r)
				defer server.Close()

				// make HTTP calls
				for _, currRoute := range tc.routes {
					req, err := http.NewRequest(
						currRoute.method,
						server.URL+currRoute.reqURL,
						nil,
					)
					require.NoError(t, err, "Case %d: %s %s", i, tc.name, routerImpl.name)
					_, err = http.DefaultClient.Do(req)
					require.NoError(t, err, "Case %d: %s %s", i, tc.name, routerImpl.name)
				}

				// verify results
				var sortedKeys []string
				for k := range matched {
					sortedKeys = append(sortedKeys, k)
				}
				sort.Strings(sortedKeys)
				for _, k := range sortedKeys {
					assert.True(t, matched[k], "Case %d: %s %s\nMatcher not called for %s", i, tc.name, routerImpl.name, k)
				}
			}()
		}
	}
}

type singleRouteTest struct {
	method         string
	path           string
	reqURL         string
	wantPathParams map[string]string
}

func mustMatchHandler(t *testing.T, method, path string, pathVars map[string]string, matched map[string]bool) http.Handler {
	matched[fmt.Sprintf("[%s] %s", method, path)] = false
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, method, r.Method, "Method did not match expected for path %s", path)
		assert.Equal(t, path, r.URL.Path, "Path did not match expected for path %s", path)
		assert.Equal(t, pathVars, wrouter.PathParams(r), "Path params did not match expected for path %s", path)
		matched[fmt.Sprintf("[%s] %s", method, path)] = true
	})
}

// Tests that RouteHandlerMiddleware are called in the right order on the right routes.
func TestRouterMiddlewareRegistration(t *testing.T) {
	type ctxKey struct{}
	newMarkingMiddleware := func(marking string) wrouter.RouteHandlerMiddleware {
		return func(rw http.ResponseWriter, req *http.Request, reqVals wrouter.RequestVals, next wrouter.RouteRequestHandler) {
			curr := req.Context().Value(ctxKey{})
			if curr == nil {
				req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, []string{marking}))
			} else {
				req = req.WithContext(context.WithValue(req.Context(), ctxKey{}, append(curr.([]string), marking)))
			}
			next(rw, req, reqVals)
		}
	}
	echoMarkingHandler := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		_, _ = rw.Write([]byte(fmt.Sprint(req.Context().Value(ctxKey{}))))
	})

	// create router
	r := wrouter.New(whttprouter.New(), wrouter.RootRouterParamAddRouteHandlerMiddleware(newMarkingMiddleware("global")))
	require.NoError(t, r.Get("/one", echoMarkingHandler))
	require.NoError(t, r.Get("/two", echoMarkingHandler, wrouter.RouteMiddleware(newMarkingMiddleware("routeHandler"))))
	// verify middleware configured after routes is still applied
	r.AddRouteHandlerMiddleware(newMarkingMiddleware("globalHandler"))

	// start server
	server := httptest.NewServer(r)
	defer server.Close()

	t.Run("endpoint one", func(t *testing.T) {
		resp, err := http.DefaultClient.Get(server.URL + "/one")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, "[global globalHandler]", string(body))
	})
	t.Run("endpoint two", func(t *testing.T) {
		resp, err := http.DefaultClient.Get(server.URL + "/two")
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()
		require.Equal(t, http.StatusOK, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		require.Equal(t, "[global globalHandler routeHandler]", string(body))
	})
}
