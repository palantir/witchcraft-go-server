// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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

package tcpjson

import (
	"io"
	"log"

	"github.com/palantir/pkg/metrics"
	werror "github.com/palantir/witchcraft-go-error"
)

const (
	// asyncWriterBufferCap is an arbitrarily high limit for the number of messages allowed to queue before writes will block.
	asyncWriterBufferCapacity = 1000
	asyncWriterBufferLenGauge = "logging.queue"
)

type asyncWriter struct {
	buffer chan []byte
	output io.Writer
	stop   chan struct{}
}

// StartAsyncWriter creates a Writer whose Write method puts the submitted byte slice onto a channel.
// In a separate goroutine, slices are pulled from the queue and written to the output writer.
// The Close method stops the consumer goroutine and will cause future writes to fail.
func StartAsyncWriter(output io.Writer, registry metrics.Registry) io.WriteCloser {
	buffer := make(chan []byte, asyncWriterBufferCapacity)
	stop := make(chan struct{})
	go func() {
		gauge := registry.Gauge(asyncWriterBufferLenGauge)
		for {
			select {
			case item := <-buffer:
				if _, err := output.Write(item); err != nil {
					// TODO(bmoylan): consider re-enqueuing message so it can be attempted again, which risks a thundering herd without careful handling.
					log.Printf("write failed: %s", werror.GenerateErrorString(err, false))
				}
				gauge.Update(int64(len(buffer)))
			case <-stop:
				return
			}
		}
	}()
	return &asyncWriter{buffer: buffer, output: output, stop: stop}
}

func (w *asyncWriter) Write(b []byte) (int, error) {
	select {
	case <-w.stop:
		return 0, werror.Error("write to closed asyncWriter")
	default:
		// copy the provided byte slice before pushing it into the buffer channel so the original
		// byte slice is not retained and thus still compliant with the io.Writer contract
		bb := make([]byte, len(b))
		copy(bb, b)
		w.buffer <- bb
		return len(b), nil
	}
}

// Close stops the consumer goroutine and will cause future writes to fail.
func (w *asyncWriter) Close() (err error) {
	close(w.stop)
	return nil
}
