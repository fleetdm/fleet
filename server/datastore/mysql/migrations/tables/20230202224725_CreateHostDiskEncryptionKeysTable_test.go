package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20230202224725(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	var (
		decryptable *bool
		key         string
	)

	insertStmt := `
          INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, decryptable)
	  VALUES (?, ?, ?)
	`

	selectStmt := `
	  SELECT base64_encrypted, decryptable
	  FROM host_disk_encryption_keys
	  WHERE host_id = ?
	`

	_, err := db.Exec(insertStmt, 1, "ABCDEFG", true)
	require.NoError(t, err)
	_, err = db.Exec(insertStmt, 2, "XYZ", false)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted) VALUES (?, ?)`, 3, "RANDOM")
	require.NoError(t, err)

	err = db.QueryRow(selectStmt, 1).Scan(&key, &decryptable)
	require.NoError(t, err)
	require.Equal(t, "ABCDEFG", key)
	require.True(t, *decryptable)

	err = db.QueryRow(selectStmt, 2).Scan(&key, &decryptable)
	require.NoError(t, err)
	require.Equal(t, "XYZ", key)
	require.False(t, *decryptable)

	err = db.QueryRow(selectStmt, 3).Scan(&key, &decryptable)
	require.NoError(t, err)
	require.Equal(t, "RANDOM", key)
	require.Nil(t, decryptable)

}
