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
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_Logger(t *testing.T) {
	var buff bytes.Buffer
	recorder := httptest.NewRecorder()

	l := NewLogger()
	l.ALogger = log.New(&buff, "[negroni] ", 0)

	n := New()
	// replace log for testing
	n.Use(l)
	n.UseHandler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusNotFound)
	}))

	req, err := http.NewRequest("GET", "http://localhost:3000/foobar", nil)
	if err != nil {
		t.Error(err)
	}

	n.ServeHTTP(recorder, req)
	expect(t, recorder.Code, http.StatusNotFound)
	refute(t, len(buff.String()), 0)
}

func Test_LoggerURLEncodedString(t *testing.T) {
	var buff bytes.Buffer
	recorder := httptest.NewRecorder()

	l := NewLogger()
	l.ALogger = log.New(&buff, "[negroni] ", 0)
	l.SetFormat("{{.Path}}")

	n := New()
	// replace log for testing
	n.Use(l)
	n.UseHandler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))

	// Test reserved characters - !*'();:@&=+$,/?%#[]
	req, err := http.NewRequest("GET", "http://localhost:3000/%21%2A%27%28%29%3B%3A%40%26%3D%2B%24%2C%2F%3F%25%23%5B%5D", nil)
	if err != nil {
		t.Error(err)
	}

	n.ServeHTTP(recorder, req)
	expect(t, recorder.Code, http.StatusOK)
	expect(t, strings.TrimSpace(buff.String()), "[negroni] /!*'();:@&=+$,/?%#[]")
	refute(t, len(buff.String()), 0)
}

func Test_LoggerCustomFormat(t *testing.T) {
	var buff bytes.Buffer
	recorder := httptest.NewRecorder()

	l := NewLogger()
	l.ALogger = log.New(&buff, "[negroni] ", 0)
	l.SetFormat("{{.Request.URL.Query.Get \"foo\"}} {{.Request.UserAgent}} - {{.Status}}")

	n := New()
	n.Use(l)
	n.UseHandler(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, _ = rw.Write([]byte("OK"))
	}))

	userAgent := "Negroni-Test"
	req, err := http.NewRequest("GET", "http://localhost:3000/foobar?foo=bar", nil)
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("User-Agent", userAgent)

	n.ServeHTTP(recorder, req)
	expect(t, strings.TrimSpace(buff.String()), "[negroni] bar "+userAgent+" - 200")
}
