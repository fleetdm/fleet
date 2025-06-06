package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230303135738, Down_20230303135738)
}

func Up_20230303135738(tx *sql.Tx) error {
	_, err := tx.Exec(`
    CREATE TABLE mdm_idp_accounts (
      uuid         varchar(255) NOT NULL,
      username     varchar(255) NOT NULL,
      salt         varchar(255) NOT NULL,
      entropy      varchar(255) NOT NULL,
      iterations   int unsigned NOT NULL,
    
      PRIMARY KEY (uuid)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	return err
}

func Down_20230303135738(tx *sql.Tx) error {
	return nil
}
