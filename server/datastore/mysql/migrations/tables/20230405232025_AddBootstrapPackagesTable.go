package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230405232025, Down_20230405232025)
}

func Up_20230405232025(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE mdm_apple_bootstrap_packages (
            team_id    int(10) unsigned NOT NULL,
            name       varchar(255),
            sha256     BINARY(32) NOT NULL,
            bytes      longblob,
            token      varchar(36),

	    PRIMARY KEY (team_id),
	    UNIQUE KEY idx_token (token)
          ) DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_unicode_ci`)
	if err != nil {
		return err
	}

	return nil
}

func Down_20230405232025(tx *sql.Tx) error {
	return nil
}
