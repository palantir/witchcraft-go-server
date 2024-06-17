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

package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/objmatcher"
	"github.com/palantir/witchcraft-go-logging/wlog"
	wlogzap "github.com/palantir/witchcraft-go-logging/wlog-zap"
	"github.com/palantir/witchcraft-go-logging/wlog/extractor"
	"github.com/palantir/witchcraft-go-logging/wlog/logreader"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/wresource"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-server/v2/wrouter/whttprouter"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
	"github.com/palantir/witchcraft-go-tracing/wzipkin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRequestTelemetryMiddleware tests the behavior of the NewRequestTelemetry request
// middleware and the NewRouteRequestLog route middleware. Verifies that service logs and request logs are emitted
// properly (and that properties like UID, SID, TokenID and TraceID are extracted from the request).
func TestRequestTelemetryMiddleware(t *testing.T) {
	// A bogus token without access to anything interesting. It encodes the UID, SID, and TokenID in testReqIDs below.
	testToken := `Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ2cDlrWFZMZ1NlbTZNZHN5a25ZVjJ3PT0iLCJzaWQiOiJyVTFLNW1XdlRpcVJvODlBR3NzZFRBPT0iLCJqdGkiOiJrbmY1cjQyWlFJcVU3L1VlZ3I0ditBPT0ifQ.JTD36MhcwmSuvfdCkfSYc-LHOGNA1UQ-0FKLKqdXbF4`
	testReqIDs := struct {
		UID, SID, TokenID, TraceID string
	}{
		UID:     "be9f645d-52e0-49e9-ba31-db32927615db",
		SID:     "ad4d4ae6-65af-4e2a-91a3-cf401acb1d4c",
		TokenID: "9277f9af-8d99-408a-94ef-f51e82be2ff8",
		TraceID: "6c2f558d62a7085f",
	}

	var svcOutput bytes.Buffer
	svcLog := svc1log.NewFromCreator(&svcOutput, wlog.InfoLevel, wlogzap.LoggerProvider().NewLeveledLogger, svc1log.Origin("origin"))
	var reqOutput bytes.Buffer
	reqLog := req2log.NewFromCreator(&reqOutput, wlogzap.LoggerProvider().NewLogger)
	var trcOutput bytes.Buffer
	trcLog := trc1log.NewFromCreator(&trcOutput, wlog.DefaultLoggerProvider().NewLogger)

	metricsRegistry := metrics.NewRootMetricsRegistry()

	// create router
	r := wrouter.New(
		whttprouter.New(),
		wrouter.RootRouterParamAddRequestHandlerMiddleware(
			NewRequestTelemetry(
				svcLog,
				nil,
				nil,
				nil,
				nil,
				reqLog,
				trcLog,
				nil,
				extractor.NewDefaultIDsExtractor(),
				metricsRegistry,
			),
		),
		wrouter.RootRouterParamAddRouteHandlerMiddleware(
			NewRouteTelemetry(),
		),
	)
	err := r.Register(http.MethodGet, "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		assert.Equal(t, testReqIDs.TraceID, string(wtracing.TraceIDFromContext(ctx)))

		svc1log.FromContext(ctx).Info("message")
		metrics.FromContext(ctx).Counter("counter").Inc(1)
	}))
	require.NoError(t, err)

	// start server
	server := httptest.NewServer(r)
	defer server.Close()
	defer func() {
		matcher := objmatcher.SliceMatcher{
			objmatcher.MapMatcher{
				"type": objmatcher.NewEqualsMatcher("trace.1"),
				"time": objmatcher.NewRegExpMatcher(".+"),
				"span": objmatcher.MapMatcher(map[string]objmatcher.Matcher{
					"name":      objmatcher.NewEqualsMatcher("GET /"),
					"traceId":   objmatcher.NewRegExpMatcher("[a-f0-9]+"),
					"parentId":  objmatcher.NewAnyMatcher(),
					"id":        objmatcher.NewRegExpMatcher("[a-f0-9]+"),
					"timestamp": objmatcher.NewAnyMatcher(),
					"duration":  objmatcher.NewAnyMatcher(),
				}),
			},
			objmatcher.MapMatcher{
				"type": objmatcher.NewEqualsMatcher("trace.1"),
				"time": objmatcher.NewRegExpMatcher(".+"),
				"span": objmatcher.MapMatcher(map[string]objmatcher.Matcher{
					"name":      objmatcher.NewEqualsMatcher("witchcraft-go-server request middleware"),
					"traceId":   objmatcher.NewRegExpMatcher("[a-f0-9]+"),
					"id":        objmatcher.NewRegExpMatcher("[a-f0-9]+"),
					"timestamp": objmatcher.NewAnyMatcher(),
					"duration":  objmatcher.NewAnyMatcher(),
					"tags": objmatcher.MapMatcher(
						map[string]objmatcher.Matcher{
							"http.status_code": objmatcher.NewEqualsMatcher("200"),
							"http.method":      objmatcher.NewEqualsMatcher("GET"),
							"http.useragent":   objmatcher.NewRegExpMatcher(".*"),
						},
					),
				}),
			},
		}
		entries, err := logreader.EntriesFromContent(trcOutput.Bytes())
		assert.NoError(t, err, "unexpected error when unmarshalling trace output")
		assert.Len(t, entries, 2, "unexpected number of trace entries")
		assert.NoError(t, (matcher[0]).Matches(map[string]any(entries[0])), "unexpected content in trace output")
		assert.NoError(t, matcher[1].Matches(map[string]any(entries[1])), "unexpected content in trace output")
	}()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", testToken)
	req.Header.Set("X-B3-TraceId", testReqIDs.TraceID)

	_, err = http.DefaultClient.Do(req)
	assert.NoError(t, err)

	testLogParams := func(t *testing.T, logBytes []byte) {
		logMap := make(map[string]interface{})
		assert.NoError(t, json.Unmarshal(logBytes, &logMap), "failed to unmarshal log output: %s", string(logBytes))
		logType := logMap[wlog.TypeKey]
		assert.Equal(t, testReqIDs.UID, logMap[wlog.UIDKey], "%s UID mismatch", logType)
		assert.Equal(t, testReqIDs.SID, logMap[wlog.SIDKey], "%s SID mismatch", logType)
		assert.Equal(t, testReqIDs.TokenID, logMap[wlog.TokenIDKey], "%s TokenID mismatch", logType)
		assert.Equal(t, testReqIDs.TraceID, logMap[wlog.TraceIDKey], "%s TraceID mismatch", logType)
	}

	testLogParams(t, svcOutput.Bytes())
	testLogParams(t, reqOutput.Bytes())

	foundCounter := false
	metricsRegistry.Each(func(name string, tags metrics.Tags, value metrics.MetricVal) {
		if name == "counter" {
			foundCounter = true
		}
	})
	assert.True(t, foundCounter, "metrics registry did not record metric inside handler")
}

