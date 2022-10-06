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

package internal

import (
	"context"
	"io"
	"net/http"

	"github.com/palantir/witchcraft-go-logging/wlog/svclog/svc1log"
)

// DrainBody reads then closes a response's body if it is non-nil.
// This function should be deferred before a response reference is
// discarded.
func DrainBody(ctx context.Context, resp *http.Response) {
	// drain and close treated as best-effort
	if resp != nil && resp.Body != nil {
		if bytes, err := io.Copy(io.Discard, resp.Body); err != nil {
			svc1log.FromContext(ctx).Warn("Failed to drain entire response body",
				svc1log.SafeParam("bytes", bytes),
				svc1log.Stacktrace(err))
		} else if bytes > 0 {
			svc1log.FromContext(ctx).Info("Drained remaining response body",
				svc1log.SafeParam("bytes", bytes))
		}

		if err := resp.Body.Close(); err != nil {
			svc1log.FromContext(ctx).Warn("Failed to close response body",
				svc1log.Stacktrace(err))
		}
	}
}
