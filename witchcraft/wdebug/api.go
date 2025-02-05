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
	"fmt"
	"io"
	"regexp"
)

var (
	diagnosticTypeRegexp = regexp.MustCompile(`^([a-z0-9]+.)+v[0-9]+$`)
)

// DiagnosticType describes the type of information provided in a given diagnostic payload.
// Type names must be specific, reasonably expected to be unique, and versioned to allow for future major changes of the payload structure.
// DiagnosticTypes must match the regular expression ([a-z0-9]+.)+v[0-9]+, i.e. be lower-case, dot-delimited, and end with a version suffix.
type DiagnosticType string

func (d DiagnosticType) Validate() error {
	if !diagnosticTypeRegexp.MatchString(string(d)) {
		return fmt.Errorf("'%s' does not match expected DiagnosticType format (%s)", d, diagnosticTypeRegexp.String())
	}
	return nil
}

// DiagnosticHandler provides methods for describing the type and nature of a diagnostic payload, as well as writing the payload to a writer.
type DiagnosticHandler interface {
	// Type returns the DiagnosticType for payloads written by this diagnostic handler.
	// See DiagnosticType docs for more information.
	Type() DiagnosticType
	// Documentation returns a human-readable description of the diagnostic payload.
	Documentation() string
	// ContentType returns the MIME type of the diagnostic payload.
	ContentType() string
	// SafeLoggable returns true if the diagnostic payload can be safely logged.
	SafeLoggable() bool
	// Extension is the file extension to use when writing the diagnostic payload to a file.
	Extension() string
	// WriteDiagnostic writes a diagnostic payload to the provided writer.
	WriteDiagnostic(ctx context.Context, w io.Writer) error
}
