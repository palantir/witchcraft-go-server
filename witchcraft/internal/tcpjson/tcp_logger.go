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
	"encoding/json"
	"io"
	"net"
	"sync"

	werror "github.com/palantir/witchcraft-go-error"
)

var _ io.Writer = (*TCPWriter)(nil)

const ErrWriterClosed = "writer is closed"

// TCPWriter writes logs to a TCP socket and wraps them with envelope metadata.
type TCPWriter struct {
	metadata LogEnvelopeMetadata
	provider ConnProvider

	// closedChan is used to signal that the writer is shutting down
	closedChan chan struct{}

	mu   sync.RWMutex // guards conn below
	conn net.Conn
}

func NewTCPWriter(metadata LogEnvelopeMetadata, provider ConnProvider) *TCPWriter {
	return &TCPWriter{
		metadata:   metadata,
		provider:   provider,
		closedChan: make(chan struct{}),
		conn:       nil,
	}
}

func (d *TCPWriter) Write(logPayload []byte) (int, error) {
	if d.closed() {
		return 0, werror.Error(ErrWriterClosed)
	}
	envelopeToWrite := SlsEnvelopeV1{
		Type:           "envelope.1",
		Deployment:     d.metadata.Deployment,
		Environment:    d.metadata.Environment,
		EnvironmentID:  d.metadata.EnvironmentID,
		Host:           d.metadata.Host,
		NodeID:         d.metadata.NodeID,
		Service:        d.metadata.Service,
		ServiceID:      d.metadata.ServiceID,
		Stack:          d.metadata.Stack,
		StackID:        d.metadata.StackID,
		Product:        d.metadata.Product,
		ProductVersion: d.metadata.ProductVersion,
		Payload:        logPayload,
	}
	b, err := json.Marshal(envelopeToWrite)
	if err != nil {
		return 0, err
	}

	conn, err := d.getConn()
	if err != nil {
		return 0, err
	}

	var total int
	for total < len(b) {
		n, err := conn.Write(b[total:])
		total += n
		if err != nil {
			if nerr, ok := err.(net.Error); !(ok && (nerr.Temporary() || nerr.Timeout())) {
				// permanent error so close the connection
				return total, d.closeConn()
			}
			return total, err
		}
	}
	return total, nil
}

func (d *TCPWriter) getConn() (net.Conn, error) {
	// Fast path: connection is already established and cached
	d.mu.RLock()
	conn := d.conn
	d.mu.RUnlock()
	if d.conn != nil {
		return conn, nil
	}

	// No active connection, so use the provider to get a new net.Conn, and cache it.
	d.mu.Lock()
	defer d.mu.Unlock()
	newConn, err := d.provider.GetConn()
	if err != nil {
		return nil, err
	}
	d.conn = newConn
	return newConn, nil
}

func (d *TCPWriter) closeConn() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.conn != nil {
		err := d.conn.Close()
		d.conn = nil
		return err
	}
	return nil
}

func (d *TCPWriter) closed() bool {
	select {
	case <-d.closedChan:
		return true
	default:
		return false
	}
}

// Close will close any existing client connections and shuts down the writer from any future writes.
func (d *TCPWriter) Close() error {
	close(d.closedChan)
	return d.closeConn()
}
