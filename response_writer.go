package mux_monitor

import (
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

// workaround to get status code on middleware
type ResponseWriter struct {
	http.ResponseWriter
	started    time.Time
	statusCode int
	count      uint64
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	// WriteHeader(int) is not called if our response implicitly returns 200 OK, so
	// we default to that status code.
	return &ResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		started:        time.Now(),
	}
}

func (r *ResponseWriter) StatusCode() int {
	return r.statusCode
}

func (r *ResponseWriter) StatusCodeStr() string {
	return strconv.Itoa(r.statusCode)
}

// Write returns underlying Write result, while counting data size
func (r *ResponseWriter) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	atomic.AddUint64(&r.count, uint64(n))
	return n, err
}

func (r *ResponseWriter) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

// Count function return counted bytes
func (r *ResponseWriter) Count() uint64 {
	return atomic.LoadUint64(&r.count)
}
