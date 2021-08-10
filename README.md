<p align="right">
<a href="https://autorelease.general.dmz.palantir.tech/palantir/witchcraft-go-server"><img src="https://img.shields.io/badge/Perform%20an-Autorelease-success.svg" alt="Autorelease"></a>
</p>

witchcraft-go-server
====================
[![](https://godoc.org/github.com/palantir/witchcraft-go-server?status.svg)](http://godoc.org/github.com/palantir/witchcraft-go-server)

`witchcraft-go-server` is a Go implementation of a Witchcraft server. It provides a way to quickly and easily create
servers that work in the Witchcraft ecosystem.

Implementation
--------------

### Configuration
A witchcraft server is provided with install configuration and runtime configuration. Install configuration specifies
configuration values that are static -- they are read in once at server startup and are known to never change. Runtime
configuration values are considered *refreshable*. When file-based configuration is used, whenever the runtime 
configuration file is updated its contents are loaded and the corresponding values are refreshed. See the section on 
refreshable configuration for more information on this.

The default configuration uses file-based configuration with the install configuration at `var/conf/install.yml` and
runtime configuration at `var/conf/runtime.yml`. It is possible to use code to specify different sources of 
configuration (for example, in-memory providers).

`witchcraft-server` also supports using `encrypted-config-value` to automatically decrypt encrypted configuration
values. The default configuration expects a key file to be at `var/conf/encrypted-config-value.key`. It is possible to 
use code to specify a different source for the key (or to specify that no key should be used). If the configuration
does not contain encrypted values, any specified ECV key will not be read. If the install configuration contains
encrypted values but the encryption key is missing or malformed, the server will fail to start. If the runtime config
contains encrypted values but fails to decrypt them, a warning will be logged and the encrypted values passed to the
server.

`witchcraft-server` defines base configuration for its install and runtime configuration. Servers that want to provide
their own install and/or runtime configuration should embed the base configuration structs within the definition of 
their configuration structs. 

### Route registration
A witchcraft server is backed by a `wrouter.Router` and allows authors to register route handlers on the server. The 
router uses a specific format for path templates to specify path parameters and has rules around the kinds of paths that
can be matched. witchcraft is opinionated about the path formats and does not support registering paths that cannot be
expressed using its template rules. All witchcraft routes are configured to emit request logs and trace logs and update
metrics for the requests using built-in middleware. The `context.Context` for the `http.Request` provided to the 
handlers  is configured with all of the standard loggers (service logger, event logger, trace logger, etc.).

When registering routes on the router, it is also possible to specify path/header/query param keys that should be
considered "safe" or "forbidden" when used as parameters in logging. These are combined with the default set of safe and
forbidden header parameters defined by the `req2log` package in `witchcraft-go-logging`.

### Liveness, readiness, and health
`witchcraft-server` registers the endpoints `/status/liveness`, `/status/readiness` and `/status/health` to report the
server's liveness, readiness and health. By default, these endpoints use a built-in provider that reports liveness,
readiness and health based on the state of the server. It is possible to configure the liveness and readiness providers
in code, and health status providers can also be added via code (health supports specifying multiple sources to report
health, and the server's built-in health status provider will always be one of them).

The default behavior serves both the user-registered endpoints and the status endpoints from the same server. However,
if a "management port" is specified in the server's install configuration and its value differs from the "port" value in
configuration, then `witchcraft-server` starts a second management server on the specified port and serves the status
endpoints on that port. This can be useful in scenarios where all of the traffic to the main endpoints require client
certificates for TLS but the status endpoints need to be served without requiring client TLS certificates.

### Debug & Diagnostic Routes

Witchcraft servers register a route on the management server at `/debug/diagnostic/{diagnosticType}`, where
diagnosticType represents a payload used for debugging a running server node. The response's `Content-Type` header
specifies the encoding and format of the response. The following types are currently supported:
* `go.goroutines.v1`: Plaintext representation of all running goroutines and their stacktraces
* `go.profile.cpu.1minute.v1`: Returns the pprof-formatted cpu profile. See [pprof.Profile](https://golang.org/pkg/net/http/pprof/#Profile).
* `go.profile.heap.v1`: Returns the pprof-formatted heap profile as of the last GC. See [pprof.Profile](https://golang.org/pkg/runtime/pprof/#Profile).
* `go.profile.allocs.v1`: Returns the pprof-formatted allocs profile for all allocations in the process lifetime. See [pprof.Profile](https://golang.org/pkg/runtime/pprof/#Profile).
* `metric.names.v1`: Records all metric names and tag sets in the process's metric registry.

#### \[Deprecated] Pprof routes
The following routes are registered on the management server (if enabled, otherwise the main server) to aid in debugging
and telemetry collection. These are generally deprecated in favor of the diagnostic routes described above.
* `/debug/pprof`: Provides an HTML index of the other endpoints at this route.
* `/debug/pprof/profile`: Returns the pprof-formatted cpu profile. See [pprof.Profile](https://golang.org/pkg/net/http/pprof/#Profile).
* `/debug/pprof/heap`: Returns the pprof-formatted heap profile as of the last GC. See [pprof.Profile](https://golang.org/pkg/runtime/pprof/#Profile).
* `/debug/pprof/cmdline`: Returns the process's command line invocation as `text/plain`. See [pprof.Cmdline](https://golang.org/pkg/net/http/pprof/#Cmdline).
* `/debug/pprof/symbol`: Looks up the program counters listed in the request, responding with a table mapping program counters to function names See [pprof.Symbol](https://golang.org/pkg/net/http/pprof/#Symbol).
* `/debug/pprof/trace`: Returns the execution trace in binary form. See [pprof.Trace](https://golang.org/pkg/net/http/pprof/#Trace).

### Context path
If `context-path` is specified in the install configuration, all of the routes registered on the server will be prefixed
with the specified `context-path`.

### Security
`witchcraft-server` only supports HTTPS. The TLS client authentication type is configurable in code. The base install 
configuration has fields to specify the location of server key and certificate material for TLS connections.

Although it is not possible to run `witchcraft-server` using HTTP, it is possible to configure the server in code to use
a generated self-signed certificate on start-up. Running the server in this mode and connecting to it using TLS without
server certificate verification (equivalent of `curl -k` or an `http.Transport` with 
`TLSClientConfig: &tls.Config{InsecureSkipVerify: true}`) provides an analog to using HTTP, with the benefit that the
traffic itself is still encrypted.

### Logging
`witchcraft-server` is configured with service, event, metric, request and trace loggers from the 
`witchcraft-go-logging` project and emits structured JSON logs using [`zerolog`](https://github.com/rs/zerolog) as the
logger implementation. The default behavior emits logs to the `var/log` directory (`var/log/service.log`, 
`var/log/request.log`, etc.) unless the server is run in a Docker container or has the environment variable `$CONTAINER` set, in which case the logs are always emitted to `stdout`. The `use-console-log` property in the install configuration can also be set to "true" to always output logs 
to `stdout`. The runtime configuration supports configuring the log output level for service logs.

The `context.Context` provided to request handlers is configured with all of the standard loggers (service logger, event
logger, trace logger, etc.). All of the handlers are also configured to emit request logs and trace logs.

### Service logger origin
By default, the `origin` field of the service logger is set to be the package path of the package in which the
`witchcraft-server` is started. For example, if the server is started in the file 
`github.com/palantir/project/server/server.go`, the origin for all service log lines will be
`github.com/palantir/project/server`.

It is possible to configure the origin to be a different value using code. The origin can be specified to be a string
constant or a function can be used that returns a specific package path based on supplied parameters (for example, the
function can specify that the caller package's parent package should be used as the origin). The origin can also be set
to empty, in which case it is omitted from the log output.

### Trace IDs and instrumentation
`witchcraft-server` supports zipkin-compatible tracing and ensures that every request is instrumented for tracing.
`witchcraft-server` also recognizes that some code will use trace IDs without necessarily using full zipkin-compatible
spans, so some allowances are made to support this scenario.

The built-in `witchcraft-server` middleware that registers loggers on the context also ensures that a zipkin span is
started. If the incoming request header has valid zipkin span information (that is, it specifies both a `X-B3-TraceId`
and `X-B3-SpanId` in the header), then the span created by the middleware is a child span of the incoming span. If the
incoming request does not have a trace ID header, a new root span is created. If the header specifies a trace ID but not
a span ID, the middleware creates a new root zipkin span, but ensures that the trace ID of the created span matches what 
is specified in the header. If an incoming request is routed to a registered endpoint, the built-in router middleware
will create another span (which is a child span of the one created by the request middleware) whose span name is the
HTTP method and template for the endpoint.

The trace information generated by the middleware is set on the header and will be visible to subsequent handlers. If
the request specifies a `X-B3-Sampled` header, the value specified in that header is used to determine sampling. If this
header is not present, whether or not the trace is sampled is determined by the sampling source configured for the 
`witchcraft-server` (by default, all traces are sampled). If a trace is not sampled, `witchcraft-server` will not 
generate any trace log output for it. However, the infrastructure will still perform all of the trace-related operations
(such as creating child spans and setting span information on headers). The install configuration field
`trace-sample-rate` represents a float between 0 and 1 (inclusive) to control the proportion of traces sampled by
default. If the `WithTraceSampler` server option is provided, it overrides this configuration.

`witchcraft-server` also ensures that the context for every request has a trace ID. After the logging middleware 
executes, the request is guaranteed to have a trace ID (either from the incoming request or from the newly generated 
root span), and that trace ID is registered on the context. The `witchcraft.TraceIDFromContext(context.Context) string`
function can be used to retrieve the trace ID from the context.

### Creating new spans/trace log entries
Use the `wtracing.StartSpanFromContext` function to start a new span.
This function will create a new span that is a child span of the span in the provided context.
Defer the `Finish()` function of the returned span to ensure that the span is properly marked as finished (the "finish"
operation will also generate a trace log entry if the span is sampled).

### Middleware
`witchcraft-server` supports registering middleware to perform custom handling/augmenting of incoming requests. There
are 2 different kinds of middleware: *request* and *route* middleware.

Request middleware is executed on every request received by the server. The function signature for request middleware is
`func(rw http.ResponseWriter, r *http.Request, next http.Handler)`. Request middleware is the most common kind of 
middleware. The server has built-in request middleware that adds a panic handler, sets the loggers and trade ID on the 
request context and updates request-related metrics. Any user-supplied request middleware is run after the built-in 
request middleware in the order in which they were added (which means that the context has all of the loggers 
configured). Request middleware is run before the request is handled by the router, which means that it is possible to 
rewrite the URL and other properties of the request and the router will route the _modified_ request. However, note that 
the built-in logging middleware extracts the UID, SID, TokenID and TraceID from the request and sets them on the loggers 
before user-provided middleware is invoked, so if the user-defined middleware modifies the header in a manner that would 
change any of these values, the middleware should also update the request context to have loggers that use the updated 
values.  

Route middleware is only executed on the routes that are registered on the router -- they wrap the handler registered on
the route, so they are executed after the path has been matched and the handler for the router has been located and the
path parameters have been extracted and set on the context. The function signature for route middleware is 
`func(rw http.ResponseWriter, r *http.Request, reqVals RequestVals, next RouteRequestHandler)`. The `RequestVals` struct
stores the path template for the route along with the path parameters and their values. The server has built-in route
middleware that records a request log entry after the request has completed and creates a trace log span and logs a 
trace log entry after the request has completed. Route middleware is run after all of the request middleware has run, 
and any user-supplied route middleware is run after the built-in route middleware in the order in which they were added.
Because router middleware is executed after the routing has been determined, changing the URL of the request will not
change the handler that is invoked or the path parameter values that have been extracted/stored (although it may still
impact behavior based on the content of the actual handler that is registered). In general, most users will likely use
request middleware rather than route middleware. However, if users want to only execute middleware on matched routes and
want route-specific information such as the unrendered path template and the path parameter values, then route
middleware should be used.

### Long-running execution not associated with a route
In some instances, a server may want a long-running task not associated with an endpoint. For example, the server may
want a long-running goroutine that performs an operation at some interval for the lifetime of the server.

It is recommended that such goroutines be launched in the initialization function provided to `witchcraft.With` and use
the `ctx Context` as its context. This context has the same lifecycle as the server and has all of the configured 
loggers (service loggers, metric loggers, etc.) already configured on it.

The provided context does not have a span or trace ID associated with it. If a trace ID is desired,
[create a new span](#creating-new-spanstrace-log-entries) with `wtracing.StartSpanFromContext` and the provided context to derive a new context that has a
new root span associated with it. This function also updates any loggers in the context to use the new trace ID (for 
example, service loggers will include the trace ID).

### Metrics
`witchcraft-server` initializes a metrics registry that uses the `github.com/palantir/pkg/metrics` package (which uses 
`github.com/rcrowley/go-metrics` internally) to track metrics for the server. All of the tracked metrics are emitted as
metric log entries once every metric emit interval, which is 60 seconds by default (and can be configured to be a custom
interval in the install configuration).

By default, `witchcraft-server` captures various Go runtime metrics (such as allocations, number of running goroutines, 
etc.) at the same frequency as the metric emit frequency. The collection of Go runtime statistics can be disabled with
the `WithDisableGoRuntimeMetrics` server method.

### SIGQUIT handling
`witchcraft-server` sets up a SIGQUIT handler such that, if the program is terminated using a SIGQUIT signal
(`kill -3`), a goroutine dump is written as a `diagnostic.1` log. This behavior can be disabled using
`server.WithDisableSigQuitHandler`.  If `server.WithSigQuitHandlerWriter` is used, the stacks will also be written in
their unparsed form to the provided writer.

### Shutdown signal handling
`witchcraft-server` attempts to drain active connections and gracefully shut down by calling `server.Shutdown` upon receiving a SIGTERM or SIGINT signal. This behavior can be disabled using `server.WithDisableShutdownSignalHandler`.

Example server initialization
-----------------------------

### Basic production server
The following is an example program that launches a `witchcraft-server` that registers a `GET /myNum` endpoint that
returns a randomly generated number encoded as JSON: 

```go
package main

import (
	"context"
	"math/rand"
	"net/http"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

func main() {
	if err := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			if err := registerMyNumEndpoint(info.Router); err != nil {
				return nil, err
			}
			return nil, nil
		}).
		Start(); err != nil {
		panic(err)
	}
}

func registerMyNumEndpoint(router wrouter.Router) error {
	return router.Get("/myNum", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		httpserver.WriteJSONResponse(rw, rand.Intn(100), http.StatusOK)
	}))
}
```

Creating a `witchcraft-server` starts with the `witchcraft.NewServer` function, which returns a new witchcraft server
with default configuration. The `*witchcraft.Server` struct has various `With*` functions that can be used to configure
the server, and the `Start()` function starts the server using the specified configuration.

The `WithInitFunc(InitFunc)` function is used to register routes on the server. The initialization function provided to 
`WithInitFunc` is of the type `witchcraft.InitFunc`, which has the following definition:
`type InitFunc func(ctx context.Context, info InitInfo) (cleanup func(), rErr error)`.

The `ctx` provided to the function is valid for the duration of the server and has loggers configured on it. The `info`
struct contains fields that can be used to initialize various state and configuration for the server -- refer to the
`InitInfo` documentation for more information. 

In this example, a "GET" endpoint is registered on the router using the "/myNum" path, and `rest` package is used to
write a JSON response.

This example server uses all of the `witchcraft` defaults -- it looks for install configuration in 
`var/conf/install.yml` and uses `config.Install` as its type, looks for runtime configuration in `var/conf/runtime.yml`
and uses `config.Runtime` as its type, and looks for an encrypted-config-value key in 
`var/conf/encrypted-config-value.key`. The install configuration must also specify paths to key and certificate files to
use for TLS.

### Basic local/test server
The defaults for the server make sense for a production environment, but can make running the server locally (or in 
tests) cumbersome. We can modify the `main` function as follows to configure the witchcraft server to use in-memory 
defaults:

```go
func main() {
	if err := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			if err := registerMyNumEndpoint(info.Router); err != nil {
				return nil, err
			}
			return nil, nil
		}).
		WithSelfSignedCertificate().
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithRuntimeConfig(config.Runtime{}).
		WithInstallConfig(config.Install{
			ProductName: "example-app",
			Server: config.Server{
				Port: 8100,
			},
			UseConsoleLog: true,
		}).
		Start(); err != nil {
		panic(err)
	}
}
```

The `WithSelfSignedCertificate()` function configures the server to start using a generated self-signed certificate, 
which removes the need to specify server TLS material. The `WithECVKeyProvider(witchcraft.ECVKeyNoOp())` function 
configures the server to use an empty ECV key source. The `WithRuntimeConfig(config.Runtime{})` function configures the 
server to use the  provided runtime configuration (in this case, it is empty), and the `WithInstallConfig` function 
specifies the install configuration that should be used (it specifies that port 8100 should be used and that log output 
should go to STDOUT).

With this configuration, the program can be run using `go run`:

```
➜ go run main.go
{"level":"INFO","time":"2018-11-27T05:47:02.013456Z","message":"Listening to https","type":"service.1","origin":"github.com/palantir/witchcraft-go-server/v2/app_example","params":{"address":":8100","server":"example-app"}}
```

Issuing a request to this server using `curl` produces the expected response (note that the `-k` option is used to skip
certificate verification because the server is using a self-signed certificate):

```go
➜ curl -k https://localhost:8100/myNum
81
```

You can also observe that the server emits trace and request logs based on receiving this request:

```
{"time":"2018-11-27T05:47:28.313585Z","type":"trace.1","span":{"traceId":"7e43bde2647413fc","id":"01228e628b3b3d22","name":"GET /myNum","parentId":"7e43bde2647413fc","timestamp":1543297648313551,"duration":29000}}
{"time":"2018-11-27T05:47:28.313719Z","type":"request.2","method":"GET","protocol":"HTTP/2.0","path":"/myNum","status":200,"requestSize":0,"responseSize":3,"duration":146,"traceId":"7e43bde2647413fc","params":{"Accept":"*/*","User-Agent":"curl/7.54.0","X-B3-Parentspanid":"7e43bde2647413fc","X-B3-Sampled":"1","X-B3-Spanid":"01228e628b3b3d22","X-B3-Traceid":"7e43bde2647413fc"}}
{"time":"2018-11-27T05:47:28.313802Z","type":"trace.1","span":{"traceId":"7e43bde2647413fc","id":"7e43bde2647413fc","name":"witchcraft-go-server request middleware","timestamp":1543297648313496,"duration":304000}}
```

### Server using install configuration
The previous examples used the built-in install configuration. Most real servers will use custom install configuration 
that specifies configuration for the server. Any struct can be used as install configuration, but it must support being
unmarshaled as YAML and must embed the `config.Install` struct. The install configuration is loaded once when the server
starts (it is never reloaded), so only values that are static for the lifetime of the server should be specified in this
configuration.

The following example modifies the previous example so that the endpoint returns the number defined in the install
configuration instead of a random number:

```go
package main

import (
	"context"
	"net/http"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

type AppInstallConfig struct {
	config.Install `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}

func main() {
	if err := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			if err := registerMyNumEndpoint(info.Router, info.InstallConfig.(AppInstallConfig).MyNum); err != nil {
				return nil, err
			}
			return nil, nil
		},
		).
		WithSelfSignedCertificate().
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithRuntimeConfig(config.Runtime{}).
		WithInstallConfigType(AppInstallConfig{}).
		WithInstallConfig(AppInstallConfig{
			Install: config.Install{
				ProductName: "example-app",
				Server: config.Server{
					Port: 8100,
				},
				UseConsoleLog: true,
			},
			MyNum: 13,
		}).
		Start(); err != nil {
		panic(err)
	}
}

func registerMyNumEndpoint(router wrouter.Router, num int) error {
	return router.Get("/myNum", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		httpserver.WriteJSONResponse(rw, num, http.StatusOK)
	}))
}
```

This example defines the `AppInstallConfig` struct, which embeds `config.Install` and also defines a `MyNum` field. The 
`WithInstallConfigType(AppInstallConfig{})` function call is added to specify `AppInstallConfig{}` as the install struct 
and the initialization function logic is modified to convert the provided `installConfig interface{}` into an 
`AppInstallConfig` and uses the `MyNum` value as the value that is returned by the endpoint. The `WithInstallConfig` 
function is also updated to use configuration that specifies a value for `MyNum`.

Running the updated program using `go run main.go` and issuing `curl -k https://localhost:8100/myNum` returns `13`.

A real program will generally read runtime configuration from disk rather than specifying it directly in code. We can
modify the example above to do this by simply removing the `WithInstallConfig` call:

```go
package main

import (
	"context"
	"net/http"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

type AppInstallConfig struct {
	config.Install `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}

func main() {
	if err := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			if err := registerMyNumEndpoint(info.Router, info.InstallConfig.(AppInstallConfig).MyNum); err != nil {
				return nil, err
			}
			return nil, nil
		},
		).
		WithSelfSignedCertificate().
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithInstallConfigType(AppInstallConfig{}).
		Start(); err != nil {
		panic(err)
	}
}

func registerMyNumEndpoint(router wrouter.Router, num int) error {
	return router.Get("/myNum", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		httpserver.WriteJSONResponse(rw, num, http.StatusOK)
	}))
}
```

By default, the install configuration is read from `var/conf/install.yml`. Create a file at that path relative to the
Go file and provide it with the YAML content for the configuration:

```yaml
product-name: "example-app"
use-console-log: true
server:
  port: 8100
my-num: 77
```

Running the updated program using `go run main.go` and issuing `curl -k https://localhost:8100/myNum` returns `77`.

### Server using runtime configuration
Runtime configuration is similar to install configuration. The main difference is that runtime configuration supports
reloading configuration. When file-based runtime configuration is used, whenever the configuration file is updated, the
associated values are updated as well.

The following example defines a custom runtime configuration struct and returns the refreshable int value in the runtime
from its endpoint (the example uses a basic in-memory install configuration for simplicity):

```go
package main

import (
	"context"
	"net/http"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

type AppRuntimeConfig struct {
	config.Runtime `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}

func main() {
	if err := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			myNumRefreshable := refreshable.NewInt(info.RuntimeConfig.Map(func(in interface{}) interface{} {
				return in.(AppRuntimeConfig).MyNum
			}))
			if err := registerMyNumEndpoint(info.Router, myNumRefreshable); err != nil {
				return nil, err
			}
			return nil, nil
		},
		).
		WithSelfSignedCertificate().
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		WithInstallConfig(config.Install{
			ProductName: "example-app",
			Server: config.Server{
				Port: 8100,
			},
			UseConsoleLog: true,
		}).
		WithRuntimeConfigType(AppRuntimeConfig{}).
		Start(); err != nil {
		panic(err)
	}
}

