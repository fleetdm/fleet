package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240307162019, Down_20240307162019)
}

func Up_20240307162019(tx *sql.Tx) error {

	tables := []string{
		"nano_commands", "windows_mdm_commands",
		"mdm_apple_configuration_profiles", "mdm_windows_configuration_profiles",
	}

	for _, t := range tables {
		_, err := tx.Exec(fmt.Sprintf(`
			ALTER TABLE`+" `%s` "+`
			-- user_id references the user that created the entity.
			-- it's NULL for rows created prior to this migration,
			-- and also for entities that don't have an user
			-- associated with it (eg: Fleet initiated actions)
			ADD COLUMN user_id int(10) unsigned DEFAULT NULL,

			-- fleet_owned indicates if the entity is managed by Fleet.
			ADD COLUMN fleet_owned tinyint(1) DEFAULT NULL

			-- this check is only parsed in MySQL 8+
		--	CHECK (
		--	  (user_id IS NOT NULL AND fleet_owned = 0) OR
		--	  (user_id IS NULL AND fleet_owned = 1) OR
		--	  (user_id IS NULL AND fleet_ownded IS NULL)
		--	)`, t))
		if err != nil {
			return fmt.Errorf("failed to add user_id and fleet_owned to %s: %w", t, err)
		}

		_, err = tx.Exec(fmt.Sprintf(`
			ALTER TABLE`+" `%s` "+`
			ADD CONSTRAINT`+" `fk_%s_users` "+`
			FOREIGN KEY (user_id) REFERENCES users(id)
			ON DELETE SET NULL`, t, t))
		if err != nil {
			return fmt.Errorf("failed to add user_id foreign key to %s: %w", t, err)
		}
	}

	return nil
}

func Down_20240307162019(tx *sql.Tx) error {
	return nil
}
