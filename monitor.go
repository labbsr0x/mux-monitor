package mux_monitor

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Monitor struct {
	reqDuration           *prometheus.HistogramVec
	dependencyReqDuration *prometheus.HistogramVec
	respSize              *prometheus.CounterVec
	dependencyUP          *prometheus.GaugeVec
	applicationInfo       *prometheus.GaugeVec
	errorMessageKey       string
	IsStatusError         func(statusCode int) bool
}

// DependencyStatus is the type to represent UP or DOWN states
type DependencyStatus int

// DependencyChecker specifies the methods a checker must implement.
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

	monitor := &Monitor{errorMessageKey: errorMessageKey, IsStatusError: IsStatusError}

	monitor.reqDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "request_seconds",
		Help:    "Duration in seconds of HTTP requests.",
		Buckets: buckets,
	}, []string{"type", "status", "method", "addr", "isError", "errorMessage"})

	monitor.respSize = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "response_size_bytes",
		Help: "Counts the size of each HTTP response",
	}, []string{"type", "status", "method", "addr", "isError", "errorMessage"})

	monitor.dependencyUP = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "dependency_up",
		Help: "Records if a dependency is up or down. 1 for up, 0 for down",
	}, []string{"name"})

	monitor.dependencyReqDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "dependency_request_seconds",
		Help:    "Duration of dependency requests in seconds.",
		Buckets: buckets,
	}, []string{"name", "type", "status", "method", "addr", "isError", "errorMessage"})

	monitor.applicationInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "application_info",
		Help: "Static information about the application",
	}, []string{"version"})
	monitor.applicationInfo.WithLabelValues(applicationVersion).Set(1)

	return monitor, nil
}

func (m *Monitor) collectTime(reqType, status, method, addr, isError, errorMessage string, durationSeconds float64) {
	m.reqDuration.WithLabelValues(reqType, status, method, addr, isError, errorMessage).Observe(durationSeconds)
}

func (m *Monitor) collectSize(reqType, status, method, addr, isError, errorMessage string, size float64) {
	m.respSize.WithLabelValues(reqType, status, method, addr, isError, errorMessage).Add(size)
}

// CollectDependencyTime collet the duration of dependency requests in seconds
func (m *Monitor) CollectDependencyTime(name, reqType, status, method, addr, isError, errorMessage string, durationSeconds float64) {
	m.dependencyReqDuration.WithLabelValues(name, reqType, status, method, addr, isError, errorMessage).Observe(durationSeconds)
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
		isErrorStr := strconv.FormatBool(m.IsStatusError(respWriter.statusCode))

		errorMessage := r.Header.Get(m.errorMessageKey)
		r.Header.Del(m.errorMessageKey)

		m.collectTime(r.Proto, statusCodeStr, r.Method, path, isErrorStr, errorMessage, duration.Seconds())
		m.collectSize(r.Proto, statusCodeStr, r.Method, path, isErrorStr, errorMessage, float64(respWriter.Count()))
	})
}

// AddDependencyChecker creates a ticker that periodically executes the checker and collects the dependency state metrics
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

func IsStatusError(statusCode int) bool {
	return statusCode < 200 || statusCode >= 400
}
