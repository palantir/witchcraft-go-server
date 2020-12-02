package cmd

import (
	"github.com/palantir/godel/v2/framework/pluginapi/v2/pluginapi"
	"github.com/palantir/godel/v2/framework/verifyorder"
)

var (
	Version    = "unspecified"
	PluginInfo = pluginapi.MustNewPluginInfo(
		"com.palantir.deployability",
		"sls-diagnostics-plugin",
		Version,
		pluginapi.PluginInfoUsesConfigFile(),
		pluginapi.PluginInfoGlobalFlagOptions(
			pluginapi.GlobalFlagOptionsParamDebugFlag("--"+pluginapi.DebugFlagName),
			pluginapi.GlobalFlagOptionsParamProjectDirFlag("--"+pluginapi.ProjectDirFlagName),
			pluginapi.GlobalFlagOptionsParamGodelConfigFlag("--"+pluginapi.GodelConfigFlagName),
		),
		pluginapi.PluginInfoTaskInfo(
			"sls-diagnostics",
			"Generates the sls-diagnostics.json manifest file",
			pluginapi.TaskInfoCommand(generateCmd.Use),
			pluginapi.TaskInfoVerifyOptions(
				pluginapi.VerifyOptionsOrdering(pInt(verifyorder.Generate+90)),
				pluginapi.VerifyOptionsApplyFalseArgs("--"+verifyFlagName),
			),
		),
	)
)

func pInt(i int) *int { return &i }
