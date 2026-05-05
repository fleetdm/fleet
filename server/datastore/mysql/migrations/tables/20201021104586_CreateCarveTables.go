package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20201021104586, Down_20201021104586)
}

func Up_20201021104586(tx *sql.Tx) error {
	if _, err := tx.Exec(`
		CREATE TABLE IF NOT EXISTS carve_metadata (
		id INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
		host_id INT UNSIGNED NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		name VARCHAR(255),
		block_count INT UNSIGNED NOT NULL,
		block_size INT UNSIGNED NOT NULL,
		carve_size BIGINT UNSIGNED NOT NULL,
		carve_id VARCHAR(64) NOT NULL,
		request_id VARCHAR(64) NOT NULL,
		session_id VARCHAR(64) NOT NULL,
		expired TINYINT DEFAULT 0,
		UNIQUE KEY idx_session_id (session_id),
		UNIQUE KEY idx_name (name),
		FOREIGN KEY (host_id) REFERENCES hosts (id) ON DELETE CASCADE
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`); err != nil {
		return errors.Wrap(err, "create carve_metadata")
	}

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS carve_blocks (
		metadata_id INT UNSIGNED NOT NULL,
		block_id INT NOT NULL,
		data LONGBLOB,
		PRIMARY KEY (metadata_id, block_id),
		FOREIGN KEY (metadata_id) REFERENCES carve_metadata (id) ON DELETE CASCADE
	)`); err != nil {
		return errors.Wrap(err, "create carve_blocks")
	}

	return nil
}

func Down_20201021104586(tx *sql.Tx) error {
	return nil
}
