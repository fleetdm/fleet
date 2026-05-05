package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20250513162912(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert host declarations with different statuses and operation types
	_, err := db.Exec(`
	INSERT INTO host_mdm_apple_declarations 
	(host_uuid, declaration_uuid, status, operation_type, token, declaration_identifier) VALUES
	('test-host-uuid', 'decl-uuid-1', 'pending', 'install', UNHEX('AABBCCDDEEFF'), 'com.example.decl1'),
	('test-host-uuid', 'decl-uuid-2', 'verified', 'remove', UNHEX('112233445566'), 'com.example.decl2'),
	('test-host-uuid', 'decl-uuid-3', 'verifying', 'remove', UNHEX('AABBCCDDEEFF'), 'com.example.decl3');
	`)
	require.NoError(t, err)

	// Verify initial state
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM host_mdm_apple_declarations").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 3, count)

	// Apply current migration
	applyNext(t, db)

	// Verify that rows with "remove" operations whose status is "verifying" or "verified" were deleted
	err = db.QueryRow("SELECT COUNT(*) FROM host_mdm_apple_declarations").Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)

	// Verify that only the 'install' operation remains
	var operationType string
	err = db.QueryRow("SELECT operation_type FROM host_mdm_apple_declarations").Scan(&operationType)
	require.NoError(t, err)
	require.Equal(t, "install", operationType)

}
