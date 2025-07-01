package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250609112613, Down_20250609112613)
}

func Up_20250609112613(tx *sql.Tx) error {
	stmt := `
	-- The challenges table holds generated challenges intended for single-use applications.
	-- Whenever a challenge is checked it should be deleted from this table.
	CREATE TABLE IF NOT EXISTS challenges (
		-- challenge is randomly generated string encoded with base64.URLEncoding.
		challenge CHAR(32),
		created_at TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6),
		updated_at TIMESTAMP(6) DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
		PRIMARY KEY (challenge)
	) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci;`

	if _, err := tx.Exec(stmt); err != nil {
		return fmt.Errorf("creating challenges table: %w", err)
	}

	return nil
}

func Down_20250609112613(tx *sql.Tx) error {
	return nil
}
