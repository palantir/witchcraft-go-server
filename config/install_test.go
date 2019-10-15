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

package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestInstallConfig(t *testing.T) {
	conf := `
product-name: productName
trace-sample-rate: 0.5
management-trace-sample-rate: 0.6
use-console-log: true
metrics-emit-frequency: 1s
server:
  address: address
  port: 10
  management-port: 11
  context-path: ContextPath
  client-ca-files:
    - a
    - b
  cert-file: certFile
  key-file: keyFile
`
	var install Install
	err := yaml.Unmarshal([]byte(conf), &install)
	assert.NoError(t, err)
	assert.Equal(t, Install{
		ProductName: "productName",
		Server: Server{
			Address:        "address",
			Port:           10,
			ManagementPort: 11,
			ContextPath:    "ContextPath",
			ClientCAFiles:  []string{"a", "b"},
			CertFile:       "certFile",
			KeyFile:        "keyFile",
		},
		MetricsEmitFrequency:      time.Second,
		TraceSampleRate:           asFloat(0.5),
		ManagementTraceSampleRate: asFloat(0.6),
		UseConsoleLog:             true,
	}, install)
}

func asFloat(f float64) *float64 {
	return &f
}
