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

package tcpjson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"testing"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-health/conjure/witchcraft/api/health"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testMetadata = LogEnvelopeMetadata{
		Type:           "envelope.1",
		Deployment:     "test-deployment",
		Environment:    "test-environment",
		EnvironmentID:  "test-environment-id",
		Host:           "test-host",
		NodeID:         "test-node-id",
		Product:        "test-product",
		ProductVersion: "test-product-version",
		Service:        "test-service",
		ServiceID:      "test-service-id",
		Stack:          "test-stack",
		StackID:        "test-stack-id",
	}
	logPayload = []byte(`{"type": "service.1","message":"test","level":"INFO"}\n`)
)

func TestWrite(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload []byte
	}{
		{"payload-with-newline", logPayload},
		{"payload-without-newline", bytes.TrimSuffix(logPayload, []byte("\n"))},
		{"no-payload", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expectedEnvelope := getEnvelopeBytes(t, tc.payload)
			provider := new(bufferedConnProvider)
			tcpWriter := NewTCPWriter(testMetadata, provider)
			n, err := tcpWriter.Write(tc.payload)
			require.NoError(t, err)
			require.Equal(t, len(tc.payload), n)
			buf := provider.buffer.Bytes()
			require.True(t, bytes.Equal(buf, expectedEnvelope))
		})
	}
}

// TestWriteFromSvc1log is more of an integration style test which verifies the envelopes written
// are as expected when using the TCPWriter as an io.Writer for svc1log.
func TestWriteFromSvc1log(t *testing.T) {
	provider := new(bufferedConnProvider)
	tcpWriter := NewTCPWriter(testMetadata, provider)
	logger := svc1log.NewFromCreator(tcpWriter, wlog.DebugLevel, wlog.NewJSONMarshalLoggerProvider().NewLeveledLogger)
	logger.Debug("this is a test")

	buf := provider.buffer.Bytes()
	var gotEnvelope LogEnvelopeV1
	err := json.Unmarshal(buf, &gotEnvelope)
	require.NoError(t, err)

	// Verify all envelope metadata
	assert.Equal(t, testMetadata.Type, gotEnvelope.Type)
	assert.Equal(t, testMetadata.Deployment, gotEnvelope.Deployment)
	assert.Equal(t, testMetadata.Environment, gotEnvelope.Environment)
	assert.Equal(t, testMetadata.EnvironmentID, gotEnvelope.EnvironmentID)
	assert.Equal(t, testMetadata.Host, gotEnvelope.Host)
	assert.Equal(t, testMetadata.NodeID, gotEnvelope.NodeID)
	assert.Equal(t, testMetadata.Product, gotEnvelope.Product)
	assert.Equal(t, testMetadata.ProductVersion, gotEnvelope.ProductVersion)
	assert.Equal(t, testMetadata.Service, gotEnvelope.Service)
	assert.Equal(t, testMetadata.ServiceID, gotEnvelope.ServiceID)
	assert.Equal(t, testMetadata.Stack, gotEnvelope.Stack)
	assert.Equal(t, testMetadata.StackID, gotEnvelope.StackID)

	// Verify the payload
	gotPayload := new(logging.ServiceLogV1)
	err = gotPayload.UnmarshalJSON(gotEnvelope.Payload)
	require.NoError(t, err)
	assert.Equal(t, "this is a test", gotPayload.Message)
	assert.Equal(t, logging.LogLevelDebug, gotPayload.Level)
}

// TestClosedWriter verifies the behavior of attempting to write when the writer is closed.
func TestClosedWriter(t *testing.T) {
	provider := new(bufferedConnProvider)
	tcpWriter := NewTCPWriter(testMetadata, provider)

	n, err := tcpWriter.Write(logPayload)
	require.NoError(t, err)
	require.Equal(t, len(logPayload), n)

	err = tcpWriter.Close()
	require.NoError(t, err)

	// Attempt a write and expect that the writer is closed
	n, err = tcpWriter.Write(logPayload)
	require.Error(t, err)
	require.EqualError(t, err, errWriterClosed)
	require.True(t, n == 0)
}

