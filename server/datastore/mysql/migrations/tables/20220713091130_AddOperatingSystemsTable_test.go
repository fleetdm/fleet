package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220713091130(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	// test operating systems table
	stmt := `
INSERT INTO operating_systems (
    name,
    version,
    arch,
    kernel_version,
	platform
)
VALUES (?, ?, ?, ?, ?)
`

	var (
		name           string
		version        string
		arch           string
		kernel_version string
		platform       string
	)

	// add first operating system
	_, err := db.Exec(stmt, "Ubuntu", "22.04 LTS", "x86_64", "5.10.76-linuxkit", "ubuntu")
	require.NoError(t, err)

	err = db.QueryRow(`SELECT name, version, arch, kernel_version, platform FROM operating_systems WHERE name = ? AND version = ?`, "Ubuntu", "22.04 LTS").
		Scan(&name, &version, &arch, &kernel_version, &platform)
	require.NoError(t, err)
	require.Equal(t, "Ubuntu", name)
	require.Equal(t, "22.04 LTS", version)
	require.Equal(t, "x86_64", arch)
	require.Equal(t, "5.10.76-linuxkit", kernel_version)
	require.Equal(t, "ubuntu", platform)

	// add second operating system
	_, err = db.Exec(stmt, "Ubuntu", "22.06 LTS", "x86_64", "5.10.76-linuxkit", "ubuntu")
	require.NoError(t, err)

	err = db.QueryRow(`SELECT name, version, arch, kernel_version, platform FROM operating_systems WHERE name = ? AND version = ?`, "Ubuntu", "22.06 LTS").
		Scan(&name, &version, &arch, &kernel_version, &platform)
	require.NoError(t, err)
	require.Equal(t, "Ubuntu", name)
	require.Equal(t, "22.06 LTS", version)
	require.Equal(t, "x86_64", arch)
	require.Equal(t, "5.10.76-linuxkit", kernel_version)
	require.Equal(t, "ubuntu", platform)

	// test host operating systems table
	stmt = `
INSERT INTO host_operating_system (
    host_id,
    os_id
)
VALUES (?, ?)
`
	// new host id, new os id
	_, err = db.Exec(stmt, 111, 1)
	require.NoError(t, err)

	// new host id, new os id
	_, err = db.Exec(stmt, 222, 2)
	require.NoError(t, err)

	// new host id, duplicate os id
	_, err = db.Exec(stmt, 333, 2)
	require.NoError(t, err)

	// new host id, non-existent os id, foreign key error
	_, err = db.Exec(stmt, 444, 4)
	require.Error(t, err)

	// duplicate host id, new os id, primary key error
	_, err = db.Exec(stmt, 111, 4)
	require.Error(t, err)

	// duplicate host id, duplicate os id, primary key error
	_, err = db.Exec(stmt, 111, 2)
	require.Error(t, err)

	var osID uint
	err = db.QueryRow(`SELECT os_id FROM host_operating_system WHERE host_id = 111`).
		Scan(&osID)
	require.NoError(t, err)
	require.Equal(t, uint(1), osID)

	var hostIDs []int
	err = db.Select(&hostIDs, `SELECT host_id FROM host_operating_system WHERE os_id = 2`)
	require.NoError(t, err)
	require.Len(t, hostIDs, 2)
	require.Contains(t, hostIDs, 222)
	require.Contains(t, hostIDs, 333)
}
