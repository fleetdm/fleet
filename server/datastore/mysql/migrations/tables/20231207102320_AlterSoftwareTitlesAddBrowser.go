package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231207102320, Down_20231207102320)
}

func Up_20231207102320(tx *sql.Tx) error {
	_, err := tx.Exec((`DELETE FROM software_titles;`)) // delete all software titles, it will be repopulated on the next cron
	if err != nil {
		return fmt.Errorf("failed to delete software titles: %w", err)
	}

	_, err = tx.Exec(`
		ALTER TABLE software_titles 
		ADD COLUMN browser varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '';`)
	if err != nil {
		return fmt.Errorf("failed to add browser column to software titles table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software_titles DROP KEY idx_software_titles_name_source;`)
	if err != nil {
		return fmt.Errorf("failed to drop name-source key from software titles table: %w", err)
	}

	return nil
}

func Down_20231207102320(tx *sql.Tx) error {
	return nil
}
