package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240709124958, Down_20240709124958)
}

func Up_20240709124958(tx *sql.Tx) error {
	if columnExists(tx, "software_titles", "bundle_identifier") {
		return nil
	}

	_, err := tx.Exec(`
		ALTER TABLE software_titles 
		ADD COLUMN bundle_identifier varchar(255) COLLATE utf8mb4_unicode_ci DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding bundle_identifier to software_titles: %w", err)
	}

	_, err = tx.Exec(`
	      UPDATE
		software_titles st,
		software s
	      SET
		st.bundle_identifier = NULLIF(s.bundle_identifier, '')
	      WHERE
		s.title_id = st.id
	`)
	if err != nil {
		return fmt.Errorf("adding bundle_identifier to existing software_titles rows: %w", err)
	}

	// To account for an edge case of two software_installers pointing to
	// different software_titles that have the same bundle_identifier now.
	_, err = tx.Exec(`
	    UPDATE software_installers si
	    JOIN (
		SELECT id, bundle_identifier
		FROM software_titles
	    ) st ON si.title_id = st.id
	    JOIN (
		SELECT MIN(id) as min_id, bundle_identifier
		FROM software_titles
		GROUP BY bundle_identifier
	    ) st_min ON st.bundle_identifier = st_min.bundle_identifier
	    SET si.title_id = st_min.min_id;
	`)
	if err != nil {
		return fmt.Errorf("ensuring software_installers point to the same software_title: %w", err)
	}

	// delete duplicates, keeping the row with the smallest `id`
	_, err = tx.Exec(`
	    DELETE st1
	    FROM software_titles st1
	    LEFT JOIN (
		SELECT MIN(id) as min_id, bundle_identifier
		FROM software_titles
		GROUP BY bundle_identifier
	    ) st_min ON st1.bundle_identifier = st_min.bundle_identifier
	    LEFT JOIN software_installers si ON st1.id = si.title_id
	    WHERE st1.id != st_min.min_id AND si.title_id IS NULL AND st1.bundle_identifier IS NOT NULL;
	`)
	if err != nil {
		return fmt.Errorf("deleting software_title entries with duplicated bundle_identifier: %w", err)
	}

	_, err = tx.Exec(`
	      UPDATE software s
	      JOIN software_titles st ON s.bundle_identifier = st.bundle_identifier
	      SET s.title_id = st.id
	      WHERE s.title_id IS NULL;
	`)
	if err != nil {
		return fmt.Errorf("updating orphaned software references: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software_titles ADD UNIQUE KEY idx_software_titles_bundle_identifier (bundle_identifier)`)
	if err != nil {
		return fmt.Errorf("adding unique key to bundle_identifier in software_titles: %w", err)
	}

	return nil
}

func Down_20240709124958(tx *sql.Tx) error {
	return nil
}
