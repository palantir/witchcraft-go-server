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

	werror "github.com/palantir/witchcraft-go-error"
)

// ConnProvider defines the behavior to retrieve an established net.Conn.
type ConnProvider interface {
	// GetConn returns a net.Conn or an error if there is no connection established.
	// It is the users responsibility to close the returned net.Conn.
	GetConn() (net.Conn, error)
}

const (
	ErrNoURIs           = "no URIs provided to connect to"
	ErrFailedParsingURI = "failed to parse provided URI"
	ErrFailedDial       = "failed to dial the host:port"
)

var _ ConnProvider = (*TCPConnProvider)(nil)

// TCPConnProvider implements a ConnProvider that will round-robin TCP connections to a specific set of hosts.
type TCPConnProvider struct {
	hostIdx   int32
	hosts     []string
	tlsConfig *tls.Config
}

func NewTCPConnProvider(uris []string, tlsCfg *tls.Config) (*TCPConnProvider, error) {
	if len(uris) < 1 {
		return nil, werror.Error(ErrNoURIs)
	}

	// Extract the host:port from each provided URI to simplify establishing a Dial connection
	var hosts []string
	for _, uri := range uris {
		u, err := url.Parse(uri)
		if err != nil {
			return nil, werror.Wrap(err, ErrFailedParsingURI, werror.SafeParam("uri", uri))
		}
		hosts = append(hosts, u.Host)
	}

	return &TCPConnProvider{
		hostIdx:   0,
		hosts:     hosts,
		tlsConfig: tlsCfg,
	}, nil
}

func (s *TCPConnProvider) GetConn() (net.Conn, error) {
	oldIdx := atomic.LoadInt32(&s.hostIdx)
	nextHostIdx := int(oldIdx) % len(s.hosts)
	atomic.CompareAndSwapInt32(&s.hostIdx, oldIdx, oldIdx+1)

	tlsConn, err := tls.Dial("tcp", s.hosts[nextHostIdx], s.tlsConfig)
	if err != nil {
		return nil, werror.Wrap(err, ErrFailedDial)
	}
	return tlsConn, nil
}
