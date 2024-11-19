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
	"strings"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient/internal"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient/internal/refreshingclient"
	"github.com/palantir/pkg/bytesbuffers"
	"github.com/palantir/pkg/refreshable"
	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

// A Client executes requests to a configured service.
//
// The Get/Head/Post/Put/Delete methods are for conveniently setting the method type and calling Do()
type Client interface {
	// Do executes a full request. Any input or output should be specified via params.
	// By the time it is returned, the response's body will be fully read and closed.
	// Use the WithResponse* params to unmarshal the body before Do() returns.
	//
	// In the case of a response with StatusCode >= 400, Do() will return a nil response and a non-nil error.
	// Use StatusCodeFromError(err) to retrieve the code from the error.
	// Use WithDisableRestErrors() to disable this middleware on your client.
	// Use WithErrorDecoder(errorDecoder) to replace this default behavior with custom error decoding behavior.
	Do(ctx context.Context, params ...RequestParam) (*http.Response, error)

	Get(ctx context.Context, params ...RequestParam) (*http.Response, error)
	Head(ctx context.Context, params ...RequestParam) (*http.Response, error)
	Post(ctx context.Context, params ...RequestParam) (*http.Response, error)
	Put(ctx context.Context, params ...RequestParam) (*http.Response, error)
	Delete(ctx context.Context, params ...RequestParam) (*http.Response, error)
}

type clientImpl struct {
	serviceName            refreshable.String
	client                 RefreshableHTTPClient
	middlewares            []Middleware
	errorDecoderMiddleware Middleware
	recoveryMiddleware     Middleware

	uriScorer      internal.RefreshableURIScoringMiddleware
	maxAttempts    refreshable.IntPtr // 0 means no limit. If nil, uses 2*len(uris).
	backoffOptions refreshingclient.RefreshableRetryParams
	bufferPool     bytesbuffers.Pool
}

func (c *clientImpl) Get(ctx context.Context, params ...RequestParam) (*http.Response, error) {
	return c.Do(ctx, append(params, WithRequestMethod(http.MethodGet))...)
}

func (c *clientImpl) Head(ctx context.Context, params ...RequestParam) (*http.Response, error) {
	return c.Do(ctx, append(params, WithRequestMethod(http.MethodHead))...)
}

func (c *clientImpl) Post(ctx context.Context, params ...RequestParam) (*http.Response, error) {
	return c.Do(ctx, append(params, WithRequestMethod(http.MethodPost))...)
}

func (c *clientImpl) Put(ctx context.Context, params ...RequestParam) (*http.Response, error) {
	return c.Do(ctx, append(params, WithRequestMethod(http.MethodPut))...)
}

func (c *clientImpl) Delete(ctx context.Context, params ...RequestParam) (*http.Response, error) {
	return c.Do(ctx, append(params, WithRequestMethod(http.MethodDelete))...)
}

func (c *clientImpl) Do(ctx context.Context, params ...RequestParam) (*http.Response, error) {
	uris := c.uriScorer.CurrentURIScoringMiddleware().GetURIsInOrderOfIncreasingScore()
	if len(uris) == 0 {
		return nil, werror.WrapWithContextParams(ctx, ErrEmptyURIs, "", werror.SafeParam("serviceName", c.serviceName.CurrentString()))
	}

	attempts := 2 * len(uris)
	if c.maxAttempts != nil {
		if confMaxAttempts := c.maxAttempts.CurrentIntPtr(); confMaxAttempts != nil {
			attempts = *confMaxAttempts
		}
	}

	retrier := internal.NewRequestRetrier(uris, c.backoffOptions.CurrentRetryParams().Start(ctx), attempts)
	uri, isRelocated := retrier.GetNextURI(nil, nil)
	for {
		resp, retryable, err := c.doOnce(ctx, uri, isRelocated, params...)
		if !retryable {
			return resp, err
		}
		uri, isRelocated = retrier.GetNextURI(resp, err)
		if uri == "" {
			return resp, err
		}
		if err != nil {
			svc1log.FromContext(ctx).Debug("Retrying request", svc1log.Stacktrace(err))
		}
	}
}

