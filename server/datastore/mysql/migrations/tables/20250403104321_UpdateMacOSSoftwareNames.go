package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250403104321, Down_20250403104321)
}

func Up_20250403104321(tx *sql.Tx) error {
	titleStmt := `UPDATE software_titles SET name = TRIM( TRAILING '.app' FROM name ) WHERE source = 'apps'`
	_, err := tx.Exec(titleStmt)
	if err != nil {
		return fmt.Errorf("updating software_titles.name: %w", err)
	}

	softwareStmt := `
	UPDATE software SET 
		name = TRIM( TRAILING '.app' FROM name ),
		checksum = UNHEX(
		MD5(
			-- concatenate with separator \x00
			CONCAT_WS(CHAR(0),
				version,
				source,
				bundle_identifier,
				` + "`release`" + `,
				arch,
				vendor,
				browser,
				extension_id
			)
		)
	)
		WHERE source = 'apps'
		AND bundle_identifier IS NOT NULL
	`
	_, err = tx.Exec(softwareStmt)
	if err != nil {
		return fmt.Errorf("updating software name and checksum: %w", err)
	}

	newColStmt := `ALTER TABLE software ADD COLUMN name_source enum('basic', 'bundle_4.67') DEFAULT 'basic' NOT NULL`
	_, err = tx.Exec(newColStmt)
	if err != nil {
		return fmt.Errorf("adding name_source column to software: %w", err)
	}

	return nil
}

func Down_20250403104321(tx *sql.Tx) error {
	return nil
}
