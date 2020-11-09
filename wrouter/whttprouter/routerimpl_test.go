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

package whttprouter_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/palantir/witchcraft-go-server/v2/wrouter"
	"github.com/palantir/witchcraft-go-server/v2/wrouter/whttprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	handlerCalled := false
	router := whttprouter.New(whttprouter.RedirectTrailingSlash(false))
	router.Register(http.MethodGet, []wrouter.PathSegment{
		{
			Type:  wrouter.LiteralSegment,
			Value: "hello",
		},
	}, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		handlerCalled = true
	}))

	server := httptest.NewServer(router)
	defer server.Close()

	_, err := http.Get(server.URL + "/hello/")
	require.NoError(t, err)

	assert.False(t, handlerCalled)
}
