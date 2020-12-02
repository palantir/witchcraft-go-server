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

package findimpls

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	werror "github.com/palantir/witchcraft-go-error"
	"golang.org/x/tools/go/packages"
)

type Query struct {
	// packages.Load options
	WorkDir  string // if unset, use cwd
	Tests    bool
	Packages []string

	// interface to find implementations

	InterfacePackage string
	InterfaceName    string

	// Methods subset to extract syntax
	Methods []string
}

type ResultImplementation struct {
	ImplType types.Type
	Methods  map[string]*ResultMethod
}

type ResultMethod struct {
	TypesFunc *types.Func
	ASTFunc   *ast.FuncDecl
}

// Find searches through packages specified in the Query for the InterfacePackage/InterfaceName provided.
// Once the interface is found, all packages are searched for struct types which implement the interface.
// The returned map has a possibly-empty entry for all packages so that absence can be easily asserted.
func Find(ctx context.Context, query Query) (map[*packages.Package][]ResultImplementation, error) {
	loadedPkgs, err := packages.Load(&packages.Config{
		Mode:    packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo | packages.NeedDeps,
		Context: ctx,
		Dir:     query.WorkDir,
		Fset:    token.NewFileSet(),
		Tests:   false,
	}, query.Packages...)
	if err != nil {
		return nil, werror.Wrap(err, "failed to load project packages")
	}

	ifaceType, err := findInterface(loadedPkgs, query.InterfacePackage, query.InterfaceName)
	if err != nil {
		return nil, err
	}

	result := make(map[*packages.Package][]ResultImplementation)
	for _, pkg := range loadedPkgs {
		impls, err := findInterfaceImplementations(ifaceType.Underlying().(*types.Interface), pkg)
		if err != nil {
			return nil, err
		}
		pkgResult, err := loadImplMetadata(impls, pkg, query.Methods)
		if err != nil {
			return nil, err
		}

		result[pkg] = pkgResult
	}
	return result, nil
}

func findInterface(loadedPkgs []*packages.Package, ifaceImportPath, ifaceName string) (*types.Named, error) {
	var ifacePkg *packages.Package
	var ifaceIdent *ast.Ident
	var ifaceObject types.Object
	for i := range loadedPkgs {
		if loadedPkgs[i].ID == ifaceImportPath {
			ifacePkg = loadedPkgs[i]
			for ident, object := range ifacePkg.TypesInfo.Defs {
				if ident != nil && ident.Obj != nil && ident.Obj.Kind == ast.Typ && ident.Name == ifaceName {
					ifaceIdent = ident
					ifaceObject = object
					break
				}
			}
			if ifaceIdent == nil {
				return nil, werror.Error("did not find interface type in loaded packages")
			}
			break
		}
	}
	if ifacePkg == nil {
		return nil, werror.Error("did not find interface package")
	}

	ifaceType := ifaceObject.Type().(*types.Named)
	return ifaceType, nil
}

func findInterfaceImplementations(iface *types.Interface, pkg *packages.Package) ([]types.Type, error) {
	var results []types.Type
	for ident, object := range pkg.TypesInfo.Defs {
		if ident != nil && ident.Obj != nil && ident.Obj.Kind == ast.Typ {
			typ := object.Type()
			if _, ok := typ.Underlying().(*types.Struct); !ok {
				continue
			}
			if types.Implements(typ, iface) {
				results = append(results, typ)
			} else if ptr := types.NewPointer(typ); types.Implements(ptr, iface) {
				results = append(results, ptr)
			}
		}
	}
	return results, nil
}

func loadImplMetadata(impls []types.Type, pkg *packages.Package, methodNames []string) ([]ResultImplementation, error) {
	var result []ResultImplementation
	for _, impl := range impls {
		methods := make(map[string]*ResultMethod, len(methodNames))
		for _, methodName := range methodNames {
			methodObj, _, _ := types.LookupFieldOrMethod(impl, true, pkg.Types, methodName)
			if methodObj == nil {
				return nil, fmt.Errorf("did not find method %s on type %s", methodName, impl.String())
			}
			method, ok := methodObj.(*types.Func)
			if !ok {
				return nil, fmt.Errorf("field %s on type %s should be a function", method, impl.String())
			}
			// Sets incomplete ResultMethod structs (they do not have ASTFunc populated yet)
			methods[methodName] = &ResultMethod{
				TypesFunc: method,
			}
		}
		result = append(result, ResultImplementation{
			ImplType: impl,
			Methods:  methods,
		})
	}

	var allMethodDecls []*ast.FuncDecl
	for _, astFile := range pkg.Syntax {
		for _, decl := range astFile.Decls {
			declFunc, ok := decl.(*ast.FuncDecl)
			if ok && declFunc.Recv != nil {
				allMethodDecls = append(allMethodDecls, declFunc)
			}
		}
	}

	for _, r := range result {
		for _, m := range r.Methods {
			for _, d := range allMethodDecls {
				if m.TypesFunc.Scope().Pos() == d.Body.Pos() {
					m.ASTFunc = d
				}
			}
		}
	}
	return result, nil
}
