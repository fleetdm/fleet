package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20251016100000(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert test hosts
	host1ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform) VALUES (?, ?, ?, ?)`, "host1", "key1", "uuid1", "darwin")
	host2ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform) VALUES (?, ?, ?, ?)`, "host2", "key2", "uuid2", "darwin")
	host3ID := execNoErrLastID(t, db, `INSERT INTO hosts (osquery_host_id, node_key, uuid, platform) VALUES (?, ?, ?, ?)`, "host3", "key3", "uuid3", "darwin")

	// Insert test SCIM users
	scimUser1ID := execNoErrLastID(t, db, `INSERT INTO scim_users (user_name, given_name, family_name, active) VALUES (?, ?, ?, ?)`, "user1@example.com", "User", "One", 1)
	scimUser2ID := execNoErrLastID(t, db, `INSERT INTO scim_users (user_name, given_name, family_name, active) VALUES (?, ?, ?, ?)`, "user2@example.com", "User", "Two", 1)

	// Insert host_emails with mdm_idp_accounts source
	execNoErr(t, db, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`, host1ID, "user1@example.com", "mdm_idp_accounts")
	execNoErr(t, db, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`, host2ID, "user2@example.com", "mdm_idp_accounts")
	execNoErr(t, db, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`, host3ID, "nomatch@example.com", "mdm_idp_accounts")
	execNoErr(t, db, `INSERT INTO host_emails (host_id, email, source) VALUES (?, ?, ?)`, host1ID, "other@example.com", "other_source")

	// Verify initial state - no host_scim_user mappings exist
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM host_scim_user")
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Apply the migration
	applyNext(t, db)

	// Verify that host_scim_user mappings were created for matching hosts
	err = db.Get(&count, "SELECT COUNT(*) FROM host_scim_user")
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// Verify specific mappings
	var mappings []struct {
		HostID     int64 `db:"host_id"`
		ScimUserID int64 `db:"scim_user_id"`
	}
	err = db.Select(&mappings, "SELECT host_id, scim_user_id FROM host_scim_user ORDER BY host_id")
	require.NoError(t, err)
	require.Len(t, mappings, 2)

	// Host 1 should be mapped to SCIM user 1
	require.Equal(t, host1ID, mappings[0].HostID)
	require.Equal(t, scimUser1ID, mappings[0].ScimUserID)

	// Host 2 should be mapped to SCIM user 2
	require.Equal(t, host2ID, mappings[1].HostID)
	require.Equal(t, scimUser2ID, mappings[1].ScimUserID)

	// Host 3 should not have a mapping (no matching SCIM user)
	err = db.Get(&count, "SELECT COUNT(*) FROM host_scim_user WHERE host_id = ?", host3ID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	// Test idempotency by running the migration again (simulate applying it twice)
	// This should not create duplicate mappings due to INSERT IGNORE
	_, err = db.Exec(`
		INSERT IGNORE INTO host_scim_user (host_id, scim_user_id)
		SELECT he.host_id, su.id
		FROM host_emails he
		JOIN scim_users su ON he.email = su.user_name
		LEFT JOIN host_scim_user existing ON he.host_id = existing.host_id
		WHERE he.source = 'mdm_idp_accounts' AND existing.host_id IS NULL
	`)
	require.NoError(t, err)

	// Should still have exactly 2 mappings
	err = db.Get(&count, "SELECT COUNT(*) FROM host_scim_user")
	require.NoError(t, err)
	require.Equal(t, 2, count)
}
