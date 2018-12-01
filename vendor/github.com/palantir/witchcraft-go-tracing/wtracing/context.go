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
	"context"
)

type tracerContextKeyType string

const tracerContextKey = tracerContextKeyType("wtracing.tracer")

// ContextWithTracer returns a copy of the provided context with the provided Tracer included as a value.
func ContextWithTracer(ctx context.Context, tracer Tracer) context.Context {
	return context.WithValue(ctx, tracerContextKey, tracer)
}

// TracerFromContext returns the Tracer stored in the provided context. Returns nil if no Tracer is stored in the
// context.
func TracerFromContext(ctx context.Context) Tracer {
	if tracer, ok := ctx.Value(tracerContextKey).(Tracer); ok {
		return tracer
	}
	return nil
}

type spanContextKeyType string

const spanContextKey = spanContextKeyType("wtracing.span")

// ContextWithSpan returns a copy of the provided context with the provided span set on it.
func ContextWithSpan(ctx context.Context, s Span) context.Context {
	return context.WithValue(ctx, spanContextKey, s)
}

// SpanFromContext returns the span stored in the provided context, or nil if no span is stored in the context.
func SpanFromContext(ctx context.Context) Span {
	if s, ok := ctx.Value(spanContextKey).(Span); ok {
		return s
	}
	return nil
}

// TraceIDFromContext returns the traceId associated with the span stored in the provided context. Returns an empty
// string if no span is stored in the context.
func TraceIDFromContext(ctx context.Context) TraceID {
	if span := SpanFromContext(ctx); span != nil {
		return span.Context().TraceID
	}
	return ""
}
