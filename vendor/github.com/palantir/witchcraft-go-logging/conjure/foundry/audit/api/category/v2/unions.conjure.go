// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category/v2 package.

package v2

import (
	v2 "github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category/v2"
	"github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/category"
)

type AuditCategoryV2 = v2.AuditCategoryV2

type AuditCategoryV2Visitor = v2.AuditCategoryV2Visitor

type AuditCategoryV2VisitorWithContext = v2.AuditCategoryV2VisitorWithContext

func NewAuditCategoryV2FromDataCreate(v DataCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataCreate(v)
}

func NewAuditCategoryV2FromDataDelete(v DataDelete) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataDelete(v)
}

func NewAuditCategoryV2FromDataLoad(v DataLoad) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataLoad(v)
}

func NewAuditCategoryV2FromDataMerge(v DataMerge) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataMerge(v)
}

func NewAuditCategoryV2FromDataPromote(v DataPromote) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataPromote(v)
}

func NewAuditCategoryV2FromDataTransform(v DataTransform) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataTransform(v)
}

func NewAuditCategoryV2FromDataExport(v DataExport) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataExport(v)
}

func NewAuditCategoryV2FromDataImport(v DataImport) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataImport(v)
}

func NewAuditCategoryV2FromDataSearch(v DataSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataSearch(v)
}

func NewAuditCategoryV2FromBulkDataImport(v BulkDataImport) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromBulkDataImport(v)
}

func NewAuditCategoryV2FromCodeExecution(v CodeExecution) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromCodeExecution(v)
}

func NewAuditCategoryV2FromCancelCodeExecution(v CancelCodeExecution) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromCancelCodeExecution(v)
}

func NewAuditCategoryV2FromDataShareCreate(v DataShareCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataShareCreate(v)
}

func NewAuditCategoryV2FromDataShareDisable(v DataShareDisable) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataShareDisable(v)
}

func NewAuditCategoryV2FromDataShare(v DataShare) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromDataShare(v)
}

func NewAuditCategoryV2FromMetaDataAccess(v MetaDataAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMetaDataAccess(v)
}

func NewAuditCategoryV2FromMetaDataCreate(v MetaDataCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMetaDataCreate(v)
}

func NewAuditCategoryV2FromMetaDataDelete(v MetaDataDelete) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMetaDataDelete(v)
}

func NewAuditCategoryV2FromMetaDataUpdate(v MetaDataUpdate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMetaDataUpdate(v)
}

func NewAuditCategoryV2FromMetaDataSearch(v MetaDataSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMetaDataSearch(v)
}

func NewAuditCategoryV2FromAppConfigAccess(v AppConfigAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAppConfigAccess(v)
}

func NewAuditCategoryV2FromAppConfigUpdate(v AppConfigUpdate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAppConfigUpdate(v)
}

func NewAuditCategoryV2FromAppConfigCreate(v AppConfigCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAppConfigCreate(v)
}

func NewAuditCategoryV2FromAppConfigDelete(v AppConfigDelete) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAppConfigDelete(v)
}

func NewAuditCategoryV2FromAppConfigSearch(v AppConfigSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAppConfigSearch(v)
}

func NewAuditCategoryV2FromMonitorRun(v MonitorRun) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMonitorRun(v)
}

func NewAuditCategoryV2FromMonitorCreate(v MonitorCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMonitorCreate(v)
}

func NewAuditCategoryV2FromMonitorDelete(v MonitorDelete) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMonitorDelete(v)
}

func NewAuditCategoryV2FromMonitorUpdate(v MonitorUpdate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMonitorUpdate(v)
}

func NewAuditCategoryV2FromMonitorAccess(v MonitorAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMonitorAccess(v)
}

func NewAuditCategoryV2FromMonitorSearch(v MonitorSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromMonitorSearch(v)
}

func NewAuditCategoryV2FromLogicCreate(v LogicCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromLogicCreate(v)
}

func NewAuditCategoryV2FromLogicUpdate(v LogicUpdate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromLogicUpdate(v)
}

