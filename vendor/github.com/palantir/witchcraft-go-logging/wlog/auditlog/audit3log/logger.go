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

package audit3log

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

type AuditProducerType string

const (
	AuditProducerServer = AuditProducerType(auditloginternal.Audit3AuditProducerServer)
	AuditProducerClient = AuditProducerType(auditloginternal.Audit3AuditProducerClient)
)

type Organization auditloginternal.Audit3Organization

type ContextualizedUser auditloginternal.Audit3ContextualizedUser

type Logger interface {
	Audit(name string, result AuditResultType, params ...Param)
}

type audit3LoggerAdapter struct {
	audit3logger auditloginternal.Audit3Logger
}

func (a *audit3LoggerAdapter) Audit(name string, result AuditResultType, params ...Param) {
	a.audit3logger.Audit(name, auditloginternal.AuditResultType(result), convertExternalParamsToInternalParams(params)...)
}

func convertInternalLoggerToExternalLogger(logger auditloginternal.Audit3Logger) Logger {
	return &audit3LoggerAdapter{
		audit3logger: logger,
	}
}

func New(w io.Writer) Logger {
	return NewFromCreator(w, wlog.DefaultLoggerProvider().NewLogger)
}

func NewDualLogger(audit3Writer, audit2Writer io.Writer) Logger {
	return NewDualLoggerFromCreator(audit3Writer, audit2Writer, wlog.DefaultLoggerProvider().NewLogger)
}

func NewFromCreator(w io.Writer, creator wlog.LoggerCreator) Logger {
	return NewDualLoggerFromCreator(w, nil, creator)
}

func NewDualLoggerFromCreator(audit3Writer, audit2Writer io.Writer, creator wlog.LoggerCreator) Logger {
	return convertInternalLoggerToExternalLogger(auditloginternal.Audit3NewFromCreator(audit3Writer, audit2Writer, creator))
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
