package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20230206163608, Down_20230206163608)
}

func Up_20230206163608(tx *sql.Tx) error {
	_, err := tx.Exec(`
CREATE TABLE mdm_apple_configuration_profiles (
	profile_id INT(10) UNSIGNED NOT NULL AUTO_INCREMENT,
	team_id INT(10) UNSIGNED NOT NULL DEFAULT 0, 
	-- team_id is zero for configuration profiles that are not associated with any team
	identifier VARCHAR(255) NOT NULL,
	name VARCHAR(255) NOT NULL,
	mobileconfig BLOB NOT NULL,
	created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at timestamp NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	
	PRIMARY KEY (profile_id),
	UNIQUE KEY idx_mdm_apple_config_prof_team_identifier (team_id, identifier),
	UNIQUE KEY idx_mdm_apple_config_prof_team_name (team_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`)
	if err != nil {
		return errors.Wrapf(err, "create table")
	}
	return nil
}

func Down_20230206163608(tx *sql.Tx) error {
	return nil
}
