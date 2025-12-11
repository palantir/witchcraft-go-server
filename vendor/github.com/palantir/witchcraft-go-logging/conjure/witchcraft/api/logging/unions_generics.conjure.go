// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/witchcraft-logging-api/witchcraft/api/logging.

//go:build go1.23

package logging

import (
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft-logging-api/witchcraft/api/logging"
)

type DiagnosticWithT[T any] = logging.DiagnosticWithT[T]

type DiagnosticVisitorWithT[T any] = logging.DiagnosticVisitorWithT[T]

type RequestLogWithT[T any] = logging.RequestLogWithT[T]

type RequestLogVisitorWithT[T any] = logging.RequestLogVisitorWithT[T]

type UnionEventLogWithT[T any] = logging.UnionEventLogWithT[T]

type UnionEventLogVisitorWithT[T any] = logging.UnionEventLogVisitorWithT[T]

type WrappedLogV1PayloadWithT[T any] = logging.WrappedLogV1PayloadWithT[T]

type WrappedLogV1PayloadVisitorWithT[T any] = logging.WrappedLogV1PayloadVisitorWithT[T]
