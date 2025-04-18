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
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/wrappedlog/wrapped1log"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft/internal/metricloggers"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	defaultLogOutputFormat = "var/log/%s.log"
	containerEnvVariable   = "CONTAINER"
)

// initDefaultLoggers initializes the Server loggers with instrumented loggers that record metrics in the given registry.
// If useConsoleLog is true, then all loggers log to stdout.
// The provided logLevel is used when initializing the service logs only.
// If the tcpWriter is provided, then it will be added as an additional output writer for all log types.
func (s *Server) initDefaultLoggers(useConsoleLog bool, logLevel wlog.LogLevel, registry metrics.Registry) {
	var originParam svc1log.Param
	switch {
	case s.svcLogOrigin != nil && *s.svcLogOrigin != "":
		originParam = svc1log.Origin(*s.svcLogOrigin)
	case s.svcLogOriginFromCallLine:
		// Skip two frames because we wrap the default logger in the metric logger
		originParam = svc1log.OriginFromCallLineWithSkip(2)
	default:
		// if origin param is not specified, use a param that uses the package name of the caller of Start()
		originParam = svc1log.Origin(svc1log.CallerPkg(2, 0))
	}

	var loggerStdoutWriter io.Writer = os.Stdout
	if s.loggerStdoutWriter != nil {
		loggerStdoutWriter = s.loggerStdoutWriter
	}

	logWriterFn := func(slsFilename string) io.Writer {
		internalWriter := newDefaultLogOutputWriter(slsFilename, useConsoleLog, loggerStdoutWriter)
		return metricloggers.NewMetricWriter(internalWriter, registry, slsFilename)
	}

	// initialize instrumented loggers
	s.svcLogger = metricloggers.NewSvc1Logger(svc1log.New(logWriterFn("service"), logLevel, originParam), registry)
	s.evtLogger = metricloggers.NewEvt2Logger(
		evt2log.New(logWriterFn("event")), registry)
	s.metricLogger = metricloggers.NewMetric1Logger(
		metric1log.New(logWriterFn("metrics")), registry)
	s.trcLogger = metricloggers.NewTrc1Logger(
		trc1log.New(logWriterFn("trace")), registry)

	var audit2Logger audit2log.Logger
	if s.dualLogAuditV2ToAuditV3 {
		audit2Logger = audit2log.NewDualLogger(logWriterFn("audit"), logWriterFn("audit.v3"))
	} else {
		audit2Logger = audit2log.New(logWriterFn("audit"))
	}
	s.audit2Logger = metricloggers.NewAudit2Logger(audit2Logger, registry)

	var audit3Logger audit3log.Logger
	if s.dualLogAuditV3ToAuditV2 {
		audit3Logger = audit3log.NewDualLogger(logWriterFn("audit.v3"), logWriterFn("audit"))
	} else {
		audit3Logger = audit3log.New(logWriterFn("audit.v3"))
	}
	audit3Logger = metricloggers.NewAudit3Logger(audit3Logger, registry)
	s.audit3Logger.Store(&audit3Logger)

	s.diagLogger = metricloggers.NewDiag1Logger(diag1log.New(logWriterFn("diagnostic")), registry)
	s.reqLogger = metricloggers.NewReq2Logger(req2log.New(logWriterFn("request"),
		req2log.Extractor(s.idsExtractor),
		req2log.SafePathParams(s.safePathParams...),
		req2log.SafeHeaderParams(s.safeHeaderParams...),
		req2log.SafeQueryParams(s.safeQueryParams...),
	), registry)
}

