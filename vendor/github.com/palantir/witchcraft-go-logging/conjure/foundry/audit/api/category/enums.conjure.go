// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category package.

package category

import (
	"github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category"
)

type SystemManagementAction = category.SystemManagementAction

type SystemManagementAction_Value = category.SystemManagementAction_Value

const (
	SystemManagementAction_MODIFY_ENVIRONMENT_CONFIG = category.SystemManagementAction_MODIFY_ENVIRONMENT_CONFIG
	SystemManagementAction_MODIFY_APPLICATION_CONFIG = category.SystemManagementAction_MODIFY_APPLICATION_CONFIG
	SystemManagementAction_VIEW_ENVIRONMENT_CONFIG   = category.SystemManagementAction_VIEW_ENVIRONMENT_CONFIG
	SystemManagementAction_VIEW_APPLICATION_CONFIG   = category.SystemManagementAction_VIEW_APPLICATION_CONFIG
	SystemManagementAction_UNKNOWN                   = category.SystemManagementAction_UNKNOWN
)

func SystemManagementAction_Values() []SystemManagementAction_Value {
	return category.SystemManagementAction_Values()
}

func New_SystemManagementAction(value SystemManagementAction_Value) SystemManagementAction {
	return category.New_SystemManagementAction(value)
}

type TokenType = category.TokenType

type TokenType_Value = category.TokenType_Value

const (
	TokenType_SESSION             = category.TokenType_SESSION
	TokenType_AUTHORIZATION_CODE  = category.TokenType_AUTHORIZATION_CODE
	TokenType_CLIENT_CREDENTIALS  = category.TokenType_CLIENT_CREDENTIALS
	TokenType_RESTRICTED_TOKEN    = category.TokenType_RESTRICTED_TOKEN
	TokenType_PROXY_TOKEN         = category.TokenType_PROXY_TOKEN
	TokenType_API_TOKEN           = category.TokenType_API_TOKEN
	TokenType_TRUSTED_SCOPE_TOKEN = category.TokenType_TRUSTED_SCOPE_TOKEN
	TokenType_UNKNOWN             = category.TokenType_UNKNOWN
)

func TokenType_Values() []TokenType_Value {
	return category.TokenType_Values()
}

func New_TokenType(value TokenType_Value) TokenType {
	return category.New_TokenType(value)
}

type UserJustificationType = category.UserJustificationType

type UserJustificationType_Value = category.UserJustificationType_Value

const (
	UserJustificationType_RESPONSE        = category.UserJustificationType_RESPONSE
	UserJustificationType_ACKNOWLEDGEMENT = category.UserJustificationType_ACKNOWLEDGEMENT
	UserJustificationType_DROPDOWN        = category.UserJustificationType_DROPDOWN
	UserJustificationType_UNKNOWN         = category.UserJustificationType_UNKNOWN
)

func UserJustificationType_Values() []UserJustificationType_Value {
	return category.UserJustificationType_Values()
}

func New_UserJustificationType(value UserJustificationType_Value) UserJustificationType {
	return category.New_UserJustificationType(value)
}

type UserManagementAction = category.UserManagementAction

type UserManagementAction_Value = category.UserManagementAction_Value

const (
	UserManagementAction_CREDENTIAL_UPDATE       = category.UserManagementAction_CREDENTIAL_UPDATE
	UserManagementAction_GROUP_MEMBERSHIP_CHANGE = category.UserManagementAction_GROUP_MEMBERSHIP_CHANGE
	UserManagementAction_ATTRIBUTE_CHANGE        = category.UserManagementAction_ATTRIBUTE_CHANGE
	UserManagementAction_VIEW_USER               = category.UserManagementAction_VIEW_USER
	UserManagementAction_UNKNOWN                 = category.UserManagementAction_UNKNOWN
)

// UserManagementAction_Values returns all known variants of UserManagementAction.
func UserManagementAction_Values() []UserManagementAction_Value {
	return category.UserManagementAction_Values()
}

func New_UserManagementAction(value UserManagementAction_Value) UserManagementAction {
	return category.New_UserManagementAction(value)
}
