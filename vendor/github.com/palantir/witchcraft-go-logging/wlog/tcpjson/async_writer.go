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
	"context"
	"io"
	"log"

	gometrics "github.com/palantir/go-metrics"
	"github.com/palantir/pkg/metrics"
	werror "github.com/palantir/witchcraft-go-error"
)

const (
	// asyncWriterBufferCap is an arbitrarily high limit for the number of messages allowed to queue before writes will block.
	asyncWriterBufferCapacity = 15000
	asyncWriterBufferLenGauge = "logging.queue"
	asyncWriterDroppedCounter = "logging.queue.dropped"
)

type asyncWriter struct {
	buffer  chan []byte
	output  io.Writer
	dropped gometrics.Counter
	queued  gometrics.Gauge
	stop    chan struct{}
}

type AsyncWriter interface {
	io.WriteCloser
	// Drain tries to gracefully drain the remaining buffered messages,
	// blocking until the buffer is empty or the provided context is cancelled.
	Drain(ctx context.Context)
}

// StartAsyncWriter creates a Writer whose Write method puts the submitted byte slice onto a channel.
// In a separate goroutine, slices are pulled from the queue and written to the output writer.
// The Close method stops the consumer goroutine and will cause future writes to fail.
func StartAsyncWriter(output io.Writer, registry metrics.Registry) AsyncWriter {
	droppedCounter := registry.Counter(asyncWriterDroppedCounter)
	buffer := make(chan []byte, asyncWriterBufferCapacity)
	stop := make(chan struct{})
	queued := registry.Gauge(asyncWriterBufferLenGauge)
	w := &asyncWriter{buffer: buffer, output: output, dropped: droppedCounter, queued: queued, stop: stop}
	go func() {
		for {
			// Ensure we stop when requested. Without the additional select,
			// the loop could continue to run as long as there are items in the buffer.
			select {
			case <-stop:
				return
			default:
			}

			select {
			case item := <-buffer:
				w.write(item)
			case <-stop:
				return
			}
		}
	}()
	return w
}

func (w *asyncWriter) write(item []byte) {
	w.queued.Update(int64(len(w.buffer)))
	if _, err := w.output.Write(item); err != nil {
		// TODO(bmoylan): consider re-enqueuing message so it can be attempted again, which risks a thundering herd without careful handling.
		log.Printf("write failed: %s", werror.GenerateErrorString(err, false))
		w.dropped.Inc(1)
	}
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
		select {
		case w.buffer <- bb:
		default:
			// drop logs -- nothing we can do
			w.dropped.Inc(1)
		}
		return len(b), nil
	}
}

// Close stops the consumer goroutine and will cause future writes to fail.
func (w *asyncWriter) Close() (err error) {
	close(w.stop)
	return nil
}

func (w *asyncWriter) Drain(ctx context.Context) {
	for {
		select {
		case item := <-w.buffer:
			w.write(item)
		case <-ctx.Done():
			return
		default:
			// Nothing left in the buffer, time to return
			return
		}
	}
}
