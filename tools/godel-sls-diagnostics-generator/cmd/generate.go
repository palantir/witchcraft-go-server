package cmd

import (
	"context"

	godelconfig "github.com/palantir/godel/v2/framework/godel/config"
	"github.com/palantir/pkg/matcher"
	"github.com/palantir/pkg/signals"
	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/diagnosticsplugin"
	"github.com/spf13/cobra"
)

const verifyFlagName = "verify"

var verifyFlag bool

var generateCmd = &cobra.Command{
	Use: "generate",
	RunE: func(cmd *cobra.Command, args []string) error {
		godelPackageMatcher, err := GodelConfigPackageMatcher(godelConfigFileFlagVal)
		if err != nil {
			return err
		}
		pkgs, err := diagnosticsplugin.PkgsInProject(projectDirFlagVal, godelPackageMatcher)
		if err != nil {
			return err
		}
		ctx, cancel := signals.ContextWithShutdown(context.Background())
		defer cancel()

		return diagnosticsplugin.Generate(ctx, pkgs, verifyFlag, projectDirFlagVal, cmd.OutOrStdout())
	},
}

func init() {
	generateCmd.Flags().BoolVar(&verifyFlag, verifyFlagName, false, "verify that current project matches output of the generator")
	rootCmd.AddCommand(generateCmd)
}

func GodelConfigPackageMatcher(godelConfigFile string) (matcher.Matcher, error) {
	var godelExcludes matcher.Matcher
	if godelConfigFile != "" {
		excludes, err := godelconfig.ReadGodelConfigExcludesFromFile(godelConfigFile)
		if err != nil {
			return nil, err
		}
		godelExcludes = excludes.Matcher()
	}
	return godelExcludes, nil
}
