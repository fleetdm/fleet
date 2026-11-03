package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260925182113(t *testing.T) {
	db := applyUpToPrev(t)

	existingID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, query, description, checksum) VALUES ('existing', 'SELECT 1', '', UNHEX(MD5('existing')))`,
	)

	applyNext(t, db)

	var hidden bool
	require.NoError(t, db.Get(&hidden, `SELECT hidden FROM policies WHERE id = ?`, existingID))
	require.False(t, hidden)

	hiddenID := execNoErrLastID(t, db,
		`INSERT INTO policies (name, query, description, checksum, hidden) VALUES ('hidden', 'SELECT 1', '', UNHEX(MD5('hidden')), 1)`,
	)
	require.NoError(t, db.Get(&hidden, `SELECT hidden FROM policies WHERE id = ?`, hiddenID))
	require.True(t, hidden)
}

func TestUp_20260925182113_PartiallyApplied(t *testing.T) {
	db := applyUpToPrev(t)

	// The column already exists but the migration was never recorded; the retry must be a no-op.
	execNoErr(t, db, `ALTER TABLE policies ADD COLUMN hidden TINYINT(1) NOT NULL DEFAULT 0`)

	applyNext(t, db)

	var hidden bool
	require.NoError(t, db.Get(&hidden, `SELECT hidden FROM policies WHERE id = ?`,
		execNoErrLastID(t, db, `INSERT INTO policies (name, query, description, checksum) VALUES ('p', 'SELECT 1', '', UNHEX(MD5('p')))`)))
	require.False(t, hidden)
}
