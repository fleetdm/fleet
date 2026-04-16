package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260409153714, Down_20260409153714)
}

func Up_20260409153714(tx *sql.Tx) error {
	if !tableExists(tx, "user_api_endpoints") {
		_, err := tx.Exec(`
			CREATE TABLE user_api_endpoints (
				user_id INT UNSIGNED NOT NULL,
				path VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
				method VARCHAR(10) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,

				PRIMARY KEY (user_id, path, method),
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			)
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

func Down_20260409153714(tx *sql.Tx) error {
	return nil
}
