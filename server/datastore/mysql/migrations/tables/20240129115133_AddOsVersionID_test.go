package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240129115133(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `
		INSERT INTO operating_systems (name, version, arch, kernel_version, platform, display_version)
		VALUES (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?), (?, ?, ?, ?, ?, ?)
	`
	_, err := db.Exec(insertStmt,
		"Ubuntu", "20.04", "x86_64", "5.4.0-65-generic", "linux", "",
		"Ubuntu", "20.04", "x86_64", "6.0.0-70-generic", "linux", "",
		"Windows", "10.0.22621.1234", "x86_64", "10.0.22621.1234", "windows", "22H2",
		"macOS", "14.2.1", "x86_64", "20.4.0", "darwin", "",
	)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	updateStmt := `
		UPDATE operating_systems
		SET os_version_id = ?
		WHERE name = ? AND version = ?
		`
	_, err = db.Exec(updateStmt, 1, "Ubuntu", "20.04")
	require.NoError(t, err)

	_, err = db.Exec(updateStmt, 2, "Windows", "10.0.22621.1234")
	require.NoError(t, err)

	_, err = db.Exec(updateStmt, 3, "macOS", "14.2.1")
	require.NoError(t, err)

	selectStmt := `
		SELECT name, version
		FROM operating_systems
		WHERE os_version_id = ?
		`
	var name, version string
	err = db.QueryRow(selectStmt, 1).Scan(&name, &version)
	require.NoError(t, err)
	require.Equal(t, "Ubuntu", name)
	require.Equal(t, "20.04", version)
}
