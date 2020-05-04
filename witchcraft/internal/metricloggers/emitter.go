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
	"github.com/palantir/pkg/metrics"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

const (
	slsLoggingMeterName = "logging.sls"
)

type MetricEmitter interface {
	MarkSLSLog(typ string, level string)
}

type metricEmitter struct {
	registry metrics.RootRegistry
}

func NewMetricEmitter(registry metrics.RootRegistry) MetricEmitter {
	return &metricEmitter{registry: registry}
}

func (m *metricEmitter) MarkSLSLog(typ string, level string) {
	m.registry.Meter(slsLoggingMeterName,
		metrics.MustNewTag("type", svc1log.TypeValue),
		metrics.MustNewTag("level", level)).Mark(1)
}
