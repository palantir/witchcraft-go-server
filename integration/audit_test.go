// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/palantir/pkg/httpserver"
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/pkg/refreshable/v2"
	"github.com/palantir/pkg/rid"
	"github.com/palantir/pkg/uuid"
	v2 "github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/category/v2"
	commonv2 "github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/common/v2"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log"
	"github.com/palantir/witchcraft-go-server/v3/config"
	"github.com/palantir/witchcraft-go-server/v3/witchcraft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const (
	uuidRegexp = "^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[8|9|a|b][a-f0-9]{3}-[a-f0-9]{12}$"

	forwardedForVal0 = "203.0.113.195"
	forwardedForVal1 = "70.41.3.18"
	forwardedForVal2 = "150.172.238.178"

	// Generated using http://jwtbuilder.jamiekurtz.com/ and verified using https://jwt.io/.
	// Content: {"iss":"Online JWT Builder","iat":1745876870,"exp":30178648314,"aud":"www.example.com","sub":"jrocket@example.com","org":"org_vIK75NKFvaozQsFy","sid":"eecb9bf34bbb4c8eb87dbba3aa1523c6","jti":"3dd6434d-79a9-4d15-98b5-7b51dbb2cd31"}
	// Signing key: a-string-secret-at-least-256-bits-long
	testJWTToken = "eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE3NDU4NzY4NzAsImV4cCI6MzAxNzg2NDgzMTQsImF1ZCI6Ind3dy5leGFtcGxlLmNvbSIsInN1YiI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJvcmciOiJvcmdfdklLNzVOS0Z2YW96UXNGeSIsInNpZCI6ImVlY2I5YmYzNGJiYjRjOGViODdkYmJhM2FhMTUyM2M2IiwianRpIjoiM2RkNjQzNGQtNzlhOS00ZDE1LTk4YjUtN2I1MWRiYjJjZDMxIn0.kqOztqW_I-Lt6ceXpEms1UzHrL6Pu5oURpApHPASWl8"
)

// TestAuditLogs tests the audit logging functionality of a server.
//
// Currently, servers are set up with both v2 and v3 audit loggers. It is also possible to enable dual logging for each
// logger type.
//
// The test runs through all combinations of the following options:
//   - Emit or do not emit an audit log entry to the v2 logger
//   - Emit or do not emit an audit log entry to the v3 logger
//   - Enable or disable dual logging from v2 to v3
//   - Enable or disable dual logging from v3 to v2
//   - Log audit output in server initialization/thread vs. in request to server
//
// The test verifies that the expected number of log entries are emitted to each logger and that the log entries contain
// the expected values.
func TestAuditLogs(t *testing.T) {
	for _, logAuditV2 := range []bool{true, false} {
		for _, logAuditV3 := range []bool{true, false} {
			for _, dualLogAuditV2ToAuditV3 := range []bool{true, false} {
				for _, dualLogAuditV3ToAuditV3 := range []bool{true, false} {
					for _, inServerInit := range []bool{true, false} {
						testAuditLogHelper(
							t,
							logAuditV2,
							dualLogAuditV2ToAuditV3,
							logAuditV3,
							dualLogAuditV3ToAuditV3,
							inServerInit,
						)
					}
				}
			}
		}
	}
}

