// Copyright (c) 2018 Palantir Technologies. All rights reserved.
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

package wtracing

import (
	"time"
)

// TraceID and SpanID are identifiers for the trace and span. Represented as hex strings.
type TraceID string
type SpanID string

type Span interface {
	Context() SpanContext

	// Finish the Span and send to Reporter.
	Finish()
}

type SpanModel struct {
	SpanContext

	Name           string
	Kind           Kind
	Timestamp      time.Time
	Duration       time.Duration
	LocalEndpoint  *Endpoint
	RemoteEndpoint *Endpoint
}

type SpanContext struct {
	TraceID  TraceID
	ID       SpanID
	ParentID *SpanID
	Debug    bool
	Sampled  *bool
	Err      error
}

func FromSpanOptions(opts ...SpanOption) *SpanOptionImpl {
	impl := &SpanOptionImpl{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.apply(impl)
	}
	return impl
}

type SpanOption interface {
	apply(impl *SpanOptionImpl)
}

type spanOptionFn func(impl *SpanOptionImpl)

func (t spanOptionFn) apply(impl *SpanOptionImpl) {
	t(impl)
}

type SpanOptionImpl struct {
	RemoteEndpoint *Endpoint
	ParentSpan     *SpanContext
	Kind           Kind
}

func WithKind(kind Kind) SpanOption {
	return spanOptionFn(func(impl *SpanOptionImpl) {
		impl.Kind = kind
	})
}

func WithParent(sc SpanContext) SpanOption {
	return spanOptionFn(func(impl *SpanOptionImpl) {
		impl.ParentSpan = &sc
	})
}

func WithRemoteEndpoint(endpoint *Endpoint) SpanOption {
	return spanOptionFn(func(impl *SpanOptionImpl) {
		impl.RemoteEndpoint = endpoint
	})
}
