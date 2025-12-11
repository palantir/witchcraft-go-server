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
	v2 "github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category/v2"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft-logging-api/witchcraft/api/logging"
	categoriespkg "github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit3log/internal/categories"
)

// entityExtractor is a function that extracts entities from a value of type T.
type uidExtractor[T any] func(val T, cat *categoryInfo[T]) ([]logging.UserId, error)

var _ uidExtractor[any] = defaultUIDExtractor[any]

func init() {
	categoryManagementTokens.uidExtractor = func(val v2.ManagementTokens, cat *categoryInfo[v2.ManagementTokens]) ([]logging.UserId, error) {
		return categoriespkg.ExtractUserIDsFromTokens(val.ManagedTokens), nil
	}
	categoryPassThrough.uidExtractor = func(val v2.PassThrough, cat *categoryInfo[v2.PassThrough]) ([]logging.UserId, error) {
		return nil, nil
	}
	categoryTokenAccess.uidExtractor = func(val v2.TokenAccess, cat *categoryInfo[v2.TokenAccess]) ([]logging.UserId, error) {
		return categoriespkg.ExtractUserIDsFromTokens(val.AccessedTokens), nil
	}
	categoryTokenGeneration.uidExtractor = func(val v2.TokenGeneration, cat *categoryInfo[v2.TokenGeneration]) ([]logging.UserId, error) {
		return categoriespkg.ExtractUserIDsFromTokens(val.GeneratedTokens), nil
	}
	categoryTokenRevoke.uidExtractor = func(val v2.TokenRevoke, cat *categoryInfo[v2.TokenRevoke]) ([]logging.UserId, error) {
		return categoriespkg.ExtractUserIDsFromTokens(val.RevokedTokens), nil
	}
}

func defaultUIDExtractor[T any](val T, cat *categoryInfo[T]) ([]logging.UserId, error) {
	var allUserIDs []logging.UserId
	extractFn := func(fieldVal categoryField[T]) error {
		if fieldVal.classification != categoriespkg.Classification_UID {
			return nil
		}
		resources, err := categoriespkg.CheckAndExtractUIDs(fieldVal.extractField(val))
		if err != nil {
			return err
		}
		allUserIDs = append(allUserIDs, resources...)
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

	return allUserIDs, nil
}
