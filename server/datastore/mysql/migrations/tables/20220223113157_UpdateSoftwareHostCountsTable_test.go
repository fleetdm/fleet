package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20220223113157(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO software_host_counts (software_id, hosts_count) VALUES (1, 1)`)
	execNoErr(t, db, `INSERT INTO software_host_counts (software_id, hosts_count) VALUES (2, 10)`)

	// Apply current migration.
	applyNext(t, db)

	var count int
	require.NoError(t, db.Get(&count, `SELECT count(*) FROM software_host_counts WHERE team_id = 0`))
	assert.Equal(t, 2, count)
	require.NoError(t, db.Get(&count, `SELECT SUM(hosts_count) FROM software_host_counts WHERE team_id = 0`))
	assert.Equal(t, 11, count)
}
