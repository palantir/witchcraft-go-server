// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/common/v2 package.

package v2

import (
	"github.com/palantir/pkg/rid"
	v2 "github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/common/v2"
)

type AssetFileLoadIdentifier = v2.AssetFileLoadIdentifier

type AssetFileLoadIdentifierVisitor = v2.AssetFileLoadIdentifierVisitor

type AssetFileLoadIdentifierVisitorWithContext = v2.AssetFileLoadIdentifierVisitorWithContext

func NewAssetFileLoadIdentifierFromMavenCoordinateAndPath(v MavenCoordinateAndPathIdentifier) AssetFileLoadIdentifier {
	return v2.NewAssetFileLoadIdentifierFromMavenCoordinateAndPath(v)
}

func NewAssetFileLoadIdentifierFromContentAddressableFileIdentifier(v ContentAddressableFileIdentifier) AssetFileLoadIdentifier {
	return v2.NewAssetFileLoadIdentifierFromContentAddressableFileIdentifier(v)
}

type AssetFileLoadResponse = v2.AssetFileLoadResponse

type AssetFileLoadResponseVisitor = v2.AssetFileLoadResponseVisitor

type AssetFileLoadResponseVisitorWithContext = v2.AssetFileLoadResponseVisitorWithContext

func NewAssetFileLoadResponseFromMavenCoordinate(v MavenCoordinate) AssetFileLoadResponse {
	return v2.NewAssetFileLoadResponseFromMavenCoordinate(v)
}

func NewAssetFileLoadResponseFromAssetCoordinate(v AssetCoordinate) AssetFileLoadResponse {
	return v2.NewAssetFileLoadResponseFromAssetCoordinate(v)
}

type EntityLocator = v2.EntityLocator

type EntityLocatorVisitor = v2.EntityLocatorVisitor

type EntityLocatorVisitorWithContext = v2.EntityLocatorVisitorWithContext

func NewEntityLocatorFromService(v ServiceLocator) EntityLocator {
	return v2.NewEntityLocatorFromService(v)
}

func NewEntityLocatorFromAsset(v AssetLocator) EntityLocator {
	return v2.NewEntityLocatorFromAsset(v)
}

func NewEntityLocatorFromDaemon(v DaemonLocator) EntityLocator {
	return v2.NewEntityLocatorFromDaemon(v)
}

func NewEntityLocatorFromK8sApp(v K8sAppLocator) EntityLocator {
	return v2.NewEntityLocatorFromK8sApp(v)
}

func NewEntityLocatorFromK8sPod(v K8sPodLocator) EntityLocator {
	return v2.NewEntityLocatorFromK8sPod(v)
}

func NewEntityLocatorFromK8sDeployment(v K8sDeployment) EntityLocator {
	return v2.NewEntityLocatorFromK8sDeployment(v)
}

type EntityTypeIdentifier = v2.EntityTypeIdentifier

type EntityTypeIdentifierVisitor = v2.EntityTypeIdentifierVisitor

type EntityTypeIdentifierVisitorWithContext = v2.EntityTypeIdentifierVisitorWithContext

func NewEntityTypeIdentifierFromRid(v rid.ResourceIdentifier) EntityTypeIdentifier {
	return v2.NewEntityTypeIdentifierFromRid(v)
}

func NewEntityTypeIdentifierFromId(v string) EntityTypeIdentifier {
	return v2.NewEntityTypeIdentifierFromId(v)
}

type Identifier = v2.Identifier

type IdentifierVisitor = v2.IdentifierVisitor

type IdentifierVisitorWithContext = v2.IdentifierVisitorWithContext

func NewIdentifierFromRid(v rid.ResourceIdentifier) Identifier {
	return v2.NewIdentifierFromRid(v)
}

func NewIdentifierFromRids(v []rid.ResourceIdentifier) Identifier {
	return v2.NewIdentifierFromRids(v)
}

func NewIdentifierFromOther(v OtherIdentifier) Identifier {
	return v2.NewIdentifierFromOther(v)
}

