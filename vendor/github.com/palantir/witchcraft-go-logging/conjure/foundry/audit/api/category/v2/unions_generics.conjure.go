// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category/v2 package.

//go:build go1.23

package v2

import (
	v2 "github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/category/v2"
)

type AuditCategoryV2WithT[T any] = v2.AuditCategoryV2WithT[T]

type AuditCategoryV2VisitorWithT[T any] = v2.AuditCategoryV2VisitorWithT[T]

type ExternalSystemWithT[T any] = v2.ExternalSystemWithT[T]

type ExternalSystemVisitorWithT[T any] = v2.ExternalSystemVisitorWithT[T]