func TestHealthStatus(t *testing.T) {
	for _, tc := range []struct {
		name          string
		tcpWriterFunc func() *TCPWriter
		expected      health.HealthState_Value
	}{
		{
			name: "not started",
			tcpWriterFunc: func() *TCPWriter {
				return NewTCPWriter(LogEnvelopeMetadata{}, &bufferedConnProvider{})
			},
			expected: health.HealthState_HEALTHY,
		},
		{
			name: "shutting down",
			tcpWriterFunc: func() *TCPWriter {
				w := NewTCPWriter(LogEnvelopeMetadata{}, &bufferedConnProvider{})
				_ = w.Close()
				return w
			},
			expected: health.HealthState_HEALTHY,
		},
		{
			name: "established connection",
			tcpWriterFunc: func() *TCPWriter {
				w := NewTCPWriter(LogEnvelopeMetadata{}, &bufferedConnProvider{})
				_, err := w.Write(logPayload)
				require.NoError(t, err)
				return w
			},
			expected: health.HealthState_HEALTHY,
		},
		{
			name: "failed to get connection",
			tcpWriterFunc: func() *TCPWriter {
				w := NewTCPWriter(LogEnvelopeMetadata{}, &failingConnProvider{err: fmt.Errorf("error")})
				_, err := w.Write(logPayload)
				assert.Error(t, err)
				return w
			},
			expected: health.HealthState_ERROR,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tcpWriter := tc.tcpWriterFunc()
			gotHealthState := tcpWriter.HealthStatus(context.Background())
			assert.NotNil(t, gotHealthState)
			assert.Len(t, gotHealthState.Checks, 1)
			assert.Equal(t, tc.expected, gotHealthState.Checks[tcpWriterHealthCheckName].State.Value())
		})
	}
}

func getEnvelopeBytes(t *testing.T, payload []byte) []byte {
	envelope, err := zerologSerializer(testMetadata)(payload)
	require.NoError(t, err)
	return envelope
}

// bufferedConnProvider is a mock ConnProvider that writes to an internal
// bytes buffer instead of to the net.Conn.
type bufferedConnProvider struct {
	net.Conn
	err    error
	buffer bytes.Buffer
}

func (t *bufferedConnProvider) GetConn() (net.Conn, error) {
	return t, nil
}

func (t *bufferedConnProvider) Write(d []byte) (int, error) {
	return t.buffer.Write(d)
}

func (t *bufferedConnProvider) Close() error {
	return t.err
}

type failingConnProvider struct {
	err error
}

func (t *failingConnProvider) GetConn() (net.Conn, error) {
	return nil, t.err
}

// BenchmarkEnvelopeSerializer records the total time and memory allocations for each envelope serializer.
func BenchmarkEnvelopeSerializer(b *testing.B) {
	for _, tc := range []struct {
		name           string
		serializerFunc envelopeSerializerFunc
	}{
		{"zerolog", zerologSerializer(testMetadata)},
		{"JSON-Encoder", jsonEncoderSerializer(testMetadata)},
		{"JSON-Marshaler", jsonMarshalSerializer(testMetadata)},
		{"manual", manualSerializer(testMetadata)},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				_, _ = tc.serializerFunc(logPayload)
			}
		})
	}
}

// jsonEncoderSerializer returns an envelopeSerializerFunc that uses the json.Encoder to serialize the envelope.
func jsonEncoderSerializer(metadata LogEnvelopeMetadata) envelopeSerializerFunc {
	return func(p []byte) ([]byte, error) {
		var buf bytes.Buffer
		envelopeToWrite := getEnvelopeWithPayload(metadata, p)
		if err := json.NewEncoder(&buf).Encode(&envelopeToWrite); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
}

// jsonMarshalSerializer returns an envelopeSerializerFunc that uses the json.Marshal to serialize the envelope.
func jsonMarshalSerializer(metadata LogEnvelopeMetadata) envelopeSerializerFunc {
	return func(p []byte) ([]byte, error) {
		envelopeToWrite := getEnvelopeWithPayload(metadata, p)
		b, err := json.Marshal(&envelopeToWrite)
		if err != nil {
			return nil, err
		}
		return append(b, '\n'), nil
	}
}

// manualSerializer returns an envelopeSerializerFunc that manually injects the payload.
func manualSerializer(metadata LogEnvelopeMetadata) envelopeSerializerFunc {
	metadataJSON, _ := jsonEncoderSerializer(metadata)(nil)
	return func(p []byte) ([]byte, error) {
		// manually inject the payload into the metadataJSON
		idx := bytes.LastIndexByte(metadataJSON, '}')
		if idx == -1 {
			return nil, werror.Error("invalid JSON")
		}
		envelope := bytes.NewBuffer(metadataJSON[:idx])
		envelope.Write([]byte(`,"payload":`))
		envelope.Write(p)
		envelope.Write([]byte(`}\n`))
		return envelope.Bytes(), nil
	}
}

func getEnvelopeWithPayload(metadata LogEnvelopeMetadata, payload []byte) LogEnvelopeV1 {
	return LogEnvelopeV1{
		LogEnvelopeMetadata: LogEnvelopeMetadata{
			Type:           "envelope.1",
			Deployment:     metadata.Deployment,
			Environment:    metadata.Environment,
			EnvironmentID:  metadata.EnvironmentID,
			Host:           metadata.Host,
			NodeID:         metadata.NodeID,
			Service:        metadata.Service,
			ServiceID:      metadata.ServiceID,
			Stack:          metadata.Stack,
			StackID:        metadata.StackID,
			Product:        metadata.Product,
			ProductVersion: metadata.ProductVersion,
		},
		Payload: payload,
	}
}
