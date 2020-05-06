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

package witchcraft

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"github.com/palantir/witchcraft-go-server/witchcraft/internal/metricloggers"
	"gopkg.in/natefinch/lumberjack.v2"
)

func (s *Server) initLoggers(useConsoleLog bool, logLevel wlog.LogLevel) []metricloggers.MetricWriter {
	if s.svcLogOrigin == nil {
		// if origin param is not specified, use a param that uses the package name of the caller of Start()
		origin := svc1log.CallerPkg(2, 0)
		s.svcLogOrigin = &origin
	}
	var svc1LogParams []svc1log.Param
	if *s.svcLogOrigin != "" {
		svc1LogParams = append(svc1LogParams, svc1log.Origin(*s.svcLogOrigin))
	}
	var loggerStdoutWriter io.Writer = os.Stdout
	if s.loggerStdoutWriter != nil {
		loggerStdoutWriter = s.loggerStdoutWriter
	}

	logOutputFn := func(logOutputPath string) metricloggers.MetricWriter {
		return newDefaultLogOutput(logOutputPath, useConsoleLog, loggerStdoutWriter)
	}

	svcLogWriter := logOutputFn("service")
	s.svcLogger = svc1log.New(svcLogWriter, logLevel, svc1LogParams...)
	evtLogWriter := logOutputFn("event")
	s.evtLogger = evt2log.New(evtLogWriter)
	metricLogWriter := logOutputFn("metrics")
	s.metricLogger = metric1log.New(metricLogWriter)
	trcLogWriter := logOutputFn("trace")
	s.trcLogger = trc1log.New(trcLogWriter)
	auditLogWriter := logOutputFn("audit")
	s.auditLogger = audit2log.New(auditLogWriter)
	diagnosticLogWriter := logOutputFn("diagnostic")
	s.diagLogger = diag1log.New(diagnosticLogWriter)
	reqLogWriter := logOutputFn("request")
	s.reqLogger = req2log.New(reqLogWriter,
		req2log.Extractor(s.idsExtractor),
		req2log.SafePathParams(s.safePathParams...),
		req2log.SafeHeaderParams(s.safeHeaderParams...),
		req2log.SafeQueryParams(s.safeQueryParams...),
	)
	return []metricloggers.MetricWriter{
		svcLogWriter,
		evtLogWriter,
		metricLogWriter,
		trcLogWriter,
		auditLogWriter,
		diagnosticLogWriter,
		reqLogWriter,
	}
}

func newDefaultLogOutput(logOutputPath string, logToStdout bool, stdoutWriter io.Writer) metricloggers.MetricWriter {
	var internalWriter io.Writer
	if logToStdout || logToStdoutBasedOnEnv() {
		internalWriter = stdoutWriter
	} else {
		internalWriter = &lumberjack.Logger{
			Filename:   fmt.Sprintf("var/log/%s.log", logOutputPath),
			MaxSize:    1000,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
		}
	}
	return metricloggers.NewMetricWriter(internalWriter, logOutputPath)
}

// logToStdoutBasedOnEnv returns true if the runtime environment is a non-jail Docker container, false otherwise.
func logToStdoutBasedOnEnv() bool {
	return isDocker() && !isJail()
}

func isDocker() bool {
	fi, err := os.Stat("/.dockerenv")
	return err == nil && !fi.IsDir()
}

func isJail() bool {
	hostname, err := os.Hostname()
	return err == nil && strings.Contains(hostname, "-jail-")
}