func registerMyNumEndpoint(router wrouter.Router, numProvider refreshable.Int) error {
	return router.Get("/myNum", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		httpserver.WriteJSONResponse(rw, numProvider.CurrentInt(), http.StatusOK)
	}))
}
``` 

The refreshable configuration warrants some closer examination. Note that the `registerMyNumEndpoint` takes a
`numProvider refreshable.Int` as an argument rather than an `int` and returns the result of `CurrentInt()`. 
Conceptually, the `numProvider` is guaranteed to always return the current value of the number specified in the runtime
configuration. Using this pattern removes the need for writing code that listens for updates -- the code can simply
assume that the provider always returns the most recent value. `refreshable.Int` and `refreshable.String` are helper
types that provide functions that return the current value of the correct type. For types without helper functions, the
general `refreshable.Refreshable` should be used, and the `interface{}` returned by `Current()` must be explicitly
converted to the proper target type (this is required because Go does not support generics/templatization).

The `numProvider` provided to `registerMyNumEndpoint` is derived by applying a mapping function to the
`runtimeConfig refreshable.Refreshable` parameter. `runtimeConfig.Map` is provided with a function that, given an
updated runtime configuration, returns the portion of the configuration that is required. The input to the mapping 
function must be explicitly cast to the runtime configuration type (in this case, `in.(AppRuntimeConfig)`), and then the
relevant section can be accessed (or derived) and returned. The result of the `Map` function is a `Refreshable` that
returns the mapped portion. In this case, because we know the result will always be an `int`, we wrap the returned
`Refreshable` in a `refreshable.NewInt` call, which provides the convenience function `CurrentInt()` that performs the
type conversion of the result to an `int`.

By default, the runtime configuration is read from `var/conf/runtime.yml`. Create a file at that path relative to the
Go file and provide it with the YAML content for the configuration:

```yaml
my-num: 99
```

Running the updated program using `go run main.go` and issuing `curl -k https://localhost:8100/myNum` returns `99`.
While the program is still running, update the content of the file to be `my-num: 88`, save it, then run the `curl`
command again. The output is `88`. 

