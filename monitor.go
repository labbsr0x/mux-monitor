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
	errorMessageKey string
}

type DependencyStatus int
type DependencyChecker interface {
	GetDependencyName() string
	Check() DependencyStatus
}

const (
	DOWN DependencyStatus = iota
	UP
)

const DefaultErrorMessageKey = "error-message"

var (
	DefaultBuckets = []float64{0.1, 0.3, 1.5, 10.5}
)

//New create new Monitor instance
func New(applicationVersion string, errorMessageKey string, buckets []float64) (*Monitor, error) {
	if strings.TrimSpace(applicationVersion) == "" {
		return nil, errors.New("application version must be a non-empty string")
	}

	if strings.TrimSpace(applicationVersion) == "" {
		errorMessageKey = DefaultErrorMessageKey
	}

	if buckets == nil {
		buckets = DefaultBuckets
	}

	monitor := &Monitor{errorMessageKey: errorMessageKey}

	monitor.reqDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_seconds",
		Help:    "duration in seconds of HTTP requests.",
		Buckets: buckets,
	}, []string{"type", "status", "method", "addr", "isError", "errorMessage"})

	monitor.respSize = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "response_size_bytes",
		Help: "counts the size of each HTTP response",
	}, []string{"type", "status", "method", "addr", "isError", "errorMessage"})

	monitor.dependencyUP = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dependency_up",
		Help: "records if a dependency is up or down. 1 for up, 0 for down",
	}, []string{"name"})

	monitor.applicationInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "application_info",
		Help: "static information about the application",
	}, []string{"version"})
	monitor.applicationInfo.WithLabelValues(applicationVersion).Set(1)

	return monitor, nil
}

func (m *Monitor) collectTime(reqType, status, method, addr string, isError string, errorMessage string, durationSeconds float64) {
	m.reqDuration.WithLabelValues(reqType, status, method, addr, isError, errorMessage).Observe(durationSeconds)
}

func (m *Monitor) collectSize(reqType, status, method, addr string, isError string, errorMessage string, size float64) {
	m.respSize.WithLabelValues(reqType, status, method, addr, isError, errorMessage).Add(size)
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
		errorMessage := w.Header().Get(m.errorMessageKey)

		m.collectTime(r.Proto, statusCodeStr, r.Method, path, isErrorStr, errorMessage, duration.Seconds())
		m.collectSize(r.Proto, statusCodeStr, r.Method, path, isErrorStr, errorMessage, float64(respWriter.Count()))
	})
}

func (m *Monitor) AddDependencyChecker(checker DependencyChecker, checkingPeriod time.Duration) {
	ticker := time.NewTicker(checkingPeriod)
	go func() {
		for {
			select {
			case <-ticker.C:
				status := checker.Check()
				m.dependencyUP.WithLabelValues(checker.GetDependencyName()).Set(float64(status))
			}
		}
	}()
}
