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

package rest

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	werror "github.com/palantir/witchcraft-go-error"
	"github.com/stretchr/testify/assert"
)

func TestNewErrorOneWrap(t *testing.T) {
	werr := werror.ErrorWithContextParams(context.Background(), "Test error", werror.SafeParam("param1", "val1"),
		werror.UnsafeParam("param2", "val2"))

	restErr := NewError(context.Background(), werr, StatusCode(400))
	errString := fmt.Sprintf("%+v", restErr)
	assert.Contains(t, errString, "Test error")
	assert.Equal(t, map[string]interface{}{
		"param1":               "val1",
		"param2":               "val2",
		httpStatusCodeParamKey: 400,
	}, mergeMaps(werror.ParamsFromError(restErr)))
}

func TestNewErrorTwoWrap(t *testing.T) {
	werr1 := werror.ErrorWithContextParams(context.Background(), "werr1", werror.SafeParam("param1", "val1"))
	werr2 := werror.WrapWithContextParams(context.Background(), werr1, "werr2", werror.SafeParam("param2", "val2"))
	restErr := NewError(context.Background(), werr2, StatusCode(http.StatusBadRequest))
	errString := fmt.Sprintf("%+v", restErr)
	assert.Contains(t, errString, "werr1")
	assert.Contains(t, errString, "werr2")
	assert.Equal(t, map[string]interface{}{
		"param1":               "val1",
		"param2":               "val2",
		httpStatusCodeParamKey: 400,
	}, mergeMaps(werror.ParamsFromError(restErr)))
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	ret := map[string]interface{}{}
	for k, v := range a {
		ret[k] = v
	}
	for k, v := range b {
		ret[k] = v
	}
	return ret
}
