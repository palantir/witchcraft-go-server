//go:generate go run $GOFILE

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/palantir/pkg/signals"
	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/cmd"
	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/diagnosticsplugin"
)

func main() {
	ctx, cancel := signals.ContextWithShutdown(context.Background())
	defer cancel()

	if err := doGenerate(ctx); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func doGenerate(ctx context.Context) error {
	projectDir, err := filepath.Abs("../..")
	if err != nil {
		return err
	}
	if err := os.Chdir(projectDir); err != nil {
		return err
	}

	godelPackageMatcher, err := cmd.GodelConfigPackageMatcher("godel/config/godel.yml")
	if err != nil {
		return err
	}
	pkgs, err := diagnosticsplugin.PkgsInProject(projectDir, godelPackageMatcher)
	if err != nil {
		return err
	}

	return diagnosticsplugin.Generate(ctx, pkgs, false, projectDir, os.Stdout)
}
