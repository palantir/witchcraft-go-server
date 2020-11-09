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

package rest_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/palantir/witchcraft-go-server/v2/rest"
	"github.com/stretchr/testify/require"
)

func TestWriteJSONResponse_Error(t *testing.T) {
	for _, test := range []struct {
		Name         string
		Err          error
		ExpectedCode int
		ExpectedJSON string
	}{
		{
			Name:         "standard werror",
			Err:          werror.Error("bad things happened"),
			ExpectedCode: 500,
			ExpectedJSON: "\"bad things happened\"\n",
		},
		{
			Name:         "rest.Error with code",
			Err:          rest.NewError(werror.Error("bad things happened"), rest.StatusCode(400)),
			ExpectedCode: 400,
			ExpectedJSON: "\"witchcraft-server rest error: bad things happened\"\n",
		},
	} {
		t.Run(test.Name, func(t *testing.T) {
			server := httptest.NewServer(rest.NewJSONHandler(func(http.ResponseWriter, *http.Request) error {
				return test.Err
			}, rest.StatusCodeMapper, nil))
			defer server.Close()

			resp, err := server.Client().Get(server.URL)
			require.NoError(t, err)
			require.Equal(t, resp.StatusCode, test.ExpectedCode)
			body, err := ioutil.ReadAll(resp.Body)
			require.NoError(t, err)
			require.Equal(t, test.ExpectedJSON, string(body))
		})
	}
}
