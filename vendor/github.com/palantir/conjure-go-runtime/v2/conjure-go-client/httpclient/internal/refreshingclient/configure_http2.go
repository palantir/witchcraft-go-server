// Copyright (c) 2021 Palantir Technologies. All rights reserved.
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

//go:build !go1.16
// +build !go1.16

package refreshingclient

import (
	"net/http"
	"time"

	werror "github.com/palantir/witchcraft-go-error"
	"golang.org/x/net/http2"
)

// configureHTTP2 will attempt to configure net/http HTTP/1 Transport to use HTTP/2.
// It returns an error if t1 has already been HTTP/2-enabled.
func configureHTTP2(t1 *http.Transport, _, _ time.Duration) error {
	if err := http2.ConfigureTransport(t1); err != nil {
		return werror.Wrap(err, "failed to configure transport for http2")
	}
	return nil
}