func (c *clientImpl) doOnce(
	ctx context.Context,
	baseURI string,
	useBaseURIOnly bool,
	params ...RequestParam,
) (_ *http.Response, retryable bool, _ error) {

	// 1. create the request
	b := &requestBuilder{
		headers:        make(http.Header),
		query:          make(url.Values),
		bodyMiddleware: &bodyMiddleware{bufferPool: c.bufferPool},
	}

	for _, p := range params {
		if p == nil {
			continue
		}
		if err := p.apply(b); err != nil {
			return nil, false, err
		}
	}
	if useBaseURIOnly {
		b.path = ""
	}

	for _, c := range b.configureCtx {
		ctx = c(ctx)
	}

	if b.method == "" {
		return nil, false, werror.ErrorWithContextParams(ctx, "httpclient: use WithRequestMethod() to specify HTTP method")
	}
	reqURI := joinURIAndPath(baseURI, b.path)
	req, err := http.NewRequestWithContext(ctx, b.method, reqURI, nil)
	if err != nil {
		return nil, false, werror.WrapWithContextParams(ctx, err, "failed to build new HTTP request")
	}

	req.Header = b.headers
	if q := b.query.Encode(); q != "" {
		req.URL.RawQuery = q
	}

	// 2. create the transport and client
	// shallow copy so we can overwrite the Transport with a wrapped one.
	clientCopy := *c.client.CurrentHTTPClient()

	// use request-specific timeout if set
	if b.requestTimeout != nil {
		clientCopy.Timeout = *b.requestTimeout
	}

	transport := clientCopy.Transport // start with the client's transport configured with default middleware

	// must precede the error decoders to read the status code of the raw response.
	transport = wrapTransport(transport, c.uriScorer.CurrentURIScoringMiddleware())
	// request decoder must precede the client decoder
	// must precede the body middleware to read the response body
	transport = wrapTransport(transport, b.errorDecoderMiddleware, c.errorDecoderMiddleware)
	// must precede the body middleware to read the request body
	transport = wrapTransport(transport, c.middlewares...)
	// must wrap inner middlewares to mutate the return values
	transport = wrapTransport(transport, b.bodyMiddleware)
	// must be the outermost middleware to recover panics in the rest of the request flow
	// there is a second, inner recoveryMiddleware in the client's default middlewares so that panics
	// inside the inner-most RoundTrip benefit from traceIDs and loggers set on the context.
	transport = wrapTransport(transport, c.recoveryMiddleware)

	clientCopy.Transport = transport

	// 3. execute the request using the client to get and handle the response
	resp, respErr := clientCopy.Do(req)

	// unless this is exactly the scenario where the caller has opted into being responsible for draining and closing
	// the response body, be sure to do so here.
	if !(respErr == nil && b.bodyMiddleware.rawOutput) {
		internal.DrainBody(ctx, resp)
	}

	// doOnce should be retried unless the body specifically indicates it can not be replayed.
	if respErr != nil {
		if !b.bodyMiddleware.noRetriesRequestBody() {
			retryable = true
		} else {
			svc1log.FromContext(ctx).Debug("Request body can not be replayed, not retrying.")
		}
		return nil, retryable, unwrapURLError(ctx, respErr)
	}

	return resp, false, nil
}

// unwrapURLError converts a *url.Error to a werror. We need this because all
// errors from the stdlib's client.Do are wrapped in *url.Error, and if we
// were to blindly return that we would lose any werror params stored on the
// underlying Err.
func unwrapURLError(ctx context.Context, respErr error) error {
	if respErr == nil {
		return nil
	}

	urlErr, ok := respErr.(*url.Error)
	if !ok {
		// We don't recognize this as a url.Error, just return the original.
		return respErr
	}
	params := []werror.Param{werror.SafeParam("requestMethod", urlErr.Op)}

	if parsedURL, _ := url.Parse(urlErr.URL); parsedURL != nil {
		params = append(params,
			werror.SafeParam("requestHost", parsedURL.Host),
			werror.UnsafeParam("requestPath", parsedURL.Path))
	}

	return werror.WrapWithContextParams(ctx, urlErr.Err, "httpclient request failed", params...)
}

func joinURIAndPath(baseURI, reqPath string) string {
	fullURI := strings.TrimRight(baseURI, "/")
	if reqPath != "" {
		fullURI += "/" + strings.TrimLeft(reqPath, "/")
	}
	return fullURI
}
