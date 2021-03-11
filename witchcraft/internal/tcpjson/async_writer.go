package tcpjson

import (
	"io"
	"log"

	"github.com/palantir/pkg/metrics"
	werror "github.com/palantir/witchcraft-go-error"
)

const (
	asyncWriterBufferCap = 1000
	asyncWriterBufferLenGauge = "sls.logging.queued"
)

type asyncWriter struct {
	buffer chan []byte
	output io.Writer
	stop chan struct{}
}

func StartAsyncWriter(output io.Writer, registry metrics.Registry) io.WriteCloser {
	buffer := make(chan []byte, asyncWriterBufferCap)
	stop := make(chan struct{})
	go func() {
		gauge := registry.Gauge(asyncWriterBufferLenGauge)
		for {
			select {
			case item := <-buffer:
				if _, err := output.Write(item); err != nil {
					log.Printf("write failed: %s", werror.GenerateErrorString(err, false))
				}
				gauge.Update(int64(len(buffer)))
			case <-stop:
				return
			}
		}
	}()
	return &asyncWriter{buffer: buffer, output: output}
}

func (w *asyncWriter) Write(b []byte) (int, error) {
	w.buffer <- b
	return len(b), nil
}

func (w *asyncWriter) Close() (err error) {
	close(w.stop)
	return nil
}
