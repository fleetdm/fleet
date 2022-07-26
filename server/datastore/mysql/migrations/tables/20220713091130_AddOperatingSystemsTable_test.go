package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220713091130(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	stmt := `
INSERT INTO operating_systems (
    name,
    version,
    arch,
    kernel_version
)
VALUES (?, ?, ?, ?)
`
	_, err := db.Exec(stmt, "Ubuntu", "22.04 LTS", "x86_64", "5.10.76-linuxkit")
	require.NoError(t, err)

	var (
		name           string
		version        string
		arch           string
		kernel_version string
	)
	err = db.QueryRow(`SELECT name, version, arch, kernel_version FROM operating_systems WHERE name = ? AND version = ?`, "Ubuntu", "22.04 LTS").
		Scan(&name, &version, &arch, &kernel_version)
	require.NoError(t, err)
	require.Equal(t, "Ubuntu", name)
	require.Equal(t, "22.04 LTS", version)
	require.Equal(t, "x86_64", arch)
	require.Equal(t, "5.10.76-linuxkit", kernel_version)
}
