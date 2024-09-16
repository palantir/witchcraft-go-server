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
	"strings"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient"
	"github.com/palantir/witchcraft-go-logging/wlog"
)

// Runtime specifies the base runtime configuration fields that should be included in all witchcraft-server-go
// server runtime configurations.
type Runtime struct {
	DiagnosticsConfig DiagnosticsConfig         `yaml:"diagnostics,omitempty"`
	HealthChecks      HealthChecksConfig        `yaml:"health-checks,omitempty"`
	LoggerConfig      *LoggerConfig             `yaml:"logging,omitempty"`
	ServiceDiscovery  httpclient.ServicesConfig `yaml:"service-discovery,omitempty"`
}

// BaseRuntimeConfig is an interface implemented by Runtime and structs that embed it.
type BaseRuntimeConfig interface {
	BaseRuntimeConfig() Runtime
}

func (r Runtime) BaseRuntimeConfig() Runtime {
	return r
}

type DiagnosticsConfig struct {
	// DebugSharedSecret is a string which, if provided, will be used as the access key to the diagnostics route.
	// This takes precedence over DebugSharedSecretFile.
	DebugSharedSecret string `yaml:"debug-shared-secret"`
	// DebugSharedSecretFile is an on-disk location containing the shared secret. If DebugSharedSecretFile is provided and
	// DebugSharedSecret is not, the content of the file will be used as the shared secret.
	DebugSharedSecretFile string `yaml:"debug-shared-secret-file"`
}

type HealthChecksConfig struct {
	SharedSecret string `yaml:"shared-secret"`
}

type LoggerConfig struct {
	// Level configures the log level for leveled loggers (such as service logs). Does not impact non-leveled loggers
	// (such as request logs).
	Level wlog.LogLevel `yaml:"level"`
}

func (c *LoggerConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type loggerConfigAlias LoggerConfig
	var cfg loggerConfigAlias
	if err := unmarshal(&cfg); err != nil {
		return err
	}
	*c = LoggerConfig(cfg)
	c.Level = wlog.LogLevel(strings.ToLower(string(c.Level)))
	return nil
}
