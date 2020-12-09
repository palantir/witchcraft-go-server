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

package findimpls_test

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/internal/findimpls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/packages"
)

const testCodePkg = "github.com/palantir/witchcraft-go-server/tools/godel-sls-diagnostics-generator/internal/findimpls/testcode"

type lineMatcher struct {
	File string
	Line int
}

func TestFind(t *testing.T) {
	fset := token.NewFileSet()

	for _, test := range []struct {
		Name             string
		Packages         []string
		InterfacePackage string
		InterfaceName    string
		Methods          []string
		Expected         map[string]map[string]map[string]lineMatcher
	}{
		{
			Name:             "test1.MyInterface",
			Packages:         []string{testCodePkg + "/test1", testCodePkg + "/test1/mypackage"},
			InterfacePackage: testCodePkg + "/test1",
			InterfaceName:    "MyInterface",
			Methods:          []string{"String", "Int"},
			Expected: map[string]map[string]map[string]lineMatcher{
				testCodePkg + "/test1": {
					"MyInterfaceImpl1": {
						"String": {
							File: "impls.go",
							Line: 8,
						},
						"Int": {
							File: "impls.go",
							Line: 12,
						},
					},
					"MyInterfaceImpl2": {
						"String": {
							File: "impls.go",
							Line: 19,
						},
						"Int": {
							File: "impls.go",
							Line: 23,
						},
					},
					"myInterfaceImpl3": {
						"String": {
							File: "impls.go",
							Line: 40,
						},
						"Int": {
							File: "impls.go",
							Line: 44,
						},
					},
					"MyInterfaceImpl4": {
						"String": {
							File: "impls.go",
							Line: 58,
						},
						"Int": {
							File: "impls.go",
							Line: 62,
						},
					},
					"MyInterfaceImpl5": {
						"String": {
							File: "impls.go",
							Line: 68,
						},
						"Int": {
							File: "impls.go",
							Line: 72,
						},
					},
				},
				testCodePkg + "/test1/mypackage": {
					"MyInterfaceImpl1": {
						"String": {
							File: "mypackage.go",
							Line: 8,
						},
						"Int": {
							File: "mypackage.go",
							Line: 12,
						},
					},
					"MyInterfaceImpl2": {
						"String": {
							File: "mypackage.go",
							Line: 19,
						},
						"Int": {
							File: "mypackage.go",
							Line: 23,
						},
					},
					"myInterfaceImpl3": {
						"String": {
							File: "mypackage.go",
							Line: 40,
						},
						"Int": {
							File: "mypackage.go",
							Line: 44,
						},
					},
				},
			},
		},
		{
			Name:             "test1.myPrivateInterface",
			Packages:         []string{testCodePkg + "/test1", testCodePkg + "/test1/mypackage"},
			InterfacePackage: testCodePkg + "/test1",
			InterfaceName:    "myPrivateInterface",
			Methods:          []string{"PublicString"},
			Expected: map[string]map[string]map[string]lineMatcher{
				testCodePkg + "/test1": {
					"MyInterfaceImpl2": {
						"PublicString": {
							File: "impls.go",
							Line: 27,
						},
					},
					"myInterfaceImpl3": {
						"PublicString": {
							File: "impls.go",
							Line: 48,
						},
					},
				},
				testCodePkg + "/test1/mypackage": {
					"MyInterfaceImpl2": {
						"PublicString": {
							File: "mypackage.go",
							Line: 27,
						},
					},
					"myInterfaceImpl3": {
						"PublicString": {
							File: "mypackage.go",
							Line: 48,
						},
					},
				},
			},
		},
		{
			Name:             "test1.myPrivateInterfaceWithPrivateMethod",
			Packages:         []string{testCodePkg + "/test1", testCodePkg + "/test1/mypackage"},
			InterfacePackage: testCodePkg + "/test1",
			InterfaceName:    "myPrivateInterfaceWithPrivateMethod",
			Methods:          []string{"PublicString", "privateString"},
			Expected: map[string]map[string]map[string]lineMatcher{
				testCodePkg + "/test1": {
					"MyInterfaceImpl2": {
						"PublicString": {
							File: "impls.go",
							Line: 27,
						},
						"privateString": {
							File: "impls.go",
							Line: 31,
						},
					},
					"myInterfaceImpl3": {
						"PublicString": {
							File: "impls.go",
							Line: 48,
						},
						"privateString": {
							File: "impls.go",
							Line: 52,
						},
					},
				},
				testCodePkg + "/test1/mypackage": {},
			},
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			result, err := findimpls.Find(&findimpls.Query{
				LoadConfig: &packages.Config{
					Dir:  ".",
					Fset: fset,
				},
				Packages:         test.Packages,
				InterfacePackage: test.InterfacePackage,
				InterfaceName:    test.InterfaceName,
				Methods:          test.Methods,
			})
			require.NoError(t, err)
			convertedResult := convertResultsToTestableMap(t, result)
			assert.Equal(t, test.Expected, convertedResult)
		})
	}
}

func convertResultsToTestableMap(t *testing.T, results findimpls.ResultPackages) map[string]map[string]map[string]lineMatcher {
	packageMap := map[string]map[string]map[string]lineMatcher{}
	for resultPackage, resultTypes := range results {
		typesMap := map[string]map[string]lineMatcher{}
		for resultType, resultMethods := range resultTypes {
			typeName := trimFullTypeName(resultType.String())
			methodsMap := map[string]lineMatcher{}
			for methodName, resultMethod := range resultMethods {
				assert.Equal(t, typeName, astFuncReceiverTypeName(resultMethod), "method receiver did not match resultType in map")
				assert.Equal(t, methodName, resultMethod.Name.String(), "method decl name did not match name in map")
				position := resultPackage.Fset.Position(resultMethod.Pos())
				methodsMap[methodName] = lineMatcher{
					File: filepath.Base(position.Filename),
					Line: position.Line,
				}
			}
			typesMap[typeName] = methodsMap
		}
		packageMap[resultPackage.ID] = typesMap
	}
	return packageMap
}

func astFuncReceiverTypeName(decl *ast.FuncDecl) string {
	return decl.Recv.List[0].Type.(*ast.Ident).Name
}

func trimFullTypeName(fullName string) string {
	separator := strings.LastIndex(fullName, ".")
	return fullName[separator+1:]
}
