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

type Audit2Logger interface {
	Audit(name string, result AuditResultType, params ...Audit2Param)
}

type audit2LoggerImpl struct {
	logger *logger
}

func (a *audit2LoggerImpl) Audit(name string, result AuditResultType, params ...Audit2Param) {
	a.logger.Audit2(name, result, params...)
}

func Audit2NewFromCreator(audit2Writer, audit3Writer io.Writer, creator wlog.LoggerCreator) Audit2Logger {
	var audit3Logger wlog.Logger
	if audit3Writer != nil {
		audit3Logger = creator(audit3Writer)
	}
	return &audit2LoggerImpl{
		logger: &logger{
			audit2Logger: creator(audit2Writer),
			audit3Logger: audit3Logger,
		},
	}
}
