package tables

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUp_20230520153236(t *testing.T) {
	db := applyUpToPrev(t)
	applyNext(t, db)

	hostID := uint(12)

	insertStmt := `INSERT INTO host_dep_assignments (host_id) VALUES (?)`

	_, err := db.Exec(insertStmt, hostID)
	require.NoError(t, err)

	_, err = db.Exec(insertStmt, hostID)
	require.ErrorContains(t, err, "Error 1062")

	type assignment struct {
		HostID    uint       `db:"host_id"`
		AddedAt   time.Time  `db:"added_at"`
		DeletedAt *time.Time `db:"deleted_at"`
	}

	var a assignment
	selectStmt := `SELECT host_id, added_at, deleted_at FROM host_dep_assignments WHERE host_id = ?`
	err = db.Get(&a, selectStmt, hostID)
	require.NoError(t, err)
	require.Equal(t, hostID, a.HostID)
	require.NotZero(t, a.AddedAt)
	require.Nil(t, a.DeletedAt)

	_, err = db.Exec(`UPDATE host_dep_assignments SET deleted_at = NOW()`)
	require.NoError(t, err)

	a = assignment{}
	err = db.Get(&a, selectStmt, hostID)
	require.NoError(t, err)
	require.Equal(t, hostID, a.HostID)
	require.NotZero(t, a.AddedAt)
	require.NotNil(t, a.DeletedAt)
}
