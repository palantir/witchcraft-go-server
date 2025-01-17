package wdebug

import (
	"context"
	"io"
)

type DiagnosticType string

type DiagnosticHandler interface {
	Type() DiagnosticType
	Documentation() string
	ContentType() string
	SafeLoggable() bool
	Extension() string
	WriteDiagnostic(ctx context.Context, w io.Writer) error
}
