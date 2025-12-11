// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/witchcraft-logging-api/witchcraft/api/logging.

package logging

import (
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft-logging-api/witchcraft/api/logging"
)

type AuditProducer = logging.AuditProducer

type AuditProducer_Value = logging.AuditProducer_Value

const (
	AuditProducer_SERVER  = logging.AuditProducer_SERVER
	AuditProducer_CLIENT  = logging.AuditProducer_CLIENT
	AuditProducer_UNKNOWN = logging.AuditProducer_UNKNOWN
)

func AuditProducer_Values() []AuditProducer_Value {
	return logging.AuditProducer_Values()
}

func New_AuditProducer(value AuditProducer_Value) AuditProducer {
	return logging.New_AuditProducer(value)
}

type AuditResult = logging.AuditResult

type AuditResult_Value = logging.AuditResult_Value

const (
	AuditResult_SUCCESS      = logging.AuditResult_SUCCESS
	AuditResult_ERROR        = logging.AuditResult_ERROR
	AuditResult_UNAUTHORIZED = logging.AuditResult_UNAUTHORIZED
	AuditResult_PARTIAL      = logging.AuditResult_PARTIAL
	AuditResult_UNKNOWN      = logging.AuditResult_UNKNOWN
)

func AuditResult_Values() []AuditResult_Value {
	return logging.AuditResult_Values()
}

func New_AuditResult(value AuditResult_Value) AuditResult {
	return logging.New_AuditResult(value)
}

type LogLevel = logging.LogLevel

type LogLevel_Value = logging.LogLevel_Value

const (
	LogLevel_FATAL   = logging.LogLevel_FATAL
	LogLevel_ERROR   = logging.LogLevel_ERROR
	LogLevel_WARN    = logging.LogLevel_WARN
	LogLevel_INFO    = logging.LogLevel_INFO
	LogLevel_DEBUG   = logging.LogLevel_DEBUG
	LogLevel_TRACE   = logging.LogLevel_TRACE
	LogLevel_UNKNOWN = logging.LogLevel_UNKNOWN
)

func LogLevel_Values() []LogLevel_Value {
	return logging.LogLevel_Values()
}

func New_LogLevel(value LogLevel_Value) LogLevel {
	return logging.New_LogLevel(value)
}
