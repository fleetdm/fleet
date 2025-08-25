package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20250818165154, Down_20250818165154)
}

func Up_20250818165154(tx *sql.Tx) error {
	// The AUTO_INCREMENT columns are used to determine if a row was updated by
	// an INSERT ... ON DUPLICATE KEY UPDATE statement. This is needed because we
	// are currently using CLIENT_FOUND_ROWS option to determine if a row was
	// found. And in order to find if the row was updated, we need to check
	// LAST_INSERT_ID(). MySQL docs:
	// https://dev.mysql.com/doc/refman/8.4/en/insert-on-duplicate.html

	// TODO: thoughts about naming it `google` as suggested by Sarah at standup?
	// Makes it a bit clearer that the 'g' prefix is used for profile_uuid (and
	// not 'a' - apple, nor 'd' - apple declarations).

	// TODO: not adding `secrets_updated_at` / `variables_updated_at` anywhere
	// just yet as we won't support them in Android profiles for now.

	createProfilesTable := `
CREATE TABLE mdm_google_configuration_profiles (
	-- profile_uuid is length 37 as it has a single char prefix (of 'g') before the actual uuid
  profile_uuid   VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	-- no FK constraint on teams, the profile is manually deleted when the team is deleted
  team_id        INT UNSIGNED NOT NULL DEFAULT '0',
	-- unique across all profiles (all platforms), must be checked on insert with the apple,
	-- windows and apple declaration names.
  name           VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
	-- same content column as for the apple declarations json
  raw_json       MEDIUMTEXT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,

	-- see note comment above, needed for upserts
	auto_increment BIGINT NOT NULL AUTO_INCREMENT UNIQUE,

  created_at     TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  uploaded_at    TIMESTAMP(6) NULL DEFAULT CURRENT_TIMESTAMP(6),

  PRIMARY KEY (profile_uuid),
  UNIQUE KEY idx_mdm_google_configuration_profiles_team_id_name (team_id, name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
	if _, err := tx.Exec(createProfilesTable); err != nil {
		return fmt.Errorf("create mdm_google_configuration_profiles table: %w", err)
	}

	// TODO: do we get to define the policyId? Could it simply be the host's
	// uuid? TBD if we have to store this info, and where.

	// TODO: not adding the `retries` column as we don't deliver the profiles to
	// the hosts like apple/windows profiles, it's more similar to apple
	// declarations in that sense (we call the Google API, if it succeeds then
	// it's "delivered", if not we have to resend it regardless of the number of
	// retries).

	createHostProfilesTable := `
CREATE TABLE host_mdm_google_profiles (
  host_uuid varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
  status varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  operation_type varchar(20) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
  detail text CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci,
	profile_uuid varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	profile_name varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',

	created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),


  PRIMARY KEY (host_uuid, profile_uuid),
  FOREIGN KEY (status) REFERENCES mdm_delivery_status (status) ON UPDATE CASCADE,
  FOREIGN KEY (operation_type) REFERENCES mdm_operation_types (operation_type) ON UPDATE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`
	if _, err := tx.Exec(createHostProfilesTable); err != nil {
		return fmt.Errorf("create host_mdm_google_profiles table: %w", err)
	}

	// TODO: we want the profile name to be unique across all platforms (as we do for apple+windows)
	// which means checking the apple (profiles+decls) and windows tables on insert.
	// https://www.figma.com/design/sPlICOpfq9w3FG8vAuJFEO/-25557-Configuration-profiles-for-Android?node-id=5378-2783&t=RNOQg4AOkuIu31Wo-0

	// TODO: add google_profile_uuid to mdm_configuration_profile_labels
	// 	alterProfileLabelsTable := `
	// ALTER TABLE mdm_configuration_profile_labels
	// `
	// TODO: add google_profile_uuid to mdm_configuration_profile_variables (even
	// if unsupported for now)? Opted not too, as I also did not add the
	// variables updated at column.
	return nil
}

func Down_20250818165154(tx *sql.Tx) error {
	return nil
}
