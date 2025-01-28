package tables

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestUp_20240408085837(t *testing.T) {
	db := applyUpToPrev(t)

	// Insert data into orbit_info
	id := 1
	execNoErr(t, db, "INSERT INTO host_orbit_info (host_id, version) VALUES (?, ?)", id, "")

	applyNext(t, db)

	type orbitInfo struct {
		HostID         int64   `db:"host_id"`
		Version        string  `db:"version"`
		DesktopVersion *string `db:"desktop_version"`
		ScriptsEnabled *bool   `db:"scripts_enabled"`
	}

	var results []orbitInfo
	err := db.SelectContext(context.Background(), &results, `SELECT * FROM host_orbit_info WHERE host_id = ?`, id)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Nil(t, results[0].DesktopVersion)
	assert.Nil(t, results[0].ScriptsEnabled)

	id = 2
	results = nil
	execNoErr(t, db, "INSERT INTO host_orbit_info (host_id, version) VALUES (?, ?)", id, "")
	err = db.SelectContext(context.Background(), &results, `SELECT * FROM host_orbit_info WHERE host_id = ?`, id)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Nil(t, results[0].DesktopVersion)
	assert.Nil(t, results[0].ScriptsEnabled)

	id = 3
	results = nil
	const desktopVersion = "1.0.0"
	const scriptsEnabled = true
	execNoErr(
		t, db, "INSERT INTO host_orbit_info (host_id, version, desktop_version, scripts_enabled) VALUES (?, ?, ?, ?)", id, "",
		desktopVersion,
		scriptsEnabled,
	)
	err = db.SelectContext(context.Background(), &results, `SELECT * FROM host_orbit_info WHERE host_id = ?`, id)
	require.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, desktopVersion, *results[0].DesktopVersion)
	assert.Equal(t, scriptsEnabled, *results[0].ScriptsEnabled)
}
