// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category package.

//go:build go1.23

package category

import (
	"github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category"
)

type AuditCategoryWithT[T any] = category.AuditCategoryWithT[T]

type AuditCategoryVisitorWithT[T any] = category.AuditCategoryVisitorWithT[T]
