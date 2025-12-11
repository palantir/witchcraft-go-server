// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/witchcraft-logging-api/witchcraft/api/logging.

package logging

import (
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft-logging-api/witchcraft/api/logging"
)

type Diagnostic = logging.Diagnostic

type DiagnosticVisitor = logging.DiagnosticVisitor

type DiagnosticVisitorWithContext = logging.DiagnosticVisitorWithContext

func NewDiagnosticFromGeneric(v GenericDiagnostic) Diagnostic {
	return logging.NewDiagnosticFromGeneric(v)
}

func NewDiagnosticFromThreadDump(v ThreadDumpV1) Diagnostic {
	return logging.NewDiagnosticFromThreadDump(v)
}

type RequestLog = logging.RequestLog

type RequestLogVisitor = logging.RequestLogVisitor

type RequestLogVisitorWithContext = logging.RequestLogVisitorWithContext

func NewRequestLogFromV1(v RequestLogV1) RequestLog {
	return logging.NewRequestLogFromV1(v)
}

func NewRequestLogFromV2(v RequestLogV2) RequestLog {
	return logging.NewRequestLogFromV2(v)
}

type UnionEventLog = logging.UnionEventLog

type UnionEventLogVisitor = logging.UnionEventLogVisitor

type UnionEventLogVisitorWithContext = logging.UnionEventLogVisitorWithContext

func NewUnionEventLogFromEventLog(v EventLogV1) UnionEventLog {
	return logging.NewUnionEventLogFromEventLog(v)
}

func NewUnionEventLogFromEventLogV2(v EventLogV2) UnionEventLog {
	return logging.NewUnionEventLogFromEventLogV2(v)
}

type WrappedLogV1Payload = logging.WrappedLogV1Payload

type WrappedLogV1PayloadVisitor = logging.WrappedLogV1PayloadVisitor

type WrappedLogV1PayloadVisitorWithContext = logging.WrappedLogV1PayloadVisitorWithContext

func NewWrappedLogV1PayloadFromServiceLogV1(v ServiceLogV1) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromServiceLogV1(v)
}

func NewWrappedLogV1PayloadFromRequestLogV2(v RequestLogV2) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromRequestLogV2(v)
}

func NewWrappedLogV1PayloadFromTraceLogV1(v TraceLogV1) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromTraceLogV1(v)
}

func NewWrappedLogV1PayloadFromEventLogV2(v EventLogV2) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromEventLogV2(v)
}

func NewWrappedLogV1PayloadFromMetricLogV1(v MetricLogV1) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromMetricLogV1(v)
}

func NewWrappedLogV1PayloadFromAuditLogV2(v AuditLogV2) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromAuditLogV2(v)
}

func NewWrappedLogV1PayloadFromAuditLogV3(v AuditLogV3) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromAuditLogV3(v)
}

func NewWrappedLogV1PayloadFromDiagnosticLogV1(v DiagnosticLogV1) WrappedLogV1Payload {
	return logging.NewWrappedLogV1PayloadFromDiagnosticLogV1(v)
}
