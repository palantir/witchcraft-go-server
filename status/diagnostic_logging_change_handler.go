package status

import (
	"bytes"
	"context"
	"runtime/pprof"

	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-server/conjure/witchcraft/api/health"
)

// NewDiagnosticLoggingChangeHandler will emit a diagnostic log whenever the latest health status' state is more severe
// than HealthStateRepairing.
func NewDiagnosticLoggingChangeHandler() HealthStatusChangeHandler {
	return healthStatusChangeHandlerFn(func(ctx context.Context, prev, curr health.HealthStatus) {
		if HealthStatusCode(curr) > HealthStateStatusCodes[health.HealthStateRepairing] {
			var buf bytes.Buffer
			_ = pprof.Lookup("goroutine").WriteTo(&buf, 2) // bytes.Buffer's Write never returns an error, so we swallow it
			diag1log.FromContext(ctx).Diagnostic(logging.NewDiagnosticFromThreadDump(diag1log.ThreadDumpV1FromGoroutines(buf.Bytes())))
		}
	})
}
