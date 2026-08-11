package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260811124912, Down_20260811124912)
}

// software_titles.additional_identifier discriminates titles that share a bundle
// identifier across Apple platforms, and it is part of the bundle_identifier
// unique key. tvOS apps need their own value, otherwise a tvOS title collides
// with the macOS title of the same app (both would resolve to 0).
func Up_20260811124912(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE software_titles
		MODIFY COLUMN additional_identifier TINYINT UNSIGNED GENERATED ALWAYS AS
			(CASE
				WHEN source = 'ios_apps' then 1
				WHEN source = 'ipados_apps' then 2
				WHEN source = 'tvos_apps' then 3
				WHEN bundle_identifier IS NOT NULL THEN 0
				ELSE NULL
			END) VIRTUAL`)
	if err != nil {
		return fmt.Errorf("adding tvos_apps to software_titles.additional_identifier: %w", err)
	}

	return nil
}

func Down_20260811124912(_ *sql.Tx) error {
	return nil
}
