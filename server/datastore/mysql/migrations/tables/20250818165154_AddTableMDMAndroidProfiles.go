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
	-- store as JSON so we can use JSON_CONTAINS to check if the applied JSON document
	-- contains this profile's JSON.
  raw_json       JSON NOT NULL,

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

	alterProfileLabelsTable := `
ALTER TABLE mdm_configuration_profile_labels
	ADD COLUMN google_profile_uuid VARCHAR(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NULL DEFAULT NULL,
	ADD FOREIGN KEY (google_profile_uuid) REFERENCES mdm_google_configuration_profiles(profile_uuid) ON DELETE CASCADE,
	ADD UNIQUE KEY idx_mdm_configuration_profile_labels_google_label_name (google_profile_uuid, label_name),
	DROP CONSTRAINT ck_mdm_configuration_profile_labels_apple_or_windows,
	-- only one of apple, google or windows profile uuid must be set
	ADD CONSTRAINT ck_mdm_configuration_profile_labels_profile_uuid
		CHECK (IF(ISNULL(apple_profile_uuid), 0, 1) + IF(ISNULL(windows_profile_uuid), 0, 1) + IF(ISNULL(google_profile_uuid), 0, 1) = 1)
`
	if _, err := tx.Exec(alterProfileLabelsTable); err != nil {
		return fmt.Errorf("alter mdm_configuration_profile_labels table: %w", err)
	}

	// TODO: add google_profile_uuid to mdm_configuration_profile_variables (even
	// if unsupported for now)? Opted not to, as I also did not add the
	// variables/secrets updated at columns.

	// TODO: I think we should keep track of the fully merged profile and the
	// requests/responses made to the android management API, a bit like we track
	// declaration requests and mdm commands. We'd then add the API request uuid
	// reference to the host mdm google profiles table. This table would store
	// the fully resulved/merged profile (android policy) that was sent to be
	// applied for the host.

	// TODO: I think we may need a table to keep each setting set by each profile
	// file, so that we can show it as failed or succeeded based on the status
	// report from the Google API. E.g. top-level keys and associated value
	// (maybe in JSON column). (because the setting in file a.json may be
	// overwritten by file b.json, so in order to know if a.json succeeded or
	// not, we'd check the status for the applied settings, see that the
	// corresponding value is not the one declared in a.json but instead the one
	// in b.json, so we mark a.json as failed and b.json as succeeded).
	//
	// Not having this lookup table would mean loading and unmarshaling the JSON
	// for every applicable profile whenever we receive a status update, which
	// would not be efficient.
	//
	// It looks like mysql's storage of JSON is normalized/canonicalized, so we
	// can likely compare the JSON values directly even if the value is an object
	// or array.
	// https://dev.mysql.com/doc/refman/8.4/en/json.html#json-normalization
	//
	// But it may not even matter, as we could possibly use the JSON_CONTAINS
	// JSON function to check if the profile's JSON document is contained as-is
	// in the applied settings, and if so it succeeded, otherwise it failed.
	// https://dev.mysql.com/doc/refman/8.4/en/json-search-functions.html#function_json-contains
	//
	// In which case we may not need a separate table for the key-values of the
	// profile file's JSON document, we could just store the profile's JSON
	// document as JSON in the mdm_google_configuration_profiles table.
	// I'll go with that approach, we can revisit if the verification is not
	// as straightforward as expected.

	return nil
}

func Down_20250818165154(tx *sql.Tx) error {
	return nil
}
