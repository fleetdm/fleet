package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261007200650, Down_20261007200650)
}

func Up_20261007200650(tx *sql.Tx) error {
	// eulas held a single row pinned at id = 1. Keying it by platform instead
	// lets macOS (PDF) and Windows (markdown) each have their own agreement.
	// The default only backfills the existing row, then it is dropped so every
	// later insert has to name its platform.
	if !columnExists(tx, "eulas", "platform") {
		if _, err := tx.Exec(`
			ALTER TABLE eulas
			MODIFY id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			ADD COLUMN platform VARCHAR(10) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT 'darwin'
		`); err != nil {
			return fmt.Errorf("adding platform column to eulas: %w", err)
		}
	}

	// Outside the guard so a retry still drops it if the run failed after adding the column.
	if _, err := tx.Exec(`ALTER TABLE eulas ALTER COLUMN platform DROP DEFAULT`); err != nil {
		return fmt.Errorf("dropping platform default on eulas: %w", err)
	}

	if !indexExistsTx(tx, "eulas", "idx_eulas_platform") {
		if _, err := tx.Exec(`ALTER TABLE eulas ADD UNIQUE KEY idx_eulas_platform (platform)`); err != nil {
			return fmt.Errorf("adding unique platform key to eulas: %w", err)
		}
	}

	return nil
}

func Down_20261007200650(tx *sql.Tx) error {
	return nil
}
