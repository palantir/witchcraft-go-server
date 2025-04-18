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
	"github.com/palantir/witchcraft-go-logging/wlog"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/audit2log"
	"github.com/palantir/witchcraft-go-logging/wlog/auditlog/internal/auditloginternal"
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
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataCreate").getParam().Audit3ParamFn(entry)
					RequestField("createdResources", v.CreatedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataDelete(ctx context.Context, v v2.DataDelete) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataDelete(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataDelete").getParam().Audit3ParamFn(entry)
					RequestField("deletedResources", v.DeletedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataLoad(ctx context.Context, v v2.DataLoad) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataLoad(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataLoad").getParam().Audit3ParamFn(entry)
					RequestField("loadedResources", v.LoadedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataMerge(ctx context.Context, v v2.DataMerge) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataMerge(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataMerge").getParam().Audit3ParamFn(entry)
					RequestField("resourcesToMerge", v.ResourcesToMerge).getParam().Audit3ParamFn(entry)
					ResultField("mergedResult", v.MergedResult).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataPromote(ctx context.Context, v v2.DataPromote) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataPromote(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataPromote").getParam().Audit3ParamFn(entry)
					RequestField("promotionDestinations", v.PromotionDestinations).getParam().Audit3ParamFn(entry)
					RequestField("promotionDescription", v.PromotionDescription).getParam().Audit3ParamFn(entry)
					RequestField("promotedResources", v.PromotedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataTransform(ctx context.Context, v v2.DataTransform) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataTransform(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataTransform").getParam().Audit3ParamFn(entry)
					RequestField("transformTargets", v.TransformTargets).getParam().Audit3ParamFn(entry)
					RequestField("transformDescription", v.TransformDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataExport(ctx context.Context, v v2.DataExport) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataExport(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataExport").getParam().Audit3ParamFn(entry)
					RequestField("downloadedResources", v.DownloadedResources).getParam().Audit3ParamFn(entry)
					ResultField("downloadedSize", v.DownloadedSize).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataImport(ctx context.Context, v v2.DataImport) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataImport(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataImport").getParam().Audit3ParamFn(entry)
					RequestField("importedFilename", v.ImportedFilename).getParam().Audit3ParamFn(entry)
					RequestField("importedFileType", v.ImportedFileType).getParam().Audit3ParamFn(entry)
					RequestField("importParentResourceId", v.ImportParentResourceId).getParam().Audit3ParamFn(entry)
					ResultField("importResourceId", v.ImportResourceId).getParam().Audit3ParamFn(entry)
					ResultField("importedSize", v.ImportedSize).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataSearch(ctx context.Context, v v2.DataSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataSearch").getParam().Audit3ParamFn(entry)
					RequestField("dataSearchQuery", v.DataSearchQuery).getParam().Audit3ParamFn(entry)
					RequestField("dataSearchContext", v.DataSearchContext).getParam().Audit3ParamFn(entry)
					ResultField("dataSearchResults", v.DataSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitBulkDataImport(ctx context.Context, v v2.BulkDataImport) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromBulkDataImport(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("bulkDataImport").getParam().Audit3ParamFn(entry)
					RequestField("bulkImportedFiles", v.BulkImportedFiles).getParam().Audit3ParamFn(entry)
					ResultField("bulkImportDestinations", v.BulkImportDestinations).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitCodeExecution(ctx context.Context, v v2.CodeExecution) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromCodeExecution(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("codeExecution").getParam().Audit3ParamFn(entry)
					RequestField("executedResourceEnvironment", v.ExecutedResourceEnvironment).getParam().Audit3ParamFn(entry)
					ResultField("executedResources", v.ExecutedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitCancelCodeExecution(ctx context.Context, v v2.CancelCodeExecution) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromCancelCodeExecution(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("cancelCodeExecution").getParam().Audit3ParamFn(entry)
					RequestField("cancelledExecutedResources", v.CancelledExecutedResources).getParam().Audit3ParamFn(entry)
					RequestField("cancelledExecutedResourceEnvironment", v.CancelledExecutedResourceEnvironment).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataShareCreate(ctx context.Context, v v2.DataShareCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataShareCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataShareCreate").getParam().Audit3ParamFn(entry)
					RequestField("dataShareCreateId", v.DataShareCreateId).getParam().Audit3ParamFn(entry)
					RequestField("dataShareCreateTargets", v.DataShareCreateTargets).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataShareDisable(ctx context.Context, v v2.DataShareDisable) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataShareDisable(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataShareDisable").getParam().Audit3ParamFn(entry)
					RequestField("dataShareDisableId", v.DataShareDisableId).getParam().Audit3ParamFn(entry)
					RequestField("dataShareDisableTargets", v.DataShareDisableTargets).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitDataShare(ctx context.Context, v v2.DataShare) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromDataShare(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("dataShare").getParam().Audit3ParamFn(entry)
					RequestField("dataShareId", v.DataShareId).getParam().Audit3ParamFn(entry)
					RequestField("dataShareTargets", v.DataShareTargets).getParam().Audit3ParamFn(entry)
					RequestField("dataShareReason", v.DataShareReason).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataAccess(ctx context.Context, v v2.MetaDataAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMetaDataAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("metaDataAccess").getParam().Audit3ParamFn(entry)
					RequestField("accessedMetaDataResources", v.AccessedMetaDataResources).getParam().Audit3ParamFn(entry)
					RequestField("accessedMetaDataDescription", v.AccessedMetaDataDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataCreate(ctx context.Context, v v2.MetaDataCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMetaDataCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("metaDataCreate").getParam().Audit3ParamFn(entry)
					RequestField("createdMetaDataDescription", v.CreatedMetaDataDescription).getParam().Audit3ParamFn(entry)
					ResultField("createdMetaDataResources", v.CreatedMetaDataResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataDelete(ctx context.Context, v v2.MetaDataDelete) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMetaDataDelete(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("metaDataDelete").getParam().Audit3ParamFn(entry)
					RequestField("deletedMetaDataResources", v.DeletedMetaDataResources).getParam().Audit3ParamFn(entry)
					RequestField("deletedMetaDataDescription", v.DeletedMetaDataDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataUpdate(ctx context.Context, v v2.MetaDataUpdate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMetaDataUpdate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("metaDataUpdate").getParam().Audit3ParamFn(entry)
					RequestField("updatedMetaDataResources", v.UpdatedMetaDataResources).getParam().Audit3ParamFn(entry)
					RequestField("updatedMetaDataDescription", v.UpdatedMetaDataDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMetaDataSearch(ctx context.Context, v v2.MetaDataSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMetaDataSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("metaDataSearch").getParam().Audit3ParamFn(entry)
					RequestField("metaDataSearchQuery", v.MetaDataSearchQuery).getParam().Audit3ParamFn(entry)
					ResultField("metaDataSearchResults", v.MetaDataSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigAccess(ctx context.Context, v v2.AppConfigAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAppConfigAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("appConfigAccess").getParam().Audit3ParamFn(entry)
					RequestField("accessedAppConfigIds", v.AccessedAppConfigIds).getParam().Audit3ParamFn(entry)
					RequestField("accessAppConfigDescription", v.AccessAppConfigDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigUpdate(ctx context.Context, v v2.AppConfigUpdate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAppConfigUpdate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("appConfigUpdate").getParam().Audit3ParamFn(entry)
					RequestField("updatedAppConfigIds", v.UpdatedAppConfigIds).getParam().Audit3ParamFn(entry)
					RequestField("updateAppConfigDescription", v.UpdateAppConfigDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigCreate(ctx context.Context, v v2.AppConfigCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAppConfigCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("appConfigCreate").getParam().Audit3ParamFn(entry)
					RequestField("createAppConfigDescription", v.CreateAppConfigDescription).getParam().Audit3ParamFn(entry)
					ResultField("createdAppConfigIds", v.CreatedAppConfigIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigDelete(ctx context.Context, v v2.AppConfigDelete) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAppConfigDelete(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("appConfigDelete").getParam().Audit3ParamFn(entry)
					RequestField("deletedAppConfigIds", v.DeletedAppConfigIds).getParam().Audit3ParamFn(entry)
					RequestField("deleteAppConfigDescription", v.DeleteAppConfigDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAppConfigSearch(ctx context.Context, v v2.AppConfigSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAppConfigSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("appConfigSearch").getParam().Audit3ParamFn(entry)
					RequestField("appConfigSearchQuery", v.AppConfigSearchQuery).getParam().Audit3ParamFn(entry)
					ResultField("appConfigSearchResults", v.AppConfigSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMonitorRun(ctx context.Context, v v2.MonitorRun) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMonitorRun(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("monitorRun").getParam().Audit3ParamFn(entry)
					RequestField("runMonitorTargets", v.RunMonitorTargets).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMonitorCreate(ctx context.Context, v v2.MonitorCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMonitorCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("monitorCreate").getParam().Audit3ParamFn(entry)
					RequestField("createdMonitorDescription", v.CreatedMonitorDescription).getParam().Audit3ParamFn(entry)
					ResultField("createdMonitorResources", v.CreatedMonitorResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMonitorDelete(ctx context.Context, v v2.MonitorDelete) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMonitorDelete(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("monitorDelete").getParam().Audit3ParamFn(entry)
					RequestField("deletedMonitorResources", v.DeletedMonitorResources).getParam().Audit3ParamFn(entry)
					RequestField("deletedMonitorDescription", v.DeletedMonitorDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMonitorUpdate(ctx context.Context, v v2.MonitorUpdate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMonitorUpdate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("monitorUpdate").getParam().Audit3ParamFn(entry)
					RequestField("updatedMonitorResources", v.UpdatedMonitorResources).getParam().Audit3ParamFn(entry)
					RequestField("updatedMonitorDescription", v.UpdatedMonitorDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMonitorAccess(ctx context.Context, v v2.MonitorAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMonitorAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("monitorAccess").getParam().Audit3ParamFn(entry)
					RequestField("accessedMonitorResources", v.AccessedMonitorResources).getParam().Audit3ParamFn(entry)
					RequestField("accessedMonitorDescription", v.AccessedMonitorDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitMonitorSearch(ctx context.Context, v v2.MonitorSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromMonitorSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("monitorSearch").getParam().Audit3ParamFn(entry)
					RequestField("monitorSearchQuery", v.MonitorSearchQuery).getParam().Audit3ParamFn(entry)
					ResultField("monitorSearchResults", v.MonitorSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitLogicCreate(ctx context.Context, v v2.LogicCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromLogicCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("logicCreate").getParam().Audit3ParamFn(entry)
					ResultField("createdLogicResources", v.CreatedLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitLogicUpdate(ctx context.Context, v v2.LogicUpdate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromLogicUpdate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("logicUpdate").getParam().Audit3ParamFn(entry)
					RequestField("updatedLogicResources", v.UpdatedLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitLogicAccess(ctx context.Context, v v2.LogicAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromLogicAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("logicAccess").getParam().Audit3ParamFn(entry)
					RequestField("accessedLogicResources", v.AccessedLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitLogicDelete(ctx context.Context, v v2.LogicDelete) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromLogicDelete(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("logicDelete").getParam().Audit3ParamFn(entry)
					RequestField("deletedLogicResources", v.DeletedLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitLogicSearch(ctx context.Context, v v2.LogicSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromLogicSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("logicSearch").getParam().Audit3ParamFn(entry)
					RequestField("logicSearchQuery", v.LogicSearchQuery).getParam().Audit3ParamFn(entry)
					ResultField("logicSearchResults", v.LogicSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestCreate(ctx context.Context, v v2.RequestCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestCreate").getParam().Audit3ParamFn(entry)
					RequestField("createdRequestAffectedResources", v.CreatedRequestAffectedResources).getParam().Audit3ParamFn(entry)
					RequestField("createdRequestDescription", v.CreatedRequestDescription).getParam().Audit3ParamFn(entry)
					ResultField("createdRequestIds", v.CreatedRequestIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestAccess(ctx context.Context, v v2.RequestAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestAccess").getParam().Audit3ParamFn(entry)
					RequestField("accessedRequestIds", v.AccessedRequestIds).getParam().Audit3ParamFn(entry)
					RequestField("accessedRequestDescription", v.AccessedRequestDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestSearch(ctx context.Context, v v2.RequestSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestSearch").getParam().Audit3ParamFn(entry)
					RequestField("requestSearchQuery", v.RequestSearchQuery).getParam().Audit3ParamFn(entry)
					ResultField("requestSearchResults", v.RequestSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestUpdate(ctx context.Context, v v2.RequestUpdate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestUpdate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestUpdate").getParam().Audit3ParamFn(entry)
					RequestField("updatedRequestIds", v.UpdatedRequestIds).getParam().Audit3ParamFn(entry)
					RequestField("updatedRequestDescription", v.UpdatedRequestDescription).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestApprove(ctx context.Context, v v2.RequestApprove) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestApprove(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestApprove").getParam().Audit3ParamFn(entry)
					RequestField("approvedRequestIds", v.ApprovedRequestIds).getParam().Audit3ParamFn(entry)
					RequestField("approveRequestUserId", v.ApproveRequestUserId).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestDisapprove(ctx context.Context, v v2.RequestDisapprove) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestDisapprove(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestDisapprove").getParam().Audit3ParamFn(entry)
					RequestField("disapprovedRequestIds", v.DisapprovedRequestIds).getParam().Audit3ParamFn(entry)
					RequestField("disapproveRequestUserId", v.DisapproveRequestUserId).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestExecute(ctx context.Context, v v2.RequestExecute) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestExecute(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestExecute").getParam().Audit3ParamFn(entry)
					RequestField("executedRequestIds", v.ExecutedRequestIds).getParam().Audit3ParamFn(entry)
					ResultField("executeRequestAffectedResources", v.ExecuteRequestAffectedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRequestCancel(ctx context.Context, v v2.RequestCancel) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRequestCancel(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("requestCancel").getParam().Audit3ParamFn(entry)
					RequestField("canceledRequestIds", v.CanceledRequestIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitManagementUsers(ctx context.Context, v v2.ManagementUsers) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromManagementUsers(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("managementUsers").getParam().Audit3ParamFn(entry)
					RequestField("managedUserIds", v.ManagedUserIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitManagementGroups(ctx context.Context, v v2.ManagementGroups) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromManagementGroups(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("managementGroups").getParam().Audit3ParamFn(entry)
					RequestField("groupPatches", v.GroupPatches).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitManagementMarkings(ctx context.Context, v v2.ManagementMarkings) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromManagementMarkings(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("managementMarkings").getParam().Audit3ParamFn(entry)
					RequestField("markingPatches", v.MarkingPatches).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitManagementPermissions(ctx context.Context, v v2.ManagementPermissions) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromManagementPermissions(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("managementPermissions").getParam().Audit3ParamFn(entry)
					RequestField("resourcesWithPermissionsChanges", v.ResourcesWithPermissionsChanges).getParam().Audit3ParamFn(entry)
					RequestField("permissionChangeContext", v.PermissionChangeContext).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitManagementTokens(ctx context.Context, v v2.ManagementTokens) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromManagementTokens(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("managementTokens").getParam().Audit3ParamFn(entry)
					RequestField("managedTokens", v.ManagedTokens).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAuthenticationCheck(ctx context.Context, v v2.AuthenticationCheck) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAuthenticationCheck(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("authenticationCheck").getParam().Audit3ParamFn(entry)
					RequestField("authenticationCheckTargets", v.AuthenticationCheckTargets).getParam().Audit3ParamFn(entry)
					ResultField("authenticationCheckResult", v.AuthenticationCheckResult).getParam().Audit3ParamFn(entry)
					ResultField("authenticationCheckResultMessage", v.AuthenticationCheckResultMessage).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAuthorizationCheck(ctx context.Context, v v2.AuthorizationCheck) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAuthorizationCheck(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("authorizationCheck").getParam().Audit3ParamFn(entry)
					RequestField("authorizationCheckTargets", v.AuthorizationCheckTargets).getParam().Audit3ParamFn(entry)
					RequestField("authorizationCheckOperations", v.AuthorizationCheckOperations).getParam().Audit3ParamFn(entry)
					ResultField("authorizationCheckSucceededTargets", v.AuthorizationCheckSucceededTargets).getParam().Audit3ParamFn(entry)
					ResultField("authorizationCheckFailedTargets", v.AuthorizationCheckFailedTargets).getParam().Audit3ParamFn(entry)
					ResultField("authorizationCheckResultMessage", v.AuthorizationCheckResultMessage).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitUserLogin(ctx context.Context, v v2.UserLogin) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromUserLogin(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("userLogin").getParam().Audit3ParamFn(entry)
					ResultField("loginUserId", v.LoginUserId).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitUserLogout(ctx context.Context, v v2.UserLogout) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromUserLogout(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("userLogout").getParam().Audit3ParamFn(entry)
					RequestField("logoutUserId", v.LogoutUserId).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitTokenGeneration(ctx context.Context, v v2.TokenGeneration) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromTokenGeneration(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("tokenGeneration").getParam().Audit3ParamFn(entry)
					RequestField("generateTokensDescription", v.GenerateTokensDescription).getParam().Audit3ParamFn(entry)
					ResultField("generatedTokens", v.GeneratedTokens).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitTokenRevoke(ctx context.Context, v v2.TokenRevoke) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromTokenRevoke(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("tokenRevoke").getParam().Audit3ParamFn(entry)
					RequestField("revokeTokensDescription", v.RevokeTokensDescription).getParam().Audit3ParamFn(entry)
					ResultField("revokedTokens", v.RevokedTokens).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitTokenAccess(ctx context.Context, v v2.TokenAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromTokenAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("tokenAccess").getParam().Audit3ParamFn(entry)
					ResultField("accessedTokens", v.AccessedTokens).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOauth2InitiateAuthFlow(ctx context.Context, v v2.Oauth2InitiateAuthFlow) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOauth2InitiateAuthFlow(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("oauth2InitiateAuthFlow").getParam().Audit3ParamFn(entry)
					RequestField("oauth2InitiateAuthFlowUser", v.Oauth2InitiateAuthFlowUser).getParam().Audit3ParamFn(entry)
					RequestField("oauth2InitiateAuthClientId", v.Oauth2InitiateAuthClientId).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAssetFileLoad(ctx context.Context, v v2.AssetFileLoad) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAssetFileLoad(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("assetFileLoad").getParam().Audit3ParamFn(entry)
					RequestField("requestMavenCoordinate", v.RequestMavenCoordinate).getParam().Audit3ParamFn(entry)
					ResultField("responseMavenCoordinate", v.ResponseMavenCoordinate).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitContainerLaunch(ctx context.Context, v v2.ContainerLaunch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromContainerLaunch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("containerLaunch").getParam().Audit3ParamFn(entry)
					RequestField("requestedContainerIdsToLaunch", v.RequestedContainerIdsToLaunch).getParam().Audit3ParamFn(entry)
					ResultField("launchedContainerIds", v.LaunchedContainerIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitContainerLoad(ctx context.Context, v v2.ContainerLoad) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromContainerLoad(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("containerLoad").getParam().Audit3ParamFn(entry)
					RequestField("requestedContainerLoadIds", v.RequestedContainerLoadIds).getParam().Audit3ParamFn(entry)
					ResultField("loadedContainerLoadIds", v.LoadedContainerLoadIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitContainerSearch(ctx context.Context, v v2.ContainerSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromContainerSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("containerSearch").getParam().Audit3ParamFn(entry)
					RequestField("containerSearchQuery", v.ContainerSearchQuery).getParam().Audit3ParamFn(entry)
					ResultField("containerSearchResults", v.ContainerSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitContainerStop(ctx context.Context, v v2.ContainerStop) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromContainerStop(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("containerStop").getParam().Audit3ParamFn(entry)
					RequestField("stoppedContainerIds", v.StoppedContainerIds).getParam().Audit3ParamFn(entry)
					RequestField("containerStopReason", v.ContainerStopReason).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitInfraLogsAccess(ctx context.Context, v v2.InfraLogsAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromInfraLogsAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("infraLogsAccess").getParam().Audit3ParamFn(entry)
					RequestField("infraLogsAccessTarget", v.InfraLogsAccessTarget).getParam().Audit3ParamFn(entry)
					ResultField("infraLogsAccessRequestId", v.InfraLogsAccessRequestId).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitCreateInfra(ctx context.Context, v v2.CreateInfra) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromCreateInfra(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("createInfra").getParam().Audit3ParamFn(entry)
					RequestField("createInfraTargets", v.CreateInfraTargets).getParam().Audit3ParamFn(entry)
					ResultField("createdInfraResources", v.CreatedInfraResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitConfigureInfra(ctx context.Context, v v2.ConfigureInfra) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromConfigureInfra(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("configureInfra").getParam().Audit3ParamFn(entry)
					RequestField("configureInfraTargets", v.ConfigureInfraTargets).getParam().Audit3ParamFn(entry)
					ResultField("configureInfraRequestId", v.ConfigureInfraRequestId).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitReviewInfraAction(ctx context.Context, v v2.ReviewInfraAction) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromReviewInfraAction(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("reviewInfraAction").getParam().Audit3ParamFn(entry)
					RequestField("reviewInfraActionRequestId", v.ReviewInfraActionRequestId).getParam().Audit3ParamFn(entry)
					RequestField("reviewInfraActionUser", v.ReviewInfraActionUser).getParam().Audit3ParamFn(entry)
					ResultField("reviewInfraActionWasApproved", v.ReviewInfraActionWasApproved).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitRestartInfra(ctx context.Context, v v2.RestartInfra) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromRestartInfra(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("restartInfra").getParam().Audit3ParamFn(entry)
					RequestField("restartedResources", v.RestartedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitUpgradeInfra(ctx context.Context, v v2.UpgradeInfra) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromUpgradeInfra(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("upgradeInfra").getParam().Audit3ParamFn(entry)
					RequestField("upgradedResources", v.UpgradedResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyDataLoad(ctx context.Context, v v2.OntologyDataLoad) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyDataLoad(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyDataLoad").getParam().Audit3ParamFn(entry)
					RequestField("ontologyDataLoadContext", v.OntologyDataLoadContext).getParam().Audit3ParamFn(entry)
					RequestField("requestedOntologyDataResources", v.RequestedOntologyDataResources).getParam().Audit3ParamFn(entry)
					ResultField("loadedOntologyDataResources", v.LoadedOntologyDataResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyDataSearch(ctx context.Context, v v2.OntologyDataSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyDataSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyDataSearch").getParam().Audit3ParamFn(entry)
					RequestField("ontologyDataSearchContext", v.OntologyDataSearchContext).getParam().Audit3ParamFn(entry)
					RequestField("searchedOntologyLogicResources", v.SearchedOntologyLogicResources).getParam().Audit3ParamFn(entry)
					ResultField("ontologyDataSearchResults", v.OntologyDataSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyDataTransform(ctx context.Context, v v2.OntologyDataTransform) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyDataTransform(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyDataTransform").getParam().Audit3ParamFn(entry)
					RequestField("ontologyDataTransformTargets", v.OntologyDataTransformTargets).getParam().Audit3ParamFn(entry)
					RequestField("ontologyDataTransformContext", v.OntologyDataTransformContext).getParam().Audit3ParamFn(entry)
					RequestField("ontologyDataTransformDescription", v.OntologyDataTransformDescription).getParam().Audit3ParamFn(entry)
					ResultField("transformedOntologyDataResources", v.TransformedOntologyDataResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicAccess(ctx context.Context, v v2.OntologyLogicAccess) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyLogicAccess(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyLogicAccess").getParam().Audit3ParamFn(entry)
					RequestField("requestedOntologyLogicResources", v.RequestedOntologyLogicResources).getParam().Audit3ParamFn(entry)
					ResultField("loadedOntologyLogicResources", v.LoadedOntologyLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicCreate(ctx context.Context, v v2.OntologyLogicCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyLogicCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyLogicCreate").getParam().Audit3ParamFn(entry)
					RequestField("createOntologyLogicContext", v.CreateOntologyLogicContext).getParam().Audit3ParamFn(entry)
					ResultField("createdOntologyLogicResources", v.CreatedOntologyLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicDelete(ctx context.Context, v v2.OntologyLogicDelete) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyLogicDelete(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyLogicDelete").getParam().Audit3ParamFn(entry)
					RequestField("deleteOntologyLogicContext", v.DeleteOntologyLogicContext).getParam().Audit3ParamFn(entry)
					ResultField("deletedOntologyLogicResources", v.DeletedOntologyLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyLogicUpdate(ctx context.Context, v v2.OntologyLogicUpdate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyLogicUpdate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyLogicUpdate").getParam().Audit3ParamFn(entry)
					RequestField("updateOntologyLogicContext", v.UpdateOntologyLogicContext).getParam().Audit3ParamFn(entry)
					ResultField("updatedOntologyLogicResources", v.UpdatedOntologyLogicResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataLoad(ctx context.Context, v v2.OntologyMetaDataLoad) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyMetaDataLoad(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyMetaDataLoad").getParam().Audit3ParamFn(entry)
					RequestField("requestedOntologyMetaDataResources", v.RequestedOntologyMetaDataResources).getParam().Audit3ParamFn(entry)
					ResultField("loadedOntologyMetaDataResources", v.LoadedOntologyMetaDataResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataCreate(ctx context.Context, v v2.OntologyMetaDataCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyMetaDataCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyMetaDataCreate").getParam().Audit3ParamFn(entry)
					ResultField("createdOntologyMetaDataResources", v.CreatedOntologyMetaDataResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataDelete(ctx context.Context, v v2.OntologyMetaDataDelete) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyMetaDataDelete(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyMetaDataDelete").getParam().Audit3ParamFn(entry)
					RequestField("deletedOntologyMetaDataResources", v.DeletedOntologyMetaDataResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataUpdate(ctx context.Context, v v2.OntologyMetaDataUpdate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyMetaDataUpdate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyMetaDataUpdate").getParam().Audit3ParamFn(entry)
					RequestField("updatedOntologyMetaDataResources", v.UpdatedOntologyMetaDataResources).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOntologyMetaDataSearch(ctx context.Context, v v2.OntologyMetaDataSearch) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOntologyMetaDataSearch(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("ontologyMetaDataSearch").getParam().Audit3ParamFn(entry)
					RequestField("ontologyMetaDataSearchedResources", v.OntologyMetaDataSearchedResources).getParam().Audit3ParamFn(entry)
					RequestField("ontologyMetaDataSearchContext", v.OntologyMetaDataSearchContext).getParam().Audit3ParamFn(entry)
					ResultField("ontologyMetaDataSearchResults", v.OntologyMetaDataSearchResults).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitSecretCreate(ctx context.Context, v v2.SecretCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromSecretCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("secretCreate").getParam().Audit3ParamFn(entry)
					RequestField("createdSecretType", v.CreatedSecretType).getParam().Audit3ParamFn(entry)
					ResultField("createdSecretIdentifiers", v.CreatedSecretIdentifiers).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitSecretUse(ctx context.Context, v v2.SecretUse) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromSecretUse(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("secretUse").getParam().Audit3ParamFn(entry)
					RequestField("usedSecretOperation", v.UsedSecretOperation).getParam().Audit3ParamFn(entry)
					RequestField("usedSecretIdentifiers", v.UsedSecretIdentifiers).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitSecretLoad(ctx context.Context, v v2.SecretLoad) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromSecretLoad(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("secretLoad").getParam().Audit3ParamFn(entry)
					RequestField("loadedSecretIdentifiers", v.LoadedSecretIdentifiers).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitSecretDeprecate(ctx context.Context, v v2.SecretDeprecate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromSecretDeprecate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("secretDeprecate").getParam().Audit3ParamFn(entry)
					RequestField("deprecatedSecretIdentifier", v.DeprecatedSecretIdentifier).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitOnBehalfOf(ctx context.Context, v v2.OnBehalfOf) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromOnBehalfOf(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("onBehalfOf").getParam().Audit3ParamFn(entry)
					RequestField("onBehalfOfUserIds", v.OnBehalfOfUserIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitInApplicationContext(ctx context.Context, v v2.InApplicationContext) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromInApplicationContext(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("inApplicationContext").getParam().Audit3ParamFn(entry)
					RequestField("applicationRid", v.ApplicationRid).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitInEnrollmentContext(ctx context.Context, v v2.InEnrollmentContext) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromInEnrollmentContext(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("inEnrollmentContext").getParam().Audit3ParamFn(entry)
					RequestField("enrollmentRids", v.EnrollmentRids).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitInternal(ctx context.Context, v category.Internal) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("internal").getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitUserJustify(ctx context.Context, v v2.UserJustify) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromUserJustify(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("userJustify").getParam().Audit3ParamFn(entry)
					RequestField("userJustifyId", v.UserJustifyId).getParam().Audit3ParamFn(entry)
					RequestField("userJustification", v.UserJustification).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitPassThrough(ctx context.Context, v v2.PassThrough) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromPassThrough(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("passThrough").getParam().Audit3ParamFn(entry)
					RequestField("passThroughRequestParams", v.PassThroughRequestParams).getParam().Audit3ParamFn(entry)
					ResultField("passThroughResponseParams", v.PassThroughResponseParams).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitLlmInference(ctx context.Context, v v2.LlmInference) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromLlmInference(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("llmInference").getParam().Audit3ParamFn(entry)
					RequestField("llmInferenceContext", v.LlmInferenceContext).getParam().Audit3ParamFn(entry)
					RequestField("llmInferenceInputs", v.LlmInferenceInputs).getParam().Audit3ParamFn(entry)
					ResultField("llmInferenceResponses", v.LlmInferenceResponses).getParam().Audit3ParamFn(entry)
					ResultField("llmInferenceResponseContext", v.LlmInferenceResponseContext).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitLlmRoute(ctx context.Context, v v2.LlmRoute) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromLlmRoute(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("llmRoute").getParam().Audit3ParamFn(entry)
					RequestField("llmRouteRequest", v.LlmRouteRequest).getParam().Audit3ParamFn(entry)
					ResultField("llmRouteResponse", v.LlmRouteResponse).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAuditDataTransform(ctx context.Context, v v2.AuditDataTransform) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAuditDataTransform(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("auditDataTransform").getParam().Audit3ParamFn(entry)
					RequestField("transformTarget", v.TransformTarget).getParam().Audit3ParamFn(entry)
					RequestField("transformDescriptions", v.TransformDescriptions).getParam().Audit3ParamFn(entry)
					ResultField("transformDestination", v.TransformDestination).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitAuditDataShareCreate(ctx context.Context, v v2.AuditDataShareCreate) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromAuditDataShareCreate(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("auditDataShareCreate").getParam().Audit3ParamFn(entry)
					RequestField("shareTargets", v.ShareTargets).getParam().Audit3ParamFn(entry)
					ResultField("shareIds", v.ShareIds).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitApiGatewayRequest(ctx context.Context, v v2.ApiGatewayRequest) (Param, error) {
	return &paramStruct{
		param: auditloginternal.Audit3Param{
			Param: auditloginternal.AuditParam{
				Audit2ParamFn: func(entry wlog.LogEntry) {
					audit2log.ApplyParam(audit2log.RequestParam("_category", v2.NewAuditCategoryV2FromApiGatewayRequest(v)), entry)
				},
				Audit3ParamFn: func(entry wlog.LogEntry) {
					categories("apiGatewayRequest").getParam().Audit3ParamFn(entry)
					RequestField("operationNames", v.OperationNames).getParam().Audit3ParamFn(entry)
				},
			},
		},
	}, nil
}

func (a *auditCategoryV2Visitor) VisitUnknown(ctx context.Context, typ string) (Param, error) {
	return nil, werror.Error("unhandled type", werror.SafeParam("type", typ))
}
