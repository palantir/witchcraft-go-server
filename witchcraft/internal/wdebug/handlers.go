// Copyright (c) 2020 Palantir Technologies. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wdebug

import (
	"context"
	"io"
	"runtime/pprof"
	"runtime/trace"
	"time"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/codecs"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	"github.com/palantir/pkg/metrics"
	werror "github.com/palantir/witchcraft-go-error"
	wparams "github.com/palantir/witchcraft-go-params"
)

const (
	DiagnosticTypeCPUProfile1MinuteV1 DiagnosticType = "go.profile.cpu.1minute.v1"
	DiagnosticTypeHeapProfileV1       DiagnosticType = "go.profile.heap.v1"
	DiagnosticTypeAllocsProfileV1     DiagnosticType = "go.profile.allocs.v1"
	DiagnosticTypeTrace1MinuteV1      DiagnosticType = "go.trace.1minute.v1"
	DiagnosticTypeGoroutinesV1        DiagnosticType = "go.goroutines.v1"
	DiagnosticTypeMetricNamesV1       DiagnosticType = "metric.names.v1"
	DiagnosticTypeSystemTimeV1        DiagnosticType = "os.system.clock.v1"
)

var diagnosticHandlers = map[DiagnosticType]DiagnosticHandler{
	DiagnosticTypeCPUProfile1MinuteV1: handlerCPUProfile1MinuteV1{},
	DiagnosticTypeHeapProfileV1:       handlerHeapProfileV1{},
	DiagnosticTypeAllocsProfileV1:     handlerAllocsProfileV1{},
	DiagnosticTypeTrace1MinuteV1:      handlerTrace1MinuteV1{},
	DiagnosticTypeGoroutinesV1:        handlerGoroutinesV1{},
	DiagnosticTypeMetricNamesV1:       handlerMetricNamesV1{},
	DiagnosticTypeSystemTimeV1:        handlerSystemTimeV1{},
}

type DiagnosticHandler interface {
	Type() DiagnosticType
	Documentation() string
	ContentType() string
	SafeLoggable() bool
	Extension() string
	WriteDiagnostic(ctx context.Context, w io.Writer) error
}

type handlerGoroutinesV1 struct{}

func (h handlerGoroutinesV1) Type() DiagnosticType {
	return DiagnosticTypeGoroutinesV1
}

func (h handlerGoroutinesV1) ContentType() string {
	return codecs.Binary.ContentType()
}

func (h handlerGoroutinesV1) Documentation() string {
	return "Returns the plaintext representation of currently running goroutines and their stacktraces"
}

func (h handlerGoroutinesV1) SafeLoggable() bool {
	return true
}

func (h handlerGoroutinesV1) Extension() string {
	return "prof"
}

func (h handlerGoroutinesV1) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	if err := pprof.Lookup("goroutine").WriteTo(w, 0); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to write goroutine dump")
	}
	return nil
}

type handlerCPUProfile1MinuteV1 struct{}

func (h handlerCPUProfile1MinuteV1) Type() DiagnosticType {
	return DiagnosticTypeCPUProfile1MinuteV1
}

func (h handlerCPUProfile1MinuteV1) ContentType() string {
	return codecs.Binary.ContentType()
}

func (h handlerCPUProfile1MinuteV1) Documentation() string {
	return `A profile recording CPU usage for one minute. See golang docs for analysis tooling: https://golang.org/doc/diagnostics.html#profiling`
}

func (h handlerCPUProfile1MinuteV1) SafeLoggable() bool {
	return true
}

func (h handlerCPUProfile1MinuteV1) Extension() string {
	return "prof"
}

func (h handlerCPUProfile1MinuteV1) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	const duration = time.Minute

	if err := pprof.StartCPUProfile(w); err != nil {
		err = werror.WrapWithContextParams(ctx, err, "failed to start CPU profile")
		return errors.WrapWithConflict(err, wparams.NewSafeParamStorer(map[string]interface{}{
			"message": err.Error(),
		}))
	}
	defer pprof.StopCPUProfile()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
	}
	return nil
}

type handlerHeapProfileV1 struct{}

func (h handlerHeapProfileV1) Type() DiagnosticType {
	return DiagnosticTypeHeapProfileV1
}

func (h handlerHeapProfileV1) ContentType() string {
	return codecs.Binary.ContentType()
}

