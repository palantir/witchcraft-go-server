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

package categories

import (
	"fmt"
	"reflect"

	werror "github.com/palantir/witchcraft-go-error"
	commonv2 "github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/common/v2"
	"github.com/palantir/witchcraft-go-logging/conjure/witchcraft/api/logging"
)

// Converts the provided data to a slice of v2.Resource. Based on the Java "promote" method.
func toResource(data any) ([]commonv2.Resource, error) {
	switch val := data.(type) {
	case commonv2.Resource:
		return []commonv2.Resource{val}, nil
	case commonv2.ApplicationResource:
		return []commonv2.Resource{commonv2.NewResourceFromApplication(val)}, nil
	case commonv2.DataResource:
		return []commonv2.Resource{commonv2.NewResourceFromData(val)}, nil
	case commonv2.MonitorResource:
		return []commonv2.Resource{commonv2.NewResourceFromMonitor(val)}, nil
	case commonv2.SystemResource:
		return []commonv2.Resource{commonv2.NewResourceFromSystem(val)}, nil
	case commonv2.LogicResource:
		return []commonv2.Resource{commonv2.NewResourceFromLogic(val)}, nil
	case commonv2.RequestResource:
		return []commonv2.Resource{commonv2.NewResourceFromRequest(val)}, nil
	case commonv2.OntologyLogicResource:
		return []commonv2.Resource{commonv2.NewResourceFromOntologyLogic(val)}, nil
	case commonv2.OntologyDataResource:
		return []commonv2.Resource{commonv2.NewResourceFromOntologyData(val)}, nil
	case commonv2.OntologyDataResourceList:
		return []commonv2.Resource{commonv2.NewResourceFromOntologyDataList(val)}, nil
	case commonv2.OntologyMetaDataResource:
		return []commonv2.Resource{commonv2.NewResourceFromOntologyMetaData(val)}, nil
	case commonv2.Identifier:
		return []commonv2.Resource{commonv2.NewResourceFromExternal(val)}, nil
	case commonv2.ImportDestination:
		resources := []commonv2.Resource{commonv2.NewResourceFromData(val.Id)}
		if val.Parent != nil {
			resources = append(resources, commonv2.NewResourceFromData(*val.Parent))
		}
		return resources, nil
	}
	return nil, werror.Error("Cannot convert non-resource type to resource", werror.SafeParam("type", fmt.Sprintf("%T", data)))
}

// CheckAndExtractResources takes an input and checks if it is a resource or a collection of resources and, if so,
// returns the v2.Resource objects for the input. Returns an error if the provided input is not a recognized resource
// type (or a slice of recognized resource types).
//
// Based on the Java "public static Set<Resource> checkAndExtractResources(Object data)" function.
func CheckAndExtractResources(input any) ([]commonv2.Resource, error) {
	if input == nil {
		return nil, nil
	}

	// if input is a slice, treat it as a collection of resources
	if valViaReflection := reflect.ValueOf(input); valViaReflection.Kind() == reflect.Slice {
		var collectedResources []commonv2.Resource
		for i := 0; i < valViaReflection.Len(); i++ {
			sliceVal := valViaReflection.Index(i).Interface()

			resources, err := toResource(sliceVal)
			if err != nil {
				return nil, werror.Wrap(err, "Data is not a Resource in a collection of Resources")
			}
			collectedResources = append(collectedResources, resources...)
		}
		return collectedResources, nil
	}

	// otherwise, treat it as a single resource
	return toResource(input)
}

// CheckAndExtractUIDs takes an input and checks if it is a UserId or a slice of UserIds and, if so, returns a slice of
// UserIds that contains all the UserIds. Returns an error if the provided input is not a UserId or a slice of UserIds.
//
// Based on the Java "public static Set<UserId> checkAndExtractUids(Object data)" function.
func CheckAndExtractUIDs(input any) ([]logging.UserId, error) {
	var uids []logging.UserId

	// if input is a slice, treat it as a collection of resources
	if valViaReflection := reflect.ValueOf(input); valViaReflection.Kind() == reflect.Slice {
		for i := 0; i < valViaReflection.Len(); i++ {
			sliceVal := valViaReflection.Index(i).Interface()
			userID, err := toUserID(sliceVal)
			if err != nil {
				return nil, werror.Wrap(err, "Data is not a UserId in a collection of UserIds")
			}
			uids = append(uids, userID)
		}
	} else {
		userID, err := toUserID(input)
		if err != nil {
			return nil, err
		}
		uids = append(uids, userID)
	}

	return uids, nil
}

func toUserID(in any) (logging.UserId, error) {
	userID, ok := in.(logging.UserId)
	if !ok {
		return "", werror.Error("Cannot convert non-UserId type to UserId", werror.SafeParam("type", fmt.Sprintf("%T", in)))
	}
	return userID, nil
}

func ExtractUserIDsFromTokens(tokens []commonv2.Token) []logging.UserId {
	var userIDs []logging.UserId
	for _, token := range tokens {
		if token.UserId == nil {
			continue
		}
		userIDs = append(userIDs, logging.UserId(*token.UserId))
	}
	return userIDs
}
