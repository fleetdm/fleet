package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250507170845(t *testing.T) {
	db := applyUpToPrev(t)

	_, err := db.Exec(`
	INSERT INTO scim_users (id, user_name, given_name, family_name, active) VALUES
	(1, 'user1@example.com', 'User', 'One', 1),
	(2, 'user2@example.com', 'User', 'Two', 1),
	(3, 'user3@example.com', 'User', 'Three', 1);
	`)
	require.NoError(t, err)

	// Insert host_scim_user entries with duplicate host_ids
	_, err = db.Exec(`
	INSERT INTO host_scim_user (host_id, scim_user_id) VALUES
	(100, 1),  -- This should be kept (smallest scim_user_id for host_id 100)
	(100, 2),  -- This should be removed (duplicate host_id)
	(200, 2),  -- This should be kept (only entry for host_id 200)
	(300, 3);  -- This should be kept (only entry for host_id 300)
	`)
	require.NoError(t, err)

	// Apply current migration
	applyNext(t, db)

	// Verify that only one row exists per host_id and it's the one with the smallest scim_user_id
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM host_scim_user").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Check that host_id 100 is associated with scim_user_id 1 (the smallest)
	var scimUserID int
	err = db.QueryRow("SELECT scim_user_id FROM host_scim_user WHERE host_id = 100").Scan(&scimUserID)
	require.NoError(t, err)
	require.Equal(t, 1, scimUserID)

	// Verify we can insert a new row with a unique host_id
	_, err = db.Exec("INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (400, 1)")
	require.NoError(t, err)

	// Verify we cannot insert a duplicate host_id
	_, err = db.Exec("INSERT INTO host_scim_user (host_id, scim_user_id) VALUES (100, 3)")
	require.Error(t, err)
}
