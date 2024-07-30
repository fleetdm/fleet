package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20240130115133(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert test data
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

	// Function to query os_version_id for a given name and version
	getOsVersionID := func(name, version string) int {
		var osVersionID int
		selectStmt := `
			SELECT os_version_id
			FROM operating_systems
			WHERE name = ? AND version = ?
			LIMIT 1
		`
		err := db.QueryRow(selectStmt, name, version).Scan(&osVersionID)
		require.NoError(t, err)
		return osVersionID
	}

	// Query os_version_id for each distinct name and version
	ubuntuOsVersionID := getOsVersionID("Ubuntu", "20.04")
	windowsOsVersionID := getOsVersionID("Windows", "10.0.22621.1234")
	macosOsVersionID := getOsVersionID("macOS", "14.2.1")

	// assert that os version IDs are unique
	require.NotEqual(t, ubuntuOsVersionID, windowsOsVersionID)
	require.NotEqual(t, ubuntuOsVersionID, macosOsVersionID)
	require.NotEqual(t, windowsOsVersionID, macosOsVersionID)

	// Assert that rows with the same name and version have the same os_version_id
	selectStmt := `
		SELECT os_version_id
		FROM operating_systems
		WHERE name = ? AND version = ?
	`
	var ubuntuIDs []int
	err = db.Select(&ubuntuIDs, selectStmt, "Ubuntu", "20.04")
	require.NoError(t, err)
	require.Equal(t, []int{ubuntuOsVersionID, ubuntuOsVersionID}, ubuntuIDs)

	var windowsIDs []int
	err = db.Select(&windowsIDs, selectStmt, "Windows", "10.0.22621.1234")
	require.NoError(t, err)
	require.Equal(t, []int{windowsOsVersionID}, windowsIDs)

	var macosIDs []int
	err = db.Select(&macosIDs, selectStmt, "macOS", "14.2.1")
	require.NoError(t, err)
	require.Equal(t, []int{macosOsVersionID}, macosIDs)
}
