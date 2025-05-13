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

const (
	Audit3TypeValue = "audit.3"

	Audit3DeploymentKey     = "deployment"
	Audit3HostKey           = "host"
	Audit3ProductKey        = "product"
	Audit3ProductVersionKey = "productVersion"
	Audit3StackKey          = "stack"
	Audit3ServiceKey        = "service"
	Audit3EnvironmentKey    = "environment"
	Audit3ProducerTypeKey   = "producerType"
	Audit3OrganizationsKey  = "organizations"
	Audit3EventIDKey        = "eventId"
	Audit3LogEntryIDKey     = "logEntryId"
	Audit3UserAgentKey      = "userAgent"
	Audit3CategoriesKey     = "categories"
	Audit3EntitiesKey       = "entities"
	Audit3UsersKey          = "users"
	Audit3OriginsKey        = "origins"
	Audit3SourceOriginKey   = "sourceOrigin"
	Audit3RequestFieldsKey  = "requestFields"
	Audit3ResultFieldsKey   = "resultFields"
	Audit3OriginKey         = "origin"
	Audit3NameKey           = "name"
	Audit3ResultKey         = "result"
)

// Audit3Param is effectively a marker type that wraps AuditParam.
// The AuditParam interface requires specifying outputs for both audit.v2 and audit.v3, but the Audit3Param type should
// only be used for parameters that are "native" to audit.v3.
type Audit3Param struct {
	Param AuditParam
}

func Audit3ToParams(atLogTimeValuesProvider *atLogTimeValues, name string, result AuditResultType, inParams []Audit3Param) []wlog.Param {
	if atLogTimeValuesProvider == nil {
		atLogTimeValuesProvider = &atLogTimeValues{}
	}
	outParams := make([]wlog.Param, 1+len(inParams))
	outParams[0] = wlog.NewParam(func(entry wlog.LogEntry) {
		entry.StringValue(wlog.TypeKey, Audit3TypeValue)
		entry.StringValue(wlog.TimeKey, atLogTimeValuesProvider.getOrComputeTimeValue())
		auditNameResultParam(name, result).Audit2ParamFn(entry)
		Audit3LogEntryID(atLogTimeValuesProvider.getOrComputeLogEntryIDValue()).Param.Audit3ParamFn(entry)
	})
	for idx := range inParams {
		outParams[1+idx] = wlog.NewParam(inParams[idx].Param.Audit3ParamFn)
	}
	return outParams
}

func Audit3Deployment(deployment string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.StringValue(Audit3DeploymentKey, deployment)
	})
}

func Audit3Host(host string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.StringValue(Audit3HostKey, host)
	})
}

func Audit3Product(product string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.StringValue(Audit3ProductKey, product)
	})
}

func Audit3ProductVersion(productVersion string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.StringValue(Audit3ProductVersionKey, productVersion)
	})
}

func Audit3Stack(stack string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(Audit3StackKey, stack)
	})
}

func Audit3Service(service string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(Audit3ServiceKey, service)
	})
}

func Audit3Environment(environment string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.OptionalStringValue(Audit3EnvironmentKey, environment)
	})
}

func Audit3ProducerType(producerType Audit3AuditProducerType) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.StringValue(Audit3ProducerTypeKey, string(producerType))
	})
}

func Audit3Organizations(organizations []Audit3Organization) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.ObjectListAppendValue(Audit3OrganizationsKey, toAnySlice(organizations))
	})
}

func Audit3EventID(eventID string) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				Audit2RequestParam("_auditEventId", eventID).Param.Audit2ParamFn(entry)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				entry.StringValue(Audit3EventIDKey, eventID)
			},
		},
	}
}

func Audit3LogEntryID(logEntryID string) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				Audit2RequestParam("_auditLogEntryId", logEntryID).Param.Audit2ParamFn(entry)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				entry.StringValue(Audit3LogEntryIDKey, logEntryID)
			},
		},
	}
}

func Audit3UserAgent(userAgent string) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				Audit2RequestParam("_userAgent", userAgent).Param.Audit2ParamFn(entry)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				entry.OptionalStringValue(Audit3UserAgentKey, userAgent)
			},
		},
	}
}

func Audit3Categories(categories []string) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.StringListAppendValue(Audit3CategoriesKey, categories)
	})
}

func Audit3Entities(entities []any) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.ObjectListAppendValue(Audit3EntitiesKey, entities)
	})
}

func Audit3Users(users []Audit3ContextualizedUser) Audit3Param {
	return audit3OnlyParamFn(func(entry wlog.LogEntry) {
		entry.ObjectListAppendValue(Audit3UsersKey, toAnySlice(users))
	})
}

func Audit3Origins(origins []string) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				Audit2RequestParam("_forwardedOrigins", origins).Param.Audit2ParamFn(entry)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				entry.StringListAppendValue(Audit3OriginsKey, origins)
			},
		},
	}
}

func Audit3SourceOrigin(sourceOrigin string) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				Audit2RequestParam("_sourceOrigin", sourceOrigin).Param.Audit2ParamFn(entry)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				entry.OptionalStringValue(Audit3SourceOriginKey, sourceOrigin)
			},
		},
	}
}

func Audit3RequestField(key string, value interface{}) Audit3Param {
	return Audit3RequestFields(map[string]interface{}{
		key: value,
	})
}

func Audit3RequestFields(requestFields map[string]any) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				entry.AnyMapValue(Audit2RequestParamsKey, requestFields)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				entry.AnyMapValue(Audit3RequestFieldsKey, requestFields)
			},
		},
	}
}

func Audit3ResultField(key string, value interface{}) Audit3Param {
	return Audit3ResultFields(map[string]interface{}{
		key: value,
	})
}

func Audit3ResultFields(resultFields map[string]any) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				entry.AnyMapValue(Audit2ResultParamsKey, resultFields)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				entry.AnyMapValue(Audit3ResultFieldsKey, resultFields)
			},
		},
	}
}

func Audit3UID(uid string) Audit3Param {
	return Audit3Param{
		Param: uidParam(uid),
	}
}

func Audit3SID(sid string) Audit3Param {
	return Audit3Param{
		Param: sidParam(sid),
	}
}

func Audit3TokenID(tokenID string) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				tokenIDParam(tokenID).Audit2ParamFn(entry)
				Audit2RequestParam("_tokenId", tokenID).Param.Audit2ParamFn(entry)
			},
			Audit3ParamFn: func(entry wlog.LogEntry) {
				tokenIDParam(tokenID).Audit3ParamFn(entry)
			},
		},
	}
}

func Audit3OrgID(orgID string) Audit3Param {
	return Audit3Param{
		Param: orgIDParam(orgID),
	}
}

func Audit3TraceID(traceID string) Audit3Param {
	return Audit3Param{
		Param: traceIDParam(traceID),
	}
}

func Audit3Origin(origin string) Audit3Param {
	return Audit3Param{
		Param: originParam(origin),
	}
}

func audit3OnlyParamFn(fn func(entry wlog.LogEntry)) Audit3Param {
	return Audit3Param{
		Param: AuditParam{
			Audit2ParamFn: func(entry wlog.LogEntry) {
				// do nothing: no equivalent field in audit.v2
			},
			Audit3ParamFn: fn,
		},
	}
}

func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}
