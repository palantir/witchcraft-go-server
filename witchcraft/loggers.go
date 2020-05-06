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

// initLoggers initializes the Server loggers with the provided logLevel.
// The loggers are instrumented and will record metrics in the given registry.
// If useConsoleLog is true, then all loggers log to stdout.
func (s *Server) initLoggers(registry metrics.Registry, useConsoleLog bool, logLevel wlog.LogLevel) {
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

	logWriterFn := func(slsFilename string) metricloggers.MetricWriter {
		return newDefaultLogOutputWriter(registry, slsFilename, useConsoleLog, loggerStdoutWriter)
	}

	// initialize instrumented loggers
	s.svcLogger = metricloggers.NewSvc1Logger(
		svc1log.New(logWriterFn("service"), logLevel, svc1LogParams...), registry)
	s.evtLogger = metricloggers.NewEvt2Logger(
		evt2log.New(logWriterFn("event")), registry)
	s.metricLogger = metricloggers.NewMetric1Logger(
		metric1log.New(logWriterFn("metrics")), registry)
	s.trcLogger = metricloggers.NewTrc1Logger(
		trc1log.New(logWriterFn("trace")), registry)
	s.auditLogger = metricloggers.NewAudit2Logger(
		audit2log.New(logWriterFn("audit")), registry)
	s.diagLogger = diag1log.New(logWriterFn("diagnostic"))
	s.reqLogger = req2log.New(logWriterFn("request"),
		req2log.Extractor(s.idsExtractor),
		req2log.SafePathParams(s.safePathParams...),
		req2log.SafeHeaderParams(s.safeHeaderParams...),
		req2log.SafeQueryParams(s.safeQueryParams...),
	)
}

// Returns a MetricWriter which can be used as the underlying writer for a logger.
// If either logToStdout or logToStdoutBasedOnEnv() is true, then the provided stdoutWriter is used within the MetricWriter.
// Otherwise, the returned MetricWriter will use a default writer that writes to slsFilename.
func newDefaultLogOutputWriter(registry metrics.Registry, slsFilename string, logToStdout bool, stdoutWriter io.Writer) metricloggers.MetricWriter {
	var internalWriter io.Writer
	if logToStdout || logToStdoutBasedOnEnv() {
		internalWriter = stdoutWriter
	} else {
		internalWriter = &lumberjack.Logger{
			Filename:   fmt.Sprintf(defaultLogOutputFormat, slsFilename),
			MaxSize:    1000,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
		}
	}
	return metricloggers.NewMetricWriter(internalWriter, registry, slsFilename)
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
