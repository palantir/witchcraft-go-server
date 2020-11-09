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

package reporter

import (
	"github.com/palantir/witchcraft-go-server/v2/status"
)

type ComponentName string

// Component represents an individual readiness check that's owned by the ReadinessReporter.
type Component interface {
	status.Source
	SetStatus(respStatus int, metadata interface{})
}

type readinessComponent struct {
	name     ComponentName
	status   int
	metadata interface{}
}

func (r *readinessComponent) Status() (respStatus int, metadata interface{}) {
	return r.status, r.metadata
}

func (r *readinessComponent) SetStatus(respStatus int, metadata interface{}) {
	r.status = respStatus
	r.metadata = metadata
}
