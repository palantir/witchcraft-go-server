// Copyright (c) 2020 Palantir Technologies. All rights reserved.
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

package metricloggers

import (
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

var _ svc1log.Logger = (*metricEmittingSvcLogger)(nil)

type metricEmittingSvcLogger struct {
	logger  svc1log.Logger
	emitter MetricEmitter
}

func NewSvcLogger(logger svc1log.Logger, emitter MetricEmitter) svc1log.Logger {
	return &metricEmittingSvcLogger{
		logger:  logger,
		emitter: emitter,
	}
}

func (m *metricEmittingSvcLogger) Debug(msg string, params ...svc1log.Param) {
	m.emitter.MarkSLSLog(svc1log.TypeValue, svc1log.LevelDebugValue)
	m.logger.Debug(msg, params...)
}

func (m *metricEmittingSvcLogger) Info(msg string, params ...svc1log.Param) {
	m.emitter.MarkSLSLog(svc1log.TypeValue, svc1log.LevelInfoValue)
	m.logger.Info(msg, params...)
}

func (m *metricEmittingSvcLogger) Warn(msg string, params ...svc1log.Param) {
	m.emitter.MarkSLSLog(svc1log.TypeValue, svc1log.LevelWarnValue)
	m.logger.Warn(msg, params...)
}

func (m *metricEmittingSvcLogger) Error(msg string, params ...svc1log.Param) {
	m.emitter.MarkSLSLog(svc1log.TypeValue, svc1log.LevelErrorValue)
	m.logger.Error(msg, params...)
}

func (m *metricEmittingSvcLogger) SetLevel(level wlog.LogLevel) {
	m.logger.SetLevel(level)
}
