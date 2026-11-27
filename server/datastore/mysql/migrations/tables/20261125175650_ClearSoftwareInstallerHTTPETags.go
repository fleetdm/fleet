package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration("Clear software installer HTTP ETags", Up_20261125175650, Down_20261125175650)
}

// Up_20261125175650 clears stored ETags. A replaced package used to keep the ETag of its URL's download, and conditional downloads
// copied that ETag along with the replaced file, so a stored ETag doesn't reliably describe the stored bytes. Each package downloads
// in full once on its next GitOps run and stores a fresh ETag.
func Up_20261125175650(tx *sql.Tx) error {
	if _, err := tx.Exec(`UPDATE software_installers SET http_etag = NULL WHERE http_etag IS NOT NULL`); err != nil {
		return fmt.Errorf("clearing software installer ETags: %w", err)
	}
	return nil
}

func Down_20261125175650(tx *sql.Tx) error {
	return nil
}
