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
	"sync"

	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/metriclog/metric1log"
	"github.com/palantir/witchcraft-go-logging/wlog/reqlog/req2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
	"github.com/palantir/witchcraft-go-logging/wlog/trclog/trc1log"
	"gopkg.in/natefinch/lumberjack.v2"
)

func (s *Server) initLoggers(useConsoleLog bool, logLevel wlog.LogLevel) {
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

	loggerFileWriterProvider := DefaultFileWriterProvider()
	if s.loggerFileWriterProvider != nil {
		loggerFileWriterProvider = s.loggerFileWriterProvider
	}

	loggerCreator := NewLoggerCreator(useConsoleLog, loggerStdoutWriter, loggerFileWriterProvider)

	s.svcLogger = loggerCreator.Service1Logger(logLevel, svc1LogParams...)
	s.evtLogger = loggerCreator.Event2Logger()
	s.metricLogger = loggerCreator.Metric1Logger()
	s.trcLogger = loggerCreator.Trace1Logger()
	s.auditLogger = loggerCreator.Audit2Logger()
	s.diagLogger = loggerCreator.Diagnostic1Logger()
	s.reqLogger = loggerCreator.Request2Logger(
		req2log.Extractor(s.idsExtractor),
		req2log.SafePathParams(s.safePathParams...),
		req2log.SafeHeaderParams(s.safeHeaderParams...),
		req2log.SafeQueryParams(s.safeQueryParams...),
	)
}

type FileWriterProvider interface {
	FileWriter(logOutputPath string) io.Writer
}

func DefaultFileWriterProvider() FileWriterProvider {
	return &defaultFileWriterProvider{}
}

type defaultFileWriterProvider struct{}

func (p *defaultFileWriterProvider) FileWriter(logOutputPath string) io.Writer {
	return &lumberjack.Logger{
		Filename:   logOutputPath,
		MaxSize:    1000,
		MaxBackups: 10,
		MaxAge:     30,
		Compress:   true,
	}
}

type cachingFileWriterProvider struct {
	delegate FileWriterProvider
	cache    map[string]io.Writer
	lock     sync.Mutex
}

// NewCachingFileWriterProvider returns a FileWriterProvider that uses the provided FileWriterProvider to create writers
// and always returns the same io.Writer for a given path. Is thread-safe.
func NewCachingFileWriterProvider(delegate FileWriterProvider) FileWriterProvider {
	return &cachingFileWriterProvider{
		delegate: delegate,
		cache:    make(map[string]io.Writer),
	}
}

func (p *cachingFileWriterProvider) FileWriter(logOutputPath string) io.Writer {
	p.lock.Lock()
	defer p.lock.Unlock()
	if _, ok := p.cache[logOutputPath]; !ok {
		// if not in cache, create and set
		p.cache[logOutputPath] = p.delegate.FileWriter(logOutputPath)
	}
	return p.cache[logOutputPath]
}

type LoggerCreator struct {
	useConsoleLog      bool
	consoleLogWriter   io.Writer
	fileWriterProvider FileWriterProvider
}

// NewLoggerCreator returns a *LoggerCreator that creates loggers using the provided parameters. This function is used
// internally by witchcraft.Server, and is thus useful in cases where code that executes before a witchcraft.Server
// wants to perform logging operations using the same writer as the one that the witchcraft.Server will use later (as
// opposed to the invoking code and server both using separate io.Writers for the same output destination, which could
// cause issues like overwriting the same file). Note that the loggers returned by this function are not updated based
// on configuration: if the client wants runtime configuration updates to modify the loggers, they must set this up
// themselves.
//
// The following is an example usage:
//
//   cachingWriterProvider := witchcraft.NewCachingFileWriterProvider(witchcraft.DefaultFileWriterProvider())
//   svc1Logger := witchcraft.NewLoggerCreator(false, os.Stdout, cachingWriterProvider).Service1Logger(wlog.DebugLevel)
//   server := witchcraft.NewServer().
//	 	 WithLoggerFileWriterProvider(cachingWriterProvider).
//		 ...
//
// In this example, the svc1Logger and the service logger for witchcraft.NewServer will both use the same writer
// provided by cachingWriterProvider.
func NewLoggerCreator(useConsoleLog bool, consoleLogWriter io.Writer, fileWriterProvider FileWriterProvider) *LoggerCreator {
	return &LoggerCreator{
		useConsoleLog:      useConsoleLog,
		consoleLogWriter:   consoleLogWriter,
		fileWriterProvider: fileWriterProvider,
	}
}

func (l *LoggerCreator) Service1Logger(logLevel wlog.LogLevel, svc1LogParams ...svc1log.Param) svc1log.Logger {
	return svc1log.New(l.logOutputFn("service"), logLevel, svc1LogParams...)
}

func (l *LoggerCreator) Event2Logger() evt2log.Logger {
	return evt2log.New(l.logOutputFn("event"))
}

func (l *LoggerCreator) Metric1Logger() metric1log.Logger {
	return metric1log.New(l.logOutputFn("metrics"))
}

func (l *LoggerCreator) Trace1Logger() trc1log.Logger {
	return trc1log.New(l.logOutputFn("trace"))
}

func (l *LoggerCreator) Audit2Logger() audit2log.Logger {
	return audit2log.New(l.logOutputFn("audit"))
}

func (l *LoggerCreator) Diagnostic1Logger() diag1log.Logger {
	return diag1log.New(l.logOutputFn("diagnostic"))
}

func (l *LoggerCreator) Request2Logger(params ...req2log.LoggerCreatorParam) req2log.Logger {
	return req2log.New(l.logOutputFn("request"), params...)
}

func (l *LoggerCreator) logOutputFn(logFileName string) io.Writer {
	return createLogWriter(fmt.Sprintf("var/log/%s.log", logFileName), l.useConsoleLog, l.consoleLogWriter, l.fileWriterProvider)
}

func createLogWriter(logOutputPath string, useConsoleLog bool, consoleLogWriter io.Writer, fileWriterProvider FileWriterProvider) io.Writer {
	if useConsoleLog || logToConsoleBasedOnEnv() {
		return consoleLogWriter
	}
	return fileWriterProvider.FileWriter(logOutputPath)
}

// logToConsoleBasedOnEnv returns true if the runtime environment is a non-jail Docker container, false otherwise.
func logToConsoleBasedOnEnv() bool {
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
