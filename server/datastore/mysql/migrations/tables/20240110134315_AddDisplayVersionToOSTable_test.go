package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240110134315(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert into OS table
	insertStmt := `
		INSERT INTO operating_systems (
			name, version, arch, kernel_version, platform
			)
		VALUES (?, ?, ?, ?, ?)
	`
	_, err := db.Exec(insertStmt, "Windows", "10.0.19042", "x86_64", "10.0.19042", "windows")
	require.NoError(t, err)

	applyNext(t, db)

	// Check that the new column exists
	var displayVersion *string
	err = db.Get(&displayVersion, "SELECT display_version FROM operating_systems LIMIT 1")
	require.NoError(t, err)
	require.Empty(t, displayVersion)

	// Test unique constraint includes display_version
	insertStmt1 := `
		INSERT INTO operating_systems (
			name, version, arch, kernel_version, platform, display_version
			)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	// New record with display_version is not a duplicate
	_, err = db.Exec(insertStmt1, "Windows", "10.0.19042", "x86_64", "10.0.19042", "windows", "22H2")
	require.NoError(t, err)

	insertStmt2 := `
		INSERT INTO operating_systems (
			name, version, arch, kernel_version, platform
			)
		VALUES (?, ?, ?, ?, ?)
	`

	// No error when display_version is NULL
	_, err = db.Exec(insertStmt2, "Windows", "10.0.19042", "x86_64", "10.0.19042", "windows")
	require.NoError(t, err)

	// Unique constraint violation when display_version is not NULL
	_, err = db.Exec(insertStmt1, "Windows", "10.0.19042", "x86_64", "10.0.19042", "windows", "22H2")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Duplicate entry")
}
