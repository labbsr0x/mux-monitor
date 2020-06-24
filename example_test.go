package mux_monitor_test

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	muxMonitor "github.com/labbsr0x/mux-monitor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type FakeDependencyChecker struct{}

func (m *FakeDependencyChecker) GetDependencyName() string {
	return "fake-dependency"
}

func (m *FakeDependencyChecker) Check() muxMonitor.DependencyStatus {
	return muxMonitor.DOWN
}

func YourHandler(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte("mux-monitor!\n"))
}

func main() {
	// Creates mux-monitor instance
	monitor, err := muxMonitor.New("v1.0.0", muxMonitor.DefaultErrorMessageKey, muxMonitor.DefaultBuckets)
	if err != nil {
		panic(err)
	}

	dependencyChecker := &FakeDependencyChecker{}
	monitor.AddDependencyChecker(dependencyChecker, time.Second*30)

	r := mux.NewRouter()

	// Register mux-monitor middleware
	r.Use(monitor.Prometheus)
	// Register metrics endpoint
	r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	// Routes consist of a path and a handler function.
	r.HandleFunc("/", YourHandler)

	// Bind to a port and pass our router in
	log.Fatal(http.ListenAndServe(":8000", r))
}
