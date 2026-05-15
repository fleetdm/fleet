package tables

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20260512173250(t *testing.T) {
	db := applyUpToPrev(t)

	// Pre-existing rows have no managed_apple_id concept.
	_, err := db.DB.Exec(`INSERT INTO host_mdm (host_id, enrolled, server_url, installed_from_dep, is_server, fleet_enroll_ref, is_personal_enrollment)
		VALUES
			(1, 1, 'https://example.com', 0, 0, '', 0),
			(2, 1, 'https://example.com', 0, 0, '', 1)`)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	// Existing rows have NULL managed_apple_id post-migration.
	var existing sql.NullString
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
