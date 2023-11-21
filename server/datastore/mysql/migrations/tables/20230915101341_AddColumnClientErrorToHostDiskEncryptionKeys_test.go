package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20230915101341(t *testing.T) {
	db := applyUpToPrev(t)
	insertStmt := `
          INSERT INTO host_disk_encryption_keys
            (host_id, base64_encrypted)
          VALUES
            (?, ?)
	`
	execNoErr(t, db, insertStmt, 1, "test-key")

	applyNext(t, db)

	// retrieve the stored value, verify that the new column is present
	var hdek struct {
		HostID          uint      `db:"host_id"`
		Base64Encrypted string    `db:"base64_encrypted"`
		Decryptable     *bool     `db:"decryptable"`
		CreatedAt       time.Time `db:"created_at"`
		UpdatedAt       time.Time `db:"updated_at"`
		ResetRequested  bool      `db:"reset_requested"`
		ClientError     string    `db:"client_error"`
	}
	err := db.Get(&hdek, "SELECT * FROM host_disk_encryption_keys WHERE host_id = ?", 1)
	require.NoError(t, err)
	require.Equal(t, uint(1), hdek.HostID)
	require.Equal(t, "test-key", hdek.Base64Encrypted)
	require.Nil(t, hdek.Decryptable)
	require.NotZero(t, hdek.CreatedAt)
	require.NotZero(t, hdek.UpdatedAt)
	require.False(t, hdek.ResetRequested)
	require.Equal(t, "", hdek.ClientError)

	insertStmt = `
          INSERT INTO host_disk_encryption_keys
            (host_id, base64_encrypted, client_error)
          VALUES
            (?, ?, ?)
	`
	execNoErr(t, db, insertStmt, 2, "", "test-error")
	err = db.Get(&hdek, "SELECT * FROM host_disk_encryption_keys WHERE host_id = ?", 2)
	require.NoError(t, err)
	require.Equal(t, uint(2), hdek.HostID)
	require.Equal(t, "", hdek.Base64Encrypted)
	require.Nil(t, hdek.Decryptable)
	require.NotZero(t, hdek.CreatedAt)
	require.NotZero(t, hdek.UpdatedAt)
	require.False(t, hdek.ResetRequested)
	require.Equal(t, "test-error", hdek.ClientError)
}