func TestRequestMetricRequestMeterMiddleware(t *testing.T) {
	reqMiddleware := NewRouteTelemetry()

	now = func() time.Time { return time.UnixMilli(0) }
	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://localhost", bytes.NewBufferString("content"))
	require.NoError(t, err)
	reqMiddleware(w, req, wrouter.RequestVals{}, func(rw http.ResponseWriter, r *http.Request, reqVals wrouter.RequestVals) {
		now = func() time.Time { return time.UnixMilli(1) }
		_, _ = fmt.Fprint(rw, "ok")
	})

	m := make(map[string]interface{})
	metrics.DefaultMetricsRegistry.Each(func(name string, tags metrics.Tags, metric metrics.MetricVal) {
		vals := metric.Values()
		m[name] = vals
	})

	respMap := m["server.response"]
	if objmatcher.MapMatcher(getTimerObjectMatcher(1, 1000)).Matches(respMap) != nil {
		t.Errorf("Does not match: %v", err)
	}
}

func TestRequestMetricHandlerWithTags(t *testing.T) {
	for _, currCase := range []struct {
		metricName          string
		metricObjectMatcher map[string]objmatcher.Matcher
	}{
		{
			metricName:          "server.response",
			metricObjectMatcher: objmatcher.MapMatcher(getTimerObjectMatcher(1, 1000)),
		},
		{
			metricName:          "server.request.size",
			metricObjectMatcher: objmatcher.MapMatcher(getHistogramObjectMatcher(1)),
		},
		{
			metricName:          "server.response.size",
			metricObjectMatcher: objmatcher.MapMatcher(getHistogramObjectMatcher(1)),
		},
		{
			metricName:          "server.response.error",
			metricObjectMatcher: objmatcher.MapMatcher(getMeterObjectMatcher(1)),
		},
	} {
		r := metrics.NewRootMetricsRegistry()

		wRouter := wrouter.New(
			whttprouter.New(),

			wrouter.RootRouterParamAddRequestHandlerMiddleware(NewRequestTelemetry(
				nil,
				nil,
				nil,
				nil,
				nil,
				req2log.New(io.Discard),
				nil,
				nil,
				nil,
				nil,
			)),
			wrouter.RootRouterParamAddRouteHandlerMiddleware(NewRouteTelemetry()),
		)

		authResource := wresource.New("AuthResource", wRouter)
		now = func() time.Time { return time.UnixMilli(0) }
		err := authResource.Get("userAuth", "/userAuth", http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			now = func() time.Time { return time.UnixMilli(1) }
			rw.WriteHeader(http.StatusInternalServerError)
		}))
		require.NoError(t, err)

		w := httptest.NewRecorder()
		req, err := http.NewRequest(http.MethodGet, "http://localhost/userAuth", bytes.NewBufferString("content"))
		require.NoError(t, err)
		wRouter.ServeHTTP(w, req)

		m := make(map[string]interface{})
		tagsMap := make(map[string]metrics.Tags)
		r.Each(func(name string, tags metrics.Tags, metric metrics.MetricVal) {
			vals := metrics.ToMetricVal(metric).Values()
			m[name] = vals
			tagsMap[name] = tags
		})

		respMap := m[currCase.metricName]
		err = objmatcher.MapMatcher(currCase.metricObjectMatcher).Matches(respMap)
		if err != nil {
			t.Errorf("Does not match: %v for %v", err, currCase.metricName)
		}

		respTags := tagsMap[currCase.metricName]
		assert.Equal(t, metrics.Tags{
			metrics.MustNewTag("endpoint", "userauth"),
			metrics.MustNewTag("method", "get"),
			metrics.MustNewTag("service-name", "authresource"),
		}, respTags, currCase.metricName)
	}
}

