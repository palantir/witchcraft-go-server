// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

package zeroimpl

import (
	"fmt"
	"io"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/rs/zerolog"
)

func LoggerProvider() wlog.LoggerProvider {
	return &loggerProvider{}
}

type loggerProvider struct{}

func (lp *loggerProvider) NewLogger(w io.Writer) wlog.Logger {
	return &zeroLogger{
		logger: newZeroLogger(w, wlog.InfoLevel),
	}
}

func (lp *loggerProvider) NewLeveledLogger(w io.Writer, level wlog.LogLevel) wlog.LeveledLogger {
	return &zeroLogger{
		logger: newZeroLogger(w, level),
		level:  toZeroLevel(level),
	}
}

func newZeroLogger(w io.Writer, level wlog.LogLevel) zerolog.Logger {
	return zerolog.New(w).Level(toZeroLevel(level))
}

func toZeroLevel(lvl wlog.LogLevel) zerolog.Level {
	switch lvl {
	case wlog.DebugLevel:
		return zerolog.DebugLevel
	case wlog.LogLevel(""), wlog.InfoLevel:
		return zerolog.InfoLevel
	case wlog.WarnLevel:
		return zerolog.WarnLevel
	case wlog.ErrorLevel:
		return zerolog.ErrorLevel
	case wlog.FatalLevel:
		return zerolog.FatalLevel
	default:
		panic(fmt.Errorf("Invalid log level %q", lvl))
	}
}
