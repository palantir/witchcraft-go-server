// Copyright (c) 2020 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"

	"github.com/palantir/pkg/signals"
	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/diagnosticsplugin"
	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/internal/godelproject"
	"github.com/spf13/cobra"
)

const verifyFlagName = "verify"

var verifyFlag bool

var generateCmd = &cobra.Command{
	Use: "generate",
	RunE: func(cmd *cobra.Command, args []string) error {
		pkgs, err := godelproject.PkgsInProject(projectDirFlagVal, godelConfigFileFlagVal)
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
