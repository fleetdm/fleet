package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260825094053(t *testing.T) {
	db := applyUpToPrev(t)

	const insertStmt = `INSERT INTO host_disks (host_id, encrypted, bitlocker_protection_status) VALUES (?, ?, ?)`
	execNoErr(t, db, insertStmt, 1, true, 0)

	applyNext(t, db)

	// Existing rows survive and default to NULL.
	var protectionError *string
	err := db.Get(&protectionError, `SELECT bitlocker_protection_error FROM host_disks WHERE host_id = ?`, 1)
	require.NoError(t, err)
	require.Nil(t, protectionError)

	// The new column is writable and readable.
	execNoErr(t, db, `UPDATE host_disks SET bitlocker_protection_error = ? WHERE host_id = ?`,
		"policy does not permit the use of TPM-only at startup", 1)
	err = db.Get(&protectionError, `SELECT bitlocker_protection_error FROM host_disks WHERE host_id = ?`, 1)
	require.NoError(t, err)
	require.NotNil(t, protectionError)
	require.Equal(t, "policy does not permit the use of TPM-only at startup", *protectionError)
}
