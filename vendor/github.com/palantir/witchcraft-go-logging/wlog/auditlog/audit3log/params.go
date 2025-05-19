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

package audit3log

import (
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/internal/auditloginternal"
)

const (
	TypeValue = auditloginternal.Audit3TypeValue

	NameKey       = auditloginternal.Audit3NameKey
	ResultKey     = auditloginternal.Audit3ResultKey
	LogEntryIDKey = auditloginternal.Audit3LogEntryIDKey
)

type Param interface {
	getParam() auditloginternal.AuditParam
}

var _ Param = (*paramStruct)(nil)

type paramStruct struct {
	param auditloginternal.Audit3Param
}

func (p *paramStruct) getParam() auditloginternal.AuditParam {
	return p.param.Param
}

func convertInternalParamToExportedParam(param auditloginternal.Audit3Param) Param {
	return &paramStruct{
		param: param,
	}
}

func convertExternalParamsToInternalParams(params []Param) []auditloginternal.Audit3Param {
	if params == nil {
		return nil
	}
	out := make([]auditloginternal.Audit3Param, len(params))
	for i := range out {
		out[i] = auditloginternal.Audit3Param{
			Param: params[i].getParam(),
		}
	}
	return out
}

func Deployment(deployment string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Deployment(deployment))
}

func Host(host string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Host(host))
}

func Product(product string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Product(product))
}

func ProductVersion(productVersion string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3ProductVersion(productVersion))
}

func Stack(stack string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Stack(stack))
}

func Service(service string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Service(service))
}

func Environment(environment string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Environment(environment))
}

func ProducerType(producerType AuditProducerType) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3ProducerType(auditloginternal.Audit3AuditProducerType(producerType)))
}

func Organizations(organizations []Organization) Param {
	var audit3Organizations []auditloginternal.Audit3Organization
	if len(organizations) > 0 {
		audit3Organizations = make([]auditloginternal.Audit3Organization, len(organizations))
		for i, org := range organizations {
			audit3Organizations[i] = auditloginternal.Audit3Organization(org)
		}
	}
	return convertInternalParamToExportedParam(auditloginternal.Audit3Organizations(audit3Organizations))
}

func EventID(eventID string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3EventID(eventID))
}

func LogEntryID(logEntryID string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3LogEntryID(logEntryID))
}

func UserAgent(userAgent string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3UserAgent(userAgent))
}

// unexported because this should only be set by the "Category" param
func categories(categories ...string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Categories(categories))
}

func Entities(entities []any) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Entities(entities))
}

func Users(users []ContextualizedUser) Param {
	var audit3ContextualizedUsers []auditloginternal.Audit3ContextualizedUser
	if len(users) > 0 {
		audit3ContextualizedUsers = make([]auditloginternal.Audit3ContextualizedUser, len(users))
		for i, user := range users {
			audit3ContextualizedUsers[i] = auditloginternal.Audit3ContextualizedUser(user)
		}
	}
	return convertInternalParamToExportedParam(auditloginternal.Audit3Users(audit3ContextualizedUsers))
}

func Origins(origins []string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Origins(origins))
}

func SourceOrigin(sourceOrigin string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3SourceOrigin(sourceOrigin))
}

func RequestField(key string, value interface{}) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3RequestField(key, value))
}

func RequestFields(requestFields map[string]any) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3RequestFields(requestFields))
}

func ResultField(key string, value interface{}) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3ResultField(key, value))
}

func ResultFields(resultFields map[string]any) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3ResultFields(resultFields))
}

func UID(uid string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3UID(uid))
}

func SID(sid string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3SID(sid))
}

func TokenID(tokenID string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3TokenID(tokenID))
}

func OrgID(orgID string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3OrgID(orgID))
}

func TraceID(traceID string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3TraceID(traceID))
}

func Origin(origin string) Param {
	return convertInternalParamToExportedParam(auditloginternal.Audit3Origin(origin))
}
