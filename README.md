# mux-monitor

A Prometheus middleware to add basic but very useful metrics for your gorilla/mux app.

## Metrics

The exposed metrics are the following:

```
request_seconds_bucket{type,status, method, addr, isError, le}
request_seconds_count{type, status, method, addr, isError}
request_seconds_sum{type, status, method, addr, isError}
response_size_bytes{type, status, method, addr, isError}
dependency_up{name}
application_info{version}
```

Where, for a specific request, `type` tells which request protocol was used (e.g. `grpc` or `http`), `status` registers the response HTTP status, `method` registers the request method, `addr` registers the requested endpoint address, `version` tells which version of your app handled the request and `isError` lets us know if the status code reported is an error or not.

In detail:

1. `request_seconds_bucket` is a metric defines the histogram of how many requests are falling into the well defined buckets represented by the label `le`;

2. `request_seconds_count` is a counter that counts the overall number of requests with those exact label occurrences;

3. `request_seconds_sum` is a counter that counts the overall sum of how long the requests with those exact label occurrences are taking;

4. `response_size_bytes` is a counter that computes how much data is being sent back to the user for a given request type. It captures the response size from the `content-length` response header. If there is no such header, the value exposed as metric will be zero;

5. `dependency_up` is a metric to register weather a specific dependency is up (1) or down (0). The label `name` registers the dependency name;

6. Finally, `application_info` holds static info of an application, such as it's semantic version number;

Labels:

1. `type` tells which request protocol was used (e.g. `grpc` or `http`);

2. `status` registers the response status (e.g. HTTP status code);

3. `method` registers the request method;

4. `addr` registers the requested endpoint address;

5. `version` tells which version of your app handled the request;

6. `isError` tells status code reported is an error or not;

7. `errorMessage` registers the error message;

## How to

### Install

With a [correctly configured](https://golang.org/doc/install#testing) Go toolchain:

```sh
go get -u github.com/labbsr0x/mux-monitor
```

### Register Metrics Middleware 
You must register the metrics middleware to enable metric collection. 

Metrics Middleware can be added to a router using `Router.Use()`:

```go
// Creates mux-monitor instance
monitor, err := muxMonitor.New("v1.0.0", muxMonitor.DefaultErrorMessageKey, muxMonitor.DefaultBuckets)
if err != nil {
    panic(err)
}

r := mux.NewRouter()
// Register mux-monitor middleware
r.Use(monitor.Prometheus)
```

> :warning: **NOTE**: 
> This middleware must be the first in the middleware chain file so that you can get the most accurate measurement of latency and response size.

### Expose Metrics Endpoint

You must register a specific router to expose the application metrics:

```go
// Register metrics endpoint
r.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
```

### Register Dependency State Checkers

To add a dependency state metrics to the Monitor, you must create a checker implementing the interface `DependencyChecker` and add an instance to the Monitor with the period interval that the dependency must be checked.

Implementing the `DependencyChecker` interface:
```go
type FakeDependencyChecker struct {}

func (m *FakeDependencyChecker) GetDependencyName() string {
	return "fake-dependency"
}

func (m *FakeDependencyChecker) Check() muxMonitor.DependencyStatus {
    // Do your things and return muxMonitor.UP or muxMonitor.DOWN
	return muxMonitor.DOWN
}
```

Adding the dependency checker to the monitor:
```go
func main() {
	// Creates mux-monitor instance
	monitor, err := muxMonitor.New("v1.0.0", muxMonitor.DefaultErrorMessageKey, muxMonitor.DefaultBuckets)
	if err != nil {
		panic(err)
	}

	dependencyChecker := &FakeDependencyChecker{}
	monitor.AddDependencyChecker(dependencyChecker, time.Second * 30)
}
```

## Example

Here's a runnable example of a small `mux` based server configured with `mux-monitor`:

```go
import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	muxMonitor "github.com/labbsr0x/mux-monitor"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type FakeDependencyChecker struct {}

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
	monitor.AddDependencyChecker(dependencyChecker, time.Second * 30)

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
```

## Big Brother

This project is part of a more large application called [Big Brother](https://github.com/labbsr0x/big-brother).