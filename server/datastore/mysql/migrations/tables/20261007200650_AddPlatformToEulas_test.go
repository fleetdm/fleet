package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261007200650(t *testing.T) {
	db := applyUpToPrev(t)

	// The existing single EULA, pinned at id = 1 as the code did before.
	execNoErr(t, db, `INSERT INTO eulas (id, name, bytes, token) VALUES (1, 'eula.pdf', 'pdf', 'token-macos')`)

	applyNext(t, db)

	var platform string
	require.NoError(t, db.Get(&platform, `SELECT platform FROM eulas WHERE token = 'token-macos'`))
	require.Equal(t, "darwin", platform)

	// A Windows agreement can now sit next to it, with a generated id.
	id := execNoErrLastID(t, db, `INSERT INTO eulas (name, bytes, token, platform) VALUES ('terms.md', 'md', 'token-windows', 'windows')`)
	require.NotEqual(t, int64(1), id)

	// Still one agreement per platform.
	_, err := db.Exec(`INSERT INTO eulas (name, bytes, token, platform) VALUES ('other.pdf', 'pdf', 'token-other', 'darwin')`)
	require.ErrorContains(t, err, "Duplicate entry")

	// The backfill default was dropped, so an insert must name its platform.
	_, err = db.Exec(`INSERT INTO eulas (name, bytes, token) VALUES ('none.pdf', 'pdf', 'token-none')`)
	require.Error(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)
	require.NoError(t, Up_20261007200650(tx))
	require.NoError(t, tx.Commit())
}

func TestUp_20261007200650_PartiallyApplied(t *testing.T) {
	db := applyUpToPrev(t)
	// A run that failed right after adding the column: default still set, no unique key.
	execNoErr(t, db, `ALTER TABLE eulas
		MODIFY id INT UNSIGNED NOT NULL AUTO_INCREMENT,
		ADD COLUMN platform VARCHAR(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'darwin'`)

	applyNext(t, db)

	_, err := db.Exec(`INSERT INTO eulas (name, bytes, token) VALUES ('none.pdf', 'pdf', 'token-none')`)
	require.Error(t, err)
	require.Equal(t, []string{"platform"}, indexColumns(t, db, "eulas", "idx_eulas_platform"))
}
