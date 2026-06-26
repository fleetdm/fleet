package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260624152755, Down_20260624152755)
}

func Up_20260624152755(tx *sql.Tx) error {
	_, err := tx.Exec(`
		ALTER TABLE mdm_configuration_profile_variables
			ADD COLUMN certificate_template_id int unsigned DEFAULT NULL,
			ADD UNIQUE KEY idx_mdm_configuration_profile_variables_cert_template_variable (certificate_template_id, fleet_variable_id),
			ADD CONSTRAINT fk_mdm_configuration_profile_variables_cert_template_id
				FOREIGN KEY (certificate_template_id) REFERENCES certificate_templates (id) ON DELETE CASCADE,
			ADD COLUMN android_app_configuration_id int unsigned DEFAULT NULL,
			ADD UNIQUE KEY idx_mdm_configuration_profile_variables_app_config_variable (android_app_configuration_id, fleet_variable_id),
			ADD CONSTRAINT fk_mdm_configuration_profile_variables_app_config_id
				FOREIGN KEY (android_app_configuration_id) REFERENCES android_app_configurations (id) ON DELETE CASCADE,
			DROP CHECK ck_mdm_configuration_profile_variables_exactly_one,
			ADD CONSTRAINT ck_mdm_configuration_profile_variables_exactly_one
				CHECK ((
					(IF(apple_profile_uuid IS NULL, 0, 1) +
					 IF(windows_profile_uuid IS NULL, 0, 1) +
					 IF(apple_declaration_uuid IS NULL, 0, 1) +
					 IF(android_profile_uuid IS NULL, 0, 1) +
					 IF(certificate_template_id IS NULL, 0, 1) +
					 IF(android_app_configuration_id IS NULL, 0, 1)) = 1
				))
	`)
	if err != nil {
		return fmt.Errorf("alter mdm_configuration_profile_variables for cert templates and app configs: %w", err)
	}
	return nil
}

func Down_20260624152755(tx *sql.Tx) error {
	return nil
}
