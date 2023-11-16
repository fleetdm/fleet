package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20221014084130(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	query := `
		INSERT INTO host_orbit_info (
			host_id,
			version
		)
		VALUES (?, ?)
	`
	_, err := db.Exec(query, 42, "1.1")
	require.NoError(t, err)

	var (
		hostID  uint
		version string
	)
	err = db.QueryRow(`SELECT host_id, version FROM host_orbit_info WHERE host_id = ?`, 42).
		Scan(&hostID, &version)
	require.NoError(t, err)
	require.Equal(t, uint(42), hostID)
	require.Equal(t, "1.1", version)
}
