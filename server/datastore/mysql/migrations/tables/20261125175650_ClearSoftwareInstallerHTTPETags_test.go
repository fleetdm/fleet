package tables

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUp_20261125175650(t *testing.T) {
	db := applyUpToPrev(t)

	scriptID := execNoErrLastID(t, db, `INSERT INTO script_contents (md5_checksum, contents) VALUES (UNHEX(MD5('echo')), 'echo')`)
	insert := func(storageID string, etag *string) {
		execNoErr(t, db, `
			INSERT INTO software_installers
				(filename, version, platform, install_script_content_id, uninstall_script_content_id, storage_id, package_ids, patch_query, url, http_etag)
			VALUES ('app.pkg', '1.0', 'darwin', ?, ?, ?, '', '', 'https://example.com/app.pkg', ?)`,
			scriptID, scriptID, storageID, etag)
	}
	etag := `"abc123"`
	insert("with_etag", &etag)
	insert("without_etag", nil)

	applyNext(t, db)

	var rows []struct {
		StorageID string  `db:"storage_id"`
		URL       string  `db:"url"`
		ETag      *string `db:"http_etag"`
	}
	require.NoError(t, db.Select(&rows, `SELECT storage_id, url, http_etag FROM software_installers ORDER BY storage_id`))
	require.Len(t, rows, 2)
	for _, row := range rows {
		require.Nil(t, row.ETag, row.StorageID)
		require.Equal(t, "https://example.com/app.pkg", row.URL)
	}
}
