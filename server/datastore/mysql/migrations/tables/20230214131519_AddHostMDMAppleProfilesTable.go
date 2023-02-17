package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20230214131519, Down_20230214131519)
}

func Up_20230214131519(tx *sql.Tx) error {
	_, err := tx.Exec(`
          CREATE TABLE mdm_apple_profile_status (
            status VARCHAR(20) PRIMARY KEY
          )`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
          INSERT INTO mdm_apple_profile_status (status)
          VALUES ('FAILED'), ('INSTALLED'), ('INSTALLING'), ('REMOVING')
	`)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
          CREATE TABLE host_mdm_apple_profiles (
            profile_id int(10) UNSIGNED NOT NULL,
            host_uuid  varchar(255) NOT NULL,
	    status     varchar(20) DEFAULT NULL,
	    error      text,

	    PRIMARY KEY (host_uuid, profile_id),
            FOREIGN KEY (profile_id) REFERENCES mdm_apple_configuration_profiles (profile_id) ON UPDATE CASCADE,
	    FOREIGN KEY (status) REFERENCES mdm_apple_profile_status (status) ON UPDATE CASCADE
          )`)
	return err
}

func Down_20230214131519(tx *sql.Tx) error {
	return nil
}
