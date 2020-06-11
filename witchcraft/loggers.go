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

	"github.com/palantir/pkg/metrics"
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

const (
	defaultLogOutputFormat = "var/log/%s.log"
)

// initLoggers initializes the Server loggers with instrumented loggers that record metrics in the given registry.
// If useConsoleLog is true, then all loggers log to stdout.
// The provided logLevel is used when initializing the service logs only.
func (s *Server) initLoggers(useConsoleLog bool, logLevel wlog.LogLevel, registry metrics.Registry) {
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

	logWriterFn := func(slsFilename string) io.Writer {
		internalWriter := newDefaultLogOutputWriter(slsFilename, useConsoleLog, loggerStdoutWriter)
		return metricloggers.NewMetricWriter(internalWriter, registry, slsFilename)
	}

	// initialize default loggers
	s.svcLogger = svc1log.New(logWriterFn("service"), logLevel, svc1LogParams...)
	s.evtLogger = evt2log.New(logWriterFn("event"))
	s.metricLogger = metric1log.New(logWriterFn("metrics"))
	s.trcLogger = trc1log.New(logWriterFn("trace"))
	s.auditLogger = audit2log.New(logWriterFn("audit"))
	s.diagLogger = diag1log.New(logWriterFn("diagnostic"))
	s.reqLogger = req2log.New(logWriterFn("request"),
		req2log.Extractor(s.idsExtractor),
		req2log.SafePathParams(s.safePathParams...),
		req2log.SafeHeaderParams(s.safeHeaderParams...),
		req2log.SafeQueryParams(s.safeQueryParams...),
	)

	if registry != nil {
		s.instrumentExistingLoggers(registry)
	}
}

func (s *Server) instrumentExistingLoggers(registry metrics.Registry) {
	s.svcLogger = metricloggers.NewSvc1Logger(s.svcLogger, registry)
	s.evtLogger = metricloggers.NewEvt2Logger(s.evtLogger, registry)
	s.metricLogger = metricloggers.NewMetric1Logger(s.metricLogger, registry)
	s.trcLogger = metricloggers.NewTrc1Logger(s.trcLogger, registry)
	s.auditLogger = metricloggers.NewAudit2Logger(s.auditLogger, registry)
	s.diagLogger = metricloggers.NewDiag1Logger(s.diagLogger, registry)
	s.reqLogger = metricloggers.NewReq2Logger(s.reqLogger, registry)
}

// Returns a io.Writer that can be used as the underlying writer for a logger.
// If either logToStdout or logToStdoutBasedOnEnv() is true, then stdoutWriter is returned.
// Otherwise, a default writer that writes to slsFilename is returned.
func newDefaultLogOutputWriter(slsFilename string, logToStdout bool, stdoutWriter io.Writer) io.Writer {
	if logToStdout || logToStdoutBasedOnEnv() {
		return stdoutWriter
	}
	return &lumberjack.Logger{
		Filename:   fmt.Sprintf(defaultLogOutputFormat, slsFilename),
		MaxSize:    1000,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}
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
