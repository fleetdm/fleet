package tables

import (
	"database/sql"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20241210094935, Down_20241210094935)
}

func Up_20241210094935(tx *sql.Tx) error {
	createVppAppStmt := `
CREATE TABLE IF NOT EXISTS vpp_app_team_labels (
	id                   INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	vpp_app_team_id      INT(10) UNSIGNED NOT NULL,

	-- unlike for configuration profiles, the referenced label for software
	-- cannot be deleted, so we make it NOT NULL and no need to capture the name.
	label_id             INT(10) UNSIGNED NOT NULL,

	-- if exclude is true, "exclude_any" condition, otherwise "include_any"
	-- (we don't support include/exclude all for now, so not adding a 
	-- "require_all" column).
	exclude              TINYINT(1) NOT NULL DEFAULT 0,

	created_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	UNIQUE KEY idx_vpp_app_team_labels_vpp_app_team_id_label_id (vpp_app_team_id, label_id),

	FOREIGN KEY (vpp_app_team_id) REFERENCES vpp_apps_teams(id) ON DELETE CASCADE,

	-- because we want to prevent deleting a label if it is referenced by a vpp app,
	-- we explicitly enforce this at the database level with the RESTRICT clause.
	FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := tx.Exec(createVppAppStmt); err != nil {
		return errors.Wrap(err, "create vpp_app_team_labels table")
	}

	createSoftwareInstallerStmt := `
CREATE TABLE IF NOT EXISTS software_installer_labels (
	id                    INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
	software_installer_id INT(10) UNSIGNED NOT NULL,

	-- unlike for configuration profiles, the referenced label for software
	-- cannot be deleted, so we make it NOT NULL and no need to capture the name.
	label_id              INT(10) UNSIGNED NOT NULL,

	-- if exclude is true, "exclude_any" condition, otherwise "include_any"
	-- (we don't support include/exclude all for now, so not adding a 
	-- "require_all" column).
	exclude               TINYINT(1) NOT NULL DEFAULT 0,

	created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

	UNIQUE KEY idx_software_installer_labels_software_installer_id_label_id (software_installer_id, label_id),

	FOREIGN KEY (software_installer_id) REFERENCES software_installers(id) ON DELETE CASCADE,

	-- because we want to prevent deleting a label if it is referenced by an installer,
	-- we explicitly enforce this at the database level with the RESTRICT clause.
	FOREIGN KEY (label_id) REFERENCES labels(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := tx.Exec(createSoftwareInstallerStmt); err != nil {
		return errors.Wrap(err, "create software_installer_labels table")
	}

	return nil
}

func Down_20241210094935(tx *sql.Tx) error {
	return nil
}
