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

package negroni

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatic(t *testing.T) {
	response := httptest.NewRecorder()
	response.Body = new(bytes.Buffer)

	n := New()
	n.Use(NewStatic(http.Dir(".")))

	req, err := http.NewRequest("GET", "http://localhost:3000/negroni.go", nil)
	if err != nil {
		t.Error(err)
	}
	n.ServeHTTP(response, req)
	expect(t, response.Code, http.StatusOK)
	expect(t, response.Header().Get("Expires"), "")
	if response.Body.Len() == 0 {
		t.Errorf("Got empty body for GET request")
	}
}

func TestStaticHead(t *testing.T) {
	response := httptest.NewRecorder()
	response.Body = new(bytes.Buffer)

	n := New()
	n.Use(NewStatic(http.Dir(".")))
	n.UseHandler(http.NotFoundHandler())

	req, err := http.NewRequest("HEAD", "http://localhost:3000/negroni.go", nil)
	if err != nil {
		t.Error(err)
	}

	n.ServeHTTP(response, req)
	expect(t, response.Code, http.StatusOK)
	if response.Body.Len() != 0 {
		t.Errorf("Got non-empty body for HEAD request")
	}
}

func TestStaticAsPost(t *testing.T) {
	response := httptest.NewRecorder()

	n := New()
	n.Use(NewStatic(http.Dir(".")))
	n.UseHandler(http.NotFoundHandler())

	req, err := http.NewRequest("POST", "http://localhost:3000/negroni.go", nil)
	if err != nil {
		t.Error(err)
	}

	n.ServeHTTP(response, req)
	expect(t, response.Code, http.StatusNotFound)
}

func TestStaticBadDir(t *testing.T) {
	response := httptest.NewRecorder()

	n := Classic()
	n.UseHandler(http.NotFoundHandler())

	req, err := http.NewRequest("GET", "http://localhost:3000/negroni.go", nil)
	if err != nil {
		t.Error(err)
	}

	n.ServeHTTP(response, req)
	refute(t, response.Code, http.StatusOK)
}

func TestStaticOptionsServeIndex(t *testing.T) {
	response := httptest.NewRecorder()

	n := New()
	s := NewStatic(http.Dir("."))
	s.IndexFile = "negroni.go"
	n.Use(s)

	req, err := http.NewRequest("GET", "http://localhost:3000/", nil)
	if err != nil {
		t.Error(err)
	}

	n.ServeHTTP(response, req)
	expect(t, response.Code, http.StatusOK)
}

func TestStaticOptionsPrefix(t *testing.T) {
	response := httptest.NewRecorder()

	n := New()
	s := NewStatic(http.Dir("."))
	s.Prefix = "/public"
	n.Use(s)

	// Check file content behaviour
	req, err := http.NewRequest("GET", "http://localhost:3000/public/negroni.go", nil)
	if err != nil {
		t.Error(err)
	}

	n.ServeHTTP(response, req)
	expect(t, response.Code, http.StatusOK)
}
