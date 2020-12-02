package main

import (
	"os"

	"github.com/palantir/godel/v2/framework/pluginapi/v2/pluginapi"
	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/cmd"
)

func main() {
	if ok := pluginapi.InfoCmd(os.Args, os.Stdout, cmd.PluginInfo); ok {
		return
	}
	os.Exit(cmd.Execute())
}
