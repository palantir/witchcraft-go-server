package diagnosticsplugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

func TestGenerateDiagnosticEntriesFileContent(t *testing.T) {
	for _, test := range []struct {
		Name      string
		In        map[*packages.Package][]DiagnosticEntry
		Expected  map[string][]byte
		ExpectErr string
	}{
		{
			Name: "2 packages, one with no impls",
			In: map[*packages.Package][]DiagnosticEntry{
				{GoFiles: []string{"/go/src/mypkg/main.go"}}: {
					{
						Type: "diagnostic1",
						Docs: "some docs",
						Safe: boolPtr(true),
					},
				},
				{GoFiles: []string{"/go/src/otherpkg/main.go"}}: {},
			},
			Expected: map[string][]byte{
				"/go/src/mypkg/sls-diagnostics.json": []byte(`{
  "diagnostics": [
    {
      "type": "diagnostic1",
      "docs": "some docs",
      "safe": true
    }
  ]
}
`),
				"/go/src/otherpkg/sls-diagnostics.json": nil,
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			out, err := generateDiagnosticEntriesFileContent(test.In)
			if test.ExpectErr == "" {
				require.NoError(t, err)
				assert.Equal(t, test.Expected, out)
			} else {
				require.EqualError(t, err, test.ExpectErr)
			}
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