func NewAuditCategoryV2FromLogicAccess(v LogicAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromLogicAccess(v)
}

func NewAuditCategoryV2FromLogicDelete(v LogicDelete) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromLogicDelete(v)
}

func NewAuditCategoryV2FromLogicSearch(v LogicSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromLogicSearch(v)
}

func NewAuditCategoryV2FromRequestCreate(v RequestCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestCreate(v)
}

func NewAuditCategoryV2FromRequestAccess(v RequestAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestAccess(v)
}

func NewAuditCategoryV2FromRequestSearch(v RequestSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestSearch(v)
}

func NewAuditCategoryV2FromRequestUpdate(v RequestUpdate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestUpdate(v)
}

func NewAuditCategoryV2FromRequestApprove(v RequestApprove) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestApprove(v)
}

func NewAuditCategoryV2FromRequestDisapprove(v RequestDisapprove) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestDisapprove(v)
}

func NewAuditCategoryV2FromRequestExecute(v RequestExecute) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestExecute(v)
}

func NewAuditCategoryV2FromRequestCancel(v RequestCancel) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRequestCancel(v)
}

func NewAuditCategoryV2FromManagementUsers(v ManagementUsers) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromManagementUsers(v)
}

func NewAuditCategoryV2FromManagementGroups(v ManagementGroups) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromManagementGroups(v)
}

func NewAuditCategoryV2FromManagementMarkings(v ManagementMarkings) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromManagementMarkings(v)
}

func NewAuditCategoryV2FromManagementPermissions(v ManagementPermissions) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromManagementPermissions(v)
}

func NewAuditCategoryV2FromManagementTokens(v ManagementTokens) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromManagementTokens(v)
}

func NewAuditCategoryV2FromAuthenticationCheck(v AuthenticationCheck) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAuthenticationCheck(v)
}

func NewAuditCategoryV2FromAuthorizationCheck(v AuthorizationCheck) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAuthorizationCheck(v)
}

func NewAuditCategoryV2FromUserLogin(v UserLogin) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromUserLogin(v)
}

func NewAuditCategoryV2FromUserLogout(v UserLogout) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromUserLogout(v)
}

func NewAuditCategoryV2FromTokenGeneration(v TokenGeneration) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromTokenGeneration(v)
}

func NewAuditCategoryV2FromTokenRevoke(v TokenRevoke) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromTokenRevoke(v)
}

func NewAuditCategoryV2FromTokenAccess(v TokenAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromTokenAccess(v)
}

func NewAuditCategoryV2FromOauth2InitiateAuthFlow(v Oauth2InitiateAuthFlow) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOauth2InitiateAuthFlow(v)
}

func NewAuditCategoryV2FromAssetFileLoad(v AssetFileLoad) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAssetFileLoad(v)
}

func NewAuditCategoryV2FromAssetFileLoadV2(v AssetFileLoadV2) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAssetFileLoadV2(v)
}

func NewAuditCategoryV2FromContainerLaunch(v ContainerLaunch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromContainerLaunch(v)
}

func NewAuditCategoryV2FromContainerLoad(v ContainerLoad) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromContainerLoad(v)
}

func NewAuditCategoryV2FromContainerSearch(v ContainerSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromContainerSearch(v)
}

func NewAuditCategoryV2FromContainerStop(v ContainerStop) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromContainerStop(v)
}

func NewAuditCategoryV2FromInfraLogsAccess(v InfraLogsAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromInfraLogsAccess(v)
}

func NewAuditCategoryV2FromCreateInfra(v CreateInfra) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromCreateInfra(v)
}

func NewAuditCategoryV2FromConfigureInfra(v ConfigureInfra) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromConfigureInfra(v)
}

func NewAuditCategoryV2FromReviewInfraAction(v ReviewInfraAction) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromReviewInfraAction(v)
}

func NewAuditCategoryV2FromRestartInfra(v RestartInfra) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromRestartInfra(v)
}

