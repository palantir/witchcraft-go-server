package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/diagnosticsplugin"
	_ "github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/stretchr/testify/require"
)

func TestFindImplementations(t *testing.T) {
	projectDir := filepath.Join(os.Getenv("GOPATH"), "src", "github.com/palantir/witchcraft-go-server")

	pkgs, err := diagnosticsplugin.PkgsInProject(projectDir, nil)
	require.NoError(t, err)

	impls, err := diagnosticsplugin.LoadDiagnosticHandlerImpls(context.Background(), pkgs, projectDir)
	require.NoError(t, err)

	require.NotEmpty(t, impls)
	for _, impl := range impls {
		fmt.Println(impl)
	}
}
