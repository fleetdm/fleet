package tables

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUp_20220215152203(t *testing.T) {
	db := applyUpToPrev(t)

	execNoErr(t, db, `INSERT INTO host_munki_info (host_id, version) VALUES (1, "6.2.8")`)
	execNoErr(t, db, `INSERT INTO host_munki_info (host_id, version) VALUES (2, "6.2.8")`)
	execNoErr(t, db, `INSERT INTO host_munki_info (host_id, version) VALUES (3, "")`)

	var count int
	require.NoError(t, db.Get(&count, `SELECT count(*) FROM host_munki_info ORDER BY host_id`))
	assert.Equal(t, 3, count)

	// Apply current migration.
	applyNext(t, db)

	require.NoError(t, db.Get(&count, `SELECT count(*) FROM host_munki_info WHERE deleted_at is NULL ORDER BY host_id`))
	assert.Equal(t, 2, count)
}
