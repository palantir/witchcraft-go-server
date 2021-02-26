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
	"io"
	"io/ioutil"
	"net"
	"sync"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/rs/zerolog"
)

var _ io.Writer = (*TCPWriter)(nil)

const errWriterClosed = "writer is closed"

// envelopeSerializerFunc provides a way to change the serialization method for the provided payload.
type envelopeSerializerFunc func(payload []byte) ([]byte, error)

// TCPWriter writes logs to a TCP socket and wraps them with envelope metadata.
type TCPWriter struct {
	provider           ConnProvider
	envelopeSerializer envelopeSerializerFunc

	// closedChan is used to signal that the writer is shutting down
	closedChan chan struct{}

	mu   sync.RWMutex // guards conn below
	conn net.Conn
}

func NewTCPWriter(metadata LogEnvelopeMetadata, provider ConnProvider) *TCPWriter {
	return newTCPWriterInternal(provider, zerologSerializer(metadata))
}

func newTCPWriterInternal(provider ConnProvider, serializerFunc envelopeSerializerFunc) *TCPWriter {
	return &TCPWriter{
		envelopeSerializer: serializerFunc,
		provider:           provider,
		closedChan:         make(chan struct{}),
		conn:               nil,
	}
}

func (d *TCPWriter) Write(p []byte) (int, error) {
	if d.closed() {
		return 0, werror.Error(errWriterClosed)
	}

	// remove the trailing new-line delimiter if it exists before wrapping in the envelope
	envelope, err := d.envelopeSerializer(trimNewLine(p))
	if err != nil {
		return 0, werror.Wrap(err, "failed to serialize the envelope")
	}

	conn, err := d.getConn()
	if err != nil {
		return 0, err
	}

	var total int
	for total < len(envelope) {
		n, err := conn.Write(envelope[total:])
		total += n
		if err != nil {
			if nerr, ok := err.(net.Error); !(ok && (nerr.Temporary() || nerr.Timeout())) {
				// permanent error so close the connection
				_ = d.closeConn()
				return total, err
			}
			return total, err
		}
	}
	return len(p), nil
}

// trimNewLine will remove a trailing new line character and return a new byte slice.
func trimNewLine(b []byte) []byte {
	if len(b) >= 1 && b[len(b)-1] == '\n' {
		return b[:len(b)-1]
	}
	return b
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

func zerologSerializer(metadata LogEnvelopeMetadata) envelopeSerializerFunc {
	// create a new top level logger with no a scratch output since each
	// serialization will write to it's own local buffer instead of a single writer.
	logger := zerolog.New(ioutil.Discard).With().
		Str("type", "envelope.1").
		Str("deployment", metadata.Deployment).
		Str("environment", metadata.Environment).
		Str("environmentId", metadata.EnvironmentID).
		Str("host", metadata.Host).
		Str("nodeId", metadata.NodeID).
		Str("service", metadata.Service).
		Str("serviceId", metadata.ServiceID).
		Str("stack", metadata.Stack).
		Str("stackId", metadata.StackID).
		Str("product", metadata.Product).
		Str("productVersion", metadata.ProductVersion).
		Logger()
	return func(p []byte) ([]byte, error) {
		var buf bytes.Buffer
		l := logger.Output(&buf)
		if p != nil {
			l = l.With().RawJSON("payload", p).Logger()
		}
		l.Log().Send()
		return buf.Bytes(), nil
	}
}
