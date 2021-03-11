package tcpjson

import (
	"bytes"
	"strconv"
	"testing"
	"time"

	"github.com/palantir/pkg/metrics"
	"github.com/stretchr/testify/assert"
)

func TestAsyncWriter(t *testing.T) {
	out := &bytes.Buffer{}
	w := StartAsyncWriter(out, metrics.DefaultMetricsRegistry)
	for i := 0; i < 5; i ++ {
		str := strconv.Itoa(i)
		go func() {
			_, _ = w.Write([]byte(str))
		}()
	}
	time.Sleep(time.Millisecond)

	written := out.String()
	t.Log(written)
	assert.Len(t, written, 5)
	for i := 0; i < 5; i ++ {
		assert.Contains(t, written, strconv.Itoa(i))
	}
}