func NewIdentifierFromOthers(v []OtherIdentifier) Identifier {
	return v2.NewIdentifierFromOthers(v)
}

type LlmInput = v2.LlmInput

type LlmInputVisitor = v2.LlmInputVisitor
type LlmInputVisitorWithContext = v2.LlmInputVisitorWithContext

func NewLlmInputFromTextPrompt(v string) LlmInput {
	return v2.NewLlmInputFromTextPrompt(v)
}

func NewLlmInputFromResourceId(v Identifier) LlmInput {
	return v2.NewLlmInputFromResourceId(v)
}

type LlmResponse = v2.LlmResponse

type LlmResponseVisitor = v2.LlmResponseVisitor

type LlmResponseVisitorWithContext = v2.LlmResponseVisitorWithContext

func NewLlmResponseFromTextResponse(v string) LlmResponse {
	return v2.NewLlmResponseFromTextResponse(v)
}

func NewLlmResponseFromResourceId(v Identifier) LlmResponse {
	return v2.NewLlmResponseFromResourceId(v)
}

type ObjectPropertyIdentifier = v2.ObjectPropertyIdentifier

type ObjectPropertyIdentifierVisitor = v2.ObjectPropertyIdentifierVisitor

type ObjectPropertyIdentifierVisitorWithContext = v2.ObjectPropertyIdentifierVisitorWithContext

func NewObjectPropertyIdentifierFromRid(v rid.ResourceIdentifier) ObjectPropertyIdentifier {
	return v2.NewObjectPropertyIdentifierFromRid(v)
}

func NewObjectPropertyIdentifierFromId(v string) ObjectPropertyIdentifier {
	return v2.NewObjectPropertyIdentifierFromId(v)
}

type Resource = v2.Resource

type ResourceVisitor = v2.ResourceVisitor

type ResourceVisitorWithContext = v2.ResourceVisitorWithContext

func NewResourceFromApplication(v ApplicationResource) Resource {
	return v2.NewResourceFromApplication(v)
}

func NewResourceFromData(v DataResource) Resource {
	return v2.NewResourceFromData(v)
}

func NewResourceFromMonitor(v MonitorResource) Resource {
	return v2.NewResourceFromMonitor(v)
}

func NewResourceFromSystem(v SystemResource) Resource {
	return v2.NewResourceFromSystem(v)
}

func NewResourceFromLogic(v LogicResource) Resource {
	return v2.NewResourceFromLogic(v)
}

func NewResourceFromRequest(v RequestResource) Resource {
	return v2.NewResourceFromRequest(v)
}

func NewResourceFromOntologyData(v OntologyDataResource) Resource {
	return v2.NewResourceFromOntologyData(v)
}

func NewResourceFromOntologyDataList(v OntologyDataResourceList) Resource {
	return v2.NewResourceFromOntologyDataList(v)
}

func NewResourceFromOntologyLogic(v OntologyLogicResource) Resource {
	return v2.NewResourceFromOntologyLogic(v)
}

func NewResourceFromOntologyMetaData(v OntologyMetaDataResource) Resource {
	return v2.NewResourceFromOntologyMetaData(v)
}

func NewResourceFromExternal(v Identifier) Resource {
	return v2.NewResourceFromExternal(v)
}

type SystemResource = v2.SystemResource

type SystemResourceVisitor = v2.SystemResourceVisitor

type SystemResourceVisitorWithContext = v2.SystemResourceVisitorWithContext

func NewSystemResourceFromNode(v NodeId) SystemResource {
	return v2.NewSystemResourceFromNode(v)
}

func NewSystemResourceFromEntity(v EntityId) SystemResource {
	return v2.NewSystemResourceFromEntity(v)
}

func NewSystemResourceFromEnvironment(v EnvironmentId) SystemResource {
	return v2.NewSystemResourceFromEnvironment(v)
}

func NewSystemResourceFromExternal(v ExternalSystemResource) SystemResource {
	return v2.NewSystemResourceFromExternal(v)
}
