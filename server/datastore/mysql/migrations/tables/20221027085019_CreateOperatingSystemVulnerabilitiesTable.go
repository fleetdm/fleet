package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20221027085019, Down_20221027085019)
}

func Up_20221027085019(tx *sql.Tx) error {
	logger.Info.Println("Creating table operating_system_vulnerabilities...")

	_, err := tx.Exec(`
		CREATE TABLE operating_system_vulnerabilities
		(
			id                  INT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
			host_id             INT UNSIGNED NOT NULL,
			operating_system_id INT UNSIGNED NOT NULL,
			cve                 VARCHAR(255) NOT NULL,
			source              SMALLINT              DEFAULT 0,
			created_at          TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,

			UNIQUE KEY idx_operating_system_vulnerabilities_unq_cve (host_id, cve),
			INDEX idx_operating_system_vulnerabilities_operating_system_id_cve (operating_system_id, cve)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
	`)
	if err != nil {
		return errors.Wrapf(err, "operating_system_vulnerabilities")
	}

	logger.Info.Println("Done creating table operating_system_vulnerabilities...")

	return nil
}

func Down_20221027085019(tx *sql.Tx) error {
	return nil
}
