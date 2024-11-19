// Copyright (c) 2024 Palantir Technologies. All rights reserved.
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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/palantir/conjure-go-runtime/v2/conjure-go-contract/codecs"
)

// RequestBody is an interface that can be used to set the body of an http.Request.
// Implementations of this interface should set the fields Body, GetBody, and ContentLength (if known).
type RequestBody interface {
	setRequestBody(req *http.Request) error
}

// requestBodyFunc is a function that returns the length, body reader, and
// getBody function that should be set on an http.Request. It implements the
// RequestBody interface by setting the "ContentLength", "Body", and "GetBody"
// fields of the provided *http.Request to the values returned by the function.
type requestBodyFunc func() (length int64, body io.ReadCloser, getBody func() (io.ReadCloser, error), err error)

func (f requestBodyFunc) setRequestBody(req *http.Request) (err error) {
	req.ContentLength, req.Body, req.GetBody, err = f()
	return err
}

// RequestBodyEmpty sets the *http.Request Body field to nil for upload.
func RequestBodyEmpty() RequestBody {
	return requestBodyFunc(func() (int64, io.ReadCloser, func() (io.ReadCloser, error), error) {
		return 0, http.NoBody, nil, nil
	})
}

// RequestBodyInMemory sets the *http.Request Body field to the provided *bytes.Buffer, *bytes.Reader, or *strings.Reader for upload.
// The GetBody field is set to a function that returns the same io.ReadCloser.
func RequestBodyInMemory[T bytes.Buffer | bytes.Reader | strings.Reader](input *T) RequestBody {
	return requestBodyFunc(func() (int64, io.ReadCloser, func() (io.ReadCloser, error), error) {
		if input == nil {
			return 0, nil, nil, nil
		}
		contentLen := contentLengthInMemory(input)
		snapshot := *input
		getBody := func() (io.ReadCloser, error) {
			r := snapshot
			return io.NopCloser(any(&r).(io.Reader)), nil
		}
		firstBody, _ := getBody()
		return contentLen, firstBody, getBody, nil
	})
}

// contentLengthInMemory returns the length of the provided *bytes.Buffer, *bytes.Reader, or *strings.Reader.
func contentLengthInMemory[T bytes.Buffer | bytes.Reader | strings.Reader](input *T) int64 {
	if input == nil {
		return 0
	}

	// lenInterface is implemented by all three buffer variants' pointer types.
	type lenInterface interface{ Len() int }

	return int64(any(input).(lenInterface).Len())
}

// noRetriesRequestBodyFunc is a marker type to indicate the body can only be used once.
type noRetriesRequestBody struct {
	requestBodyFunc
}

// requestBodyStreamInput is a generic constraint for functions that return a reader with optional content length and error.
type requestBodyStreamInput interface {
	func() io.ReadCloser | func() (io.ReadCloser, error) | func() (io.ReadCloser, int64, error)
}

// RequestBodyStreamOnce sets the *http.Request Body field for upload.
//
// The GetBody field is left nil. The http.Transport will return an error if it is unable to replay the request body.
// Use this function when the body should be read only once (e.g. it is a response body stream from another request).
//
// The body's Close() method will be called when the request is completed. To disable, wrap the value in io.NopCloser.
//
// The input argument must be a function of one of the following types:
//   - func() io.ReadCloser                 // Returns the body and no error
//   - func() (io.ReadCloser, error)        // Returns the body and an error
//   - func() (io.ReadCloser, int64, error) // Returns the body, content length, and an error
func RequestBodyStreamOnce[T requestBodyStreamInput](input T) RequestBody {
	return noRetriesRequestBody{requestBodyFunc: func() (contentLen int64, body io.ReadCloser, getBody func() (io.ReadCloser, error), err error) {
		switch v := any(input).(type) {
		default:
			// Cases below MUST be exhaustive of the generic type!
			return 0, nil, nil, fmt.Errorf("httpclient.RequestBodyStreamOnce: unexpected type %T", v)
		case nil:
			return 0, nil, nil, nil
		case func() io.ReadCloser:
			return -1, v(), nil, nil
		case func() (io.ReadCloser, error):
			body, err = v()
			return -1, body, nil, err
		case func() (io.ReadCloser, int64, error):
			body, contentLen, err = v()
			return contentLen, body, nil, err
		}
	}}
}

// RequestBodyStreamWithReplay sets the *http.Request Body and GetBody fields for upload.
//
// The GetBody field is set to a function that returns the same io.ReadCloser. The http.Transport will be able to replay the request body
// when a request is redirected. Use this function when the body reader can be recreated multiple times.
//
// The body's Close() method will be called when the request is completed. To disable, wrap the value in io.NopCloser.
//
// The input argument must be a function of one of the following types:
//   - func() io.ReadCloser                 // Returns the body and no error
//   - func() (io.ReadCloser, error)        // Returns the body and an error
//   - func() (io.ReadCloser, int64, error) // Returns the body, content length, and an error
func RequestBodyStreamWithReplay[T requestBodyStreamInput](input T) RequestBody {
	return requestBodyFunc(func() (contentLen int64, body io.ReadCloser, getBody func() (io.ReadCloser, error), err error) {
		switch v := any(input).(type) {
		default:
			// Cases below MUST be exhaustive of the generic type!
			return 0, nil, nil, fmt.Errorf("httpclient.RequestBodyStreamWithReplay: unexpected type %T", v)
		case nil:
			return 0, nil, nil, nil
		case func() io.ReadCloser:
			body = v()
			getBody = func() (io.ReadCloser, error) {
				return v(), nil
			}
			return -1, body, getBody, nil
		case func() (io.ReadCloser, error):
			body, err = v()
			return -1, body, v, err
		case func() (io.ReadCloser, int64, error):
			getBody = func() (io.ReadCloser, error) {
				b, _, e := v()
				return b, e
			}
			body, contentLen, err = v()
			return contentLen, body, getBody, err
		}
	})
}

// RequestBodyEncoderObject sets the *http.Request Body field for upload using the provided encoder.
func RequestBodyEncoderObject(input any, encoder codecs.Encoder) RequestBody {
	return requestBodyFunc(func() (contentLen int64, body io.ReadCloser, getBody func() (io.ReadCloser, error), err error) {
		raw, err := encoder.Marshal(input)
		if err != nil {
			return 0, nil, nil, err
		}
		getBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(raw)), nil
		}
		body, _ = getBody()
		return int64(len(raw)), body, getBody, nil
	})
}

// RequestBodyEncoderObjectBuffer is like RequestBodyEncoderObject but writes the encoded object to the provided buffer.
func RequestBodyEncoderObjectBuffer(input any, encoder codecs.Encoder, buffer *bytes.Buffer) RequestBody {
	return requestBodyFunc(func() (contentLen int64, body io.ReadCloser, getBody func() (io.ReadCloser, error), err error) {
		if err := encoder.Encode(buffer, input); err != nil {
			return 0, nil, nil, err
		}
		raw := buffer.Bytes()

		getBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(raw)), nil
		}
		body, _ = getBody()
		return int64(len(raw)), body, getBody, nil
	})
}
