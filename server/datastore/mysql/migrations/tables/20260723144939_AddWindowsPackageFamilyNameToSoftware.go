package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260723144939, Down_20260723144939)
}

func Up_20260723144939(tx *sql.Tx) error {
	// package_family_name is the MSIX/AppX package family (e.g. "Microsoft.Copilot_8wekyb3d8bbwe"),
	// informational inventory data that identifies a packaged Windows app store app. It is populated
	// only for packaged apps and left NULL otherwise.
	//
	// No backfill is needed: the column is only meaningful when non-empty, and ingestion fills it in
	// automatically. Packaged apps get the value on their next report (a non-empty package_family_name
	// changes the software checksum, so a new row is inserted); classic programs never carry a value.
	// ALGORITHM=INSTANT keeps this a metadata-only change on the (potentially very large) software table.
	if _, err := tx.Exec(`ALTER TABLE software ADD COLUMN package_family_name VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL, ALGORITHM=INSTANT`); err != nil {
		return fmt.Errorf("failed to add software.package_family_name column: %w", err)
	}

	return nil
}

func Down_20260723144939(tx *sql.Tx) error {
	return nil
}
