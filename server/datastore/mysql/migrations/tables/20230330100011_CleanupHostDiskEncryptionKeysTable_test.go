package tables

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/require"
)

func TestUp_20230330100011(t *testing.T) {
	db := applyUpToPrev(t)

	//
	// Insert data to test the migration
	//
	// ...
	insertHostDisksQuery := `INSERT INTO host_disks (host_id, encrypted) VALUES (?, ?)`

	execNoErr(t, db, insertHostDisksQuery, 1, 0)
	execNoErr(t, db, insertHostDisksQuery, 2, 0)
	execNoErr(t, db, insertHostDisksQuery, 3, 1)

	insertHostDiskEncryptionKeysQuery := `
	INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted)
	VALUES (?, ?)`

	execNoErr(t, db, insertHostDiskEncryptionKeysQuery, 1, "")
	execNoErr(t, db, insertHostDiskEncryptionKeysQuery, 2, "")
	execNoErr(t, db, insertHostDiskEncryptionKeysQuery, 3, "")

	// Apply current migration.
	applyNext(t, db)

	//
	// Check data, insert new entries, e.g. to verify migration is safe.
	//
	// ...
	selectHostDiskEncryptionKeysQuery := `SELECT host_id from host_disk_encryption_keys`

	var rows []fleet.HostDiskEncryptionKey
	require.NoError(t, db.SelectContext(context.Background(), &rows, selectHostDiskEncryptionKeysQuery))
	require.Len(t, rows, 1)
	require.Equal(t, rows[0].HostID, uint(3))
}
