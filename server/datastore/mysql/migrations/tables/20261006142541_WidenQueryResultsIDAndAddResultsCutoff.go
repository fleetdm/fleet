package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261006142541, Down_20261006142541)
}

func Up_20261006142541(tx *sql.Tx) error {
	return withSteps([]migrationStep{
		widenQueryResultsID,
		addQueryResultsCutoff,
	}, tx)
}

// Every result write deletes and re-inserts the host's rows, using new ids. With the report cap
// raised to the host count, large deployments can use up an INT UNSIGNED in days, after which the
// INSERT IGNORE that stores results silently drops them.
func widenQueryResultsID(tx *sql.Tx) error {
	var dataType string
	if err := tx.QueryRow(`
		SELECT DATA_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'query_results' AND COLUMN_NAME = 'id'
	`).Scan(&dataType); err != nil {
		return fmt.Errorf("reading query_results.id type: %w", err)
	}
	if dataType == "bigint" {
		return nil
	}
	if _, err := tx.Exec(`ALTER TABLE query_results MODIFY id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT`); err != nil {
		return fmt.Errorf("widening query_results.id: %w", err)
	}
	return nil
}

func addQueryResultsCutoff(tx *sql.Tx) error {
	if columnExists(tx, "queries", "results_valid_from_id") {
		return nil
	}
	if _, err := tx.Exec(`
		ALTER TABLE queries
		ADD COLUMN results_valid_from_id BIGINT UNSIGNED NOT NULL DEFAULT 0,
		ADD COLUMN results_cleanup_pending TINYINT(1) NOT NULL DEFAULT 0,
		ALGORITHM=INSTANT
	`); err != nil {
		return fmt.Errorf("adding results_valid_from_id and results_cleanup_pending to queries table: %w", err)
	}
	return nil
}

func Down_20261006142541(tx *sql.Tx) error {
	return nil
}