func (h handlerHeapProfileV1) Documentation() string {
	return `A profile recording in-use objects on the heap as of the last garbage collection. See golang docs for analysis tooling: https://golang.org/doc/diagnostics.html#profiling`
}

func (h handlerHeapProfileV1) SafeLoggable() bool {
	return true
}

func (h handlerHeapProfileV1) Extension() string {
	return "prof"
}

func (h handlerHeapProfileV1) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	if err := pprof.Lookup("heap").WriteTo(w, 0); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to write heap in-use profile")
	}
	return nil
}

type handlerAllocsProfileV1 struct{}

func (h handlerAllocsProfileV1) Type() DiagnosticType {
	return DiagnosticTypeAllocsProfileV1
}

func (h handlerAllocsProfileV1) ContentType() string {
	return codecs.Binary.ContentType()
}

func (h handlerAllocsProfileV1) Documentation() string {
	return `A profile recording all allocated objects since the process started. See golang docs for analysis tooling: https://golang.org/doc/diagnostics.html#profiling`
}

func (h handlerAllocsProfileV1) SafeLoggable() bool {
	return true
}

func (h handlerAllocsProfileV1) Extension() string {
	return "prof"
}

func (h handlerAllocsProfileV1) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	if err := pprof.Lookup("allocs").WriteTo(w, 0); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to write heap allocs profile")
	}
	return nil
}

type handlerTrace1MinuteV1 struct{}

func (h handlerTrace1MinuteV1) Type() DiagnosticType {
	return DiagnosticTypeTrace1MinuteV1
}

func (h handlerTrace1MinuteV1) ContentType() string {
	return codecs.Binary.ContentType()
}

func (h handlerTrace1MinuteV1) Documentation() string {
	return `An execution trace of the program for 1 minute. See golang docs for analysis tooling: https://golang.org/doc/diagnostics.html#profiling`
}

func (h handlerTrace1MinuteV1) SafeLoggable() bool {
	return true
}

func (h handlerTrace1MinuteV1) Extension() string {
	return "prof"
}

func (h handlerTrace1MinuteV1) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	const duration = time.Minute

	if err := trace.Start(w); err != nil {
		err = werror.WrapWithContextParams(ctx, err, "failed to start execution tracer")
		return errors.WrapWithConflict(err, wparams.NewSafeParamStorer(map[string]interface{}{
			"message": err.Error(),
		}))
	}
	defer trace.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
	}
	return nil
}

type handlerMetricNamesV1 struct{}

func (h handlerMetricNamesV1) Type() DiagnosticType {
	return DiagnosticTypeMetricNamesV1
}

func (h handlerMetricNamesV1) ContentType() string {
	return codecs.JSON.ContentType()
}

func (h handlerMetricNamesV1) Documentation() string {
	return `Records all metric names and tag sets in the process's metric registry`
}

func (h handlerMetricNamesV1) SafeLoggable() bool {
	return true
}

func (h handlerMetricNamesV1) Extension() string {
	return "json"
}

func (h handlerMetricNamesV1) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	// Assume that the metric registry is bound to the context by WC middleware
	registry := metrics.FromContext(ctx)

	result := make([]map[string]interface{}, 0)
	registry.Each(func(name string, tags metrics.Tags, value metrics.MetricVal) {
		result = append(result, map[string]interface{}{
			"name": name,
			"tags": tags.ToMap(),
		})
	})

	if err := codecs.JSON.Encode(w, result); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to write metric names")
	}
	return nil
}

type handlerSystemTimeV1 struct{}

func (h handlerSystemTimeV1) Type() DiagnosticType {
	return DiagnosticTypeSystemTimeV1
}

func (h handlerSystemTimeV1) Documentation() string {
	return `This diagnostic prints the current system timestamp as observed by the service. This could be useful for diagnosing clock skew.`
}

func (h handlerSystemTimeV1) ContentType() string {
	return codecs.Plain.ContentType()
}

func (h handlerSystemTimeV1) SafeLoggable() bool {
	return true
}

func (h handlerSystemTimeV1) Extension() string {
	return "txt"
}

func (h handlerSystemTimeV1) WriteDiagnostic(ctx context.Context, w io.Writer) error {
	if err := codecs.Plain.Encode(w, time.Now().Format(time.RFC3339Nano)); err != nil {
		return werror.WrapWithContextParams(ctx, err, "failed to write system time")
	}
	return nil
}
