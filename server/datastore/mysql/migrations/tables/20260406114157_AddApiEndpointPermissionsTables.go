package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260406114157, Down_20260406114157)
}

func Up_20260406114157(tx *sql.Tx) error {
	if !tableExists(tx, "user_api_endpoints") {
		_, err := tx.Exec(`
			CREATE TABLE user_api_endpoints (
				user_id INT UNSIGNED NOT NULL,
				endpoint_id VARCHAR(64) NOT NULL,

				is_allowed BOOLEAN DEFAULT TRUE,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				created_by_id INT UNSIGNED NULL,

				PRIMARY KEY (user_id, endpoint_id),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (created_by_id) REFERENCES users(id) ON DELETE SET NULL
			)
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

func Down_20260406114157(tx *sql.Tx) error {
	return nil
}