func TestAuditLogRuntimeConfigLiveReloaded(t *testing.T) {
	logOutputBuffer := &bytes.Buffer{}
	port, err := httpserver.AvailablePort()
	require.NoError(t, err)

	tmpDir := t.TempDir()

	runtimeCfg := config.Runtime{
		LoggerConfig: &config.LoggerConfig{
			Level: wlog.InfoLevel,
		},
		AuditConfig: &config.AuditConfig{
			Deployment:     "test-deployment",
			Product:        productName,
			ProductVersion: productVersion,
			Stack:          "test-stack",
			Service:        "test-service",
			Environment:    "test-environment",
		},
	}
	runtimeCfgYML, err := yaml.Marshal(runtimeCfg)
	require.NoError(t, err)

	runtimeConfigPath := filepath.Join(tmpDir, "runtime.yml")
	err = os.WriteFile(runtimeConfigPath, runtimeCfgYML, 0644)
	require.NoError(t, err)

	fileRefreshable := refreshable.NewFileRefreshable(context.Background(), runtimeConfigPath)
	_, err = fileRefreshable.Validation()
	require.NoError(t, err)
	fileRefreshableR, _ := refreshable.MapFromValidated(fileRefreshable, func(b []byte) []byte { return b })

	server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (deferFn func(), rErr error) {
		// if "inServerInit" is false, emit audit logs as a result of calling the endpoint. Verifies that audit
		// logging is properly set up on the context used in requests.
		if err := info.Router.Register("GET", "/testAuditLog", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			audit3log.FromContext(r.Context()).Audit("LogToAudit3", audit3log.AuditResultSuccess)
			w.WriteHeader(http.StatusOK)
		})); err != nil {
			return nil, err
		}

		return nil, nil
	}, logOutputBuffer, func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
		return createTestServerWithRuntimeConfigProvider(initFn, installCfg, logOutputBuffer, fileRefreshableR)
	})
	defer func() {
		require.NoError(t, server.Close())
	}()
	defer cleanup()

	client := testServerClient()

	// make a request to trigger audit log entry
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/testAuditLog", port, basePath), nil)
	require.NoError(t, err)
	_, err = client.Do(request)
	require.NoError(t, err)

	// update parts of audit runtime configuration
	runtimeCfg = config.Runtime{
		LoggerConfig: &config.LoggerConfig{
			Level: wlog.InfoLevel,
		},
		AuditConfig: &config.AuditConfig{
			Deployment:     "test-deployment-updated",
			Product:        productName,
			ProductVersion: productVersion,
			Stack:          "test-stack-updated",
			Service:        "test-service-updated",
			Environment:    "test-environment-updated",
		},
	}
	runtimeCfgYML, err = yaml.Marshal(runtimeCfg)
	require.NoError(t, err)

	err = os.WriteFile(runtimeConfigPath, runtimeCfgYML, 0644)
	require.NoError(t, err)

	// wait for runtime config to update
	time.Sleep(1000 * time.Millisecond)

	// make a request to trigger audit log entry after config was updated
	_, err = client.Do(request)
	require.NoError(t, err)

	audit3LogEntries := extractLogEntries[logging.AuditLogV3](t, logOutputBuffer.String(), "audit.3")

	assert.Equal(t, 2, len(audit3LogEntries))

	firstAudit3LogEntry := audit3LogEntries[0]
	assert.Equal(t, "LogToAudit3", firstAudit3LogEntry.Name)
	logEntryIDValue := uuidPtrValue(firstAudit3LogEntry.LogEntryId)
	assert.Regexp(t, uuidRegexp, logEntryIDValue)
	logEventIDValue := firstAudit3LogEntry.EventId.String()
	assert.Regexp(t, uuidRegexp, logEventIDValue)
	assert.Equal(t, "test-deployment", firstAudit3LogEntry.Deployment)
	assert.Equal(t, productName, firstAudit3LogEntry.Product)
	assert.Equal(t, productVersion, firstAudit3LogEntry.ProductVersion)
	assert.Equal(t, "test-stack", stringPtrValue(firstAudit3LogEntry.Stack))
	assert.Equal(t, "test-service", stringPtrValue(firstAudit3LogEntry.Service))
	assert.Equal(t, "test-environment", stringPtrValue(firstAudit3LogEntry.Environment))

	secondAudit3LogEntry := audit3LogEntries[1]
	assert.Equal(t, "LogToAudit3", secondAudit3LogEntry.Name)
	secondLogEntryIDValue := uuidPtrValue(secondAudit3LogEntry.LogEntryId)
	assert.Regexp(t, uuidRegexp, secondLogEntryIDValue)
	assert.NotEqual(t, logEntryIDValue, secondLogEntryIDValue)
	secondLogEventIDValue := secondAudit3LogEntry.EventId.String()
	assert.Regexp(t, uuidRegexp, secondLogEventIDValue)
	assert.NotEqual(t, logEventIDValue, secondLogEventIDValue)
	assert.Equal(t, "test-deployment-updated", secondAudit3LogEntry.Deployment)
	assert.Equal(t, productName, secondAudit3LogEntry.Product)
	assert.Equal(t, productVersion, secondAudit3LogEntry.ProductVersion)
	assert.Equal(t, "test-stack-updated", stringPtrValue(secondAudit3LogEntry.Stack))
	assert.Equal(t, "test-service-updated", stringPtrValue(secondAudit3LogEntry.Service))
	assert.Equal(t, "test-environment-updated", stringPtrValue(secondAudit3LogEntry.Environment))

	select {
	case err := <-serverErr:
		require.NoError(t, err)
	default:
	}
}

