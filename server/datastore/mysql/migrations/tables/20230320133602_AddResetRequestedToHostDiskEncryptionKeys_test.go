package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230320133602(t *testing.T) {
	db := applyUpToPrev(t)
	_, err := db.Exec(`INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, decryptable) VALUES (1, 'asdf', 0)
`)
	require.NoError(t, err)

	applyNext(t, db)

	var decryptable, resetRequested *bool
	var base64Encrypted string
	err = db.QueryRow(`SELECT base64_encrypted, decryptable, reset_requested FROM host_disk_encryption_keys WHERE host_id = 1`).Scan(&base64Encrypted, &decryptable, &resetRequested)
	require.NoError(t, err)
	require.Equal(t, "asdf", base64Encrypted)
	require.False(t, *decryptable)
	require.False(t, *resetRequested)

	_, err = db.Exec(`INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, decryptable, reset_requested) VALUES (2, 'zxy', 1, 1)`)
	require.NoError(t, err)

	err = db.QueryRow(`SELECT base64_encrypted, decryptable, reset_requested FROM host_disk_encryption_keys WHERE host_id = 2`).Scan(&base64Encrypted, &decryptable, &resetRequested)
	require.NoError(t, err)
	require.Equal(t, "zxy", base64Encrypted)
	require.True(t, *decryptable)
	require.True(t, *resetRequested)
}
