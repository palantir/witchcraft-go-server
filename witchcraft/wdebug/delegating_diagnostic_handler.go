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
	"context"
	"io"
	"sync/atomic"

	werror "github.com/palantir/witchcraft-go-error"
)

// DiagnosticWriter is the interface that resource-specific diagnostic implementations must satisfy.
type DiagnosticWriter interface {
	WriteDiagnostic(ctx context.Context, w io.Writer) error
}

// DelegatingDiagnosticHandler extends DiagnosticHandler with the ability to lazily set
// the underlying writer after construction.
type DelegatingDiagnosticHandler interface {
	DiagnosticHandler
	SetWriter(writer DiagnosticWriter)
}

// DiagnosticHandlerConfig holds the static metadata for a diagnostic handler.
type DiagnosticHandlerConfig struct {
	DiagnosticType DiagnosticType
	Documentation  string
	ContentType    string
	SafeLoggable   bool
	Extension      string
}

// diagnosticHandler is the concrete implementation of DelegatingDiagnosticHandler.
type diagnosticHandler struct {
	config DiagnosticHandlerConfig
	writer atomic.Pointer[DiagnosticWriter]
}

var _ DelegatingDiagnosticHandler = &diagnosticHandler{}

// NewDelegatingDiagnosticHandler creates a new DelegatingDiagnosticHandler with the given configuration.
// Call SetWriter once the resource-specific dependencies are available.
func NewDelegatingDiagnosticHandler(config DiagnosticHandlerConfig) DelegatingDiagnosticHandler {
	return &diagnosticHandler{
		config: config,
	}
}

// SetWriter sets the resource-specific writer that handles WriteDiagnostic calls.
func (h *diagnosticHandler) SetWriter(writer DiagnosticWriter) {
	h.writer.Store(&writer)
}

func (h *diagnosticHandler) Type() DiagnosticType {
	return h.config.DiagnosticType
}

func (h *diagnosticHandler) Documentation() string {
	return h.config.Documentation
}

func (h *diagnosticHandler) ContentType() string {
	return h.config.ContentType
}

func (h *diagnosticHandler) SafeLoggable() bool {
	return h.config.SafeLoggable
}

func (h *diagnosticHandler) Extension() string {
	return h.config.Extension
}

func (h *diagnosticHandler) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	writer := h.writer.Load()
	if writer == nil {
		return werror.ErrorWithContextParams(ctx, "diagnostic writer not yet initialized")
	}
	return (*writer).WriteDiagnostic(ctx, w)
}