// TestAuditLog_ProduceAudit2LogsConfig verifies that the ProduceAudit2Logs runtime config field controls audit.2
// log output suppression, including via dual-logging from audit.3, and supports live reload.
func TestAuditLog_ProduceAudit2LogsConfig(t *testing.T) {
	for _, tc := range []struct {
		name              string
		produceAudit2     *bool
		logAuditV3        bool
		enableDualV3ToV2  bool
		wantAudit2Entries int
	}{
		{
			name:              "nil ProduceAudit2Logs emits audit.2 logs",
			produceAudit2:     nil,
			wantAudit2Entries: 1,
		},
		{
			name:              "true ProduceAudit2Logs emits audit.2 logs",
			produceAudit2:     toPtr(true),
			wantAudit2Entries: 1,
		},
		{
			name:              "false ProduceAudit2Logs suppresses audit.2 logs",
			produceAudit2:     toPtr(false),
			wantAudit2Entries: 0,
		},
		{
			name:              "false ProduceAudit2Logs suppresses dual-logged audit.3-to-audit.2 entries",
			produceAudit2:     toPtr(false),
			logAuditV3:        true,
			enableDualV3ToV2:  true,
			wantAudit2Entries: 0,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var logOutputBuffer bytes.Buffer
			port, err := httpserver.AvailablePort()
			require.NoError(t, err)

			runtimeCfg := config.Runtime{
				LoggerConfig: &config.LoggerConfig{Level: wlog.InfoLevel},
				AuditConfig: &config.AuditConfig{
					Deployment:        "test-deployment",
					Product:           productName,
					ProductVersion:    productVersion,
					ProduceAudit2Logs: tc.produceAudit2,
				},
			}
			runtimeCfgYML, err := yaml.Marshal(runtimeCfg)
			require.NoError(t, err)

			server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port,
				func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
					return nil, info.Router.Register("GET", "/testAuditLog", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if tc.logAuditV3 {
							audit3log.FromContext(r.Context()).Audit("LogToAudit3", audit3log.AuditResultSuccess)
						} else {
							audit2log.FromContext(r.Context()).Audit("LogToAudit2", audit2log.AuditResultSuccess)
						}
						w.WriteHeader(http.StatusOK)
					}))
				},
				&logOutputBuffer,
				func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
					srv := createTestServerWithRuntimeConfigProvider(initFn, installCfg, logOutputBuffer, refreshable.New(runtimeCfgYML))
					if tc.enableDualV3ToV2 {
						srv = srv.WithEnableDualLogAuditV3ToAuditV2()
					}
					return srv
				},
			)
			defer func() { require.NoError(t, server.Close()) }()
			defer cleanup()

			client := testServerClient()
			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/testAuditLog", port, basePath), nil)
			require.NoError(t, err)
			_, err = client.Do(request)
			require.NoError(t, err)

			audit2LogEntries := extractLogEntries[logging.AuditLogV2](t, logOutputBuffer.String(), "audit.2")
			assert.Equal(t, tc.wantAudit2Entries, len(audit2LogEntries), "unexpected number of audit.2 log entries")

			select {
			case err := <-serverErr:
				require.NoError(t, err)
			default:
			}
		})
	}

	t.Run("live reload toggles audit.2 emission", func(t *testing.T) {
		var logOutputBuffer bytes.Buffer
		port, err := httpserver.AvailablePort()
		require.NoError(t, err)

		tmpDir := t.TempDir()
		runtimeCfg := config.Runtime{
			LoggerConfig: &config.LoggerConfig{Level: wlog.InfoLevel},
			AuditConfig: &config.AuditConfig{
				Deployment:        "test-deployment",
				Product:           productName,
				ProductVersion:    productVersion,
				ProduceAudit2Logs: nil, // audit.2 logs should be emitted by default
			},
		}
		runtimeCfgYML, err := yaml.Marshal(runtimeCfg)
		require.NoError(t, err)

		runtimeConfigPath := filepath.Join(tmpDir, "runtime.yml")
		err = os.WriteFile(runtimeConfigPath, runtimeCfgYML, 0644)
		require.NoError(t, err)

		fileRefreshable := refreshable.NewFileRefreshable(context.Background(), runtimeConfigPath)
		_, err = fileRefreshable.Validation()
		require.NoError(t, err)
		fileRefreshableR, _ := refreshable.MapFromValidated(fileRefreshable, func(b []byte) []byte { return b })

		server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port,
			func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (func(), error) {
				return nil, info.Router.Register("GET", "/testAuditLog", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					audit2log.FromContext(r.Context()).Audit("LogToAudit2", audit2log.AuditResultSuccess)
					w.WriteHeader(http.StatusOK)
				}))
			},
			&logOutputBuffer,
			func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
				return createTestServerWithRuntimeConfigProvider(initFn, installCfg, logOutputBuffer, fileRefreshableR)
			},
		)
		defer func() { require.NoError(t, server.Close()) }()
		defer cleanup()

		client := testServerClient()

		// First request: audit.2 enabled
		request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/testAuditLog", port, basePath), nil)
		require.NoError(t, err)
		_, err = client.Do(request)
		require.NoError(t, err)

		audit2LogEntries := extractLogEntries[logging.AuditLogV2](t, logOutputBuffer.String(), "audit.2")
		assert.Equal(t, 1, len(audit2LogEntries), "audit.2 should be emitted when ProduceAudit2Logs is nil")

		// Update runtime config to disable audit.2 logs
		produceAudit2False := false
		runtimeCfg.AuditConfig.ProduceAudit2Logs = &produceAudit2False
		runtimeCfgYML, err = yaml.Marshal(runtimeCfg)
		require.NoError(t, err)
		err = os.WriteFile(runtimeConfigPath, runtimeCfgYML, 0644)
		require.NoError(t, err)

		// Wait for runtime config to reload
		time.Sleep(1000 * time.Millisecond)

		// Second request: audit.2 should now be suppressed
		_, err = client.Do(request)
		require.NoError(t, err)

		audit2LogEntriesAfter := extractLogEntries[logging.AuditLogV2](t, logOutputBuffer.String(), "audit.2")
		assert.Equal(t, 1, len(audit2LogEntriesAfter), "audit.2 should be suppressed after ProduceAudit2Logs set to false")

		select {
		case err := <-serverErr:
			require.NoError(t, err)
		default:
		}
	})
}

