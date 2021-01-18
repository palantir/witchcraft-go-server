package negroni

import (
	"bufio"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type closeNotifyingRecorder struct {
	*httptest.ResponseRecorder
	closed chan bool
}

func newCloseNotifyingRecorder() *closeNotifyingRecorder {
	return &closeNotifyingRecorder{
		httptest.NewRecorder(),
		make(chan bool, 1),
	}
}

func (c *closeNotifyingRecorder) close() {
	c.closed <- true
}

func (c *closeNotifyingRecorder) CloseNotify() <-chan bool {
	return c.closed
}

type hijackableResponse struct {
	Hijacked bool
}

func newHijackableResponse() *hijackableResponse {
	return &hijackableResponse{}
}

func (h *hijackableResponse) Header() http.Header           { return nil }
func (h *hijackableResponse) Write(buf []byte) (int, error) { return 0, nil }
func (h *hijackableResponse) WriteHeader(code int)          {}
func (h *hijackableResponse) Flush()                        {}
func (h *hijackableResponse) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h.Hijacked = true
	return nil, nil, nil
}

func TestResponseWriterBeforeWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	assert.Equal(t, rw.Status(), 200) // Defaults to 200 when not set
	assert.Equal(t, rw.Written(), false)
}

func TestResponseWriterBeforeFuncHasAccessToStatus(t *testing.T) {
	var status int

	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.Before(func(w ResponseWriter) {
		status = w.Status()
	})
	rw.WriteHeader(http.StatusCreated)

	assert.Equal(t, status, http.StatusCreated)
}

func TestResponseWriterWritingString(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	_, _ = rw.Write([]byte("Hello world"))

	assert.Equal(t, rec.Code, rw.Status())
	assert.Equal(t, rec.Body.String(), "Hello world")
	assert.Equal(t, rw.Status(), http.StatusOK)
	assert.Equal(t, rw.Size(), 11)
	assert.Equal(t, rw.Written(), true)
}

func TestResponseWriterWritingStrings(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	_, _ = rw.Write([]byte("Hello world"))
	_, _ = rw.Write([]byte("foo bar bat baz"))

	assert.Equal(t, rec.Code, rw.Status())
	assert.Equal(t, rec.Body.String(), "Hello worldfoo bar bat baz")
	assert.Equal(t, rw.Status(), http.StatusOK)
	assert.Equal(t, rw.Size(), 26)
}

func TestResponseWriterWritingHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, rec.Code, rw.Status())
	assert.Equal(t, rec.Body.String(), "")
	assert.Equal(t, rw.Status(), http.StatusNotFound)
	assert.Equal(t, rw.Size(), 0)
}

func TestResponseWriterBefore(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)
	result := ""

	rw.Before(func(ResponseWriter) {
		result += "foo"
	})
	rw.Before(func(ResponseWriter) {
		result += "bar"
	})

	rw.WriteHeader(http.StatusNotFound)

	assert.Equal(t, rec.Code, rw.Status())
	assert.Equal(t, rec.Body.String(), "")
	assert.Equal(t, rw.Status(), http.StatusNotFound)
	assert.Equal(t, rw.Size(), 0)
	assert.Equal(t, result, "barfoo")
}

func TestResponseWriterHijack(t *testing.T) {
	hijackable := newHijackableResponse()
	rw := NewResponseWriter(hijackable)
	hijacker, ok := rw.(http.Hijacker)
	assert.Equal(t, ok, true)
	_, _, err := hijacker.Hijack()
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, hijackable.Hijacked, true)
}

func TestResponseWriteHijackNotOK(t *testing.T) {
	hijackable := new(http.ResponseWriter)
	rw := NewResponseWriter(*hijackable)
	hijacker, ok := rw.(http.Hijacker)
	assert.Equal(t, ok, true)
	_, _, err := hijacker.Hijack()

	assert.Error(t, err)
}

func TestResponseWriterCloseNotify(t *testing.T) {
	rec := newCloseNotifyingRecorder()
	rw := NewResponseWriter(rec)
	closed := false
	notifier := rw.(http.CloseNotifier).CloseNotify()
	rec.close()
	select {
	case <-notifier:
		closed = true
	case <-time.After(time.Second):
	}
	assert.Equal(t, closed, true)
}

func TestResponseWriterNonCloseNotify(t *testing.T) {
	rw := NewResponseWriter(httptest.NewRecorder())
	_, ok := rw.(http.CloseNotifier)
	assert.Equal(t, ok, false)
}

func TestResponseWriterFlusher(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	_, ok := rw.(http.Flusher)
	assert.Equal(t, ok, true)
}

func TestResponseWriter_Flush_marksWritten(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter(rec)

	rw.Flush()
	assert.Equal(t, rw.Status(), http.StatusOK)
	assert.Equal(t, rw.Written(), true)
}
