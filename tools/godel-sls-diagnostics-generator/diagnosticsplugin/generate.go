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

package diagnosticsplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	"golang.org/x/tools/go/packages"
)

func Generate(ctx context.Context, pkgs []string, verify bool, projectDir string, stdout io.Writer) error {
	metadata, err := LoadDiagnosticHandlerImpls(ctx, pkgs, projectDir)
	if err != nil {
		return err
	}
	fileContent, err := generateDiagnosticEntriesFileContent(metadata)
	if err != nil {
		return err
	}
	return writeDiagnosticEntriesFileContent(fileContent, verify, stdout)
}

func generateDiagnosticEntriesFileContent(entries map[*packages.Package][]DiagnosticEntry) (map[string][]byte, error) {
	files := map[string][]byte{}
	for pkg, impls := range entries {
		outputPath, err := getPackageDiagnosticsOutputPath(pkg)
		if err != nil {
			if len(impls) > 0 {
				return nil, fmt.Errorf("package contained interface implementations but no detectable files: %v", err)
			}
			continue
		}

		if len(impls) == 0 {
			files[outputPath] = nil
		} else {
			outputContent, err := renderPackageDiagnosticsJSON(impls)
			if err != nil {
				return nil, err
			}
			files[outputPath] = outputContent
		}
	}
	return files, nil
}

func writeDiagnosticEntriesFileContent(files map[string][]byte, verify bool, stdout io.Writer) error {
	for outputPath, outputContent := range files {
		exists := false
		if _, err := os.Stat(outputPath); err == nil {
			exists = true
		}
		if len(outputContent) == 0 {
			if exists {
				if verify {
					return fmt.Errorf("%s should not exist as it would have no entries", diagnosticsJSONPath)
				}
				_, _ = fmt.Fprintf(stdout, "Removing %s as there are no DiagnosticHandler implementations\n", diagnosticsJSONPath)
				if err := os.Remove(outputPath); err != nil {
					return err
				}
			}
			continue
		}

		// Write package file
		if verify {
			if !exists {
				return fmt.Errorf("%s does not exist and must be regenerated", outputPath)
			}
			existingContent, err := ioutil.ReadFile(outputPath)
			if err != nil {
				return fmt.Errorf("failed to read existing path: %v", err)
			}
			if string(outputContent) != string(existingContent) {
				return fmt.Errorf("%s content differs from what is on disk and must be regenerated", outputPath)
			}
		} else {
			if err := ioutil.WriteFile(outputPath, outputContent, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %v", diagnosticsJSONPath, err)
			}
		}
	}
	return nil
}

func getPackageDiagnosticsOutputPath(pkg *packages.Package) (string, error) {
	var outputDir string
	if len(pkg.GoFiles) > 0 {
		outputDir = filepath.Dir(pkg.GoFiles[0])
	} else if len(pkg.OtherFiles) > 0 {
		outputDir = filepath.Dir(pkg.OtherFiles[0])
	} else {
		return "", fmt.Errorf("failed to detect package %q output directory: no go files in package", pkg.ID)
	}

	return filepath.Join(outputDir, diagnosticsJSONPath), nil
}

func renderPackageDiagnosticsJSON(impls []DiagnosticEntry) ([]byte, error) {
	sort.Slice(impls, func(i, j int) bool {
		return impls[i].Type < impls[j].Type
	})

	metadataJSON, err := json.MarshalIndent(SLSDiagnosticsWrapper{impls}, "", "  ")
	if err != nil {
		return nil, err
	}
	if metadataJSON[len(metadataJSON)-1] != '\n' {
		metadataJSON = append(metadataJSON, '\n')
	}

	return metadataJSON, nil
}
