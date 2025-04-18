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

package metricloggers

import (
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log"
)

var _ audit3log.Logger = (*audit3Logger)(nil)

type audit3Logger struct {
	logger   audit3log.Logger
	recorder metricRecorder
}

func NewAudit3Logger(logger audit3log.Logger, registry metrics.Registry) audit3log.Logger {
	return &audit3Logger{
		logger:   logger,
		recorder: newMetricRecorder(registry, audit3log.TypeValue),
	}
}

func (m *audit3Logger) Audit(name string, result audit3log.AuditResultType, params ...audit3log.Param) {
	m.logger.Audit(name, result, params...)
	m.recorder.RecordSLSLog()
}
