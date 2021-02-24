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

package tcpjson_test

import (
	"bytes"
	"encoding/json"
	"net"
	"testing"

	"github.com/palantir/witchcraft-go-server/v2/witchcraft/internal/tcpjson"
	"github.com/stretchr/testify/require"
)

var (
	testMetadata = tcpjson.LogEnvelopeMetadata{
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
	tcpWriter := tcpjson.NewTCPWriter(testMetadata, provider)
	n, err := tcpWriter.Write(logPayload)
	require.NoError(t, err)
	require.Equal(t, len(expectedEnvelope), n)

	require.True(t, bytes.Equal(provider.buffer.Bytes(), expectedEnvelope))
}

// TestClosedWriter verifies the behavior of attempting to write when the writer is closed.
func TestClosedWriter(t *testing.T) {
	expectedEnvelope := getEnvelopeBytes(t, logPayload)

	provider := new(bufferedConnProvider)
	tcpWriter := tcpjson.NewTCPWriter(testMetadata, provider)

	n, err := tcpWriter.Write(logPayload)
	require.Error(t, err)
	require.Equal(t, len(expectedEnvelope), n)

	err = tcpWriter.Close()
	require.NoError(t, err)

	// Attempt a write and expect that the writer is closed
	n, err = tcpWriter.Write(logPayload)
	require.Error(t, err)
	require.EqualError(t, err, tcpjson.ErrWriterClosed)
	require.True(t, n == 0)
}

func getEnvelopeBytes(t *testing.T, payload []byte) []byte {
	e := tcpjson.SlsEnvelopeV1{
		Type:           "envelope.1",
		Deployment:     testMetadata.Deployment,
		Environment:    testMetadata.Environment,
		EnvironmentID:  testMetadata.EnvironmentID,
		Host:           testMetadata.Host,
		NodeID:         testMetadata.NodeID,
		Service:        testMetadata.Service,
		ServiceID:      testMetadata.ServiceID,
		Stack:          testMetadata.Stack,
		StackID:        testMetadata.StackID,
		Product:        testMetadata.Product,
		ProductVersion: testMetadata.ProductVersion,
		Payload:        payload,
	}
	b, err := json.Marshal(&e)
	require.NoError(t, err)
	return b
}

// bufferedConnProvider is a mock ConnProvider that writes to an internal
// bytes buffer instead of to the net.Conn.
type bufferedConnProvider struct {
	net.Conn
	buffer bytes.Buffer
}

func (t *bufferedConnProvider) GetConn() (net.Conn, error) {
	return t, nil
}

func (t *bufferedConnProvider) Write(d []byte) (int, error) {
	return t.buffer.Write(d)
}
