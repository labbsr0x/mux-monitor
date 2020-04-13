package mux_monitor

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Monitor struct {
	reqDuration     *prometheus.HistogramVec
	respSize        *prometheus.CounterVec
	dependencyUP    *prometheus.GaugeVec
	applicationInfo *prometheus.GaugeVec
}

var (
	defaultBuckets = []float64{0.1, 0.3, 1.5, 10.5}
)

func New(applicationVersion string) (*Monitor, error) {
	if strings.TrimSpace(applicationVersion) == "" {
		return nil, errors.New("application version must be a non-empty string")
	}

	monitor := &Monitor{}

	monitor.reqDuration    = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_seconds",
		Help:    "Duration in seconds of HTTP requests.",
		Buckets: defaultBuckets,
	}, []string{"type", "status", "method", "addr", "isError"})

	monitor.respSize = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "response_size_bytes",
		Help: "counts the size of each HTTP response",
	}, []string{"type", "status", "method", "addr", "isError"})

	monitor.dependencyUP = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dependency_up",
		Help: "records if a dependency is up or down. 1 for up, 0 for down",
	}, []string{"name"})

	monitor.applicationInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "application_info",
		Help: "static information about the application",
	}, []string{"name"})

	return monitor, nil
}

func (m *Monitor) collectTime(reqType, status, method, addr string, isError string, durationSeconds float64) {
	m.reqDuration.WithLabelValues(reqType, status, method, addr, isError).Observe(durationSeconds)
}

func (m *Monitor) collectSize(reqType, status, method, addr string, isError string, size float64) {
	m.respSize.WithLabelValues(reqType, status, method, addr, isError).Add(size)
}

// Prometheus implements mux.MiddlewareFunc.
func (m *Monitor) Prometheus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		respWriter := NewResponseWriter(w)

		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()

		next.ServeHTTP(respWriter, r)

		duration := time.Since(respWriter.started)

		statusCodeStr := respWriter.StatusCodeStr()
		isErrorStr := respWriter.IsErrorStr()

		m.collectTime(r.Proto, statusCodeStr, r.Method, path, isErrorStr, duration.Seconds())
		m.collectSize(r.Proto, statusCodeStr, r.Method, path, isErrorStr, float64(respWriter.Count()))
	})
}
