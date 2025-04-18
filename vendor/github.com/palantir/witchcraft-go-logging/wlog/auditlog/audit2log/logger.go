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

package audit2log

import (
	"io"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/internal/auditloginternal"
)

type AuditResultType string

const (
	AuditResultSuccess      = AuditResultType(auditloginternal.AuditResultSuccess)
	AuditResultUnauthorized = AuditResultType(auditloginternal.AuditResultUnauthorized)
	AuditResultError        = AuditResultType(auditloginternal.AuditResultError)
)

type Logger interface {
	Audit(name string, result AuditResultType, params ...Param)
}

type audit2LoggerAdapter struct {
	audit2logger auditloginternal.Audit2Logger
}

func (a *audit2LoggerAdapter) Audit(name string, result AuditResultType, params ...Param) {
	a.audit2logger.Audit(name, auditloginternal.AuditResultType(result), convertExternalParamsToInternalParams(params)...)
}

func convertInternalLoggerToExternalLogger(logger auditloginternal.Audit2Logger) Logger {
	return &audit2LoggerAdapter{
		audit2logger: logger,
	}
}

func New(w io.Writer) Logger {
	return NewFromCreator(w, wlog.DefaultLoggerProvider().NewLogger)
}

func NewDualLogger(audit2Writer, audit3Writer io.Writer) Logger {
	return newDualLoggerFromCreator(audit2Writer, audit3Writer, wlog.DefaultLoggerProvider().NewLogger)
}

func NewFromCreator(w io.Writer, creator wlog.LoggerCreator) Logger {
	return newDualLoggerFromCreator(w, nil, creator)
}

func newDualLoggerFromCreator(audit2Writer, audit3Writer io.Writer, creator wlog.LoggerCreator) Logger {
	return convertInternalLoggerToExternalLogger(auditloginternal.Audit2NewFromCreator(audit2Writer, audit3Writer, creator))
}

func WithParams(logger Logger, params ...Param) Logger {
	if len(params) == 0 {
		return logger
	}

	if innerWrapped, ok := logger.(*wrappedLogger); ok {
		return &wrappedLogger{
			logger: innerWrapped.logger,
			params: append(innerWrapped.params, params...),
		}
	}

	return &wrappedLogger{
		logger: logger,
		params: params,
	}
}
