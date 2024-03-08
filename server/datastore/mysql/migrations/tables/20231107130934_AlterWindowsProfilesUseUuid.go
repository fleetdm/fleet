package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231107130934, Down_20231107130934)
}

func Up_20231107130934(tx *sql.Tx) error {
	// add the profile_uuid column to the profiles table, keeping the old id.
	// Note that we cannot set the default to uuid() as functions cannot be used
	// as defaults in mysql 5.7. It will have to be generated in code when
	// inserting. Cannot be set as primary key yet as it may have duplicates until
	// we generate the uuids.
	_, err := tx.Exec(`
ALTER TABLE mdm_windows_configuration_profiles
	-- required to remove AUTO_INCREMENT because it must be a primary key
	CHANGE COLUMN profile_id profile_id INT(10) UNSIGNED NOT NULL,
	DROP PRIMARY KEY,
	ADD COLUMN profile_uuid VARCHAR(36) NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_windows_configuration_profiles table: %w", err)
	}

	// add the profile_uuid column to the host profiles table, keeping the old
	// id. Cannot be set as primary key yet as it may have duplicates until we
	// generate the uuids.
	_, err = tx.Exec(`
ALTER TABLE host_mdm_windows_profiles
	DROP PRIMARY KEY,
	ADD COLUMN profile_uuid VARCHAR(36) NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter host_mdm_windows_profiles table: %w", err)
	}

	// generate the uuids for the profiles table
	_, err = tx.Exec(`
UPDATE
	mdm_windows_configuration_profiles
SET
	profile_uuid = uuid()
`)
	if err != nil {
		return fmt.Errorf("failed to update mdm_windows_configuration_profiles table: %w", err)
	}

	// update the host profiles table's profile_uuid based on its profile_id
	_, err = tx.Exec(`
UPDATE
	host_mdm_windows_profiles
SET
	profile_uuid = COALESCE((
		SELECT
			mwcp.profile_uuid
		FROM
			mdm_windows_configuration_profiles mwcp
		WHERE
			host_mdm_windows_profiles.profile_id = mwcp.profile_id
	), uuid())
`)
	if err != nil {
		return fmt.Errorf("failed to update host_mdm_windows_profiles table: %w", err)
	}

	// drop the now unused profile_id column from both tables
	_, err = tx.Exec(`ALTER TABLE mdm_windows_configuration_profiles
		ADD PRIMARY KEY (profile_uuid),
		DROP COLUMN profile_id`)
	if err != nil {
		return fmt.Errorf("failed to drop column from mdm_windows_configuration_profiles table: %w", err)
	}
	_, err = tx.Exec(`ALTER TABLE host_mdm_windows_profiles
		ADD PRIMARY KEY (host_uuid, profile_uuid),
		DROP COLUMN profile_id`)
	if err != nil {
		return fmt.Errorf("failed to drop column from host_mdm_windows_profiles table: %w", err)
	}

	return nil
}

func Down_20231107130934(tx *sql.Tx) error {
	return nil
}
