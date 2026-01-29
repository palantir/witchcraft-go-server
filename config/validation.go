// Copyright (c) 2026 Palantir Technologies. All rights reserved.
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
	"context"

	"github.com/palantir/witchcraft-go-health/v2/conjure/witchcraft/api/health"
)

const (
	// ValidationFailureMetric is the name of the counter metric emitted when
	// [Validator.Validate] returns an error and MetricOnFailure is enabled.
	ValidationFailureMetric = "server.config.validation.failure"
)

// Validator is an optional interface that config structs can implement
// to provide validation logic. If a config struct implements this interface
// and validation is enabled via WithInstallConfigValidation or
// WithRuntimeConfigValidation, the Validate method will be called
// after unmarshaling the config.
type Validator interface {
	Validate(ctx context.Context) error
}

// InstallConfigValidationOptions configures validation behavior for install configuration.
// Install config validation has limited options since logging, health checks, and metrics
// all depend on install config being loaded first. Validation errors always fail at startup
// since install config cannot be changed at runtime.
type InstallConfigValidationOptions struct {
	// StrictUnmarshaling enables strict YAML unmarshaling that fails on unknown fields.
	StrictUnmarshaling bool
}

// RuntimeConfigValidationOptions configures validation behavior for runtime configuration.
type RuntimeConfigValidationOptions struct {
	// StrictUnmarshaling enables strict YAML unmarshaling that fails on unknown fields.
	// Strict unmarshaling errors always fail at startup and reject reloads.
	StrictUnmarshaling bool

	// LogOnFailure logs [Validator.Validate] errors if they occur.
	LogOnFailure bool

	// MetricOnFailure emits a counter metric when [Validator.Validate] returns an error.
	// The metric name is [ValidationFailureMetric].
	MetricOnFailure bool

	// HealthCheckOnFailure sets a health check to the specified state when
	// [Validator.Validate] fails. If nil, no health check is affected.
	HealthCheckOnFailure *HealthCheckOnFailure

	// FailStartupOnError causes the server to fail to start if the initial runtime config
	// validation fails. When false, the server starts even with invalid config, but other
	// failure actions (logging, metrics, health checks) still apply if configured.
	FailStartupOnError bool

	// RejectInvalidReload causes config reloads to be rejected if validation fails,
	// keeping the previous config. When false, invalid config is accepted on reload,
	// but other failure actions (logging, metrics, health checks) still apply if configured.
	RejectInvalidReload bool
}

// HealthCheckOnFailure configures health check behavior when validation fails.
type HealthCheckOnFailure struct {
	// HealthState is the health state to set when validation fails.
	// This should be a non-HEALTHY state such as ERROR or WARNING.
	// The health check will report HEALTHY when validation succeeds.
	HealthState health.HealthState_Value
}
