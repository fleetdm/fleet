package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20260922124402(t *testing.T) {
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
