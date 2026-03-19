// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package wdebug

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockDiagnosticWriter struct {
	writeFn func(ctx context.Context, w io.Writer) error
}

func (m *mockDiagnosticWriter) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	return m.writeFn(ctx, w)
}

func TestNewDelegatingDiagnosticHandler_ReturnsConfigValues(t *testing.T) {
	handler := NewDelegatingDiagnosticHandler(DiagnosticHandlerConfig{
		DiagnosticType: "test.type.v1",
		Documentation:  "test documentation",
		ContentType:    "text/plain",
		SafeLoggable:   false,
		Extension:      "txt",
	})
	assert.Equal(t, DiagnosticType("test.type.v1"), handler.Type())
	assert.Equal(t, "test documentation", handler.Documentation())
	assert.Equal(t, "text/plain", handler.ContentType())
	assert.False(t, handler.SafeLoggable())
	assert.Equal(t, "txt", handler.Extension())
}

func TestDelegatingDiagnosticHandler_WriteDiagnostic_BeforeSetWriter(t *testing.T) {
	handler := NewDelegatingDiagnosticHandler(DiagnosticHandlerConfig{
		DiagnosticType: "test.type.v1",
		Documentation:  "test",
		ContentType:    "application/json",
		Extension:      "json",
	})
	var buf bytes.Buffer
	err := handler.WriteDiagnostic(context.Background(), &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "diagnostic writer not yet initialized")
}

func TestDelegatingDiagnosticHandler_WriteDiagnostic_DelegatesToWriter(t *testing.T) {
	handler := NewDelegatingDiagnosticHandler(DiagnosticHandlerConfig{
		DiagnosticType: "test.type.v1",
		Documentation:  "test",
		ContentType:    "application/json",
		Extension:      "json",
	})
	handler.SetWriter(&mockDiagnosticWriter{
		writeFn: func(ctx context.Context, w io.Writer) error {
			return nil
		},
	})
	var buf bytes.Buffer
	err := handler.WriteDiagnostic(context.Background(), &buf)
	assert.NoError(t, err)
}

func TestDelegatingDiagnosticHandler_WriteDiagnostic_PropagatesWriterError(t *testing.T) {
	handler := NewDelegatingDiagnosticHandler(DiagnosticHandlerConfig{
		DiagnosticType: "test.type.v1",
		Documentation:  "test",
		ContentType:    "application/json",
		Extension:      "json",
	})
	handler.SetWriter(&mockDiagnosticWriter{
		writeFn: func(ctx context.Context, w io.Writer) error {
			return errors.New("writer error")
		},
	})
	var buf bytes.Buffer
	err := handler.WriteDiagnostic(context.Background(), &buf)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "writer error")
}
