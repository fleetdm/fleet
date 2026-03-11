package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260217181748, Down_20260217181748)
}

func Up_20260217181748(tx *sql.Tx) error {
	_, err := tx.Exec(`UPDATE software_titles SET source = 'apps' WHERE source = 'pkg_packages' AND bundle_identifier != ''`)
	if err != nil {
		return fmt.Errorf("failed to change source for software titles: %w", err)
	}
	return nil
}

func Down_20260217181748(tx *sql.Tx) error {
	return nil
}
