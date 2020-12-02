package cmd

import (
	"github.com/palantir/godel/v2/framework/pluginapi"
	"github.com/palantir/pkg/cobracli"
	"github.com/spf13/cobra"
)

var (
	projectDirFlagVal      string
	godelConfigFileFlagVal string
)

var rootCmd = &cobra.Command{
	Use:   "sls-diagnostics-plugin",
	Short: "Provides sls-diagnostics tasks",
}

func Execute() int {
	return cobracli.ExecuteWithDefaultParams(rootCmd)
}

func init() {
	pluginapi.AddProjectDirPFlagPtr(rootCmd.PersistentFlags(), &projectDirFlagVal)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.ProjectDirFlagName); err != nil {
		panic(err)
	}
	pluginapi.AddGodelConfigPFlagPtr(rootCmd.PersistentFlags(), &godelConfigFileFlagVal)
	if err := rootCmd.MarkPersistentFlagRequired(pluginapi.GodelConfigFlagName); err != nil {
		panic(err)
	}
}
