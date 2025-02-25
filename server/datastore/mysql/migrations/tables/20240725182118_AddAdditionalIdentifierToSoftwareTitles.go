package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240725182118, Down_20240725182118)
}

func Up_20240725182118(tx *sql.Tx) error {
	if columnExists(tx, "software_titles", "additional_identifier") {
		return nil
	}

	_, err := tx.Exec(`
		ALTER TABLE software_titles 
		ADD COLUMN additional_identifier TINYINT UNSIGNED GENERATED ALWAYS AS 
			(CASE
				WHEN source = 'ios_apps' then 1
				WHEN source = 'ipados_apps' then 2
				WHEN bundle_identifier IS NOT NULL THEN 0
				ELSE NULL
			END) VIRTUAL`)
	if err != nil {
		return fmt.Errorf("adding additional_identifier to software_titles: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software_titles DROP INDEX idx_software_titles_bundle_identifier`)
	if err != nil {
		return fmt.Errorf("dropping unique key for bundle_identifier in software_titles: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software_titles ADD UNIQUE KEY idx_software_titles_bundle_identifier (bundle_identifier, additional_identifier)`)
	if err != nil {
		return fmt.Errorf("adding unique key for identifiers in software_titles: %w", err)
	}

	return nil
}

func Down_20240725182118(_ *sql.Tx) error {
	return nil
}
