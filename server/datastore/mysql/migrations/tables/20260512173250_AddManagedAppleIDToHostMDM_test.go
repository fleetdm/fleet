package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260512173250(t *testing.T) {
	db := applyUpToPrev(t)

	// Seed hosts (the migration joins host_mdm -> hosts.uuid for backfill).
	insertHostStmt := `INSERT INTO hosts (id, uuid, hostname, osquery_host_id, node_key, platform) VALUES (?, ?, ?, ?, ?, 'ios')`
	for _, h := range []struct {
		id   int
		uuid string
	}{
		{1, "host-uuid-1"},
		{2, "host-uuid-2"},
		{4, "host-uuid-4"}, // personal-enrollment host with IDP account, will be backfilled
		{5, "host-uuid-5"}, // personal-enrollment host without IDP account, stays NULL
		{6, "host-uuid-6"}, // non-personal-enrollment host with IDP account, stays NULL
	} {
		_, err := db.DB.Exec(insertHostStmt, h.id, h.uuid, h.uuid, h.uuid, h.uuid)
		require.NoError(t, err)
	}

	// Seed IDP accounts and their host links.
	_, err := db.DB.Exec(`INSERT INTO mdm_idp_accounts (uuid, username, email)
		VALUES
			('idp-uuid-4', 'user4', 'user4@example.com'),
			('idp-uuid-6', 'user6', 'user6@example.com')`)
	require.NoError(t, err)

	// Link host-uuid-4 (personal) and host-uuid-6 (non-personal) to IDP accounts.
	_, err = db.DB.Exec(`INSERT INTO host_mdm_idp_accounts (host_uuid, account_uuid)
		VALUES
			('host-uuid-4', 'idp-uuid-4'),
			('host-uuid-6', 'idp-uuid-6')`)
	require.NoError(t, err)

	// Pre-existing host_mdm rows.
	_, err = db.DB.Exec(`INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server, fleet_enroll_ref, is_personal_enrollment)
		VALUES
			(1, 1, 'https://example.com', 0, 0, '', 0),
			(2, 1, 'https://example.com', 0, 0, '', 1),
			(4, 1, 'https://example.com', 0, 0, '', 1),
			(5, 1, 'https://example.com', 0, 0, '', 1),
			(6, 1, 'https://example.com', 0, 0, '', 0)`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Backfill: personal-enrollment host with IDP account got the email.
	var existing sql.NullString
	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 4).Scan(&existing)
	require.NoError(t, err)
	require.True(t, existing.Valid, "personal-enrollment host with IDP account should be backfilled")
	assert.Equal(t, "user4@example.com", existing.String)

	// No backfill: non-personal-enrollment host stays NULL even with IDP account.
	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 6).Scan(&existing)
	require.NoError(t, err)
	assert.False(t, existing.Valid, "non-personal-enrollment hosts must not be backfilled")

	// No backfill: personal-enrollment host without IDP account stays NULL.
	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 5).Scan(&existing)
	require.NoError(t, err)
	assert.False(t, existing.Valid)

	// No backfill: rows that didn't match any join condition stay NULL.
	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 1).Scan(&existing)
	require.NoError(t, err)
	assert.False(t, existing.Valid)

	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 2).Scan(&existing)
	require.NoError(t, err)
	assert.False(t, existing.Valid)

	// Insert with explicit managed_apple_id.
	_, err = db.DB.Exec(`INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server, fleet_enroll_ref, is_personal_enrollment, managed_apple_id)
		VALUES (3, 1, 'https://example.com', 0, 0, '', 1, ?)`, "user@example.com")
	require.NoError(t, err)

	var got sql.NullString
	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 3).Scan(&got)
	require.NoError(t, err)
	require.True(t, got.Valid)
	assert.Equal(t, "user@example.com", got.String)

	// Update sets the value on a row that previously had NULL.
	_, err = db.DB.Exec(`UPDATE host_mdm SET managed_apple_id = ? WHERE host_id = ?`, "second@example.com", 2)
	require.NoError(t, err)

	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 2).Scan(&got)
	require.NoError(t, err)
	require.True(t, got.Valid)
	assert.Equal(t, "second@example.com", got.String)

	// Update overwrites a previously-set value (re-assignment scenario).
	_, err = db.DB.Exec(`UPDATE host_mdm SET managed_apple_id = ? WHERE host_id = ?`, "reassigned@example.com", 3)
	require.NoError(t, err)

	err = db.DB.QueryRow(`SELECT managed_apple_id FROM host_mdm WHERE host_id = ?`, 3).Scan(&got)
	require.NoError(t, err)
	require.True(t, got.Valid)
	assert.Equal(t, "reassigned@example.com", got.String)
}