// Helper function that starts a server, logs audit entries, and verifies the log output based on the provided
// parameters.
func testAuditLogHelper(t *testing.T, logAuditV2, dualLogAuditV2ToAuditV3, logAuditV3, dualLogAuditV3ToV2, inServerInit bool) {
	t.Run(fmt.Sprintf("Test audit log output: logAuditV2=%t dualLogAuditV2ToAuditV3=%t logAuditV3=%t dualLogAuditV3ToV2=%t inServerInit=%t", logAuditV2, dualLogAuditV2ToAuditV3, logAuditV3, dualLogAuditV3ToV2, inServerInit), func(t *testing.T) {
		logOutputBuffer := &bytes.Buffer{}
		port, err := httpserver.AvailablePort()
		require.NoError(t, err)

		searchResultsVal := []commonv2.RequestResource{
			{
				Id: commonv2.NewIdentifierFromRid(rid.MustNew("service", "instance", "resource-type", uuid.NewUUID().String())),
				Context: []commonv2.ResourceContext{
					{
						Value:       "test-context-value",
						Description: "test-context-description",
					},
				},
			},
		}
		searchResultsValInterface := jsonRoundTrip(t, searchResultsVal)

		categoryVal := v2.NewAuditCategoryV2FromRequestSearch(
			v2.RequestSearch{
				RequestSearchQuery:   toPtr("test-search-query"),
				RequestSearchResults: searchResultsVal,
			},
		)
		categoryValInterface := jsonRoundTrip(t, categoryVal)

		metrics.DefaultMetricsRegistry = metrics.NewRootMetricsRegistry()
		server, serverErr, cleanup := createAndRunCustomTestServer(t, port, port, func(ctx context.Context, info witchcraft.InitInfo[config.Install, config.Runtime]) (deferFn func(), rErr error) {
			emitAuditLogs := func(ctx context.Context) {
				if logAuditV2 {
					audit2log.FromContext(ctx).Audit("LogToAudit2", audit2log.AuditResultSuccess)
				}
				if logAuditV3 {
					audit3log.FromContext(ctx).Audit("LogToAudit3", audit3log.AuditResultSuccess,
						audit3log.Category(categoryVal),
					)
				}
			}

			if inServerInit {
				// if "inServerInit" is true, emit audit logs in the server init phase. Verifies that audit logging is
				// properly set up on the context used in server initialization.
				emitAuditLogs(ctx)
			} else {
				// if "inServerInit" is false, emit audit logs as a result of calling the endpoint. Verifies that audit
				// logging is properly set up on the context used in requests.
				if err := info.Router.Register("GET", "/testAuditLog", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					emitAuditLogs(r.Context())
					w.WriteHeader(http.StatusOK)
				})); err != nil {
					return nil, err
				}
			}

			return nil, nil
		}, logOutputBuffer, func(t *testing.T, initFn witchcraft.InitFunc[config.Install, config.Runtime], installCfg config.Install, logOutputBuffer io.Writer) *witchcraft.Server[config.Install, config.Runtime] {
			installCfg.MetricsEmitFrequency = 25 * time.Millisecond
			server := createTestServer(t, initFn, installCfg, logOutputBuffer)
			if dualLogAuditV2ToAuditV3 {
				server = server.WithEnableDualLogAuditV2ToAuditV3()
			}
			if dualLogAuditV3ToV2 {
				server = server.WithEnableDualLogAuditV3ToAuditV2()
			}
			return server
		})
		defer func() {
			require.NoError(t, server.Close())
		}()
		defer cleanup()

		// Allow startup
		time.Sleep(150 * time.Millisecond)

		if !inServerInit {
			// Make a request to trigger the audit log entries
			client := testServerClient()

			request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://localhost:%d/%s/testAuditLog", port, basePath), nil)
			require.NoError(t, err)
			request.Header.Add("X-Forwarded-For", forwardedForVal0)
			request.Header.Add("X-Forwarded-For", forwardedForVal1)
			request.Header.Add("X-Forwarded-For", forwardedForVal2)
			request.Header.Set("Authorization", "Bearer "+testJWTToken)
			_, err = client.Do(request)
			require.NoError(t, err)
		}

		// allow metric emitter to run
		time.Sleep(50 * time.Millisecond)

		audit2LogEntries := extractLogEntries[logging.AuditLogV2](t, logOutputBuffer.String(), "audit.2")
		audit3LogEntries := extractLogEntries[logging.AuditLogV3](t, logOutputBuffer.String(), "audit.3")
		metricLogEntries := extractLogEntries[logging.MetricLogV1](t, logOutputBuffer.String(), "metric.1")

		wantNumAudit2LogEntries := 0
		wantNumAudit3LogEntries := 0

		if logAuditV2 {
			wantNumAudit2LogEntries++
			if dualLogAuditV2ToAuditV3 {
				wantNumAudit3LogEntries++
			}
		}
		if logAuditV3 {
			wantNumAudit3LogEntries++
			if dualLogAuditV3ToV2 {
				wantNumAudit2LogEntries++
			}
		}

		assert.Equal(t, wantNumAudit2LogEntries, len(audit2LogEntries))
		assert.Equal(t, wantNumAudit2LogEntries, getNumEntriesFromMetricLogs(metricLogEntries, "audit.2"))
		assert.Equal(t, wantNumAudit3LogEntries, len(audit3LogEntries))
		assert.Equal(t, wantNumAudit3LogEntries, getNumEntriesFromMetricLogs(metricLogEntries, "audit.3"))

		auditLog2Idx := 0
		auditLog3Idx := 0
		if logAuditV2 {
			assert.Equal(t, "LogToAudit2", audit2LogEntries[auditLog2Idx].Name)
			auditLog2Idx++
			if dualLogAuditV2ToAuditV3 {
				audit3LogEntry := audit3LogEntries[auditLog3Idx]
				assert.Equal(t, "LogToAudit2", audit3LogEntry.Name)

				// all audit.3 log entries should have an entry ID (even when dual-logged)
				logEntryIDValue := uuidPtrValue(audit3LogEntry.LogEntryId)
				assert.Regexp(t, uuidRegexp, logEntryIDValue)

				// dual-logged entries do not inherit from request context: they are translated directly from audit.2
				assert.Nil(t, audit3LogEntry.Origin)
				assert.Nil(t, audit3LogEntry.SourceOrigin)
				assert.Nil(t, audit3LogEntry.UserAgent)
				assert.Equal(t, "00000000-0000-0000-0000-000000000000", audit3LogEntry.EventId.String())
				auditLog3Idx++
			}
		}

		if logAuditV3 {
			audit3LogEntry := audit3LogEntries[auditLog3Idx]

			assert.Equal(t, "LogToAudit3", audit3LogEntry.Name)
			logEntryIDValue := uuidPtrValue(audit3LogEntry.LogEntryId)
			assert.Regexp(t, uuidRegexp, logEntryIDValue)
			assert.Equal(t, []string{"requestSearch"}, audit3LogEntry.Categories)
			assert.Equal(t, "test-search-query", audit3LogEntry.RequestFields["requestSearchQuery"])
			assert.Equal(t, searchResultsValInterface, audit3LogEntry.ResultFields["requestSearchResults"])
			assert.Equal(t, "test-deployment", audit3LogEntry.Deployment)
			assert.Equal(t, productName, audit3LogEntry.Product)
			assert.Equal(t, productVersion, audit3LogEntry.ProductVersion)
			assert.Equal(t, "test-stack", stringPtrValue(audit3LogEntry.Stack))
			assert.Equal(t, "test-service", stringPtrValue(audit3LogEntry.Service))
			assert.Equal(t, "test-environment", stringPtrValue(audit3LogEntry.Environment))

			sourceOriginValue := stringPtrValue(audit3LogEntry.SourceOrigin)
			userAgentValue := stringPtrValue(audit3LogEntry.UserAgent)
			auditEventIDValue := audit3LogEntry.EventId.String()
			originsValue := audit3LogEntry.Origins
			tokenIDValue := stringPtrValue(audit3LogEntry.TokenId)
			uidValue := stringPtrValue(audit3LogEntry.Uid)
			sidValue := stringPtrValue(audit3LogEntry.Sid)
			orgIDValue := stringPtrValue(audit3LogEntry.OrgId)
			originValue := stringPtrValue(audit3LogEntry.Origin)

			if !inServerInit {
				// params set by request context
				assert.Equal(t, "3dd6434d-79a9-4d15-98b5-7b51dbb2cd31", tokenIDValue)
				assert.Equal(t, "jrocket@example.com", uidValue)
				assert.Equal(t, "eecb9bf34bbb4c8eb87dbba3aa1523c6", sidValue)
				assert.Equal(t, "org_vIK75NKFvaozQsFy", orgIDValue)
				assert.Equal(t, []logging.ContextualizedUser{
					{
						Uid:    "jrocket@example.com",
						Groups: []string{},
					},
				}, audit3LogEntry.Users)
				assert.Regexp(t, ".+", originValue)
				assert.Regexp(t, ".+", sourceOriginValue)
				assert.Equal(t, "Go-http-client/1.1", userAgentValue)
				assert.Regexp(t, uuidRegexp, auditEventIDValue)
				assert.Equal(t, []string{"203.0.113.195", "70.41.3.18", "150.172.238.178"}, originsValue)
			} else {
				// params not set when using server init context (which is not associated with a request)
				assert.Nil(t, audit3LogEntry.TokenId)
				assert.Nil(t, audit3LogEntry.Uid)
				assert.Nil(t, audit3LogEntry.Sid)
				assert.Nil(t, audit3LogEntry.OrgId)
				assert.Nil(t, audit3LogEntry.Origin)
				assert.Nil(t, audit3LogEntry.SourceOrigin)
				assert.Nil(t, audit3LogEntry.UserAgent)
				assert.Equal(t, "00000000-0000-0000-0000-000000000000", auditEventIDValue)
				assert.Empty(t, originsValue)
			}

			if dualLogAuditV3ToV2 {
				audit2LogEntry := audit2LogEntries[auditLog2Idx]
				assert.Equal(t, "LogToAudit3", audit2LogEntry.Name)
				assert.Equal(t, logEntryIDValue, audit2LogEntry.RequestParams["_auditLogEntryId"])
				assert.Equal(t, categoryValInterface, audit2LogEntry.RequestParams["_category"])

				if !inServerInit {
					assert.Equal(t, tokenIDValue, stringPtrValue(audit2LogEntry.TokenId))
					assert.Equal(t, uidValue, stringPtrValue(audit2LogEntry.Uid))
					assert.Equal(t, sidValue, stringPtrValue(audit2LogEntry.Sid))
					assert.Equal(t, orgIDValue, stringPtrValue(audit2LogEntry.OrgId))
					assert.Equal(t, originValue, stringPtrValue(audit2LogEntry.Origin))

					assert.Equal(t, sourceOriginValue, audit2LogEntry.RequestParams["_sourceOrigin"])
					assert.Equal(t, originsValue, toStringSlice(audit2LogEntry.RequestParams["_forwardedOrigins"]))
					assert.Equal(t, tokenIDValue, audit2LogEntry.RequestParams["_tokenId"])
					assert.Equal(t, userAgentValue, audit2LogEntry.RequestParams["_userAgent"])
					assert.Equal(t, auditEventIDValue, audit2LogEntry.RequestParams["_auditEventId"])
				} else {
					assert.Nil(t, audit2LogEntry.TokenId)
					assert.Nil(t, audit2LogEntry.Uid)
					assert.Nil(t, audit2LogEntry.Sid)
					assert.Nil(t, audit2LogEntry.OrgId)
					assert.Nil(t, audit2LogEntry.Origin)

					assert.True(t, mapEntryNotPresent(audit2LogEntry.RequestParams, "_sourceOrigin"))
					assert.True(t, mapEntryNotPresent(audit2LogEntry.RequestParams, "_forwardedOrigins"))
					assert.True(t, mapEntryNotPresent(audit2LogEntry.RequestParams, "_tokenId"))
					assert.True(t, mapEntryNotPresent(audit2LogEntry.RequestParams, "_userAgent"))
					assert.True(t, mapEntryNotPresent(audit2LogEntry.RequestParams, "_auditEventId"))
				}
			}
		}

		select {
		case err := <-serverErr:
			require.NoError(t, err)
		default:
		}
	})
}

