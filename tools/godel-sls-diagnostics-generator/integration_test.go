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

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/diagnosticsplugin"
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
