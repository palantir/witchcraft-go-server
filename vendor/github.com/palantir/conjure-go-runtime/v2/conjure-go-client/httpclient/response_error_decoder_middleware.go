// Copyright (c) 2019 Palantir Technologies. All rights reserved.
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
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-client/httpclient/internal"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/codecs"
	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/errors"
	werror "github.com/palantir/witchcraft-go-error"
)

// ErrorDecoder implementations declare whether or not they should be used to handle certain http responses, and return
// decoded errors when invoked. Custom implementations can be used when consumers expect structured errors in response bodies.
type ErrorDecoder interface {
	// Handles returns whether or not the decoder considers the response an error.
	Handles(resp *http.Response) bool
	// DecodeError returns a decoded error, or an error encountered while trying to decode.
	// DecodeError should never return nil.
	DecodeError(resp *http.Response) error
}

// errorDecoderMiddleware intercepts a round trip's response.
// If the supplied ErrorDecoder handles the response, we return the error as decoded by ErrorDecoder.
// In this case, the *http.Response returned will be nil.
type errorDecoderMiddleware struct {
	errorDecoder ErrorDecoder
}

func (e errorDecoderMiddleware) RoundTrip(req *http.Request, next http.RoundTripper) (*http.Response, error) {
	resp, err := next.RoundTrip(req)
	// if error is already set, it is more severe than our HTTP error. Just return it.
	if resp == nil || err != nil {
		return nil, err
	}
	if e.errorDecoder.Handles(resp) {
		defer internal.DrainBody(req.Context(), resp)
		return nil, e.errorDecoder.DecodeError(resp)
	}
	return resp, nil
}

// restErrorDecoder is our default error decoder.
// It handles responses of status code >= 307. In this case,
// we create and return a werror with the 'statusCode' parameter
// set to the integer value from the response.
//
// Use StatusCodeFromError(err) to retrieve the code from the error,
// and WithDisableRestErrors() to disable this middleware on your client.
//
// If the response has a Content-Type containing 'application/json', we attempt
// to unmarshal the error as a conjure error. See TestErrorDecoderMiddlewares for
// example error messages and parameters.
type restErrorDecoder struct {
	conjureErrorDecoder errors.ConjureErrorDecoder
}

var _ ErrorDecoder = restErrorDecoder{}

func (d restErrorDecoder) Handles(resp *http.Response) bool {
	return resp.StatusCode >= http.StatusTemporaryRedirect
}

func (d restErrorDecoder) DecodeError(resp *http.Response) error {
	safeParams := map[string]interface{}{
		"statusCode": resp.StatusCode,
	}
	unsafeParams := map[string]interface{}{}
	if resp.StatusCode >= http.StatusTemporaryRedirect &&
		resp.StatusCode < http.StatusBadRequest {
		location, err := resp.Location()
		if err == nil {
			unsafeParams["location"] = location.String()
		}
	}
	wSafeParams := werror.SafeParams(safeParams)
	wUnsafeParams := werror.UnsafeParams(unsafeParams)

	// TODO(#98): If a byte buffer pool is configured, use it to avoid an allocation.
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return werror.Wrap(err, "server returned an error and failed to read body", wSafeParams, wUnsafeParams)
	}
	if len(body) == 0 {
		return werror.Error(resp.Status, wSafeParams, wUnsafeParams)
	}

	// If JSON, try to unmarshal as conjure error
	if isJSON := strings.Contains(resp.Header.Get("Content-Type"), codecs.JSON.ContentType()); !isJSON {
		return werror.Error(resp.Status, wSafeParams, wUnsafeParams, werror.UnsafeParam("responseBody", string(body)))
	}
	var conjureErr errors.Error
	var jsonErr error
	if d.conjureErrorDecoder != nil {
		conjureErr, jsonErr = errors.UnmarshalErrorWithDecoder(d.conjureErrorDecoder, body)
	} else {
		conjureErr, jsonErr = errors.UnmarshalError(body)
	}
	if jsonErr != nil {
		return werror.Error(resp.Status, wSafeParams, wUnsafeParams, werror.UnsafeParam("responseBody", string(body)))
	}
	return werror.Wrap(conjureErr, "", wSafeParams, wUnsafeParams)
}

// StatusCodeFromError wraps the internal StatusCodeFromError func. For behavior details, see its docs.
func StatusCodeFromError(err error) (statusCode int, ok bool) {
	return internal.StatusCodeFromError(err)
}

// LocationFromError wraps the internal LocationFromError func. For behavior details, see its docs.
func LocationFromError(err error) (location string, ok bool) {
	return internal.LocationFromError(err)
}
