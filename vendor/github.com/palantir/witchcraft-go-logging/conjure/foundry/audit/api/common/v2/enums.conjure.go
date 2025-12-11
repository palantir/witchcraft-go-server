// Deprecated: exists only for backwards compatibility.
// New usages should use the github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/common/v2 package.

package v2

import (
	v2 "github.com/palantir/witchcraft-go-logging/conjure/audit-api/foundry/audit/api/common/v2"
)

type SecretOperation = v2.SecretOperation

type SecretOperation_Value = v2.SecretOperation_Value

const (
	SecretOperation_HASH    = v2.SecretOperation_HASH
	SecretOperation_ENCRYPT = v2.SecretOperation_ENCRYPT
	SecretOperation_DECRYPT = v2.SecretOperation_DECRYPT
	SecretOperation_UNKNOWN = v2.SecretOperation_UNKNOWN
)

// SecretOperation_Values returns all known variants of SecretOperation.
func SecretOperation_Values() []SecretOperation_Value {
	return v2.SecretOperation_Values()
}

func New_SecretOperation(value SecretOperation_Value) SecretOperation {
	return v2.New_SecretOperation(value)
}

type SecretType = v2.SecretType

type SecretType_Value = v2.SecretType_Value

const (
	SecretType_PEPPER              = v2.SecretType_PEPPER
	SecretType_ASYMMETRIC_KEY_PAIR = v2.SecretType_ASYMMETRIC_KEY_PAIR
	SecretType_SYMMETRIC_KEY       = v2.SecretType_SYMMETRIC_KEY
	SecretType_UNKNOWN             = v2.SecretType_UNKNOWN
)

// SecretType_Values returns all known variants of SecretType.
func SecretType_Values() []SecretType_Value {
	return v2.SecretType_Values()
}

func New_SecretType(value SecretType_Value) SecretType {
	return v2.New_SecretType(value)
}
