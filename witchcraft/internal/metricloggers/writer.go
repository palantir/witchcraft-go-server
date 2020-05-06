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
	"io"

	"github.com/palantir/pkg/metrics"
)

var _ io.Writer = (*metricWriter)(nil)

type MetricWriter interface {
	io.Writer
	SetMetricRegistry(metrics.Registry)
}

type metricWriter struct {
	writer   io.Writer
	typ      string
	recorder metricRecorder
}

func NewMetricWriter(writer io.Writer, typ string) MetricWriter {
	return &metricWriter{
		writer:   writer,
		typ:      typ,
		recorder: nil,
	}
}

func (m *metricWriter) SetMetricRegistry(registry metrics.Registry) {
	m.recorder = newMetricRecorder(registry, m.typ)
}

func (m *metricWriter) Write(p []byte) (int, error) {
	n, err := m.writer.Write(p)
	if m.recorder != nil {
		// Commented out because I still need to fix the integration tests if we chose this approach
		//m.recorder.RecordSLSLogLength(n)
	}
	return n, err
}
