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
	"context"

	werror "github.com/palantir/witchcraft-go-error"
	v2 "github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/category/v2"
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	categoriespkg "github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log/internal/categories"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/internal/auditloginternal"
)

type categoryInfo[T any] struct {
	categoryName    string
	valueToCategory func(T) v2.AuditCategoryV2

	entityExtractor entityExtractor[T]
	uidExtractor    uidExtractor[T]

	requestFields  map[string]categoryField[T]
	responseFields map[string]categoryField[T]
}

type categoryField[T any] struct {
	extractField   func(T) any
	classification categoriespkg.Classification
	isRequired     bool
}

func newCategoryInfo[T any](categoryName string, valueToCategory func(T) v2.AuditCategoryV2) *categoryInfo[T] {
	return &categoryInfo[T]{
		categoryName:    categoryName,
		valueToCategory: valueToCategory,
		requestFields:   make(map[string]categoryField[T]),
		responseFields:  make(map[string]categoryField[T]),
	}
}

func (c *categoryInfo[T]) requiredRequest(name string, classification categoriespkg.Classification, valueFn func(T) any) *categoryInfo[T] {
	setFieldHelper(c.requestFields, name, classification, valueFn, true)
	return c
}

func (c *categoryInfo[T]) optionalRequest(name string, classification categoriespkg.Classification, valueFn func(T) any) *categoryInfo[T] {
	setFieldHelper(c.requestFields, name, classification, valueFn, false)
	return c
}

func (c *categoryInfo[T]) requiredResponse(name string, classification categoriespkg.Classification, valueFn func(T) any) *categoryInfo[T] {
	setFieldHelper(c.responseFields, name, classification, valueFn, true)
	return c
}

func (c *categoryInfo[T]) optionalResponse(name string, classification categoriespkg.Classification, valueFn func(T) any) *categoryInfo[T] {
	setFieldHelper(c.responseFields, name, classification, valueFn, false)
	return c
}

func setFieldHelper[T any](m map[string]categoryField[T], name string, classification categoriespkg.Classification, valueFn func(T) any, isRequired bool) {
	m[name] = categoryField[T]{
		extractField:   valueFn,
		classification: classification,
		isRequired:     isRequired,
	}
}

func (c *categoryInfo[T]) toParam(v T) Param {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", c.valueToCategory(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories(c.categoryName).getParam().Audit3ParamFn(entry)

					for k, fieldVal := range c.requestFields {
						RequestField(k, fieldVal.extractField(v)).getParam().Audit3ParamFn(entry)
					}
					for k, fieldVal := range c.responseFields {
						ResultField(k, fieldVal.extractField(v)).getParam().Audit3ParamFn(entry)
					}

					// add all extracted entities
					entityExtractorFn := c.entityExtractor
					if entityExtractorFn == nil {
						entityExtractorFn = defaultEntityExtractor[T]
					}
					if entities, _ := entityExtractorFn(v, c); len(entities) > 0 {
						anyEntriesSlice := make([]any, len(entities))
						for i, entity := range entities {
							anyEntriesSlice[i] = entity
						}
						Entities(anyEntriesSlice).getParam().Audit3ParamFn(entry)
					}

					// add all extracted user IDs
					userIDExtractorFn := c.uidExtractor
					if userIDExtractorFn == nil {
						userIDExtractorFn = defaultUIDExtractor[T]
					}
					if userIDs, _ := userIDExtractorFn(v, c); len(userIDs) > 0 {
						var users []ContextualizedUser
						for _, userID := range userIDs {
							users = append(users, ContextualizedUser{
								UID: string(userID),
							})
						}
						Users(users).getParam().Audit3ParamFn(entry)
					}
				},
			},
		},
	}
}

func Category(category v2.AuditCategoryV2) Param {
	categoryV2WithT := (v2.AuditCategoryV2WithT[Param])(category)
	param, _ := categoryV2WithT.Accept(context.TODO(), &auditCategoryV2Visitor{})
	return param
}

var _ v2.AuditCategoryV2VisitorWithT[Param] = (*auditCategoryV2Visitor)(nil)

type auditCategoryV2Visitor struct {
}

func (a *auditCategoryV2Visitor) VisitUnknown(ctx context.Context, typ string) (Param, error) {
	return nil, werror.Error("unhandled type", werror.SafeParam("type", typ))
}
