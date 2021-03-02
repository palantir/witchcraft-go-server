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
	"os"
)

type LogEnvelopeV1 struct {
	LogEnvelopeMetadata
	Payload json.RawMessage `json:"payload"`
}

type LogEnvelopeMetadata struct {
	Type           string `json:"type"`
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

func GetEnvelopeMetadata() LogEnvelopeMetadata {
	return LogEnvelopeMetadata{
		Deployment:     os.Getenv("LOG_ENVELOPE_DEPLOYMENT_NAME"),
		Environment:    os.Getenv("LOG_ENVELOPE_ENVIRONMENT_NAME"),
		EnvironmentID:  os.Getenv("LOG_ENVELOPE_ENVIRONMENT"),
		Host:           os.Getenv("LOG_ENVELOPE_HOST"),
		NodeID:         os.Getenv("LOG_ENVELOPE_NODE_ID"),
		Product:        os.Getenv("LOG_ENVELOPE_PRODUCT_NAME"),
		ProductVersion: os.Getenv("LOG_ENVELOPE_PRODUCT_VERSION"),
		Service:        os.Getenv("LOG_ENVELOPE_SERVICE_NAME"),
		ServiceID:      os.Getenv("LOG_ENVELOPE_SERVICE_ID"),
		Stack:          os.Getenv("LOG_ENVELOPE_STACK_NAME"),
		StackID:        os.Getenv("LOG_ENVELOPE_STACK_ID"),
	}
}
