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
	"fmt"
	"go/ast"
	"go/token"
	"strconv"

	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/internal/findimpls"
	"golang.org/x/tools/go/packages"
)

// LoadDiagnosticHandlerImpls returns a mapping from each package to metadata describing its implementations
// of wdebug.DiagnosticHandler. It loads all go packages specified in pkgs, finds the interface type and its methods,
// then searches all packages for types that implement the interface. For each implementing type, a DiagnosticEntry
// describes the type, documentation, and whether it is safe-loggable. These values are extracted from the methods'
// implementations. Only methods returning basic types (literals or declared constants) are supported. More complex
// function bodies will result in an error.
//
// Note that packages are loaded based on the build flags existing in our process's environment (e.g. GOOS, GOARCH).
// Files with build constraints that are excluded under these parameters will be ignored.
func LoadDiagnosticHandlerImpls(ctx context.Context, pkgs []string, projectDir string) (map[*packages.Package][]DiagnosticEntry, error) {
	findResult, err := findimpls.Find(&findimpls.Query{
		LoadConfig: &packages.Config{
			Context: ctx,
			Dir:     projectDir,
			Fset:    token.NewFileSet(),
		},
		Packages:         pkgs,
		InterfacePackage: wdebugImportPath,
		InterfaceName:    wdebugInterfaceName,
		Methods:          []string{"Type", "Documentation", "SafeLoggable"},
	})
	if err != nil {
		return nil, err
	}

	result := make(map[*packages.Package][]DiagnosticEntry)
	for pkg, pkgResults := range findResult {
		result[pkg] = nil
		for typ, methods := range pkgResults {
			typeValue, err := getStringFromFuncBody(methods, "Type")
			if err != nil {
				return nil, fmt.Errorf("failed to extract return value from Type() method for impl %s: %v", typ.String(), err)
			}
			docsValue, err := getStringFromFuncBody(methods, "Documentation")
			if err != nil {
				return nil, fmt.Errorf("failed to extract return value from Documentation() method for impl %s: %v", typ.String(), err)
			}
			safeValue, err := getBoolFromFuncBody(methods, "SafeLoggable")
			if err != nil {
				return nil, fmt.Errorf("failed to extract return value from SafeLoggable() method for impl %s: %v", typ.String(), err)
			}
			result[pkg] = append(result[pkg], DiagnosticEntry{
				Type: typeValue,
				Docs: docsValue,
				Safe: &safeValue,
			})
		}
	}
	return result, nil
}

func getStringFromFuncBody(methods findimpls.ResultMethods, methodName string) (string, error) {
	returnValue, err := getSingleReturnFromFuncBody(methods, methodName)
	if err != nil {
		return "", err
	}
	return derefStringValue(returnValue)
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
		if v.Obj == nil {
			return "", fmt.Errorf("expected ident value to have an object, got %v", v)
		}
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

func getBoolFromFuncBody(methods findimpls.ResultMethods, methodName string) (bool, error) {
	returnValue, err := getSingleReturnFromFuncBody(methods, methodName)
	if err != nil {
		return false, err
	}
	return derefBoolValue(returnValue)
}

func derefBoolValue(expr ast.Expr) (bool, error) {
	switch v := expr.(type) {
	default:
		return false, fmt.Errorf("expected literal or constant return value, got %T %v", expr, expr)
	case *ast.Ident:
		if v.Obj == nil {
			switch v.Name {
			case "true":
				return true, nil
			case "false":
				return false, nil
			}
			return false, fmt.Errorf("expected ident value to be a boolean literal or have an object, got %v", v)
		}
		valueSpec, ok := v.Obj.Decl.(*ast.ValueSpec)
		if !ok {
			return false, fmt.Errorf("expected ident value to have a value, got %T", v.Obj.Decl)
		}
		if len(valueSpec.Values) != 1 {
			return false, fmt.Errorf("expected ident value to have a single value, got %v", valueSpec.Values)
		}
		return derefBoolValue(valueSpec.Values[0])
	}
}

func getSingleReturnFromFuncBody(methods findimpls.ResultMethods, methodName string) (ast.Expr, error) {
	methodAST, ok := methods[methodName]
	if !ok {
		return nil, fmt.Errorf("method %s() not found", methodName)
	}
	bodyList := methodAST.Body.List
	if len(bodyList) != 1 {
		return nil, fmt.Errorf("expected single-line method body, got %v", bodyList)
	}
	body := bodyList[0]
	returnStmt, ok := body.(*ast.ReturnStmt)
	if !ok {
		return nil, fmt.Errorf("expected return statement, got %T %v", body, body)
	}
	return returnStmt.Results[0], nil
}
