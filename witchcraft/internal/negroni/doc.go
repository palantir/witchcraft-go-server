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

// Package negroni is an idiomatic approach to web middleware in Go. It is tiny, non-intrusive, and encourages use of net/http Handlers.
//
// If you like the idea of Martini, but you think it contains too much magic, then Negroni is a great fit.
//
// For a full guide visit http://github.com/urfave/negroni
//
//  package main
//
//  import (
//    "github.com/urfave/negroni"
//    "net/http"
//    "fmt"
//  )
//
//  func main() {
//    mux := http.NewServeMux()
//    mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
//      fmt.Fprintf(w, "Welcome to the home page!")
//    })
//
//    n := negroni.Classic()
//    n.UseHandler(mux)
//    n.Run(":3000")
//  }
package negroni
