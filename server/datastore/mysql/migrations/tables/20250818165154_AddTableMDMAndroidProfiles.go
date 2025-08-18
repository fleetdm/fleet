package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20250818165154, Down_20250818165154)
}

func Up_20250818165154(tx *sql.Tx) error {
	createProfilesTable := `
-- name it 'google' instead of 'android'? Makes it clear(er) that the 'g' prefix is used for profile_uuid
-- (and not 'a' - apple, nor 'd' - apple declarations)
CREATE TABLE mdm_google_configuration_profiles (
  profile_uuid varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  team_id      int unsigned NOT NULL DEFAULT '0',
  name         varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci  NOT NULL,
  content      mediumblob NOT NULL,
  created_at   timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  uploaded_at  timestamp(6) NULL DEFAULT NULL,
	auto_increment bigint NOT NULL AUTO_INCREMENT, -- TODO: why do we have this? Do we need it for android?
  checksum     binary(16) GENERATED ALWAYS AS (unhex(md5(content))) STORED,
	secrets_updated_at datetime(6) DEFAULT NULL, -- TODO: unused for now, should we create it immediately?

  PRIMARY KEY (profile_uuid),
  UNIQUE KEY idx_mdm_google_configuration_profiles_team_id_name (team_id, name),
	UNIQUE KEY auto_increment (auto_increment) -- TODO: why?
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
`

	// TODO(mna): presumably we want the profile name to be unique across all platforms (as we do for apple+windows)
	// TODO(mna): add google_profile_uuid to mdm_configuration_profile_labels
	// TODO(mna): add google_profile_uuid to mdm_configuration_profile_variables (even if unsupported for now)?
	return nil
}

func Down_20250818165154(tx *sql.Tx) error {
	return nil
}
