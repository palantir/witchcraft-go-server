// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package auditloginternal

import (
	"io"

	"github.com/palantir/witchcraft-go-logging/wlog"
)

type Audit3AuditProducerType string

const (
	Audit3AuditProducerServer Audit3AuditProducerType = "SERVER"
	Audit3AuditProducerClient Audit3AuditProducerType = "CLIENT"
)

type Audit3Organization struct {
	ID     string `json:"id"`
	Reason string `json:"reason"`
}

type Audit3ContextualizedUser struct {
	UID       string   `json:"uid"`
	UserName  *string  `json:"userName,omitempty"`
	FirstName *string  `json:"firstName,omitempty"`
	LastName  *string  `json:"lastName,omitempty"`
	Groups    []string `json:"groups,omitempty"`
	Realm     *string  `json:"realm,omitempty"`
}

type Audit3Logger interface {
	Audit(name string, result AuditResultType, params ...Audit3Param)
}

type audit3LoggerImpl struct {
	logger *logger
}

func (a *audit3LoggerImpl) Audit(name string, result AuditResultType, params ...Audit3Param) {
	a.logger.Audit3(name, result, params...)
}

func Audit3NewFromCreator(audit3Writer, audit2Writer io.Writer, creator wlog.LoggerCreator) Audit3Logger {
	var audit2Logger wlog.Logger
	if audit2Writer != nil {
		audit2Logger = creator(audit2Writer)
	}
	return &audit3LoggerImpl{
		logger: &logger{
			audit3Logger: creator(audit3Writer),
			audit2Logger: audit2Logger,
		},
	}
}
