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
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	testMetadata = LogEnvelopeMetadata{
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
	logPayload = []byte(`{"type": "service.1","message":"test","level":"INFO"}`)
)

func TestWrite(t *testing.T) {
	expectedEnvelope := getEnvelopeBytes(t, logPayload)

	provider := new(bufferedConnProvider)
	tcpWriter := NewTCPWriter(testMetadata, provider)
	n, err := tcpWriter.Write(logPayload)
	require.NoError(t, err)
	require.Equal(t, len(logPayload), n)

	require.True(t, bytes.Equal(provider.buffer.Bytes(), expectedEnvelope))
}

// TestClosedWriter verifies the behavior of attempting to write when the writer is closed.
func TestClosedWriter(t *testing.T) {
	expectedEnvelope := getEnvelopeBytes(t, logPayload)

	provider := new(bufferedConnProvider)
	tcpWriter := NewTCPWriter(testMetadata, provider)

	n, err := tcpWriter.Write(logPayload)
	require.NoError(t, err)
	require.Equal(t, len(expectedEnvelope), n)

	err = tcpWriter.Close()
	require.NoError(t, err)

	// Attempt a write and expect that the writer is closed
	n, err = tcpWriter.Write(logPayload)
	require.Error(t, err)
	require.EqualError(t, err, errWriterClosed)
	require.True(t, n == 0)
}

// BenchmarkEnvelopeSerializer records the total time and memory allocations for each envelope serializer.
func BenchmarkEnvelopeSerializer(b *testing.B) {
	for _, tc := range []struct {
		name           string
		serializerFunc func([]byte) ([]byte, error)
	}{
		{"zerolog", zerologSerializer(testMetadata)},
		{"JSON-Encoder", jsonEncoderSerializer(testMetadata)},
		{"JSON-Marshaler", jsonMarshalSerializer(testMetadata)},
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
