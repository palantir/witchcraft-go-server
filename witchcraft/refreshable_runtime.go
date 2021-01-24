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

package witchcraft

import (
	"github.com/palantir/pkg/refreshable"
	"github.com/palantir/witchcraft-go-server/v2/config"
)

type refreshableBaseRuntimeConfig interface {
	refreshable.Refreshable
	CurrentBaseRuntimeConfig() config.Runtime
}

func newRefreshableBaseRuntimeConfig(in refreshable.Refreshable) refreshableBaseRuntimeConfig {
	return refreshableBaseRuntimeConfigImpl{
		Refreshable: in,
	}
}

type refreshableBaseRuntimeConfigImpl struct {
	refreshable.Refreshable
}

func (r refreshableBaseRuntimeConfigImpl) CurrentBaseRuntimeConfig() config.Runtime {
	return r.Current().(config.Runtime)
}