### Full server example
The following is an example of a server that defines and uses both custom install and runtime configuration:

```go
package main

import (
	"context"
	"net/http"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-server/httpserver"
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/config"
	"github.com/palantir/witchcraft-go-server/v2/witchcraft"
	"github.com/palantir/witchcraft-go-server/v2/wrouter"
)

type AppInstallConfig struct {
	config.Install `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}

type AppRuntimeConfig struct {
	config.Runtime `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}

func main() {
	if err := witchcraft.NewServer().
		WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
			if err := registerInstallNumEndpoint(info.Router, info.InstallConfig.(AppInstallConfig).MyNum); err != nil {
				return nil, err
			}

			myNumRefreshable := refreshable.NewInt(info.RuntimeConfig.Map(func(in interface{}) interface{} {
				return in.(AppRuntimeConfig).MyNum
			}))
			if err := registerRuntimeNumEndpoint(info.Router, myNumRefreshable); err != nil {
				return nil, err
			}
			return nil, nil
		},
		).
		WithInstallConfigType(AppInstallConfig{}).
		WithRuntimeConfigType(AppRuntimeConfig{}).
		WithSelfSignedCertificate().
		WithECVKeyProvider(witchcraft.ECVKeyNoOp()).
		Start(); err != nil {
		panic(err)
	}
}

func registerInstallNumEndpoint(router wrouter.Router, num int) error {
	return router.Get("/installNum", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		httpserver.WriteJSONResponse(rw, num, http.StatusOK)
	}))
}

func registerRuntimeNumEndpoint(router wrouter.Router, numProvider refreshable.Int) error {
	return router.Get("/runtimeNum", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		httpserver.WriteJSONResponse(rw, numProvider.CurrentInt(), http.StatusOK)
	}))
}
```

With `var/conf/install.yml`:

```yaml
product-name: "example-app"
use-console-log: true
server:
  port: 8100
my-num: 7
```

And `var/conf/runtime.yml`:

```yaml
my-num: 13
```

Querying `installNum` returns `7`, while querying `runtimeNum` returns `13`:

```
➜ curl -k https://localhost:8100/installNum
7
➜ curl -k https://localhost:8100/runtimeNum
13
```

In a production server, `WithSelfSignedCertificate()` and `WithECVKeyProvider(witchcraft.ECVKeyNoOp())` would not be
called and the proper security and key material would exist in their expected locations.

Refreshable configuration
-------------------------
The runtime configuration for `witchcraft-server` uses the `refreshable.Refreshable` interface. Conceptually, a
`Refreshable` is a container that holds a value of a specific type that may be updated/refreshed. The following is the
interface definition for `Refreshable`:

```go
type Refreshable interface {
	// Current returns the most recent value of this Refreshable.
	Current() interface{}

	// Subscribe subscribes to changes of this Refreshable. The provided function is called with the value of Current()
	// whenever the value changes.
	Subscribe(consumer func(interface{})) (unsubscribe func())

	// Map returns a new Refreshable based on the current one that handles updates based on the current Refreshable.
	Map(func(interface{}) interface{}) Refreshable
}
```

The `runtimeConfig refreshable.Refreshable` parameter provided to the initialization function specified using 
`WithInitFunc` stores the latest unmarshaled runtime configuration as its current value, and the type of the value is
specified using the `WithRuntimeConfigType` function (if this function is not called, `config.Runtime` is used as the
default type).

For example, for the call: 

```go
witchcraft.NewServer().
    WithInitFunc(func(ctx context.Context, info witchcraft.InitInfo) (func(), error) {
        return nil, nil
    }).
    WithRuntimeConfigType(AppRuntimeConfig{})
```

The `WithRuntimeConfigType(AppRuntimeConfig{})` function specifies that the type of the runtime configuration is 
`AppRuntimeConfig`, so the value returned by `runtimeConfig.Current()` in `WithInitFunc` will have the type 
`AppRuntimeConfig`. Because Go does not have a notion of generics, the author must make this association manually and 
perform the conversion of the current value into the desired type when using it (for example, 
`runtimeConfig.Current().(AppRuntimeConfig)`).

The `Refreshable` interface supports using the `Map` function to derive a new refreshable based on the value of the
current refreshable. This allows downstream functions that are only interested in a subset of the refreshable to observe
just the relevant portion.

For example, consider the `AppRuntimeConfig` definition:

```go
type AppRuntimeConfig struct {
	config.Runtime `yaml:",inline"`

	MyNum int `yaml:"my-num"`
}
```

A downstream function may only be interested in updates to the `MyNum` variable -- if updates to `config.Runtime` are 
not relevant to the function, there is no need to subscribe to it. The following code derives a new `Refreshable` from
the `runtimeConfig` refreshable: 

```go
myNumRefreshable := runtimeConfig.Map(func(in interface{}) interface{} {
    return in.(AppRuntimeConfig).MyNum
})
```

The `Current()` function for `myNumRefreshable` returns the `MyNum` field of `in.(AppRuntimeConfig)`, and the derived
`Refreshable` is only updated when the derived value changes. Accessing a field is the most common usage of `Map`, but
any arbitrary logic can be performed in the mapping function. Just note that the mapping will be performed whenever the
parent refreshable is updated and the result will be compared using `reflect.DeepEqual`. 

The general `Refreshable` interface returns an `interface{}` and its result must always be converted to the actual
underlying type. However, if a `Refreshable` is known to return an `int`, `string` or `bool`, convenience wrapper types
are provided to return typed values. For example, `refreshable.NewInt(in Refreshable)` returns a `refreshable.Int`,
which is defined as:

```go
type Int interface {
	Refreshable
	CurrentInt() int
}
```

The `CurrentInt()` function returns the current value converted to an `int`, which makes it easier to use in code and
alleviates the need for clients to manually remember the type stored in the `Refreshable`.

If a `Refreshable` with a particular value/type is used widely throughout a code base, it may make sense to define a 
similar interface so that clients do not have to manually track the type information. For example, a typed `Refreshable`
for `AppRuntimeConfig` can be defined as follows:

```go
type RefreshableAppRuntimeConfig interface {
	Refreshable
	CurrentAppRuntimeConfig() AppRuntimeConfig
}

type refreshableAppRuntimeConfig struct {
	Refreshable
}

func (r refreshableAppRuntimeConfig) CurrentAppRuntimeConfig() AppRuntimeConfig {
	return rt.Current().(AppRuntimeConfig)
} 
```

### Updating refreshable configuration: provider-based vs. push-based
The "provider" model of configuration updates takes the philosophy that executing code simply needs the most up-to-date
value of a `Refreshable` when it executes. This model makes the most sense when the value is read whenever an endpoint
is executed or when a long-running or periodically executed background task executes. In these scenarios, the latest 
value of the `Refreshable` is only needed when the logic executes. This update model is typically the most common, and
is achieved by passing down specific `Refreshable` providers for the required values to the handlers/routines.

However, in some cases, an application may want to be notified of every update to a field and react to that update
immediately -- for example, if updating a specific configuration field triggers an expensive computation that should
happen immediately, the logic wants to be notified as soon as the update is made.

In this scenario, the `Subscribe` function should be used for the `Refreshable` that has the value for which updates are
needed. For example, consider the following configuration:

```go
type AppRuntimeConfig struct {
	config.Runtime `yaml:",inline"`

	AssetURLs []string `yaml:"asset-urls"`
}
```

The `AssetURLs` field specifies URLs that should be downloaded by the program whenever the value is updated. This
can be handled as follows:

```go
unsubscribe := runtimeConfig.Map(func(in interface{}) interface{} {
    return in.(AppRuntimeConfig).AssetURLs
}).Subscribe(func(in interface{}) {
	assetURLs := in.([]string)
	// perform work
})
// unsubscribe should be deferred or stored and run at shutdown 
```

The `Map` function returns a new `Refreshable` that updates only when the `AssetURLs` field is updated, and the
`Subscribe` function subscribes a listener that performs work as soon as the value is updated. This ensures that the 
logic is run as soon as the value is refreshed every time the value is updated.

License
-------
This project is made available under the [Apache 2.0 License](http://www.apache.org/licenses/LICENSE-2.0).
