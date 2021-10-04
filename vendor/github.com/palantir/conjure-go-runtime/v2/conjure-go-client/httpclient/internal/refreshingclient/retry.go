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
	"context"
	"time"

	"github.com/palantir/pkg/retry"
)

type RetryParams struct {
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

// ConfigureRetry accepts a mapping function which will be applied to the params value as it is evaluated.
// This can be used to layer/overwrite configuration before building the RefreshableRetryParams.
func ConfigureRetry(r RefreshableRetryParams, mapFn func(p RetryParams) RetryParams) RefreshableRetryParams {
	return NewRefreshingRetryParams(r.MapRetryParams(func(params RetryParams) interface{} {
		return mapFn(params)
	}))
}

func (r RetryParams) Start(ctx context.Context) retry.Retrier {
	return retry.Start(ctx,
		retry.WithInitialBackoff(r.InitialBackoff),
		retry.WithMaxBackoff(r.MaxBackoff),
	)
}