func TestStrictTransportSecurity(t *testing.T) {
	wRouter := wrouter.New(
		whttprouter.New(),
		wrouter.RootRouterParamAddRequestHandlerMiddleware(NewStrictTransportSecurityHeader()),
	)

	w := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/", nil)
	require.NoError(t, err)
	wRouter.ServeHTTP(w, req)
	assert.Equal(t, []string{"max-age=31536000"}, w.Result().Header["Strict-Transport-Security"])
}

func getHistogramObjectMatcher(count int) map[string]objmatcher.Matcher {
	return map[string]objmatcher.Matcher{
		"count":  objmatcher.NewEqualsMatcher(int64(count)),
		"mean":   objmatcher.NewAnyMatcher(),
		"stddev": objmatcher.NewAnyMatcher(),
		"max":    objmatcher.NewAnyMatcher(),
		"min":    objmatcher.NewAnyMatcher(),
		"p50":    objmatcher.NewAnyMatcher(),
		"p95":    objmatcher.NewAnyMatcher(),
		"p99":    objmatcher.NewAnyMatcher(),
	}
}

func getMeterObjectMatcher(count int) map[string]objmatcher.Matcher {
	return map[string]objmatcher.Matcher{
		"count": objmatcher.NewEqualsMatcher(int64(count)),
		"1m":    objmatcher.NewAnyMatcher(),
		"5m":    objmatcher.NewAnyMatcher(),
		"15m":   objmatcher.NewAnyMatcher(),
		"mean":  objmatcher.NewAnyMatcher(),
	}
}

func getTimerObjectMatcher(count, value int) map[string]objmatcher.Matcher {
	return map[string]objmatcher.Matcher{
		"count":    objmatcher.NewEqualsMatcher(int64(count)),
		"mean":     objmatcher.NewEqualsMatcher(float64(value)),
		"stddev":   objmatcher.NewAnyMatcher(),
		"min":      objmatcher.NewEqualsMatcher(int64(value)),
		"max":      objmatcher.NewEqualsMatcher(int64(value)),
		"meanRate": objmatcher.NewAnyMatcher(),
		"1m":       objmatcher.NewAnyMatcher(),
		"5m":       objmatcher.NewAnyMatcher(),
		"15m":      objmatcher.NewAnyMatcher(),
		"p50":      objmatcher.NewAnyMatcher(),
		"p95":      objmatcher.NewAnyMatcher(),
		"p99":      objmatcher.NewAnyMatcher(),
	}
}

func TestRequestDisableTelemetry(t *testing.T) {
	ctx := context.Background()

	var reqOutput bytes.Buffer
	reqLog := req2log.NewFromCreator(&reqOutput, wlogzap.LoggerProvider().NewLogger)
	ctx = req2log.WithLogger(ctx, reqLog)

	var spanOutput bytes.Buffer
	spanLog := trc1log.NewFromCreator(&spanOutput, wlogzap.LoggerProvider().NewLogger)

	tracer, err := wzipkin.NewTracer(spanLog)
	require.NoError(t, err)
	ctx = wtracing.ContextWithTracer(ctx, tracer)

	w := httptest.NewRecorder()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost", bytes.NewBufferString("content"))
	require.NoError(t, err)

	NewRouteTelemetry()(w, req, wrouter.RequestVals{DisableTelemetry: true}, func(rw http.ResponseWriter, r *http.Request, reqVals wrouter.RequestVals) {
		_, _ = fmt.Fprint(rw, "ok")
	})

	metrics.DefaultMetricsRegistry.Each(func(_ string, _ metrics.Tags, metric metrics.MetricVal) {
		assert.Empty(t, metric.Values(), "expected no metrics to be written when Skiptelemetry is true")
	})

	assert.Empty(t, reqOutput.Bytes(), "expected request log to be empty when DisableTelemetry is true")
	assert.Empty(t, spanOutput.Bytes(), "expected trace span log to be empty when DisableTelemetry is true")
}