// initWrappedLoggers initializes the Server loggers with instrumented loggers that record metrics in the given registry
// and emit logs in wrapped.1 format.
// If useConsoleLog is true, then all loggers log to stdout.
// The provided logLevel is used when initializing the service logs only.
// productName is used as the entityName in wrapped.1 format logs
// productVersion is used as the entityVersion in wrapped.1 format logs
func (s *Server) initWrappedLoggers(useConsoleLog bool, productName, productVersion string, logLevel wlog.LogLevel, registry metrics.Registry) {
	var originParam svc1log.Param
	switch {
	case s.svcLogOrigin != nil && *s.svcLogOrigin != "":
		originParam = svc1log.Origin(*s.svcLogOrigin)
	case s.svcLogOriginFromCallLine:
		// Wrapped five frames for wrapped logger
		originParam = svc1log.OriginFromCallLineWithSkip(5)
	default:
		// if origin param is not specified, use a param that uses the package name of the caller of Start()
		originParam = svc1log.Origin(svc1log.CallerPkg(2, 0))
	}

	var loggerStdoutWriter io.Writer = os.Stdout
	if s.loggerStdoutWriter != nil {
		loggerStdoutWriter = s.loggerStdoutWriter
	}

	logWriterFn := func(slsFilename string) io.Writer {
		internalWriter := newDefaultLogOutputWriter(slsFilename, useConsoleLog, loggerStdoutWriter)
		return metricloggers.NewMetricWriter(internalWriter, registry, slsFilename)
	}

	// initialize instrumented wrapped loggers
	s.svcLogger = metricloggers.NewSvc1Logger(
		wrapped1log.New(logWriterFn("service"), logLevel, productName, productVersion).Service(originParam), registry)
	s.evtLogger = metricloggers.NewEvt2Logger(
		wrapped1log.New(logWriterFn("event"), logLevel, productName, productVersion).Event(), registry)
	s.metricLogger = metricloggers.NewMetric1Logger(
		wrapped1log.New(logWriterFn("metrics"), logLevel, productName, productVersion).Metric(), registry)
	s.trcLogger = metricloggers.NewTrc1Logger(
		wrapped1log.New(logWriterFn("trace"), logLevel, productName, productVersion).Trace(), registry)
	s.audit2Logger = metricloggers.NewAudit2Logger(
		wrapped1log.New(logWriterFn("audit"), logLevel, productName, productVersion).Audit(), registry)
	audit3Logger := metricloggers.NewAudit3Logger(
		wrapped1log.New(logWriterFn("audit.v3"), logLevel, productName, productVersion).AuditV3(), registry)
	s.audit3Logger.Store(&audit3Logger)
	s.diagLogger = metricloggers.NewDiag1Logger(
		wrapped1log.New(logWriterFn("diagnostic"), logLevel, productName, productVersion).Diagnostic(), registry)
	s.reqLogger = metricloggers.NewReq2Logger(wrapped1log.New(logWriterFn("request"), logLevel, productName, productVersion).Request(
		req2log.Extractor(s.idsExtractor),
		req2log.SafePathParams(s.safePathParams...),
		req2log.SafeHeaderParams(s.safeHeaderParams...),
		req2log.SafeQueryParams(s.safeQueryParams...),
	), registry)
}

// resolveHostName resolves the hostname using os.Hostname. Returns an error if the hostname cannot be resolved or if
// the resolved hostname is "localhost". Used to determine the hostname field for audit logs.
func resolveHostName() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", werror.Wrap(err, "failed to get hostname")
	}
	if hostname == "localhost" {
		return "", werror.Error("hostname discovery failed", werror.SafeParam("host", hostname))
	}
	return hostname, nil
}

func (s *Server) updateAuditLoggerConfig(auditConfig *config.AuditConfig) {
	audit3Logger := *s.audit3Logger.Load()
	if audit3Logger == nil {
		return
	}

	// set values based on configuration (empty string if not set)
	var deployment, product, productVersion, stack, service, environment string
	if auditConfig != nil {
		deployment = auditConfig.Deployment
		product = auditConfig.Product
		productVersion = auditConfig.ProductVersion
		stack = auditConfig.Stack
		service = auditConfig.Service
		environment = auditConfig.Environment
	}

	host, err := resolveHostName()
	if err != nil && s.svcLogger != nil {
		s.svcLogger.Warn("failed to resolve hostname for audit logger", svc1log.Stacktrace(err))
	}

	audit3Logger = audit3log.WithParams(audit3Logger,
		audit3log.Deployment(deployment),
		audit3log.Host(host),
		audit3log.Product(product),
		audit3log.ProductVersion(productVersion),
		audit3log.Stack(stack),
		audit3log.Service(service),
		audit3log.Environment(environment),
		audit3log.ProducerType(audit3log.AuditProducerServer),
	)

	// set audit3 logger params
	s.audit3Logger.Store(&audit3Logger)
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
	return isContainer() && !isJail()
}

func isContainer() bool {
	return isDocker() || isContainerByEnvVar()
}

func isContainerByEnvVar() bool {
	_, present := os.LookupEnv(containerEnvVariable)
	return present
}

func isDocker() bool {
	fi, err := os.Stat("/.dockerenv")
	return err == nil && !fi.IsDir()
}

func isJail() bool {
	hostname, err := os.Hostname()
	return err == nil && strings.Contains(hostname, "-jail-")
}
