# mux-monitor

A Prometheus middleware to add basic but very useful metrics for your gorilla/mux app.

## Metrics

The exposed metrics are the following:

```
request_seconds_bucket{type, status, method, addr, isError, errorMessage, le}
request_seconds_count{type, status, method, addr, isError, errorMessage}
request_seconds_sum{type, status, method, addr, isError, errorMessage}
response_size_bytes{type, status, method, addr, isError, errorMessage}
dependency_up{name}
dependency_request_seconds_bucket{name, type, status, method, addr, isError, errorMessage, le}
dependency_request_seconds_count{name, type, status, method, addr, isError, errorMessage}
dependency_request_seconds_sum{name, type, status, method, addr, isError, errorMessage}
application_info{version}
```

Details:

1. The `request_seconds_bucket` metric defines the histogram of how many requests are falling into the well-defined buckets represented by the label `le`;

2. The `request_seconds_count` metric counts the overall number of requests with those exact label occurrences;

3. The `request_seconds_sum` metric counts the overall sum of how long the requests with those exact label occurrences are taking;

4. The `response_size_bytes` metric computes how much data is being sent back to the user for a given request type;

5. The `dependency_up` metric register whether a specific dependency is up (1) or down (0). The label `name` registers the dependency name;

6. The `dependency_request_seconds_bucket` metric defines the histogram of how many requests to a specific dependency are falling into the well-defined buckets represented by the label le;

7. The `dependency_request_seconds_count` metric counts the overall number of requests to a specific dependency;

8. The `dependency_request_seconds_sum` metric counts the overall sum of how long requests to a specific dependency are taking;

9. The `application_info` holds static info of an application, such as its semantic version number;

Labels:

1. `type` registers request protocol used (e.g. `grpc` or `http`);

2. `status` registers the response status (e.g. HTTP status code);

3. `method` registers the request method;

4. `addr` registers the requested endpoint address;

5. `version` registers which version of your app handled the request;

6. `isError` registers whether status code reported is an error or not;

7. `errorMessage` registers the error message;

8. `name` registers the name of the dependency;

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

### Register Error Message

It's possible to register the error message to your metrics, you must set a header to your `http.Request` with key defined on `muxMonitor.New`.

The following code creates a monitor instance with the error message key `muxMonitor.DefaultErrorMessageKey` passed by the second parameter:

```go
// Creates mux-monitor instance
monitor, err := muxMonitor.New("v1.0.0", muxMonitor.DefaultErrorMessageKey, muxMonitor.DefaultBuckets)
```

At your handler, your must set a header with the same key `muxMonitor.DefaultErrorMessageKey`:
```go
func ErrorHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set(muxMonitor.DefaultErrorMessageKey, "this is an error message - internal server error")
	w.WriteHeader(http.StatusInternalServerError)
}
``` 

> :warning: **NOTE**: 
> The cardinality of this label affect Prometheus performance 

### Dependency Metrics

#### Register Dependency State Checkers

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

### Collect Dependency Request Duration

You can also monitor request latency for dependencies calling `monitor.CollectDependencyTime` method.

e.g.
```go
monitor.CollectDependencyTime("http-dependency", "http", "200", "GET", "localhost:8001", "false", "", 10)
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