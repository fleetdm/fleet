package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20220714191431(t *testing.T) {
	db := applyUpToPrev(t)

	applyNext(t, db)

	stmt := `
INSERT INTO host_operating_system (
    host_id,
    os_id
)
VALUES (?, ?)
`
	// new host id, new os id
	_, err := db.Exec(stmt, 111, 1)
	require.NoError(t, err)

	// new host id, new os id
	_, err = db.Exec(stmt, 222, 2)
	require.NoError(t, err)

	// new host id, duplicate os id
	_, err = db.Exec(stmt, 333, 2)
	require.NoError(t, err)

	// duplicate host id, new os id
	_, err = db.Exec(stmt, 111, 4)
	require.Error(t, err)

	// duplicate host id, duplicate os id
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
