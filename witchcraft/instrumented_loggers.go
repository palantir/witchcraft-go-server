// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/diaglog/diag1log"
	"github.com/palantir/witchcraft-go-logging/wlog/evtlog/evt2log"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

const (
	infoMetricName  = "logs.service1.info"
	warnMetricName  = "logs.service1.warn"
	errorMetricName = "logs.service1.error"

	diagnosticsMetricName = "logs.diagnostic1"

	eventMetricName = "logs.event2"
)

var _ svc1log.Logger = &instrumentedSvc1Logger{}

func newInstrumentedSvc1Logger(logger svc1log.Logger, registry metrics.Registry) svc1log.Logger {
	return &instrumentedSvc1Logger{
		logger:   logger,
		registry: registry,
	}
}

type instrumentedSvc1Logger struct {
	logger   svc1log.Logger
	registry metrics.Registry
}

func (l *instrumentedSvc1Logger) Debug(msg string, params ...svc1log.Param) {
	l.logger.Debug(msg, params...)
}

func (l *instrumentedSvc1Logger) Info(msg string, params ...svc1log.Param) {
	l.registry.Counter(infoMetricName).Inc(1)
	l.logger.Info(msg, params...)
}

func (l *instrumentedSvc1Logger) Warn(msg string, params ...svc1log.Param) {
	l.registry.Counter(warnMetricName).Inc(1)
	l.logger.Warn(msg, params...)
}

func (l *instrumentedSvc1Logger) Error(msg string, params ...svc1log.Param) {
	l.registry.Counter(errorMetricName).Inc(1)
	l.logger.Error(msg, params...)
}

func (l *instrumentedSvc1Logger) SetLevel(level wlog.LogLevel) {
	l.logger.SetLevel(level)
}

func newInstrumentedDiag1Logger(logger diag1log.Logger, registry metrics.Registry) diag1log.Logger {
	return &instrumentedDiag1Logger{
		logger:   logger,
		registry: registry,
	}
}

type instrumentedDiag1Logger struct {
	logger   diag1log.Logger
	registry metrics.Registry
}

func (l *instrumentedDiag1Logger) Diagnostic(diagnostic logging.Diagnostic, params ...diag1log.Param) {
	l.registry.Counter(diagnosticsMetricName).Inc(1)
	l.logger.Diagnostic(diagnostic, params...)
}

func newInstrumentedEvt2Logger(logger evt2log.Logger, registry metrics.Registry) evt2log.Logger {
	return &instrumentedEvt2Logger{
		logger:   logger,
		registry: registry,
	}
}

type instrumentedEvt2Logger struct {
	logger   evt2log.Logger
	registry metrics.Registry
}

func (l *instrumentedEvt2Logger) Event(name string, params ...evt2log.Param) {
	l.registry.Counter(eventMetricName).Inc(1)
	l.logger.Event(name, params...)
}
