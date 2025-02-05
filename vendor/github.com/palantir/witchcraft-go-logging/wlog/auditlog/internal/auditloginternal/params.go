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
	"github.com/palantir/witchcraft-go-logging/wlog"
)

type AuditParam struct {
	Audit2ParamFn func(entry wlog.LogEntry)
	Audit3ParamFn func(entry wlog.LogEntry)
}

func auditNameResultParam(name string, resultType AuditResultType) AuditParam {
	return AuditParam{
		Audit2ParamFn: func(entry wlog.LogEntry) {
			entry.StringValue(Audit2NameKey, name)
			entry.StringValue(Audit2ResultKey, string(resultType))
		},
		Audit3ParamFn: func(entry wlog.LogEntry) {
			entry.StringValue(Audit3NameKey, name)
			entry.StringValue(Audit3ResultKey, string(resultType))
		},
	}
}

func uidParam(uid string) AuditParam {
	return sameImplParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(wlog.UIDKey, uid)
	})
}

func sidParam(sid string) AuditParam {
	return sameImplParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(wlog.SIDKey, sid)
	})
}

func tokenIDParam(tokenID string) AuditParam {
	return sameImplParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(wlog.TokenIDKey, tokenID)
	})
}

func orgIDParam(orgID string) AuditParam {
	return sameImplParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(wlog.OrgIDKey, orgID)
	})
}

func traceIDParam(traceID string) AuditParam {
	return sameImplParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(wlog.TraceIDKey, traceID)
	})
}

func originParam(origin string) AuditParam {
	return AuditParam{
		Audit2ParamFn: func(entry wlog.LogEntry) {
			entry.OptionalStringValue(Audit2OriginKey, origin)
		},
		Audit3ParamFn: func(entry wlog.LogEntry) {
			entry.OptionalStringValue(Audit3OriginKey, origin)
		},
	}
}

func sameImplParamFn(fn func(entry wlog.LogEntry)) AuditParam {
	return AuditParam{
		Audit2ParamFn: fn,
		Audit3ParamFn: fn,
	}
}
