package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261019151819(t *testing.T) {
	db := applyUpToPrev(t)

	newHost := func(name, platform string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, hostname, uuid, platform) VALUES (?, ?, ?, ?, ?)`,
			name, name, name, name, platform)
	}
	windowsWithKey := newHost("win-key", "windows")
	windowsWithError := newHost("win-error", "windows")
	windowsNoKeyRow := newHost("win-none", "windows")
	darwinWithKey := newHost("mac-key", "darwin")
	linuxWithKey := newHost("linux-key", "ubuntu")
	const deletedHost = 9999

	insertKey := func(hostID int64, key string) {
		execNoErr(t, db, `INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, client_error) VALUES (?, ?, ?)`,
			hostID, key, "")
	}
	insertKey(windowsWithKey, "legacy-windows-blob")
	// updated_at drives the host's disk encryption status and auto-updates, so the backfill must not move it.
	const backdated = "2026-01-02 03:04:05.000000"
	execNoErr(t, db, `UPDATE host_disk_encryption_keys SET updated_at = ? WHERE host_id = ?`, backdated, windowsWithKey)
	execNoErr(t, db, `INSERT INTO host_disk_encryption_keys (host_id, base64_encrypted, client_error) VALUES (?, '', 'bitlocker error')`,
		windowsWithError)
	insertKey(darwinWithKey, "filevault-blob")
	insertKey(linuxWithKey, "luks-blob")
	insertKey(deletedHost, "orphan-blob")

	insertArchived := func(hostID int64, key string) int64 {
		return execNoErrLastID(t, db, `INSERT INTO host_disk_encryption_keys_archive (host_id, base64_encrypted) VALUES (?, ?)`, hostID, key)
	}
	windowsArchivedOld := insertArchived(windowsWithKey, "legacy-windows-blob-old")
	windowsArchivedCurrent := insertArchived(windowsWithKey, "legacy-windows-blob")
	windowsArchivedNoLiveRow := insertArchived(windowsNoKeyRow, "legacy-windows-blob-gone")
	darwinArchived := insertArchived(darwinWithKey, "filevault-blob")
	orphanArchived := insertArchived(deletedHost, "orphan-blob")

	applyNext(t, db)

	marked := func(table, idColumn string, id int64) bool {
		var v bool
		require.NoError(t, db.Get(&v, `SELECT wstep_encrypted FROM `+table+` WHERE `+idColumn+` = ?`, id))
		return v
	}

	// Only Windows rows holding a key are legacy WSTEP blobs. Error-only rows, other platforms and rows whose host
	// is gone keep the default.
	require.True(t, marked("host_disk_encryption_keys", "host_id", windowsWithKey))
	require.False(t, marked("host_disk_encryption_keys", "host_id", windowsWithError))
	require.False(t, marked("host_disk_encryption_keys", "host_id", darwinWithKey))
	require.False(t, marked("host_disk_encryption_keys", "host_id", linuxWithKey))
	require.False(t, marked("host_disk_encryption_keys", "host_id", deletedHost))

	var updatedAtUnchanged bool
	require.NoError(t, db.QueryRow(`SELECT updated_at = ? FROM host_disk_encryption_keys WHERE host_id = ?`, backdated, windowsWithKey).Scan(&updatedAtUnchanged))
	require.True(t, updatedAtUnchanged, "marking a row must not count as a key change")

	// Every archived Windows key is marked, whether or not the host still has a live row.
	require.True(t, marked("host_disk_encryption_keys_archive", "id", windowsArchivedOld))
	require.True(t, marked("host_disk_encryption_keys_archive", "id", windowsArchivedCurrent))
	require.True(t, marked("host_disk_encryption_keys_archive", "id", windowsArchivedNoLiveRow))
	require.False(t, marked("host_disk_encryption_keys_archive", "id", darwinArchived))
	require.False(t, marked("host_disk_encryption_keys_archive", "id", orphanArchived))

	// Rerunning the backfill on an already migrated schema changes nothing.
	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261019151819(tx))
	require.NoError(t, tx.Commit())
	require.True(t, marked("host_disk_encryption_keys", "host_id", windowsWithKey))
	require.False(t, marked("host_disk_encryption_keys", "host_id", darwinWithKey))
}