func sortMetricByCountValueDescending(a, b logging.MetricLogV1) int {
	aCountNum, ok := a.Values["count"].(json.Number)
	if !ok {
		return 0
	}
	bCountNum, ok := b.Values["count"].(json.Number)
	if !ok {
		return 0
	}
	var (
		aCountVal, bCountVal int64
		err                  error
	)
	aCountVal, err = aCountNum.Int64()
	if err != nil {
		return 0
	}
	bCountVal, err = bCountNum.Int64()
	if err != nil {
		return 0
	}
	if aCountVal < bCountVal {
		return 1
	} else if aCountVal > bCountVal {
		return -1
	}
	return 0
}

func extractLogEntries[LogT any](t *testing.T, loggerOutput, logTypeName string) []LogT {
	var logEntries []LogT
	parts := strings.Split(loggerOutput, "\n")
	for _, curr := range parts {
		typedLogEntry := struct {
			Type string `json:"type"`
		}{}
		if err := json.Unmarshal([]byte(curr), &typedLogEntry); err != nil {
			continue
		}
		if typedLogEntry.Type != logTypeName {
			continue
		}
		var currLogEntry LogT
		require.NoError(t, json.Unmarshal([]byte(curr), &currLogEntry))
		logEntries = append(logEntries, currLogEntry)
	}
	return logEntries
}

