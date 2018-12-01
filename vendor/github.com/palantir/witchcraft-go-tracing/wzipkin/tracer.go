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

package wzipkin

import (
	"fmt"
	"strconv"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/idgenerator"
	"github.com/openzipkin/zipkin-go/model"
	"github.com/palantir/witchcraft-go-tracing/wtracing"
)

func NewTracer(rep wtracing.Reporter, opts ...wtracing.TracerOption) (wtracing.Tracer, error) {
	zipkinTracer, err := zipkin.NewTracer(newZipkinReporterAdapter(rep), toZipkinTracerOptions(wtracing.FromTracerOptions(opts...))...)
	if err != nil {
		return nil, err
	}
	return &tracerImpl{
		tracer: zipkinTracer,
	}, nil
}

func toZipkinTracerOptions(impl *wtracing.TracerOptionImpl) []zipkin.TracerOption {
	var zipkinTracerOptions []zipkin.TracerOption
	zipkinTracerOptions = append(zipkinTracerOptions, zipkin.WithSharedSpans(false))
	zipkinTracerOptions = append(zipkinTracerOptions, zipkin.WithLocalEndpoint(toZipkinEndpoint(impl.LocalEndpoint)))
	if impl.Sampler != nil {
		zipkinTracerOptions = append(zipkinTracerOptions, zipkin.WithSampler(zipkin.Sampler(impl.Sampler)))
	}
	if impl.Generator != nil {
		zipkinTracerOptions = append(zipkinTracerOptions, zipkin.WithIDGenerator(toZipkinIDGenerator(impl.Generator)))
	}
	return zipkinTracerOptions
}

type tracerImpl struct {
	tracer *zipkin.Tracer
}

func (t *tracerImpl) StartSpan(name string, options ...wtracing.SpanOption) wtracing.Span {
	return fromZipkinSpan(t.tracer.StartSpan(name, toZipkinSpanOptions(wtracing.FromSpanOptions(options...))...))
}

func toZipkinEndpoint(endpoint *wtracing.Endpoint) *model.Endpoint {
	if endpoint == nil {
		return nil
	}
	return &model.Endpoint{
		ServiceName: endpoint.ServiceName,
		IPv4:        endpoint.IPv4,
		IPv6:        endpoint.IPv6,
		Port:        endpoint.Port,
	}
}

func fromZipkinEndpoint(endpoint *model.Endpoint) *wtracing.Endpoint {
	if endpoint == nil {
		return nil
	}
	return &wtracing.Endpoint{
		ServiceName: endpoint.ServiceName,
		IPv4:        endpoint.IPv4,
		IPv6:        endpoint.IPv6,
		Port:        endpoint.Port,
	}
}

func toZipkinIDGenerator(generator wtracing.IDGenerator) idgenerator.IDGenerator {
	return &idGeneratorAdapter{
		generator: generator,
	}
}

type idGeneratorAdapter struct {
	generator wtracing.IDGenerator
}

func (gen *idGeneratorAdapter) TraceID() model.TraceID {
	traceID := gen.generator.TraceID()
	zipkinTraceID, err := model.TraceIDFromHex(string(traceID))
	if err != nil {
		panic(fmt.Sprintf("invalid TraceID %s: %v", traceID, err))
	}
	return zipkinTraceID
}

func (gen *idGeneratorAdapter) SpanID(traceID model.TraceID) model.ID {
	spanID := gen.generator.SpanID(wtracing.TraceID(traceID.String()))
	zipkinSpanID, err := strconv.ParseUint(string(spanID), 16, 64)
	if err != nil {
		panic(fmt.Sprintf("invalid SpanID %s: %v", spanID, err))
	}
	return model.ID(zipkinSpanID)
}

func FromZipkinIDGenerator(generator idgenerator.IDGenerator) wtracing.IDGenerator {
	return &zipkinIDGeneratorAdapter{
		generator: generator,
	}
}

type zipkinIDGeneratorAdapter struct {
	generator idgenerator.IDGenerator
}

func (gen *zipkinIDGeneratorAdapter) TraceID() wtracing.TraceID {
	return wtracing.TraceID(gen.generator.TraceID().String())
}

func (gen *zipkinIDGeneratorAdapter) SpanID(traceID wtracing.TraceID) wtracing.SpanID {
	zipkinTraceID, err := model.TraceIDFromHex(string(traceID))
	if err != nil {
		panic(fmt.Sprintf("invalid TraceID %s: %v", string(traceID), err))
	}
	return wtracing.SpanID(gen.generator.SpanID(zipkinTraceID).String())
}
