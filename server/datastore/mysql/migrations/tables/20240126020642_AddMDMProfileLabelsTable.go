package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240126020642, Down_20240126020642)
}

func Up_20240126020642(tx *sql.Tx) error {
	createStmt := `
    CREATE TABLE IF NOT EXISTS mdm_configuration_profile_labels (
			id                   INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,

			-- using distinct fields for the profile uuid so that proper foreign keys
			-- can be created to the apple and windows tables.
			apple_profile_uuid   VARCHAR(37) NULL,
			windows_profile_uuid VARCHAR(37) NULL,

			-- label name is stored here because we need to list the labels in the UI
			-- even if it has been deleted from the labels table.
			label_name           VARCHAR(255) NOT NULL,

			-- label id is nullable in case it gets deleted from the labels table.
			-- A row in this table with label_id = null indicates the "broken" state
			-- in the UI.
			label_id             INT(10) UNSIGNED NULL,

			created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

			-- cannot have a single unique key with apple+windows+label name because
			-- NULLs are not considered equal in unique keys (so "W1+null+L1" is not
			-- a duplicate of itself). Using two distinct unique keys instead, and there's
			-- a check constraint to ensure that only one of the apple or windows
			-- profile uuid can be set.
			UNIQUE KEY idx_mdm_configuration_profile_labels_apple_label_name (apple_profile_uuid, label_name),
			UNIQUE KEY idx_mdm_configuration_profile_labels_windows_label_name (windows_profile_uuid, label_name),

			FOREIGN KEY (apple_profile_uuid) REFERENCES mdm_apple_configuration_profiles(profile_uuid) ON DELETE CASCADE,
			FOREIGN KEY (windows_profile_uuid) REFERENCES mdm_windows_configuration_profiles(profile_uuid) ON DELETE CASCADE,
			FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE SET NULL,

			-- TODO(mna): CHECK constraint is parsed but ignored on mysql 5.7, will have to do without.

			-- exactly one of apple or windows profile uuid must be set
			CONSTRAINT ck_mdm_configuration_profile_labels_apple_or_windows
				CHECK (ISNULL(apple_profile_uuid) <> ISNULL(windows_profile_uuid))
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := tx.Exec(createStmt); err != nil {
		return errors.Wrap(err, "create mdm_configuration_profile_labels table")
	}

	return nil
}

func Down_20240126020642(tx *sql.Tx) error {
	return nil
}
