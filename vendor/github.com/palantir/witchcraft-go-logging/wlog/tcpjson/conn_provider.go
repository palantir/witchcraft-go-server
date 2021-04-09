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
	"crypto/tls"
	"net"
	"net/url"
	"sync/atomic"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
)

// ConnProvider defines the behavior to retrieve an established net.Conn.
type ConnProvider interface {
	// GetConn returns a net.Conn or an error if there is no connection established.
	// It is the caller's responsibility to close the returned net.Conn and
	// gracefully handle any closed connection errors if the net.Conn is shared across clients.
	GetConn() (net.Conn, error)
}

const (
	ErrNoURIs           = "no URIs provided to connect to"
	ErrFailedParsingURI = "failed to parse provided URI"
	ErrFailedDial       = "failed to dial the host:port"
)

var _ ConnProvider = (*tcpConnProvider)(nil)

// tcpConnProvider implements a ConnProvider that will round-robin TCP connections to a specific set of hosts.
type tcpConnProvider struct {
	// nextHostIdx contains the index of the next host to connect to from the hosts slice below.
	// The index will be reset to 0 when len(hosts) is reached to facilitate round-robin connections.
	nextHostIdx int32
	hosts       []string
	tlsConfig   *tls.Config
}

// NewTCPConnProvider returns a new ConnProvider that provides TCP connections.
// The provided uris must not be empty and must be able to be parsed as URIs. Refer to the documentation for url.Parse.
func NewTCPConnProvider(uris []string, options ...ConnProviderOption) (ConnProvider, error) {
	if len(uris) < 1 {
		return nil, werror.Error(ErrNoURIs)
	}

	// Extract the host:port from each provided URI to simplify establishing a Dial connection
	var hosts []string
	for _, uri := range uris {
		u, err := url.Parse(uri)
		if err != nil {
			return nil, werror.Error(ErrFailedParsingURI, werror.SafeParam("uri", uri))
		}
		hosts = append(hosts, u.Host)
	}

	provider := &tcpConnProvider{
		nextHostIdx: 0,
		hosts:       hosts,
	}

	for _, option := range options {
		option.apply(provider)
	}
	return provider, nil
}

var (
	defaultDialer = &net.Dialer{
		// Dial timeout is short - this connection is internal to the environment and slowness can cause logs to fall behind.
		Timeout: 5 * time.Second,
	}
)

func (s *tcpConnProvider) GetConn() (net.Conn, error) {
	hostIdx := atomic.LoadInt32(&s.nextHostIdx)
	nextHostIdx := int(hostIdx+1) % len(s.hosts)
	atomic.CompareAndSwapInt32(&s.nextHostIdx, hostIdx, int32(nextHostIdx))

	tlsConn, err := tls.DialWithDialer(defaultDialer, "tcp", s.hosts[hostIdx], s.tlsConfig)
	if err != nil {
		return nil, werror.Wrap(err, ErrFailedDial)
	}
	return tlsConn, nil
}
