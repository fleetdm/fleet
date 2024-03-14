package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20240313143039, Down_20240313143039)
}

func Up_20240313143039(tx *sql.Tx) error {
	// create a new table to store user information that's persisted after
	// users are deleted.
	_, err := tx.Exec(`
	  CREATE TABLE IF NOT EXISTS user_persistent_info (
	    -- id is an unique identifier for the row, independent from whatever is stored in 'users'
	    id int(10) unsigned NOT NULL AUTO_INCREMENT,

	    -- user_id is a nullable FK reference to the users table
	    user_id int(10) unsigned DEFAULT NULL,

	    -- user_name mirrors the users.name value
	    user_name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

	    -- user_email mirrors the users.email value
	    user_email varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL,

	    -- timestamps
	    created_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
	    updated_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	    PRIMARY KEY (id),
	    UNIQUE INDEX idx_unique_user_id (user_id),
	    FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
	  )
      `)
	if err != nil {
		return fmt.Errorf("failed to add user_persistent_info table: %w", err)
	}

	// migrate existing data.
	_, err = tx.Exec(`
	  INSERT INTO user_persistent_info (user_id, user_name, user_email)
	  SELECT id, name, email
	  FROM users
	`)
	if err != nil {
		return fmt.Errorf("failed to add user information into the user_persistent_info table: %w", err)
	}

	tables := []string{
		"nano_commands", "windows_mdm_commands",
		"mdm_apple_configuration_profiles", "mdm_windows_configuration_profiles",
	}

	for _, t := range tables {
		_, err := tx.Exec(fmt.Sprintf(`
			ALTER TABLE`+" `%s` "+`
			-- user_persistent_info_id references the user that created the entity.
			-- it's NULL for rows created prior to this migration,
			-- and also for entities that don't have an user
			-- associated with it (eg: Fleet initiated actions)
			ADD COLUMN user_persistent_info_id int(10) unsigned DEFAULT NULL,

			-- fleet_owned indicates if the entity is managed by Fleet.
			ADD COLUMN fleet_owned tinyint(1) DEFAULT NULL
			`, t))
		if err != nil {
			return fmt.Errorf("failed to add user_persistent_info_id and fleet_owned to %s: %w", t, err)
		}

		_, err = tx.Exec(fmt.Sprintf(`
			ALTER TABLE`+" `%s` "+`
			ADD CONSTRAINT`+" `fk_%s_user_info` "+`
			FOREIGN KEY (user_persistent_info_id) REFERENCES user_persistent_info(id)
			ON DELETE RESTRICT`, t, t))
		if err != nil {
			return fmt.Errorf("failed to add user_persistent_info_id foreign key to %s: %w", t, err)
		}
	}

	return nil
}

func Down_20240313143039(tx *sql.Tx) error {
	return nil
}
