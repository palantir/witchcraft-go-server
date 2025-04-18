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
	"time"

	"github.com/palantir/pkg/uuid"
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

// atLogTimeValues stores values that are computed once when first logged. Exists to support dual-writing audit v2 and
// v3 logs: when dual-writing, there are certain fields that should be computed when they are first logged, but should
// then be logged with same value when written to the other logger -- for example, "time". This struct exists so that
// such values can be computed when the logging first occurs, but subsequent invocations return the same value.
//
// Note that this structure is not safe to be used concurrently. Although it is straightforward to make it safe, it is
// on a possible hot path (audit logging) and is an internal struct that is currently only used in a non-concurrent
// manner, at this point the performance cost of making it safe does not seem to be worth the benefit.
type atLogTimeValues struct {
	timeValue       *string
	logEntryIDValue *string
}

func (v *atLogTimeValues) getOrComputeTimeValue() string {
	if v.timeValue == nil {
		value := time.Now().Format(time.RFC3339Nano)
		v.timeValue = &value
	}
	return *v.timeValue
}

func (v *atLogTimeValues) getOrComputeLogEntryIDValue() string {
	if v.logEntryIDValue == nil {
		value := uuid.NewUUID().String()
		v.logEntryIDValue = &value
	}
	return *v.logEntryIDValue
}

func (v *atLogTimeValues) getLogEntryIDValue() *string {
	return v.logEntryIDValue
}

func (l *logger) Audit2(name string, result AuditResultType, params ...Audit2Param) {
	atLogTimeValuesProvider := &atLogTimeValues{}

	l.audit2Logger.Log(Audit2ToParams(atLogTimeValuesProvider, name, result, params)...)

	// dual-log if audit3Logger specified
	if l.audit3Logger != nil {
		audit3Params := make([]Audit3Param, len(params))
		for i, param := range params {
			audit3Params[i] = Audit3Param{
				Param: param.Param,
			}
		}
		l.audit3Logger.Log(Audit3ToParams(atLogTimeValuesProvider, name, result, audit3Params)...)
	}
}

func (l *logger) Audit3(name string, result AuditResultType, params ...Audit3Param) {
	atLogTimeValuesProvider := &atLogTimeValues{}

	l.audit3Logger.Log(Audit3ToParams(atLogTimeValuesProvider, name, result, params)...)

	// dual-log if audit2Logger specified
	if l.audit2Logger != nil {
		audit2Params := make([]Audit2Param, len(params))
		for i, param := range params {
			audit2Params[i] = Audit2Param{
				Param: param.Param,
			}
		}
		l.audit2Logger.Log(Audit2ToParams(atLogTimeValuesProvider, name, result, audit2Params)...)
	}
}
