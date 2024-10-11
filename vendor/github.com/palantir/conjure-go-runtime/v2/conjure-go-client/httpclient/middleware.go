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
	"net/http"
)

// A Middleware wraps an http client's request and is able to read or modify the request and response.
type Middleware interface {
	// RoundTrip mimics the API of http.RoundTripper, but adds a 'next' argument.
	// RoundTrip is responsible for invoking next.RoundTrip(req) and returning the response.
	RoundTrip(req *http.Request, next http.RoundTripper) (*http.Response, error)
}

// MiddlewareFunc is a convenience type alias that implements Middleware.
type MiddlewareFunc func(req *http.Request, next http.RoundTripper) (*http.Response, error)

func (f MiddlewareFunc) RoundTrip(req *http.Request, next http.RoundTripper) (*http.Response, error) {
	return f(req, next)
}

// wrapTransport is used by clientBuilder to create the final Client's RoundTripper.
func wrapTransport(baseTransport http.RoundTripper, middlewares ...Middleware) http.RoundTripper {
	if baseTransport == nil {
		baseTransport = http.DefaultTransport
	}
	for i := range middlewares {
		if middleware := middlewares[i]; middleware != nil {
			baseTransport = &wrappedClient{baseTransport: baseTransport, middleware: middleware}
		}
	}
	return baseTransport
}

type wrappedClient struct {
	baseTransport http.RoundTripper
	middleware    Middleware
}

func (c *wrappedClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.middleware.RoundTrip(req, c.baseTransport)
}
