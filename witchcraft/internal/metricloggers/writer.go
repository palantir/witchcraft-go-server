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

// MetricWriter can be used for SLS logging. It emits metrics about the length of the lines writen.
type MetricWriter interface {
	io.Writer
	// SetMetricRegistry initializes the metric registry used for emitting metrics.
	// Before this is called, no metrics are emitted.
	SetMetricRegistry(metrics.Registry)
}

type metricWriter struct {
	writer   io.Writer
	typ      string
	recorder metricRecorder
}

// NewMetricWriter returns a MetricWriter that delegates to writer, and emits SLS log line length
// according to the provided slsFilename, e.g. "service", or "trace".
func NewMetricWriter(writer io.Writer, slsFilename string) MetricWriter {
	return &metricWriter{
		writer:   writer,
		typ:      slsFilename,
		recorder: nil,
	}
}

func (m *metricWriter) SetMetricRegistry(registry metrics.Registry) {
	m.recorder = newMetricRecorder(registry, m.typ)
}

func (m *metricWriter) Write(p []byte) (int, error) {
	n, err := m.writer.Write(p)
	if m.recorder != nil {
		m.recorder.RecordSLSLogLength(n)
	}
	return n, err
}
