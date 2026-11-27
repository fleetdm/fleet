package mysql

import "github.com/fleetdm/fleet/v4/server/platform/crypto"

func encrypt(plainText []byte, key string) ([]byte, error) {
	return crypto.Encrypt(plainText, key)
}

func decrypt(encrypted []byte, key string) ([]byte, error) {
	return crypto.Decrypt(encrypted, key)
}

// decryptOrPassthrough tries to decrypt versioned data; if the data
// does not have the version prefix it is returned as legacy cleartext.
// Unlike the conditional access path, MDM tokens may exist on many
// devices, so they use passthrough until a startup backfill is added.
func decryptOrPassthrough(data []byte, key string) []byte {
	if len(key) == 0 || len(data) == 0 {
		return data
	}
	decrypted, err := decrypt(data, key)
	if err != nil {
		return data
	}
	return decrypted
}
