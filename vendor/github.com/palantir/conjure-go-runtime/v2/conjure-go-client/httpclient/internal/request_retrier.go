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

package internal

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/palantir/pkg/retry"
)

const (
	meshSchemePrefix = "mesh-"
)

// RequestRetrier manages URIs for an HTTP client, providing an API which determines whether requests should be retries
// and supplying the correct URL for the client to retry.
// In the case of servers in a service-mesh, requests will never be retried and the mesh URI will only be returned on the
// first call to GetNextURI
type RequestRetrier struct {
	currentURI    string
	retrier       retry.Retrier
	uris          []string
	offset        int
	relocatedURIs map[string]struct{}
	failedURIs    map[string]struct{}
	maxAttempts   int
	attemptCount  int
}

// NewRequestRetrier creates a new request retrier.
// Regardless of maxAttempts, mesh URIs will never be retried.
func NewRequestRetrier(uris []string, retrier retry.Retrier, maxAttempts int) *RequestRetrier {
	offset := 0
	return &RequestRetrier{
		currentURI:    uris[offset],
		retrier:       retrier,
		uris:          uris,
		offset:        offset,
		relocatedURIs: map[string]struct{}{},
		failedURIs:    map[string]struct{}{},
		maxAttempts:   maxAttempts,
		attemptCount:  0,
	}
}

func (r *RequestRetrier) attemptsRemaining() bool {
	// maxAttempts of 0 indicates no limit
	if r.maxAttempts == 0 {
		return true
	}
	return r.attemptCount < r.maxAttempts
}

// GetNextURI returns the next URI a client should use, or empty string if no suitable URI remaining to retry.
// isRelocated is true when the URI comes from a redirect's Location header. In this case, it already includes the request path.
func (r *RequestRetrier) GetNextURI(resp *http.Response, respErr error) (uri string, isRelocated bool) {
	defer func() {
		r.attemptCount++
	}()
	if r.attemptCount == 0 {
		// First attempt is always successful. Trigger the first retry so later calls have backoff
		// but ignore the returned value to ensure that the client can instrument the request even
		// if the context is done.
		r.retrier.Next()
		return r.removeMeshSchemeIfPresent(r.currentURI), false
	}
	if !r.attemptsRemaining() {
		// Retries exhausted
		return "", false
	}
	if r.isMeshURI(r.currentURI) {
		// Mesh uris don't get retried
		return "", false
	}
	retryFn := r.getRetryFn(resp, respErr)
	if retryFn == nil {
		// The previous response was not retryable
		return "", false
	}
	// Updates currentURI
	if !retryFn() {
		return "", false
	}
	return r.currentURI, r.isRelocatedURI(r.currentURI)
}

func (r *RequestRetrier) getRetryFn(resp *http.Response, respErr error) func() bool {
	errCode, _ := StatusCodeFromError(respErr)
	if retryOther, _ := isThrottleResponse(resp, errCode); retryOther {
		// 429: throttle
		// Immediately backoff and select the next URI.
		// TODO(whickman): use the retry-after header once #81 is resolved
		return r.nextURIAndBackoff
	} else if isUnavailableResponse(resp, errCode) {
		// 503: go to next node
		return r.nextURIOrBackoff
	} else if shouldTryOther, otherURI := isRetryOtherResponse(resp, respErr, errCode); shouldTryOther {
		// 307 or 308: go to next node, or particular node if provided.
		if otherURI != nil {
			return func() bool {
				r.setURIAndResetBackoff(otherURI)
				return true
			}
		}
		return r.nextURIOrBackoff
	} else if resp == nil {
		// if we get a nil response, we can assume there is a problem with host and can move on to the next.
		return r.nextURIOrBackoff
	}
	return nil
}

func (r *RequestRetrier) setURIAndResetBackoff(otherURI *url.URL) {
	// If the URI returned by relocation header is a relative path
	// We will resolve it with the current URI
	if !otherURI.IsAbs() {
		if currentURI := parseLocationURL(r.currentURI); currentURI != nil {
			otherURI = currentURI.ResolveReference(otherURI)
		}
	}
	nextURI := otherURI.String()
	r.relocatedURIs[otherURI.String()] = struct{}{}
	r.retrier.Reset()
	r.currentURI = nextURI
}

// If lastURI was already marked failed, we perform a backoff as determined by the retrier before returning the next URI and its offset.
// Otherwise, we add lastURI to failedURIs and return the next URI and its offset immediately.
func (r *RequestRetrier) nextURIOrBackoff() bool {
	_, performBackoff := r.failedURIs[r.currentURI]
	r.markFailedAndMoveToNextURI()
	// If the URI has failed before, perform a backoff
	if performBackoff || len(r.uris) == 1 {
		return r.retrier.Next()
	}
	return true
}

// Marks the current URI as failed, gets the next URI, and performs a backoff as determined by the retrier.
func (r *RequestRetrier) nextURIAndBackoff() bool {
	r.markFailedAndMoveToNextURI()
	return r.retrier.Next()
}

func (r *RequestRetrier) markFailedAndMoveToNextURI() {
	r.failedURIs[r.currentURI] = struct{}{}
	nextURIOffset := (r.offset + 1) % len(r.uris)
	nextURI := r.uris[nextURIOffset]
	r.currentURI = nextURI
	r.offset = nextURIOffset
}

func (r *RequestRetrier) removeMeshSchemeIfPresent(uri string) string {
	if r.isMeshURI(uri) {
		return strings.Replace(uri, meshSchemePrefix, "", 1)
	}
	return uri
}

func (r *RequestRetrier) isMeshURI(uri string) bool {
	return strings.HasPrefix(uri, meshSchemePrefix)
}

func (r *RequestRetrier) isRelocatedURI(uri string) bool {
	_, relocatedURI := r.relocatedURIs[uri]
	return relocatedURI
}
