package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20231204155427, Down_20231204155427)
}

func Up_20231204155427(tx *sql.Tx) error {
	// update the windows profiles tables to use a 37-char uuid column for
	// the 'w' prefix.
	_, err := tx.Exec(`
ALTER TABLE host_mdm_windows_profiles
	CHANGE COLUMN profile_uuid profile_uuid VARCHAR(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter host_mdm_windows_profiles table: %w", err)
	}
	_, err = tx.Exec(`
ALTER TABLE mdm_windows_configuration_profiles
	CHANGE COLUMN profile_uuid profile_uuid VARCHAR(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_windows_configuration_profiles table: %w", err)
	}

	// add the 'w' prefix to the windows profiles table
	_, err = tx.Exec(`
UPDATE
	mdm_windows_configuration_profiles mwcp
SET
	profile_uuid = CONCAT('w', profile_uuid),
	updated_at = mwcp.updated_at
`)
	if err != nil {
		return fmt.Errorf("failed to update mdm_windows_configuration_profiles table: %w", err)
	}
	_, err = tx.Exec(`
UPDATE
	host_mdm_windows_profiles
SET
	profile_uuid = CONCAT('w', profile_uuid)
`)
	if err != nil {
		return fmt.Errorf("failed to update host_mdm_windows_profiles table: %w", err)
	}

	// update the apple profiles table to add the profile_uuid column.
	_, err = tx.Exec(`
ALTER TABLE mdm_apple_configuration_profiles
	-- 37 and not 36 because the UUID will be prefixed with 'a' to indicate
	-- that it's an Apple profile.
	ADD COLUMN profile_uuid VARCHAR(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter mdm_apple_configuration_profiles table: %w", err)
	}

	// generate the uuids for the apple profiles table
	_, err = tx.Exec(`
UPDATE
	mdm_apple_configuration_profiles macp
SET
	-- see https://stackoverflow.com/a/51393124/1094941
	profile_uuid = CONCAT('a', CONVERT(uuid() USING utf8mb4)),
	updated_at = macp.updated_at
`)
	if err != nil {
		return fmt.Errorf("failed to update mdm_apple_configuration_profiles table: %w", err)
	}

	// set the profile uuid as the new primary key
	_, err = tx.Exec(`
ALTER TABLE mdm_apple_configuration_profiles
	-- auto-increment column must have an index, so we create one before
	-- dropping the primary key.
	ADD UNIQUE KEY idx_mdm_apple_config_prof_id (profile_id),
	DROP PRIMARY KEY,
	ADD PRIMARY KEY (profile_uuid)`)
	if err != nil {
		return fmt.Errorf("failed to set primary key of mdm_apple_configuration_profiles table: %w", err)
	}

	// add the profile_uuid column to the host apple profiles table, keeping the
	// old id for now.
	_, err = tx.Exec(`
ALTER TABLE host_mdm_apple_profiles
	ADD COLUMN profile_uuid VARCHAR(37) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT ''
`)
	if err != nil {
		return fmt.Errorf("failed to alter host_mdm_apple_profiles table: %w", err)
	}

	// update the apple host profiles table's profile_uuid based on its profile_id
	_, err = tx.Exec(`
UPDATE
	host_mdm_apple_profiles
SET
	profile_uuid = COALESCE((
		SELECT
			macp.profile_uuid
		FROM
			mdm_apple_configuration_profiles macp
		WHERE
			host_mdm_apple_profiles.profile_id = macp.profile_id
	-- see https://stackoverflow.com/a/51393124/1094941
	), CONCAT('a', CONVERT(uuid() USING utf8mb4)))
`)
	if err != nil {
		return fmt.Errorf("failed to update host_mdm_apple_profiles table: %w", err)
	}

	// drop the now unused profile_id column from the host apple profiles table
	_, err = tx.Exec(`ALTER TABLE host_mdm_apple_profiles
		DROP PRIMARY KEY,
		ADD PRIMARY KEY (host_uuid, profile_uuid),
		DROP COLUMN profile_id`)
	if err != nil {
		return fmt.Errorf("failed to drop column from host_mdm_apple_profiles table: %w", err)
	}

	return nil
}

func Down_20231204155427(tx *sql.Tx) error {
	return nil
}
