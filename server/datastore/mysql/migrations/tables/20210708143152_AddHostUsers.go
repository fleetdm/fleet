package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20210708143152, Down_20210708143152)
}

func Up_20210708143152(tx *sql.Tx) error {
	sqlStatement := `
		CREATE TABLE IF NOT EXISTS host_users (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		host_id INT UNSIGNED NOT NULL,
		uid INT UNSIGNED NOT NULL,
		username VARCHAR(255),
		groupname VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		removed_at TIMESTAMP NULL,
		user_type VARCHAR(255),
		UNIQUE KEY idx_uid_username (host_id, uid, username),
		FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`
	if _, err := tx.Exec(sqlStatement); err != nil {
		return err
	}

	return nil
}

func Down_20210708143152(tx *sql.Tx) error {
	return nil
}
