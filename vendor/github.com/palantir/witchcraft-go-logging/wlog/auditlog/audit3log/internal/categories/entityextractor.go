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
	"github.com/palantir/pkg/rid"
	commonv2 "github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/common/v2"
)

func EntitiesFromResources(resource ...commonv2.Resource) ([]rid.ResourceIdentifier, error) {
	var ids []rid.ResourceIdentifier
	for _, r := range resource {
		rids, err := entitiesFromResource(r)
		if err != nil {
			return nil, err
		}
		ids = append(ids, rids...)
	}
	return ids, nil
}

func entitiesFromResource(resource commonv2.Resource) ([]rid.ResourceIdentifier, error) {
	var ids []rid.ResourceIdentifier

	ridsFromIDFn := func(id commonv2.Identifier) error {
		rids, err := ridsFromIdentifier(id)
		if err != nil {
			return err
		}
		ids = append(ids, rids...)
		return nil
	}
	ridsFn := func(rids []rid.ResourceIdentifier, err error) error {
		if err != nil {
			return err
		}
		ids = append(ids, rids...)
		return nil
	}

	if err := resource.AcceptFuncs(
		func(v commonv2.ApplicationResource) error {
			return ridsFromIDFn(v.Id)
		},
		func(v commonv2.DataResource) error {
			return ridsFromIDFn(v.Id)
		},
		func(v commonv2.MonitorResource) error {
			return ridsFromIDFn(v.Id)
		},
		resource.SystemNoopSuccess,
		func(v commonv2.LogicResource) error {
			return ridsFromIDFn(v.Id)
		},
		func(v commonv2.RequestResource) error {
			return ridsFromIDFn(v.Id)
		},
		func(v commonv2.OntologyDataResource) error {
			return ridsFn(ridsFromOntologyDataResource(v))
		},
		func(v commonv2.OntologyDataResourceList) error {
			return ridsFn(ridsFromOntologyDataResourceList(v))
		},
		func(v commonv2.OntologyLogicResource) error {
			return ridsFn(ridsFromOntologyLogicResource(v))
		},
		func(v commonv2.OntologyMetaDataResource) error {
			return ridsFromIDFn(v.Id)
		},
		func(v commonv2.Identifier) error {
			return ridsFromIDFn(v)
		},
		resource.ErrorOnUnknown,
	); err != nil {
		return nil, err
	}
	return ids, nil
}

func ridsFromIdentifier(identifier commonv2.Identifier) ([]rid.ResourceIdentifier, error) {
	var ids []rid.ResourceIdentifier
	if err := identifier.AcceptFuncs(
		func(v rid.ResourceIdentifier) error {
			ids = append(ids, v)
			return nil
		},
		func(v []rid.ResourceIdentifier) error {
			ids = append(ids, v...)
			return nil
		},
		identifier.OtherNoopSuccess,
		identifier.OthersNoopSuccess,
		identifier.ErrorOnUnknown,
	); err != nil {
		return nil, err
	}
	return ids, nil
}

func ridsFromOntologyDataResource(resource commonv2.OntologyDataResource) ([]rid.ResourceIdentifier, error) {
	var ids []rid.ResourceIdentifier
	for _, k := range resource.ObjectPrimaryKey {
		rids, err := ridsFromObjectPropertyIdentifier(k.PropertyIdentifier)
		if err != nil {
			return nil, err
		}
		ids = append(ids, rids...)
	}
	if resource.AdditionalObjectProperties != nil {
		for _, prop := range *resource.AdditionalObjectProperties {
			rids, err := ridsFromObjectPropertyIdentifier(prop.PropertyIdentifier)
			if err != nil {
				return nil, err
			}
			ids = append(ids, rids...)
		}
	}
	if resource.OntologyContext != nil {
		rids, err := ridsFromOntologyContext(*resource.OntologyContext)
		if err != nil {
			return nil, err
		}
		ids = append(ids, rids...)
	}
	return ids, nil
}

func ridsFromObjectPropertyIdentifier(identifier commonv2.ObjectPropertyIdentifier) ([]rid.ResourceIdentifier, error) {
	var ids []rid.ResourceIdentifier
	if err := identifier.AcceptFuncs(
		func(v rid.ResourceIdentifier) error {
			ids = append(ids, v)
			return nil
		},
		identifier.IdNoopSuccess,
		identifier.ErrorOnUnknown,
	); err != nil {
		return nil, err
	}
	return ids, nil
}

func ridsFromOntologyContext(ontologyContext commonv2.OntologyContext) ([]rid.ResourceIdentifier, error) {
	if ontologyContext.EntityType == nil {
		return nil, nil
	}
	var ids []rid.ResourceIdentifier
	if err := ontologyContext.EntityType.AcceptFuncs(
		func(v rid.ResourceIdentifier) error {
			ids = append(ids, v)
			return nil
		},
		ontologyContext.EntityType.IdNoopSuccess,
		ontologyContext.EntityType.ErrorOnUnknown,
	); err != nil {
		return nil, err
	}
	return ids, nil
}

func ridsFromOntologyDataResourceList(resource commonv2.OntologyDataResourceList) ([]rid.ResourceIdentifier, error) {
	var ids []rid.ResourceIdentifier

	rids, err := ridsFromOntologyContext(resource.SharedOntologyResourceContext.OntologyContext)
	if err != nil {
		return nil, err
	}
	ids = append(ids, rids...)

	for _, r := range resource.OntologyDataResources {
		rids, err := ridsFromOntologyDataResource(r)
		if err != nil {
			return nil, err
		}
		ids = append(ids, rids...)
	}
	return ids, nil
}

func ridsFromOntologyLogicResource(resource commonv2.OntologyLogicResource) ([]rid.ResourceIdentifier, error) {
	var ids []rid.ResourceIdentifier
	for _, ontologyContext := range resource.OntologyContext {
		rids, err := ridsFromOntologyContext(ontologyContext)
		if err != nil {
			return nil, err
		}
		ids = append(ids, rids...)
	}
	if resource.Id != nil {
		rids, err := ridsFromIdentifier(*resource.Id)
		if err != nil {
			return nil, err
		}
		ids = append(ids, rids...)
	}
	return ids, nil
}
