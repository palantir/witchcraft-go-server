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

package audit3log

import (
	"fmt"
	"io"
	"os"

	wloginternal "github.com/palantir/witchcraft-go-logging/wlog/internal"
)

func SetDefaultLoggerCreator(creator func() Logger) {
	defaultLoggerCreator = creator
}

var defaultLoggerCreator = func() Logger {
	return &warnLogger{
		w: os.Stderr,
	}
}

// warnLogger is a logger that writes a warning to the provided io.Writer whenever its logging function is invoked.
type warnLogger struct {
	w io.Writer
}

func (l *warnLogger) Audit(name string, result AuditResultType, params ...Param) {
	// Ignore the audit log output to prevent leaking sensitive data
	_, _ = fmt.Fprintln(l.w, wloginternal.WarnLoggerOutput("audit3log", "", 2))
}
