package mysql

import "github.com/fleetdm/fleet/v4/server/platform/crypto"

// Thin wrappers so callers in this package don't change.

func encrypt(plainText []byte, key string) ([]byte, error) {
	return crypto.Encrypt(plainText, key)
}

func decrypt(encrypted []byte, key string) ([]byte, error) {
	return crypto.Decrypt(encrypted, key)
}

func decryptOrPassthrough(data []byte, key string) []byte {
	return crypto.DecryptOrPassthrough(data, key)
}
