package tables

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240810072941, Down_20240810072941)
}

func Up_20240810072941(tx *sql.Tx) error {
	// On the first iteration we will have "manual" and "automatic", but we are planning on adding more
	// types in the future (e.g. "update", "force").
	if _, err := tx.Exec("ALTER TABLE software_installers ADD COLUMN install_type VARCHAR(20) NOT NULL DEFAULT 'manual'"); err != nil {
		return fmt.Errorf("Failed to add install_type to software_installers: %w", err)
	}

	// Table 'software_installer_labels' follows the same pattern as the 'mdm_configuration_profile_labels'
	// (for product consistency).
	//
	// NOTE: We will not allow deleting a label that is associated to a software installer. User has to delete
	// the referenced software installer and then delete the label.
	createStmt := `
    CREATE TABLE IF NOT EXISTS software_installer_labels (
			id                    INT(10) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,

			software_installer_id INT(10) UNSIGNED NOT NULL,
			label_id              INT(10) UNSIGNED NOT NULL,
			exclude 			  TINYINT(1) NOT NULL,

			created_at            TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
			updated_at            TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),

			UNIQUE KEY idx_software_installer_id_and_label_id (software_installer_id, label_id),

			FOREIGN KEY (software_installer_id) REFERENCES software_installers(id) ON DELETE CASCADE,
			FOREIGN KEY (label_id) REFERENCES labels(id)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
`
	if _, err := tx.Exec(createStmt); err != nil {
		return errors.Wrap(err, "create software_installer_labels table")
	}

	return nil
}

func Down_20240810072941(tx *sql.Tx) error {
	return nil
}
