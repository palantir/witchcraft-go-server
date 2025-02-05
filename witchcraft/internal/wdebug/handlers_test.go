package wdebug

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidDiagnosticTypes(t *testing.T) {
	for diagnosticType := range diagnosticHandlers {
		assert.NoError(t, diagnosticType.Validate())
	}
}
