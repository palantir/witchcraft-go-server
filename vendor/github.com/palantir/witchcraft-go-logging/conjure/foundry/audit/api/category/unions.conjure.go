// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category package.

package category

import (
	"github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category"
)

type AuditCategory = category.AuditCategory

type AuditCategoryVisitor = category.AuditCategoryVisitor

type AuditCategoryVisitorWithContext = category.AuditCategoryVisitorWithContext

func NewAuditCategoryFromAuthenticationCheck(v AuthenticationCheck) AuditCategory {
	return category.NewAuditCategoryFromAuthenticationCheck(v)
}

func NewAuditCategoryFromCodeExecution(v CodeExecution) AuditCategory {
	return category.NewAuditCategoryFromCodeExecution(v)
}

func NewAuditCategoryFromContainerLaunch(v ContainerLaunch) AuditCategory {
	return category.NewAuditCategoryFromContainerLaunch(v)
}

func NewAuditCategoryFromContainerStop(v ContainerStop) AuditCategory {
	return category.NewAuditCategoryFromContainerStop(v)
}

func NewAuditCategoryFromDataCreate(v DataCreate) AuditCategory {
	return category.NewAuditCategoryFromDataCreate(v)
}

func NewAuditCategoryFromDataDelete(v DataDelete) AuditCategory {
	return category.NewAuditCategoryFromDataDelete(v)
}

func NewAuditCategoryFromDataExport(v DataExport) AuditCategory {
	return category.NewAuditCategoryFromDataExport(v)
}

func NewAuditCategoryFromDataImport(v DataImport) AuditCategory {
	return category.NewAuditCategoryFromDataImport(v)
}

func NewAuditCategoryFromDataLoad(v DataLoad) AuditCategory {
	return category.NewAuditCategoryFromDataLoad(v)
}

func NewAuditCategoryFromDataMerge(v DataMerge) AuditCategory {
	return category.NewAuditCategoryFromDataMerge(v)
}

func NewAuditCategoryFromDataPromote(v DataPromote) AuditCategory {
	return category.NewAuditCategoryFromDataPromote(v)
}

func NewAuditCategoryFromDataSearch(v DataSearch) AuditCategory {
	return category.NewAuditCategoryFromDataSearch(v)
}

func NewAuditCategoryFromDataShareCreate(v DataShareCreate) AuditCategory {
	return category.NewAuditCategoryFromDataShareCreate(v)
}

func NewAuditCategoryFromDataShareDisable(v DataShareDisable) AuditCategory {
	return category.NewAuditCategoryFromDataShareDisable(v)
}

func NewAuditCategoryFromDataShare(v DataShare) AuditCategory {
	return category.NewAuditCategoryFromDataShare(v)
}

func NewAuditCategoryFromDataUpdate(v DataUpdate) AuditCategory {
	return category.NewAuditCategoryFromDataUpdate(v)
}

func NewAuditCategoryFromInternal(v Internal) AuditCategory {
	return category.NewAuditCategoryFromInternal(v)
}

func NewAuditCategoryFromLogicCreate(v LogicCreate) AuditCategory {
	return category.NewAuditCategoryFromLogicCreate(v)
}

func NewAuditCategoryFromLogicUpdate(v LogicUpdate) AuditCategory {
	return category.NewAuditCategoryFromLogicUpdate(v)
}

func NewAuditCategoryFromManagementUsers(v ManagementUsers) AuditCategory {
	return category.NewAuditCategoryFromManagementUsers(v)
}

func NewAuditCategoryFromManagementGroups(v ManagementGroups) AuditCategory {
	return category.NewAuditCategoryFromManagementGroups(v)
}

func NewAuditCategoryFromManagementPermissions(v ManagementPermissions) AuditCategory {
	return category.NewAuditCategoryFromManagementPermissions(v)
}

func NewAuditCategoryFromManagementTokens(v ManagementTokens) AuditCategory {
	return category.NewAuditCategoryFromManagementTokens(v)
}

func NewAuditCategoryFromMandatoryControlManagement(v []MandatoryControlManagement) AuditCategory {
	return category.NewAuditCategoryFromMandatoryControlManagement(v)
}

func NewAuditCategoryFromMandatoryControlApplication(v []MandatoryControlApplication) AuditCategory {
	return category.NewAuditCategoryFromMandatoryControlApplication(v)
}

func NewAuditCategoryFromMetaDataAccess(v MetaDataAccess) AuditCategory {
	return category.NewAuditCategoryFromMetaDataAccess(v)
}

func NewAuditCategoryFromSystemManagement(v SystemManagement) AuditCategory {
	return category.NewAuditCategoryFromSystemManagement(v)
}

func NewAuditCategoryFromTokenGeneration(v TokenGeneration) AuditCategory {
	return category.NewAuditCategoryFromTokenGeneration(v)
}

func NewAuditCategoryFromUserJustify(v UserJustify) AuditCategory {
	return category.NewAuditCategoryFromUserJustify(v)
}

func NewAuditCategoryFromUserLogin(v UserLogin) AuditCategory {
	return category.NewAuditCategoryFromUserLogin(v)
}

func NewAuditCategoryFromUserLogout(v UserLogout) AuditCategory {
	return category.NewAuditCategoryFromUserLogout(v)
}
