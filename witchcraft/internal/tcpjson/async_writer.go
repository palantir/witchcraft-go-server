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
	asyncWriterBufferCap      = 1000
	asyncWriterBufferLenGauge = "sls.logging.queued"
)

type asyncWriter struct {
	buffer chan []byte
	output io.Writer
	stop   chan struct{}
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
