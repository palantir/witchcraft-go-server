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

//+build go1.8

package negroni

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type pusherRecorder struct {
	*httptest.ResponseRecorder
	pushed bool
}

func newPusherRecorder() *pusherRecorder {
	return &pusherRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (c *pusherRecorder) Push(target string, opts *http.PushOptions) error {
	c.pushed = true
	return nil
}

func TestResponseWriterPush(t *testing.T) {
	pushable := newPusherRecorder()
	rw := NewResponseWriter(pushable)
	pusher, ok := rw.(http.Pusher)
	expect(t, ok, true)
	err := pusher.Push("", nil)
	if err != nil {
		t.Error(err)
	}
	expect(t, pushable.pushed, true)
}
