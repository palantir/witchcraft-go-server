// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/codecs"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	pkgserver "github.com/palantir/pkg/httpserver"
	"github.com/palantir/pkg/objmatcher"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerPanicRecoveryMiddleware(t *testing.T) {
	logOutputBuffer := &bytes.Buffer{}
	port, err := pkgserver.AvailablePort()
	require.NoError(t, err)

	var called bool
	var initCtx context.Context
	initFn := func(ctx context.Context, info witchcraft.InitInfo) (deferFn func(), rErr error) {
		initCtx = ctx
		// register handler that returns "ok"
		if err := info.Router.Get("/ok", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			httpserver.WriteJSONResponse(rw, "ok", http.StatusOK)
		})); err != nil {
			return nil, err
		}
		if err := info.Router.Get("/panic", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			called = true
			panic("panic inside handler")
		})); err != nil {
			return nil, err
		}
		if err := info.Router.Get("/panic-after-write", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			called = true
			_, _ = rw.Write([]byte("ok"))
			panic("panic inside handler after write")
		})); err != nil {
			return nil, err
		}
		if err := info.Router.Get("/before", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			called = true
			_, _ = rw.Write([]byte("ok"))
		})); err != nil {
			return nil, err
		}
		if err := info.Router.Get("/after", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			called = true
			_, _ = rw.Write([]byte("ok"))
		})); err != nil {
			return nil, err
		}
		return nil, nil
	}
	mware := wrouter.RequestHandlerMiddleware(func(rw http.ResponseWriter, r *http.Request, next http.Handler) {
		if strings.HasSuffix(r.URL.Path, "/before") {
			panic("panic before handler")
		}
		next.ServeHTTP(rw, r)
		if strings.HasSuffix(r.URL.Path, "/after") {
			panic("panic after handler")
		}
	})

	createTestServer := func(t *testing.T, initFn witchcraft.InitFunc, installCfg config.Install, logOutputBuffer io.Writer) (server *witchcraft.Server) {
		server = witchcraft.
			NewServer().
			WithInitFunc(initFn).
			WithMiddleware(mware).
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

	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, initFn, logOutputBuffer, createTestServer)
	defer func() {
		_ = server.Close()
	}()
	defer cleanup()

	client, err := httpclient.NewClient(
		httpclient.WithBaseURLs([]string{fmt.Sprintf("https://localhost:%d/example", port)}),
		httpclient.WithTLSConfig(&tls.Config{InsecureSkipVerify: true}),
		httpclient.WithMaxRetries(0))
	require.NoError(t, err)

	for _, test := range []struct {
		Name         string
		Path         string
		ExpectCalled bool
		VerifyErr    func(t *testing.T, traceID string, err errors.Error, logOutputBuffer *bytes.Buffer)
		VerifyOk     func(t *testing.T, traceID string, resp string, logOutputBuffer *bytes.Buffer)
	}{
		{
			Name:         "panic inside handler",
			Path:         "/panic",
			ExpectCalled: true,
			VerifyErr: func(t *testing.T, traceID string, err errors.Error, logOutputBuffer *bytes.Buffer) {
				svcLogs := getLogMessagesOfType(t, "service.1", logOutputBuffer.Bytes())
				if assert.Len(t, svcLogs, 2) {
					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("panic recovered"),
						"traceId": objmatcher.NewEqualsMatcher(traceID),
						"params": objmatcher.MapMatcher{
							"stacktrace": objmatcher.NewAnyMatcher(),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic inside handler"),
						},
					}.Matches(svcLogs[0]))

					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("Panic recovered in request handler. This is a bug. Responding 500 Internal Server Error."),
						"traceId": objmatcher.NewEqualsMatcher(traceID),
						"params": objmatcher.MapMatcher{
							"errorInstanceId": objmatcher.NewEqualsMatcher(err.InstanceID().String()),
							"errorName":       objmatcher.NewEqualsMatcher("Default:Internal"),
							"stacktrace":      objmatcher.NewAnyMatcher(),
						},
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic inside handler"),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
					}.Matches(svcLogs[1]))
				}

				reqLogs := getLogMessagesOfType(t, "request.2", logOutputBuffer.Bytes())
				if assert.Len(t, reqLogs, 1) {
					assert.NoError(t, objmatcher.MapMatcher{
						"time":         objmatcher.NewAnyMatcher(),
						"type":         objmatcher.NewEqualsMatcher("request.2"),
						"duration":     objmatcher.NewAnyMatcher(),
						"method":       objmatcher.NewEqualsMatcher("GET"),
						"params":       objmatcher.NewAnyMatcher(),
						"path":         objmatcher.NewEqualsMatcher("/example/panic"),
						"protocol":     objmatcher.NewEqualsMatcher("HTTP/2.0"),
						"requestSize":  objmatcher.NewAnyMatcher(),
						"responseSize": objmatcher.NewAnyMatcher(),
						"status":       objmatcher.NewEqualsMatcher(float64(500)),
						"traceId":      objmatcher.NewEqualsMatcher(traceID),
					}.Matches(reqLogs[0]))
				}
			},
		},
		{
			Name:         "panic inside handler after write",
			Path:         "/panic-after-write",
			ExpectCalled: true,
			VerifyOk: func(t *testing.T, traceID string, resp string, logOutputBuffer *bytes.Buffer) {
				assert.Equal(t, "ok", resp)

				svcLogs := getLogMessagesOfType(t, "service.1", logOutputBuffer.Bytes())
				if assert.Len(t, svcLogs, 2) {
					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("panic recovered"),
						"traceId": objmatcher.NewEqualsMatcher(traceID),
						"params": objmatcher.MapMatcher{
							"stacktrace": objmatcher.NewAnyMatcher(),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic inside handler after write"),
						},
					}.Matches(svcLogs[0]))

					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("Panic recovered in request handler. This is a bug. HTTP response status already written."),
						"traceId": objmatcher.NewEqualsMatcher(traceID),
						"params": objmatcher.MapMatcher{
							"stacktrace": objmatcher.NewAnyMatcher(),
						},
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic inside handler after write"),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
					}.Matches(svcLogs[1]))
				}

				reqLogs := getLogMessagesOfType(t, "request.2", logOutputBuffer.Bytes())
				if assert.Len(t, reqLogs, 1) {
					assert.NoError(t, objmatcher.MapMatcher{
						"time":         objmatcher.NewAnyMatcher(),
						"type":         objmatcher.NewEqualsMatcher("request.2"),
						"duration":     objmatcher.NewAnyMatcher(),
						"method":       objmatcher.NewEqualsMatcher("GET"),
						"params":       objmatcher.NewAnyMatcher(),
						"path":         objmatcher.NewEqualsMatcher("/example/panic-after-write"),
						"protocol":     objmatcher.NewEqualsMatcher("HTTP/2.0"),
						"requestSize":  objmatcher.NewAnyMatcher(),
						"responseSize": objmatcher.NewAnyMatcher(),
						"status":       objmatcher.NewEqualsMatcher(float64(200)),
						"traceId":      objmatcher.NewEqualsMatcher(traceID),
					}.Matches(reqLogs[0]))
				}
			},
		},
		{
			Name:         "panic before handler",
			Path:         "/before",
			ExpectCalled: false,
			VerifyErr: func(t *testing.T, traceID string, err errors.Error, logOutputBuffer *bytes.Buffer) {
				svcLogs := getLogMessagesOfType(t, "service.1", logOutputBuffer.Bytes())
				if assert.Len(t, svcLogs, 2) {
					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("panic recovered"),
						"params": objmatcher.MapMatcher{
							"stacktrace": objmatcher.NewAnyMatcher(),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
						"traceId":    objmatcher.NewEqualsMatcher(traceID),
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic before handler"),
						},
					}.Matches(svcLogs[0]))

					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("Panic recovered in server handler. This is a bug. Responding 500 Internal Server Error."),
						"params": objmatcher.MapMatcher{
							"errorInstanceId": objmatcher.NewEqualsMatcher(err.InstanceID().String()),
							"errorName":       objmatcher.NewEqualsMatcher("Default:Internal"),
							"stacktrace":      objmatcher.NewAnyMatcher(),
						},
						"traceId": objmatcher.NewEqualsMatcher(traceID),
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic before handler"),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
					}.Matches(svcLogs[1]))
				}

				reqLogs := getLogMessagesOfType(t, "request.2", logOutputBuffer.Bytes())
				assert.Len(t, reqLogs, 0, "panic before a request executes does not expect a request log")
			},
		},
		{
			Name:         "panic after handler",
			Path:         "/after",
			ExpectCalled: true,
			VerifyOk: func(t *testing.T, traceID string, resp string, logOutputBuffer *bytes.Buffer) {
				assert.Equal(t, "ok", resp)

				svcLogs := getLogMessagesOfType(t, "service.1", logOutputBuffer.Bytes())
				if assert.Len(t, svcLogs, 2) {
					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("panic recovered"),
						"params": objmatcher.MapMatcher{
							"stacktrace": objmatcher.NewAnyMatcher(),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
						"traceId":    objmatcher.NewEqualsMatcher(traceID),
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic after handler"),
						},
					}.Matches(svcLogs[0]))

					assert.NoError(t, objmatcher.MapMatcher{
						"time":    objmatcher.NewAnyMatcher(),
						"type":    objmatcher.NewEqualsMatcher("service.1"),
						"level":   objmatcher.NewEqualsMatcher("ERROR"),
						"origin":  objmatcher.NewEqualsMatcher("github.com/palantir/witchcraft-go-server/integration"),
						"message": objmatcher.NewEqualsMatcher("Panic recovered in server handler. This is a bug. HTTP response status already written."),
						"params": objmatcher.MapMatcher{
							"stacktrace": objmatcher.NewAnyMatcher(),
						},
						"traceId": objmatcher.NewEqualsMatcher(traceID),
						"unsafeParams": objmatcher.MapMatcher{
							"recovered": objmatcher.NewEqualsMatcher("panic after handler"),
						},
						"stacktrace": objmatcher.NewAnyMatcher(),
					}.Matches(svcLogs[1]))
				}

				reqLogs := getLogMessagesOfType(t, "request.2", logOutputBuffer.Bytes())
				if assert.Len(t, reqLogs, 1) {
					assert.NoError(t, objmatcher.MapMatcher{
						"time":         objmatcher.NewAnyMatcher(),
						"type":         objmatcher.NewEqualsMatcher("request.2"),
						"duration":     objmatcher.NewAnyMatcher(),
						"method":       objmatcher.NewEqualsMatcher("GET"),
						"params":       objmatcher.NewAnyMatcher(),
						"path":         objmatcher.NewEqualsMatcher("/example/after"),
						"protocol":     objmatcher.NewEqualsMatcher("HTTP/2.0"),
						"requestSize":  objmatcher.NewAnyMatcher(),
						"responseSize": objmatcher.NewEqualsMatcher(float64(2)),
						"status":       objmatcher.NewEqualsMatcher(float64(200)),
						"traceId":      objmatcher.NewEqualsMatcher(traceID),
					}.Matches(reqLogs[0]))
				}
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			logOutputBuffer.Reset()
			called = false

			span, ctx := wtracing.StartSpanFromTracerInContext(initCtx, test.Name)
			defer span.Finish()
			traceID := string(span.Context().TraceID)
			var respStr string
			resp, err := client.Get(ctx, httpclient.WithPath(test.Path), httpclient.WithResponseBody(&respStr, codecs.Plain))
			if test.VerifyErr != nil {
				require.Error(t, err)
				require.Nil(t, resp)
				cerr := errors.GetConjureError(err)
				require.NotNil(t, cerr)
				test.VerifyErr(t, traceID, cerr, logOutputBuffer)
			} else if test.VerifyOk != nil {
				require.NoError(t, err)
				require.NotNil(t, resp)
				test.VerifyOk(t, traceID, respStr, logOutputBuffer)
			}
			assert.Equal(t, test.ExpectCalled, called)
		})
	}

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}
