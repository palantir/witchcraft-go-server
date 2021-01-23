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

package httpclient

import (
	"context"
	"net/http"
	"net/url"

	"github.com/palantir/pkg/bytesbuffers"
)

type requestBuilder struct {
	method         string
	path           string
	address        string
	headers        http.Header
	query          url.Values
	bodyMiddleware *bodyMiddleware
	bufferPool     bytesbuffers.Pool

	errorDecoderMiddleware Middleware
	configureCtx           []func(context.Context) context.Context
}

const traceIDHeaderKey = "X-B3-TraceId"

type RequestParam interface {
	apply(*requestBuilder) error
}

type requestParamFunc func(*requestBuilder) error

func (f requestParamFunc) apply(b *requestBuilder) error {
	return f(b)
}
