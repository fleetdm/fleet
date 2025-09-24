package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20250923165754(t *testing.T) {
	db := applyUpToPrev(t)

	insertStmt := `INSERT INTO host_disks (host_id) VALUES (1)`
	_, err := db.Exec(insertStmt)
	require.NoError(t, err)

	// Apply current migration.
	applyNext(t, db)

	type diskSpace struct {
		HostID           uint    `db:"host_id"`
		GigsAllDiskSpace float64 `db:"gigs_all_disk_space"`
	}

	var ds diskSpace
	err = db.Get(&ds, `SELECT host_id, gigs_all_disk_space from host_disks where host_id = 1`)
	require.NoError(t, err)
	assert.Equal(t, nil, ds.GigsAllDiskSpace)

	_, err = db.Exec(`INSERT INTO host_disks (host_id, gigs_all_disk_space) VALUES (2, 1.5)`)
	require.NoError(t, err)
	err = db.Get(&ds, `SELECT host_id, gigs_all_disk_space from host_disks where host_id = 2`)
	require.NoError(t, err)
	assert.Equal(t, 1.5, ds.GigsAllDiskSpace)
}
