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
)

const (
	slsLoggingMeterName = "logging.sls"
)

type MetricRecorder interface {
	RecordSLSLog(typ string, level string)
}

type metricRecorder struct {
	registry metrics.Registry
}

func NewMetricRecorder(registry metrics.Registry) MetricRecorder {
	return &metricRecorder{registry: registry}
}

// RecordSLSLog increments the count of the SLS logging metric
// for the given log type and log level.
// If level is empty, it will be omitted from the recorded metric.
func (m *metricRecorder) RecordSLSLog(typ string, level string) {
	tags := metrics.Tags{
		metrics.MustNewTag("type", typ),
	}
	if len(level) > 0 {
		tags = append(tags, metrics.MustNewTag("level", level))
	}
	m.registry.Meter(slsLoggingMeterName, tags...).Mark(1)
}
