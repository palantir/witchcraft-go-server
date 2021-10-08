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

package refreshingclient

import (
	"net/http"
	"time"

	"github.com/palantir/pkg/refreshable"
)

type RefreshableHTTPClient interface {
	refreshable.Refreshable
	CurrentHTTPClient() *http.Client
}

type refreshableHTTPClient struct {
	refreshable.Refreshable
}

func (r refreshableHTTPClient) CurrentHTTPClient() *http.Client {
	return r.Current().(*http.Client)
}

func NewRefreshableHTTPClient(rt http.RoundTripper, timeout refreshable.Duration) RefreshableHTTPClient {
	return refreshableHTTPClient{
		Refreshable: timeout.MapDuration(func(timeout time.Duration) interface{} {
			return &http.Client{
				Timeout:   timeout,
				Transport: rt,
			}
		}),
	}
}
