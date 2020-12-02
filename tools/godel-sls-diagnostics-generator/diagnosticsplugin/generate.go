package diagnosticsplugin

import (
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/internal/findimpls"
	"golang.org/x/tools/go/packages"
)

func Generate(ctx context.Context, pkgs []string, verify bool, projectDir string, stdout io.Writer) error {
	metadata, err := LoadDiagnosticHandlerImpls(ctx, pkgs, projectDir)
	if err != nil {
		return err
	}

	for pkg, impls := range metadata {
		outputPath, err := getPackageDiagnosticsOutputPath(pkg)
		if err != nil {
			if len(impls) > 0 {
				return fmt.Errorf("package contained interface implementations but no detectable files: %v", err)
			}
			continue
		}
		exists := false
		if _, err := os.Stat(outputPath); err == nil {
			exists = true
		}

		// If there should be no content, ensure the file does not exist then continue
		if len(impls) == 0 {
			if exists {
				if verify {
					return fmt.Errorf("%s should not exist as it would have no entries", diagnosticsJSONPath)
				} else {
					_, _ = fmt.Fprintf(stdout, "Removing %s as there are no DiagnosticHandler implementations\n", diagnosticsJSONPath)
					if err := os.Remove(outputPath); err != nil {
						return err
					}
				}
			}
			continue
		}

		// Write package file
		metadataJSON, err := renderPackageDiagnosticsJSON(impls)
		if err != nil {
			return err
		}
		if verify {
			if !exists {
				return fmt.Errorf( "%s does not exist and must be regenerated\n", outputPath)
			}
			existingContent, err := ioutil.ReadFile(outputPath)
			if err != nil {
				return fmt.Errorf("failed to read existing path: %v", err)
			}
			if string(metadataJSON) != string(existingContent) {
				return fmt.Errorf("%s content differs from what is on disk and must be regenerated", outputPath)
			}
		} else {
			if err := ioutil.WriteFile(outputPath, metadataJSON, 0644); err != nil {
				return fmt.Errorf("failed to write %s: %v", diagnosticsJSONPath, err)
			}
		}
	}

	return nil
}

func LoadDiagnosticHandlerImpls(ctx context.Context, pkgs []string, projectDir string) (map[*packages.Package][]DiagnosticHandlerMetadata, error) {
	findResult, err := findimpls.Find(ctx, findimpls.Query{
		WorkDir:          projectDir,
		Packages:         pkgs,
		InterfacePackage: wdebugImportPath,
		InterfaceName:    wdebugInterfaceName,
		Methods:          []string{"Type", "Documentation"},
	})
	if err != nil {
		return nil, err
	}

	result := make(map[*packages.Package][]DiagnosticHandlerMetadata)
	for pkg, pkgResults := range findResult {
		result[pkg] = nil
		for _, impl := range pkgResults {
			typeMethod, ok := impl.Methods["Type"]
			if !ok {
				return nil, fmt.Errorf("method Type() was not found on the impl %s", impl.ImplType.String())
			}
			typeValue, err := getStringFromFuncBody(typeMethod.ASTFunc.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to extract return value from Type() method for impl %s: %v", impl.ImplType.String(), err)
			}

			docsMethod, ok := impl.Methods["Documentation"]
			if !ok {
				return nil, fmt.Errorf("method Documentation() was not found on the impl %s", impl.ImplType.String())
			}
			docsValue, err := getStringFromFuncBody(docsMethod.ASTFunc.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to extract return value from Documentation() method for impl %s: %v", impl.ImplType.String(), err)
			}

			result[pkg] = append(result[pkg], DiagnosticHandlerMetadata{
				DiagnosticType: typeValue,
				DiagnosticDocs: docsValue,
			})
		}
	}

	return result, nil
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

func getStringFromFuncBody(body *ast.BlockStmt) (string, error) {
	if len(body.List) != 1 {
		return "", fmt.Errorf("expected single-line method body, got %v", body.List)
	}
	returnStmt, ok := body.List[0].(*ast.ReturnStmt)
	if !ok {
		return "", fmt.Errorf("expected return statement, got %T %v", body.List[0], body.List[0])
	}
	return derefStringValue(returnStmt.Results[0])
}

func derefStringValue(expr ast.Expr) (string, error) {
	switch v := expr.(type) {
	default:
		return "", fmt.Errorf("expected literal or constant return value, got %T %v", expr, expr)
	case *ast.BasicLit:
		if v.Kind != token.STRING {
			return "", fmt.Errorf("expected basic value to be string, got %v", v.Kind.String())
		}
		return strconv.Unquote(v.Value)
	case *ast.Ident:
		valueSpec, ok := v.Obj.Decl.(*ast.ValueSpec)
		if !ok {
			return "", fmt.Errorf("expected ident value to have a value, got %T", v.Obj.Decl)
		}
		if len(valueSpec.Values) != 1 {
			return "", fmt.Errorf("expected ident value to have a single value, got %v", valueSpec.Values)
		}
		return derefStringValue(valueSpec.Values[0])
	}
}

func renderPackageDiagnosticsJSON(impls []DiagnosticHandlerMetadata) ([]byte, error) {
	sort.Slice(impls, func(i, j int) bool {
		return impls[i].DiagnosticType < impls[j].DiagnosticType
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
