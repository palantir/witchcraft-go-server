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
	"github.com/palantir/pkg/rid"
	commonv2 "github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/common/v2"
	categoriespkg "github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log/internal/categories"
)

// entityExtractor is a function that extracts entities from a value of type T.
type entityExtractor[T any] func(val T, cat *categoryInfo[T]) ([]rid.ResourceIdentifier, error)

var _ entityExtractor[any] = defaultEntityExtractor[any]

func defaultEntityExtractor[T any](val T, cat *categoryInfo[T]) ([]rid.ResourceIdentifier, error) {
	var allResources []commonv2.Resource
	extractFn := func(fieldVal categoryField[T]) error {
		if fieldVal.classification != categoriespkg.Classification_RESOURCE {
			return nil
		}
		resources, err := categoriespkg.CheckAndExtractResources(fieldVal.extractField(val))
		if err != nil {
			return err
		}
		allResources = append(allResources, resources...)
		return nil
	}

	for _, fieldVal := range cat.requestFields {
		if err := extractFn(fieldVal); err != nil {
			return nil, err
		}
	}
	for _, fieldVal := range cat.responseFields {
		if err := extractFn(fieldVal); err != nil {
			return nil, err
		}
	}

	entities, err := categoriespkg.EntitiesFromResources(allResources...)
	if err != nil {
		return nil, err
	}
	return entities, nil
}
