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
	"time"

	"github.com/palantir/pkg/metrics"
)

// ValidatedClientParams represents a set of fields derived from a snapshot of ClientConfig.
// It is designed for use within a refreshable: fields are comparable with reflect.DeepEqual
// so unnecessary updates are not pushed to subscribers.
// Values are generally known to be "valid" to minimize downstream error handling.
type ValidatedClientParams struct {
	APIToken       *string
	BasicAuth      *BasicAuth
	Dialer         DialerParams
	DisableMetrics bool
	MaxAttempts    *int
	MetricsTags    metrics.Tags
	Retry          RetryParams
	ServiceName    string
	Timeout        time.Duration
	Transport      TransportParams
	URIs           []string
}

// BasicAuth represents the configuration for HTTP Basic Authorization
type BasicAuth struct {
	User     string
	Password string
}
