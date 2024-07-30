package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231207102321, Down_20231207102321)
}

func Up_20231207102321(tx *sql.Tx) error {
	_, err := tx.Exec(`ALTER TABLE software_titles ADD UNIQUE INDEX idx_sw_titles (name, source, browser);`)
	if err != nil {
		return fmt.Errorf("failed to add unique index to software titles table: %w", err)
	}

	_, err = tx.Exec(`ALTER TABLE software ADD INDEX idx_sw_name_source_browser (name, source, browser);`)
	if err != nil {
		return fmt.Errorf("failed to add name-source-browser index to software table: %w", err)
	}
	return nil
}

func Down_20231207102321(tx *sql.Tx) error {
	return nil
}