func getNumEntriesFromMetricLogs(metricLogEntries []logging.MetricLogV1, tagType string) int {
	matchingMetricLogEntries := filterMatchingLogEntries(metricLogEntries, func(entry logging.MetricLogV1) bool {
		return entry.MetricName == "logging.sls" && entry.Tags["type"] == tagType
	})
	slices.SortFunc(matchingMetricLogEntries, sortMetricByCountValueDescending)
	return getMetricLogMaxCount(matchingMetricLogEntries)
}

func getMetricLogMaxCount(metricLogEntries []logging.MetricLogV1) int {
	maxVal := int64(0)
	for _, entry := range metricLogEntries {
		countNum, ok := entry.Values["count"].(json.Number)
		if !ok {
			return 0
		}
		countVal, err := countNum.Int64()
		if err != nil {
			return 0
		}
		maxVal = max(maxVal, countVal)
	}
	return int(maxVal)
}

func filterMatchingLogEntries[LogT any](entries []LogT, matcher func(LogT) bool) []LogT {
	var logEntries []LogT
	for _, currLogEntry := range entries {
		if matcher(currLogEntry) {
			logEntries = append(logEntries, currLogEntry)
		}
	}
	return logEntries
}

func toPtr[T any](in T) *T {
	return &in
}

func stringPtrValue[T ~string](in *T) string {
	if in == nil {
		return ""
	}
	return (string)(*in)
}

func uuidPtrValue(in *uuid.UUID) string {
	if in == nil {
		return ""
	}
	return in.String()
}

func mapEntryNotPresent(in map[string]any, key string) bool {
	_, ok := in[key]
	return !ok
}

func toStringSlice(in any) []string {
	anySlice := in.([]any)
	out := make([]string, len(anySlice))
	for i, v := range anySlice {
		out[i] = v.(string)
	}
	return out
}

func jsonRoundTrip(t *testing.T, in any) any {
	jsonBytes, err := json.Marshal(in)
	require.NoError(t, err)

	var out any
	err = json.Unmarshal(jsonBytes, &out)
	require.NoError(t, err)
	return out
}
