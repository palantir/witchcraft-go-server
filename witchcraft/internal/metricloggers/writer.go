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
