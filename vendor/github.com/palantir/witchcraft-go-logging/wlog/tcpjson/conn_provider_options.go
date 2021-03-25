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
)

// ConnProviderOption can be used to configure a ConnProvider
type ConnProviderOption interface {
	apply(*tcpConnProvider)
}

// connProviderOptionFunc wraps a func so it satisfies the ConnProviderOption interface.
type connProviderOptionFunc func(*tcpConnProvider)

func (f connProviderOptionFunc) apply(p *tcpConnProvider) {
	f(p)
}

// WithTLSConfig configures the ConnProvider to use the provided TLS configuration.
// If not set, the ConnProvider will use the default TLS config.
func WithTLSConfig(tlsConfig *tls.Config) ConnProviderOption {
	return connProviderOptionFunc(func(p *tcpConnProvider) {
		p.tlsConfig = tlsConfig
	})
}
