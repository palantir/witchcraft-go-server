godel-sls-diagnostics-generator
============

godel-sls-diagnostics-generator is a [godel](https://github.com/palantir/godel) plugin which generates a file named `sls-diagnostics.json` in each package containing types that implement `wdebug.DiagnosticHandler`.
This allows downstream projects to identify at build-time what diagnostics are supported by a dependency.

This operation happens in three steps:
1. Load all packages managed by the godel project and find the wdebug.DiagnosticHandler interface type.
1. Iterate through all loaded packages and check whether each declared type implements the interface. If so, store the type to a list.
1. In each package containing interface implementations, render the sls-diagnosics.json file. If the file exists in a package which no longer contains implementing types, delete the file.

Example JSON content:
```json
{
  "diagnostics": [
    {
      "type": "go.goroutines.v1",
      "docs": "Returns the plaintext representation of currently running goroutines and their stacktraces",
      "safe": true
    },
    {
      "type": "go.profile.allocs.v1",
      "docs": "A profile recording all allocated objects since the process started. See golang docs for analysis tooling: https://golang.org/doc/diagnostics.html#profiling",
      "safe": true
    },
    {
      "type": "go.profile.cpu.1minute.v1",
      "docs": "A profile recording CPU usage for one minute. See golang docs for analysis tooling: https://golang.org/doc/diagnostics.html#profiling",
      "safe": true
    },
    {
      "type": "go.profile.heap.v1",
      "docs": "A profile recording in-use objects on the heap as of the last garbage collection. See golang docs for analysis tooling: https://golang.org/doc/diagnostics.html#profiling",
      "safe": true
    },
    {
      "type": "metric.names.v1",
      "docs": "Records all metric names and tag sets in the process's metric registry",
      "safe": true
    }
  ]
}
```