func NewAuditCategoryV2FromUpgradeInfra(v UpgradeInfra) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromUpgradeInfra(v)
}

func NewAuditCategoryV2FromOntologyDataLoad(v OntologyDataLoad) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyDataLoad(v)
}

func NewAuditCategoryV2FromOntologyDataSearch(v OntologyDataSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyDataSearch(v)
}

func NewAuditCategoryV2FromOntologyDataTransform(v OntologyDataTransform) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyDataTransform(v)
}

func NewAuditCategoryV2FromOntologyLogicAccess(v OntologyLogicAccess) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyLogicAccess(v)
}

func NewAuditCategoryV2FromOntologyLogicCreate(v OntologyLogicCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyLogicCreate(v)
}

func NewAuditCategoryV2FromOntologyLogicDelete(v OntologyLogicDelete) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyLogicDelete(v)
}

func NewAuditCategoryV2FromOntologyLogicUpdate(v OntologyLogicUpdate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyLogicUpdate(v)
}

func NewAuditCategoryV2FromOntologyMetaDataLoad(v OntologyMetaDataLoad) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyMetaDataLoad(v)
}

func NewAuditCategoryV2FromOntologyMetaDataCreate(v OntologyMetaDataCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyMetaDataCreate(v)
}

func NewAuditCategoryV2FromOntologyMetaDataDelete(v OntologyMetaDataDelete) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyMetaDataDelete(v)
}

func NewAuditCategoryV2FromOntologyMetaDataUpdate(v OntologyMetaDataUpdate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyMetaDataUpdate(v)
}

func NewAuditCategoryV2FromOntologyMetaDataSearch(v OntologyMetaDataSearch) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOntologyMetaDataSearch(v)
}

func NewAuditCategoryV2FromSecretCreate(v SecretCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromSecretCreate(v)
}

func NewAuditCategoryV2FromSecretUse(v SecretUse) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromSecretUse(v)
}

func NewAuditCategoryV2FromSecretLoad(v SecretLoad) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromSecretLoad(v)
}

func NewAuditCategoryV2FromSecretDeprecate(v SecretDeprecate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromSecretDeprecate(v)
}

func NewAuditCategoryV2FromOnBehalfOf(v OnBehalfOf) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromOnBehalfOf(v)
}

func NewAuditCategoryV2FromInApplicationContext(v InApplicationContext) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromInApplicationContext(v)
}

func NewAuditCategoryV2FromInEnrollmentContext(v InEnrollmentContext) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromInEnrollmentContext(v)
}

func NewAuditCategoryV2FromInternal(v category.Internal) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromInternal(v)
}

func NewAuditCategoryV2FromUserJustify(v UserJustify) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromUserJustify(v)
}

func NewAuditCategoryV2FromPassThrough(v PassThrough) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromPassThrough(v)
}

func NewAuditCategoryV2FromLlmInference(v LlmInference) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromLlmInference(v)
}

func NewAuditCategoryV2FromLlmRoute(v LlmRoute) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromLlmRoute(v)
}

func NewAuditCategoryV2FromAuditDataTransform(v AuditDataTransform) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAuditDataTransform(v)
}

func NewAuditCategoryV2FromAuditDataShareCreate(v AuditDataShareCreate) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromAuditDataShareCreate(v)
}

func NewAuditCategoryV2FromApiGatewayRequest(v ApiGatewayRequest) AuditCategoryV2 {
	return v2.NewAuditCategoryV2FromApiGatewayRequest(v)
}

type ExternalSystem = v2.ExternalSystem

type ExternalSystemVisitor = v2.ExternalSystemVisitor

type ExternalSystemVisitorWithContext = v2.ExternalSystemVisitorWithContext

func NewExternalSystemFromPalantirSystem(v ExternalPalantirResource) ExternalSystem {
	return v2.NewExternalSystemFromPalantirSystem(v)
}

func NewExternalSystemFromOther(v []GenericValue) ExternalSystem {
	return v2.NewExternalSystemFromOther(v)
}
