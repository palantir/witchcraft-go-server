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
	"github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/category"
	v2 "github.com/palantir/witchcraft-go-logging/conjure/foundry/audit/api/category/v2"
)

func Category(category v2.AuditCategoryV2) Param {
	categoryV2WithT := (v2.AuditCategoryV2WithT[Param])(category)
	param, _ := categoryV2WithT.Accept(context.TODO(), &auditCategoryV2Visitor{})
	return param
}

var _ v2.AuditCategoryV2VisitorWithT[Param] = (*auditCategoryV2Visitor)(nil)

type auditCategoryV2Visitor struct {
}

func (a *auditCategoryV2Visitor) VisitDataCreate(ctx context.Context, v v2.DataCreate) (Param, error) {
	return multiParam([]Param{
		categories("dataCreate"),
		RequestField("createdResources", v.CreatedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataDelete(ctx context.Context, v v2.DataDelete) (Param, error) {
	return multiParam([]Param{
		categories("dataDelete"),
		RequestField("deletedResources", v.DeletedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataLoad(ctx context.Context, v v2.DataLoad) (Param, error) {
	return multiParam([]Param{
		categories("dataLoad"),
		RequestField("loadedResources", v.LoadedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataMerge(ctx context.Context, v v2.DataMerge) (Param, error) {
	return multiParam([]Param{
		categories("dataMerge"),
		RequestField("resourcesToMerge", v.ResourcesToMerge),
		ResultField("mergedResult", v.MergedResult),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataPromote(ctx context.Context, v v2.DataPromote) (Param, error) {
	return multiParam([]Param{
		categories("dataPromote"),
		RequestField("promotionDestinations", v.PromotionDestinations),
		RequestField("promotionDescription", v.PromotionDescription),
		RequestField("promotedResources", v.PromotedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataTransform(ctx context.Context, v v2.DataTransform) (Param, error) {
	return multiParam([]Param{
		categories("dataTransform"),
		RequestField("transformTargets", v.TransformTargets),
		RequestField("transformDescription", v.TransformDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataExport(ctx context.Context, v v2.DataExport) (Param, error) {
	return multiParam([]Param{
		categories("dataExport"),
		RequestField("downloadedResources", v.DownloadedResources),
		ResultField("downloadedSize", v.DownloadedSize),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataImport(ctx context.Context, v v2.DataImport) (Param, error) {
	return multiParam([]Param{
		categories("dataImport"),
		RequestField("importedFilename", v.ImportedFilename),
		RequestField("importedFileType", v.ImportedFileType),
		RequestField("importParentResourceId", v.ImportParentResourceId),
		ResultField("importResourceId", v.ImportResourceId),
		ResultField("importedSize", v.ImportedSize),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataSearch(ctx context.Context, v v2.DataSearch) (Param, error) {
	return multiParam([]Param{
		categories("dataSearch"),
		RequestField("dataSearchQuery", v.DataSearchQuery),
		RequestField("dataSearchContext", v.DataSearchContext),
		ResultField("dataSearchResults", v.DataSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitBulkDataImport(ctx context.Context, v v2.BulkDataImport) (Param, error) {
	return multiParam([]Param{
		categories("bulkDataImport"),
		RequestField("bulkImportedFiles", v.BulkImportedFiles),
		ResultField("bulkImportDestinations", v.BulkImportDestinations),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitCodeExecution(ctx context.Context, v v2.CodeExecution) (Param, error) {
	return multiParam([]Param{
		categories("codeExecution"),
		RequestField("executedResourceEnvironment", v.ExecutedResourceEnvironment),
		ResultField("executedResources", v.ExecutedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitCancelCodeExecution(ctx context.Context, v v2.CancelCodeExecution) (Param, error) {
	return multiParam([]Param{
		categories("cancelCodeExecution"),
		RequestField("cancelledExecutedResources", v.CancelledExecutedResources),
		RequestField("cancelledExecutedResourceEnvironment", v.CancelledExecutedResourceEnvironment),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataShareCreate(ctx context.Context, v v2.DataShareCreate) (Param, error) {
	return multiParam([]Param{
		categories("dataShareCreate"),
		RequestField("dataShareCreateId", v.DataShareCreateId),
		RequestField("dataShareCreateTargets", v.DataShareCreateTargets),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataShareDisable(ctx context.Context, v v2.DataShareDisable) (Param, error) {
	return multiParam([]Param{
		categories("dataShareDisable"),
		RequestField("dataShareDisableId", v.DataShareDisableId),
		RequestField("dataShareDisableTargets", v.DataShareDisableTargets),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitDataShare(ctx context.Context, v v2.DataShare) (Param, error) {
	return multiParam([]Param{
		categories("dataShare"),
		RequestField("dataShareId", v.DataShareId),
		RequestField("dataShareTargets", v.DataShareTargets),
		RequestField("dataShareReason", v.DataShareReason),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataAccess(ctx context.Context, v v2.MetaDataAccess) (Param, error) {
	return multiParam([]Param{
		categories("metaDataAccess"),
		RequestField("accessedMetaDataResources", v.AccessedMetaDataResources),
		RequestField("accessedMetaDataDescription", v.AccessedMetaDataDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataCreate(ctx context.Context, v v2.MetaDataCreate) (Param, error) {
	return multiParam([]Param{
		categories("metaDataCreate"),
		RequestField("createdMetaDataDescription", v.CreatedMetaDataDescription),
		ResultField("createdMetaDataResources", v.CreatedMetaDataResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataDelete(ctx context.Context, v v2.MetaDataDelete) (Param, error) {
	return multiParam([]Param{
		categories("metaDataDelete"),
		RequestField("deletedMetaDataResources", v.DeletedMetaDataResources),
		RequestField("deletedMetaDataDescription", v.DeletedMetaDataDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataUpdate(ctx context.Context, v v2.MetaDataUpdate) (Param, error) {
	return multiParam([]Param{
		categories("metaDataUpdate"),
		RequestField("updatedMetaDataResources", v.UpdatedMetaDataResources),
		RequestField("updatedMetaDataDescription", v.UpdatedMetaDataDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataSearch(ctx context.Context, v v2.MetaDataSearch) (Param, error) {
	return multiParam([]Param{
		categories("metaDataSearch"),
		RequestField("metaDataSearchQuery", v.MetaDataSearchQuery),
		ResultField("metaDataSearchResults", v.MetaDataSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigAccess(ctx context.Context, v v2.AppConfigAccess) (Param, error) {
	return multiParam([]Param{
		categories("appConfigAccess"),
		RequestField("accessedAppConfigIds", v.AccessedAppConfigIds),
		RequestField("accessAppConfigDescription", v.AccessAppConfigDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigUpdate(ctx context.Context, v v2.AppConfigUpdate) (Param, error) {
	return multiParam([]Param{
		categories("appConfigUpdate"),
		RequestField("updatedAppConfigIds", v.UpdatedAppConfigIds),
		RequestField("updateAppConfigDescription", v.UpdateAppConfigDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigCreate(ctx context.Context, v v2.AppConfigCreate) (Param, error) {
	return multiParam([]Param{
		categories("appConfigCreate"),
		RequestField("createAppConfigDescription", v.CreateAppConfigDescription),
		ResultField("createdAppConfigIds", v.CreatedAppConfigIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigDelete(ctx context.Context, v v2.AppConfigDelete) (Param, error) {
	return multiParam([]Param{
		categories("appConfigDelete"),
		RequestField("deletedAppConfigIds", v.DeletedAppConfigIds),
		RequestField("deleteAppConfigDescription", v.DeleteAppConfigDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigSearch(ctx context.Context, v v2.AppConfigSearch) (Param, error) {
	return multiParam([]Param{
		categories("appConfigSearch"),
		RequestField("appConfigSearchQuery", v.AppConfigSearchQuery),
		ResultField("appConfigSearchResults", v.AppConfigSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMonitorRun(ctx context.Context, v v2.MonitorRun) (Param, error) {
	return multiParam([]Param{
		categories("monitorRun"),
		RequestField("runMonitorTargets", v.RunMonitorTargets),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMonitorCreate(ctx context.Context, v v2.MonitorCreate) (Param, error) {
	return multiParam([]Param{
		categories("monitorCreate"),
		RequestField("createdMonitorDescription", v.CreatedMonitorDescription),
		ResultField("createdMonitorResources", v.CreatedMonitorResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMonitorDelete(ctx context.Context, v v2.MonitorDelete) (Param, error) {
	return multiParam([]Param{
		categories("monitorDelete"),
		RequestField("deletedMonitorResources", v.DeletedMonitorResources),
		RequestField("deletedMonitorDescription", v.DeletedMonitorDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMonitorUpdate(ctx context.Context, v v2.MonitorUpdate) (Param, error) {
	return multiParam([]Param{
		categories("monitorUpdate"),
		RequestField("updatedMonitorResources", v.UpdatedMonitorResources),
		RequestField("updatedMonitorDescription", v.UpdatedMonitorDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMonitorAccess(ctx context.Context, v v2.MonitorAccess) (Param, error) {
	return multiParam([]Param{
		categories("monitorAccess"),
		RequestField("accessedMonitorResources", v.AccessedMonitorResources),
		RequestField("accessedMonitorDescription", v.AccessedMonitorDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitMonitorSearch(ctx context.Context, v v2.MonitorSearch) (Param, error) {
	return multiParam([]Param{
		categories("monitorSearch"),
		RequestField("monitorSearchQuery", v.MonitorSearchQuery),
		ResultField("monitorSearchResults", v.MonitorSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitLogicCreate(ctx context.Context, v v2.LogicCreate) (Param, error) {
	return multiParam([]Param{
		categories("logicCreate"),
		ResultField("createdLogicResources", v.CreatedLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitLogicUpdate(ctx context.Context, v v2.LogicUpdate) (Param, error) {
	return multiParam([]Param{
		categories("logicUpdate"),
		RequestField("updatedLogicResources", v.UpdatedLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitLogicAccess(ctx context.Context, v v2.LogicAccess) (Param, error) {
	return multiParam([]Param{
		categories("logicAccess"),
		RequestField("accessedLogicResources", v.AccessedLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitLogicDelete(ctx context.Context, v v2.LogicDelete) (Param, error) {
	return multiParam([]Param{
		categories("logicDelete"),
		RequestField("deletedLogicResources", v.DeletedLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitLogicSearch(ctx context.Context, v v2.LogicSearch) (Param, error) {
	return multiParam([]Param{
		categories("logicSearch"),
		RequestField("logicSearchQuery", v.LogicSearchQuery),
		ResultField("logicSearchResults", v.LogicSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestCreate(ctx context.Context, v v2.RequestCreate) (Param, error) {
	return multiParam([]Param{
		categories("requestCreate"),
		RequestField("createdRequestAffectedResources", v.CreatedRequestAffectedResources),
		RequestField("createdRequestDescription", v.CreatedRequestDescription),
		ResultField("createdRequestIds", v.CreatedRequestIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestAccess(ctx context.Context, v v2.RequestAccess) (Param, error) {
	return multiParam([]Param{
		categories("requestAccess"),
		RequestField("accessedRequestIds", v.AccessedRequestIds),
		RequestField("accessedRequestDescription", v.AccessedRequestDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestSearch(ctx context.Context, v v2.RequestSearch) (Param, error) {
	return multiParam([]Param{
		categories("requestSearch"),
		RequestField("requestSearchQuery", v.RequestSearchQuery),
		ResultField("requestSearchResults", v.RequestSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestUpdate(ctx context.Context, v v2.RequestUpdate) (Param, error) {
	return multiParam([]Param{
		categories("requestUpdate"),
		RequestField("updatedRequestIds", v.UpdatedRequestIds),
		RequestField("updatedRequestDescription", v.UpdatedRequestDescription),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestApprove(ctx context.Context, v v2.RequestApprove) (Param, error) {
	return multiParam([]Param{
		categories("requestApprove"),
		RequestField("approvedRequestIds", v.ApprovedRequestIds),
		RequestField("approveRequestUserId", v.ApproveRequestUserId),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestDisapprove(ctx context.Context, v v2.RequestDisapprove) (Param, error) {
	return multiParam([]Param{
		categories("requestDisapprove"),
		RequestField("disapprovedRequestIds", v.DisapprovedRequestIds),
		RequestField("disapproveRequestUserId", v.DisapproveRequestUserId),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestExecute(ctx context.Context, v v2.RequestExecute) (Param, error) {
	return multiParam([]Param{
		categories("requestExecute"),
		RequestField("executedRequestIds", v.ExecutedRequestIds),
		ResultField("executeRequestAffectedResources", v.ExecuteRequestAffectedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRequestCancel(ctx context.Context, v v2.RequestCancel) (Param, error) {
	return multiParam([]Param{
		categories("requestCancel"),
		RequestField("canceledRequestIds", v.CanceledRequestIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitManagementUsers(ctx context.Context, v v2.ManagementUsers) (Param, error) {
	return multiParam([]Param{
		categories("managementUsers"),
		RequestField("managedUserIds", v.ManagedUserIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitManagementGroups(ctx context.Context, v v2.ManagementGroups) (Param, error) {
	return multiParam([]Param{
		categories("managementGroups"),
		RequestField("groupPatches", v.GroupPatches),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitManagementMarkings(ctx context.Context, v v2.ManagementMarkings) (Param, error) {
	return multiParam([]Param{
		categories("managementMarkings"),
		RequestField("markingPatches", v.MarkingPatches),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitManagementPermissions(ctx context.Context, v v2.ManagementPermissions) (Param, error) {
	return multiParam([]Param{
		categories("managementPermissions"),
		RequestField("resourcesWithPermissionsChanges", v.ResourcesWithPermissionsChanges),
		RequestField("permissionChangeContext", v.PermissionChangeContext),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitManagementTokens(ctx context.Context, v v2.ManagementTokens) (Param, error) {
	return multiParam([]Param{
		categories("managementTokens"),
		RequestField("managedTokens", v.ManagedTokens),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAuthenticationCheck(ctx context.Context, v v2.AuthenticationCheck) (Param, error) {
	return multiParam([]Param{
		categories("authenticationCheck"),
		RequestField("authenticationCheckTargets", v.AuthenticationCheckTargets),
		ResultField("authenticationCheckResult", v.AuthenticationCheckResult),
		ResultField("authenticationCheckResultMessage", v.AuthenticationCheckResultMessage),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAuthorizationCheck(ctx context.Context, v v2.AuthorizationCheck) (Param, error) {
	return multiParam([]Param{
		categories("authorizationCheck"),
		RequestField("authorizationCheckTargets", v.AuthorizationCheckTargets),
		RequestField("authorizationCheckOperations", v.AuthorizationCheckOperations),
		ResultField("authorizationCheckSucceededTargets", v.AuthorizationCheckSucceededTargets),
		ResultField("authorizationCheckFailedTargets", v.AuthorizationCheckFailedTargets),
		ResultField("authorizationCheckResultMessage", v.AuthorizationCheckResultMessage),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitUserLogin(ctx context.Context, v v2.UserLogin) (Param, error) {
	return multiParam([]Param{
		categories("userLogin"),
		ResultField("loginUserId", v.LoginUserId),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitUserLogout(ctx context.Context, v v2.UserLogout) (Param, error) {
	return multiParam([]Param{
		categories("userLogout"),
		RequestField("logoutUserId", v.LogoutUserId),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitTokenGeneration(ctx context.Context, v v2.TokenGeneration) (Param, error) {
	return multiParam([]Param{
		categories("tokenGeneration"),
		RequestField("generateTokensDescription", v.GenerateTokensDescription),
		ResultField("generatedTokens", v.GeneratedTokens),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitTokenRevoke(ctx context.Context, v v2.TokenRevoke) (Param, error) {
	return multiParam([]Param{
		categories("tokenRevoke"),
		RequestField("revokeTokensDescription", v.RevokeTokensDescription),
		ResultField("revokedTokens", v.RevokedTokens),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitTokenAccess(ctx context.Context, v v2.TokenAccess) (Param, error) {
	return multiParam([]Param{
		categories("tokenAccess"),
		ResultField("accessedTokens", v.AccessedTokens),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOauth2InitiateAuthFlow(ctx context.Context, v v2.Oauth2InitiateAuthFlow) (Param, error) {
	return multiParam([]Param{
		categories("oauth2InitiateAuthFlow"),
		RequestField("oauth2InitiateAuthFlowUser", v.Oauth2InitiateAuthFlowUser),
		RequestField("oauth2InitiateAuthClientId", v.Oauth2InitiateAuthClientId),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAssetFileLoad(ctx context.Context, v v2.AssetFileLoad) (Param, error) {
	return multiParam([]Param{
		categories("assetFileLoad"),
		RequestField("requestMavenCoordinate", v.RequestMavenCoordinate),
		ResultField("responseMavenCoordinate", v.ResponseMavenCoordinate),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitContainerLaunch(ctx context.Context, v v2.ContainerLaunch) (Param, error) {
	return multiParam([]Param{
		categories("containerLaunch"),
		RequestField("requestedContainerIdsToLaunch", v.RequestedContainerIdsToLaunch),
		ResultField("launchedContainerIds", v.LaunchedContainerIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitContainerLoad(ctx context.Context, v v2.ContainerLoad) (Param, error) {
	return multiParam([]Param{
		categories("containerLoad"),
		RequestField("requestedContainerLoadIds", v.RequestedContainerLoadIds),
		ResultField("loadedContainerLoadIds", v.LoadedContainerLoadIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitContainerSearch(ctx context.Context, v v2.ContainerSearch) (Param, error) {
	return multiParam([]Param{
		categories("containerSearch"),
		RequestField("containerSearchQuery", v.ContainerSearchQuery),
		ResultField("containerSearchResults", v.ContainerSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitContainerStop(ctx context.Context, v v2.ContainerStop) (Param, error) {
	return multiParam([]Param{
		categories("containerStop"),
		RequestField("stoppedContainerIds", v.StoppedContainerIds),
		RequestField("containerStopReason", v.ContainerStopReason),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitInfraLogsAccess(ctx context.Context, v v2.InfraLogsAccess) (Param, error) {
	return multiParam([]Param{
		categories("infraLogsAccess"),
		RequestField("infraLogsAccessTarget", v.InfraLogsAccessTarget),
		ResultField("infraLogsAccessRequestId", v.InfraLogsAccessRequestId),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitCreateInfra(ctx context.Context, v v2.CreateInfra) (Param, error) {
	return multiParam([]Param{
		categories("createInfra"),
		RequestField("createInfraTargets", v.CreateInfraTargets),
		ResultField("createdInfraResources", v.CreatedInfraResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitConfigureInfra(ctx context.Context, v v2.ConfigureInfra) (Param, error) {
	return multiParam([]Param{
		categories("configureInfra"),
		RequestField("configureInfraTargets", v.ConfigureInfraTargets),
		ResultField("configureInfraRequestId", v.ConfigureInfraRequestId),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitReviewInfraAction(ctx context.Context, v v2.ReviewInfraAction) (Param, error) {
	return multiParam([]Param{
		categories("reviewInfraAction"),
		RequestField("reviewInfraActionRequestId", v.ReviewInfraActionRequestId),
		RequestField("reviewInfraActionUser", v.ReviewInfraActionUser),
		ResultField("reviewInfraActionWasApproved", v.ReviewInfraActionWasApproved),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitRestartInfra(ctx context.Context, v v2.RestartInfra) (Param, error) {
	return multiParam([]Param{
		categories("restartInfra"),
		RequestField("restartedResources", v.RestartedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitUpgradeInfra(ctx context.Context, v v2.UpgradeInfra) (Param, error) {
	return multiParam([]Param{
		categories("upgradeInfra"),
		RequestField("upgradedResources", v.UpgradedResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyDataLoad(ctx context.Context, v v2.OntologyDataLoad) (Param, error) {
	return multiParam([]Param{
		categories("ontologyDataLoad"),
		RequestField("ontologyDataLoadContext", v.OntologyDataLoadContext),
		RequestField("requestedOntologyDataResources", v.RequestedOntologyDataResources),
		ResultField("loadedOntologyDataResources", v.LoadedOntologyDataResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyDataSearch(ctx context.Context, v v2.OntologyDataSearch) (Param, error) {
	return multiParam([]Param{
		categories("ontologyDataSearch"),
		RequestField("ontologyDataSearchContext", v.OntologyDataSearchContext),
		RequestField("searchedOntologyLogicResources", v.SearchedOntologyLogicResources),
		ResultField("ontologyDataSearchResults", v.OntologyDataSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyDataTransform(ctx context.Context, v v2.OntologyDataTransform) (Param, error) {
	return multiParam([]Param{
		categories("ontologyDataTransform"),
		RequestField("ontologyDataTransformTargets", v.OntologyDataTransformTargets),
		RequestField("ontologyDataTransformContext", v.OntologyDataTransformContext),
		RequestField("ontologyDataTransformDescription", v.OntologyDataTransformDescription),
		ResultField("transformedOntologyDataResources", v.TransformedOntologyDataResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicAccess(ctx context.Context, v v2.OntologyLogicAccess) (Param, error) {
	return multiParam([]Param{
		categories("ontologyLogicAccess"),
		RequestField("requestedOntologyLogicResources", v.RequestedOntologyLogicResources),
		ResultField("loadedOntologyLogicResources", v.LoadedOntologyLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicCreate(ctx context.Context, v v2.OntologyLogicCreate) (Param, error) {
	return multiParam([]Param{
		categories("ontologyLogicCreate"),
		RequestField("createOntologyLogicContext", v.CreateOntologyLogicContext),
		ResultField("createdOntologyLogicResources", v.CreatedOntologyLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicDelete(ctx context.Context, v v2.OntologyLogicDelete) (Param, error) {
	return multiParam([]Param{
		categories("ontologyLogicDelete"),
		RequestField("deleteOntologyLogicContext", v.DeleteOntologyLogicContext),
		ResultField("deletedOntologyLogicResources", v.DeletedOntologyLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicUpdate(ctx context.Context, v v2.OntologyLogicUpdate) (Param, error) {
	return multiParam([]Param{
		categories("ontologyLogicUpdate"),
		RequestField("updateOntologyLogicContext", v.UpdateOntologyLogicContext),
		ResultField("updatedOntologyLogicResources", v.UpdatedOntologyLogicResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataLoad(ctx context.Context, v v2.OntologyMetaDataLoad) (Param, error) {
	return multiParam([]Param{
		categories("ontologyMetaDataLoad"),
		RequestField("requestedOntologyMetaDataResources", v.RequestedOntologyMetaDataResources),
		ResultField("loadedOntologyMetaDataResources", v.LoadedOntologyMetaDataResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataCreate(ctx context.Context, v v2.OntologyMetaDataCreate) (Param, error) {
	return multiParam([]Param{
		categories("ontologyMetaDataCreate"),
		ResultField("createdOntologyMetaDataResources", v.CreatedOntologyMetaDataResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataDelete(ctx context.Context, v v2.OntologyMetaDataDelete) (Param, error) {
	return multiParam([]Param{
		categories("ontologyMetaDataDelete"),
		RequestField("deletedOntologyMetaDataResources", v.DeletedOntologyMetaDataResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataUpdate(ctx context.Context, v v2.OntologyMetaDataUpdate) (Param, error) {
	return multiParam([]Param{
		categories("ontologyMetaDataUpdate"),
		RequestField("updatedOntologyMetaDataResources", v.UpdatedOntologyMetaDataResources),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataSearch(ctx context.Context, v v2.OntologyMetaDataSearch) (Param, error) {
	return multiParam([]Param{
		categories("ontologyMetaDataSearch"),
		RequestField("ontologyMetaDataSearchedResources", v.OntologyMetaDataSearchedResources),
		RequestField("ontologyMetaDataSearchContext", v.OntologyMetaDataSearchContext),
		ResultField("ontologyMetaDataSearchResults", v.OntologyMetaDataSearchResults),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitSecretCreate(ctx context.Context, v v2.SecretCreate) (Param, error) {
	return multiParam([]Param{
		categories("secretCreate"),
		RequestField("createdSecretType", v.CreatedSecretType),
		ResultField("createdSecretIdentifiers", v.CreatedSecretIdentifiers),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitSecretUse(ctx context.Context, v v2.SecretUse) (Param, error) {
	return multiParam([]Param{
		categories("secretUse"),
		RequestField("usedSecretOperation", v.UsedSecretOperation),
		RequestField("usedSecretIdentifiers", v.UsedSecretIdentifiers),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitSecretLoad(ctx context.Context, v v2.SecretLoad) (Param, error) {
	return multiParam([]Param{
		categories("secretLoad"),
		RequestField("loadedSecretIdentifiers", v.LoadedSecretIdentifiers),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitSecretDeprecate(ctx context.Context, v v2.SecretDeprecate) (Param, error) {
	return multiParam([]Param{
		categories("secretDeprecate"),
		RequestField("deprecatedSecretIdentifier", v.DeprecatedSecretIdentifier),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitOnBehalfOf(ctx context.Context, v v2.OnBehalfOf) (Param, error) {
	return multiParam([]Param{
		categories("onBehalfOf"),
		RequestField("onBehalfOfUserIds", v.OnBehalfOfUserIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitInApplicationContext(ctx context.Context, v v2.InApplicationContext) (Param, error) {
	return multiParam([]Param{
		categories("inApplicationContext"),
		RequestField("applicationRid", v.ApplicationRid),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitInEnrollmentContext(ctx context.Context, v v2.InEnrollmentContext) (Param, error) {
	return multiParam([]Param{
		categories("inEnrollmentContext"),
		RequestField("enrollmentRids", v.EnrollmentRids),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitInternal(ctx context.Context, v category.Internal) (Param, error) {
	return multiParam([]Param{
		categories("internal"),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitUserJustify(ctx context.Context, v v2.UserJustify) (Param, error) {
	return multiParam([]Param{
		categories("userJustify"),
		RequestField("userJustifyId", v.UserJustifyId),
		RequestField("userJustification", v.UserJustification),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitPassThrough(ctx context.Context, v v2.PassThrough) (Param, error) {
	return multiParam([]Param{
		categories("passThrough"),
		RequestField("passThroughRequestParams", v.PassThroughRequestParams),
		ResultField("passThroughResponseParams", v.PassThroughResponseParams),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitLlmInference(ctx context.Context, v v2.LlmInference) (Param, error) {
	return multiParam([]Param{
		categories("llmInference"),
		RequestField("llmInferenceContext", v.LlmInferenceContext),
		RequestField("llmInferenceInputs", v.LlmInferenceInputs),
		ResultField("llmInferenceResponses", v.LlmInferenceResponses),
		ResultField("llmInferenceResponseContext", v.LlmInferenceResponseContext),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitLlmRoute(ctx context.Context, v v2.LlmRoute) (Param, error) {
	return multiParam([]Param{
		categories("llmRoute"),
		RequestField("llmRouteRequest", v.LlmRouteRequest),
		ResultField("llmRouteResponse", v.LlmRouteResponse),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAuditDataTransform(ctx context.Context, v v2.AuditDataTransform) (Param, error) {
	return multiParam([]Param{
		categories("auditDataTransform"),
		RequestField("transformTarget", v.TransformTarget),
		RequestField("transformDescriptions", v.TransformDescriptions),
		ResultField("transformDestination", v.TransformDestination),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitAuditDataShareCreate(ctx context.Context, v v2.AuditDataShareCreate) (Param, error) {
	return multiParam([]Param{
		categories("auditDataShareCreate"),
		RequestField("shareTargets", v.ShareTargets),
		ResultField("shareIds", v.ShareIds),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitApiGatewayRequest(ctx context.Context, v v2.ApiGatewayRequest) (Param, error) {
	return multiParam([]Param{
		categories("apiGatewayRequest"),
		RequestField("operationNames", v.OperationNames),
	}), nil
}

func (a *auditCategoryV2Visitor) VisitUnknown(ctx context.Context, typ string) (Param, error) {
	return nil, werror.Error("unhandled type", werror.SafeParam("type", typ))
}
