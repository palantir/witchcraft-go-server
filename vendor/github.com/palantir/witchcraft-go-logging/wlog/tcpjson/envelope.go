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
	"os"

	werror "github.com/palantir/witchcraft-go-error"
)

const (
	logEnvelopeV1Type = "envelope.1"
)

type LogEnvelopeMetadata struct {
	Deployment     string `json:"deployment"`
	Environment    string `json:"environment"`
	EnvironmentID  string `json:"environmentId"`
	Host           string `json:"host"`
	NodeID         string `json:"nodeId"`
	Service        string `json:"service"`
	ServiceID      string `json:"serviceId"`
	Stack          string `json:"stack"`
	StackID        string `json:"stackId"`
	Product        string `json:"product"`
	ProductVersion string `json:"productVersion"`
}

const (
	envVarDeployment     = "LOG_ENVELOPE_DEPLOYMENT_NAME"
	envVarEnvironment    = "LOG_ENVELOPE_ENVIRONMENT_NAME"
	envVarEnvironmentID  = "LOG_ENVELOPE_ENVIRONMENT_ID"
	envVarHost           = "LOG_ENVELOPE_HOST"
	envVarNodeID         = "LOG_ENVELOPE_NODE_ID"
	envVarProduct        = "LOG_ENVELOPE_PRODUCT_NAME"
	envVarProductVersion = "LOG_ENVELOPE_PRODUCT_VERSION"
	envVarService        = "LOG_ENVELOPE_SERVICE_NAME"
	envVarServiceID      = "LOG_ENVELOPE_SERVICE_ID"
	envVarStack          = "LOG_ENVELOPE_STACK_NAME"
	envVarStackID        = "LOG_ENVELOPE_STACK_ID"
)

// GetEnvelopeMetadata retrieves all log envelope environment variables
// and returns the fully populated LogEnvelopeMetadata and a nil error.
// If any expected environment variables are not found or empty, then an empty LogEnvelopeMetadata
// will be returned along with an error that contains the missing environment variables.
func GetEnvelopeMetadata() (LogEnvelopeMetadata, error) {
	metadata := LogEnvelopeMetadata{
		Deployment:     os.Getenv(envVarDeployment),
		Environment:    os.Getenv(envVarEnvironment),
		EnvironmentID:  os.Getenv(envVarEnvironmentID),
		Host:           os.Getenv(envVarHost),
		NodeID:         os.Getenv(envVarNodeID),
		Product:        os.Getenv(envVarProduct),
		ProductVersion: os.Getenv(envVarProductVersion),
		Service:        os.Getenv(envVarService),
		ServiceID:      os.Getenv(envVarServiceID),
		Stack:          os.Getenv(envVarStack),
		StackID:        os.Getenv(envVarStackID),
	}
	var missingEnvVars []string
	if metadata.Deployment == "" {
		missingEnvVars = append(missingEnvVars, envVarDeployment)
	}
	if metadata.Environment == "" {
		missingEnvVars = append(missingEnvVars, envVarEnvironment)
	}
	if metadata.EnvironmentID == "" {
		missingEnvVars = append(missingEnvVars, envVarEnvironmentID)
	}
	if metadata.Host == "" {
		missingEnvVars = append(missingEnvVars, envVarHost)
	}
	if metadata.NodeID == "" {
		missingEnvVars = append(missingEnvVars, envVarNodeID)
	}
	if metadata.Product == "" {
		missingEnvVars = append(missingEnvVars, envVarProduct)
	}
	if metadata.ProductVersion == "" {
		missingEnvVars = append(missingEnvVars, envVarProductVersion)
	}
	if metadata.Service == "" {
		missingEnvVars = append(missingEnvVars, envVarService)
	}
	if metadata.ServiceID == "" {
		missingEnvVars = append(missingEnvVars, envVarServiceID)
	}
	if metadata.Stack == "" {
		missingEnvVars = append(missingEnvVars, envVarStack)
	}
	if metadata.StackID == "" {
		missingEnvVars = append(missingEnvVars, envVarStackID)
	}
	if len(missingEnvVars) > 0 {
		return LogEnvelopeMetadata{}, werror.Error("all log envelope environment variables are not set",
			werror.SafeParam("missingEnvVars", missingEnvVars))
	}
	return metadata, nil
}
