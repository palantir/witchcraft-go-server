package diagnosticsplugin

const (
	wdebugImportPath    = "github.com/palantir/witchcraft-go-server/v2/witchcraft/internal/wdebug"
	wdebugInterfaceName = "DiagnosticHandler"

	diagnosticsJSONPath = "sls-diagnostics.json"
)

type SLSDiagnosticsWrapper struct {
	Diagnostics []DiagnosticHandlerMetadata `json:"diagnostics"`
}

type DiagnosticHandlerMetadata struct {
	DiagnosticType string `json:"type"`
	DiagnosticDocs string `json:"docs"`
}
