package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20251217120000, Down_20251217120000)
}

func Up_20251217120000(tx *sql.Tx) error {
	// Insert Security and Utilities categories into software_categories table
	// Using INSERT IGNORE to avoid errors if categories already exist
	_, err := tx.Exec(`INSERT IGNORE INTO software_categories (name) VALUES ('Security'), ('Utilities')`)
	if err != nil {
		return fmt.Errorf("inserting Security and Utilities categories into software_categories table: %w", err)
	}

	return nil
}

func Down_20251217120000(tx *sql.Tx) error {
	// Remove Security and Utilities categories
	_, err := tx.Exec(`DELETE FROM software_categories WHERE name IN ('Security', 'Utilities')`)
	if err != nil {
		return fmt.Errorf("removing Security and Utilities categories from software_categories table: %w", err)
	}

	return nil
}
