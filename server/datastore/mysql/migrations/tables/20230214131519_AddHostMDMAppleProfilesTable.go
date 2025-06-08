package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230214131519, Down_20230214131519)
}

func Up_20230214131519(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE mdm_apple_delivery_status (
            status VARCHAR(20) PRIMARY KEY
          ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
          INSERT INTO mdm_apple_delivery_status (status)
          VALUES ('failed'), ('applied'), ('pending')
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
          CREATE TABLE mdm_apple_operation_types (
            operation_type VARCHAR(20) PRIMARY KEY
          ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
          INSERT INTO mdm_apple_operation_types (operation_type)
          VALUES ('install'), ('remove')
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
          CREATE TABLE host_mdm_apple_profiles (
            profile_id          int(10) UNSIGNED NOT NULL,
            profile_identifier  varchar(255) NOT NULL,
            host_uuid           varchar(255) NOT NULL,
	    status              varchar(20) DEFAULT NULL,
	    operation_type      varchar(20) DEFAULT NULL,
	    detail              text,
            command_uuid        varchar(127) NOT NULL,

	    PRIMARY KEY (host_uuid, profile_id),
	    FOREIGN KEY (status) REFERENCES mdm_apple_delivery_status (status) ON UPDATE CASCADE,
	    FOREIGN KEY (operation_type) REFERENCES mdm_apple_operation_types (operation_type) ON UPDATE CASCADE
          ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	return err
}

func Down_20230214131519(tx *sql.Tx) error {
	return nil
}
