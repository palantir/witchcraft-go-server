// Copyright (c) 2025 Palantir Technologies. All rights reserved.
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

package auditloginternal

import (
	"sync"
	"time"

	"github.com/palantir/witchcraft-go-logging/wlog"
)

type AuditResultType string

const (
	AuditResultSuccess      AuditResultType = "SUCCESS"
	AuditResultUnauthorized AuditResultType = "UNAUTHORIZED"
	AuditResultError        AuditResultType = "ERROR"
)

type logger struct {
	audit2Logger wlog.Logger
	audit3Logger wlog.Logger
}

// currentTimeProvider is a struct that allows for the current time to be computed once and then returned.
// Exists to support dual-writing audit v2 and v3 logs: when dual-writing, it is desirable for both the v2 and v3 logs
// to have the exact same time value. However, the instant that the time is computed should still be when it is first
// written to the primary log (v2 or v3). This struct exists so that the time value is computed when the logging first
// occurs, but subsequent invocations return the same value.
type currentTimeProvider struct {
	timeValue string
	once      sync.Once
}

func (t *currentTimeProvider) getTimeValue() string {
	t.once.Do(func() {
		t.timeValue = time.Now().Format(time.RFC3339Nano)
	})
	return t.timeValue
}

func (l *logger) Audit2(name string, result AuditResultType, params ...Audit2Param) {
	timeProvider := &currentTimeProvider{}

	l.audit2Logger.Log(Audit2ToParams(timeProvider, name, result, params)...)

	// dual-log if audit3Logger specified
	if l.audit3Logger != nil {
		audit3Params := make([]Audit3Param, len(params))
		for i, param := range params {
			audit3Params[i] = Audit3Param{
				Param: param.Param,
			}
		}
		l.audit3Logger.Log(Audit3ToParams(timeProvider, name, result, audit3Params)...)
	}
}

func (l *logger) Audit3(name string, result AuditResultType, params ...Audit3Param) {
	timeProvider := &currentTimeProvider{}

	l.audit3Logger.Log(Audit3ToParams(timeProvider, name, result, params)...)

	// dual-log if audit2Logger specified
	if l.audit2Logger != nil {
		audit2Params := make([]Audit2Param, len(params))
		for i, param := range params {
			audit2Params[i] = Audit2Param{
				Param: param.Param,
			}
		}
		l.audit2Logger.Log(Audit2ToParams(timeProvider, name, result, audit2Params)...)
	}
}
