package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260729115013, Down_20260729115013)
}

func Up_20260729115013(tx *sql.Tx) error {
	// This table was created alongside the original DDM tables to model
	// many-to-many activation references, but was never written to by any code
	// path. Custom activations are 1:1 with their configuration declaration, so
	// drop it rather than carry it forward.
	if _, err := tx.Exec(`DROP TABLE IF EXISTS mdm_apple_declaration_activation_references`); err != nil {
		return fmt.Errorf("dropping mdm_apple_declaration_activation_references table: %w", err)
	}

	// Custom activations let admins override the activation Fleet otherwise
	// generates for a DDM configuration declaration, mainly to attach a
	// Predicate. The activation JSON is stored as-is (mediumtext, not json, so
	// the generated token hashes the exact stored bytes) and is never
	// interpreted by Fleet beyond validation of its envelope.
	//
	// declaration_uuid is the authoritative link to the configuration this
	// activation gates: it is 1:1 (enforced by its unique key) and cascades on
	// delete so removing a DDM profile removes its activation. The
	// configuration_identifier column keeps the declaration's Identifier
	// alongside it, since the activation JSON references its configuration by
	// Identifier rather than by UUID.
	_, err := tx.Exec(`
CREATE TABLE mdm_apple_ddm_activations (
	activation_uuid          varchar(37)  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
	team_id                  int unsigned NOT NULL DEFAULT '0',
	identifier               varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	raw_json                 mediumtext   CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	declaration_uuid         varchar(37)  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	configuration_identifier varchar(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci NOT NULL,
	secrets_updated_at       datetime(6)  DEFAULT NULL,
	created_at               timestamp(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	uploaded_at              timestamp(6) NULL DEFAULT NULL,
	token                    binary(16) GENERATED ALWAYS AS
		(unhex(md5(concat(raw_json, ifnull(secrets_updated_at, ''))))) STORED,

	PRIMARY KEY (activation_uuid),
	UNIQUE KEY idx_mdm_apple_ddm_activation_team_identifier (team_id, identifier),
	UNIQUE KEY idx_mdm_apple_ddm_activation_team_config (team_id, configuration_identifier),
	UNIQUE KEY idx_mdm_apple_ddm_activation_declaration (declaration_uuid),
	CONSTRAINT fk_mdm_apple_ddm_activations_declaration_uuid
		FOREIGN KEY (declaration_uuid) REFERENCES mdm_apple_declarations (declaration_uuid) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
	`)
	if err != nil {
		return fmt.Errorf("creating mdm_apple_ddm_activations table: %w", err)
	}

	// Activations can carry Fleet variables, so they need an owner column in
	// the profile variables table. The unique key is what makes that table's
	// ON DUPLICATE KEY UPDATE write path work, and the check constraint has to
	// be replaced to count the new column, otherwise every row setting it is
	// rejected.
	_, err = tx.Exec(`
		ALTER TABLE mdm_configuration_profile_variables
			ADD COLUMN apple_ddm_activation_uuid varchar(37) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci DEFAULT NULL,
			ADD UNIQUE KEY idx_mdm_config_profile_vars_ddm_activation_variable (apple_ddm_activation_uuid, fleet_variable_id),
			ADD CONSTRAINT mdm_config_profile_variables_ddm_activation_fk
				FOREIGN KEY (apple_ddm_activation_uuid) REFERENCES mdm_apple_ddm_activations (activation_uuid) ON DELETE CASCADE,
			DROP CHECK ck_mdm_configuration_profile_variables_exactly_one,
			ADD CONSTRAINT ck_mdm_configuration_profile_variables_exactly_one
				CHECK ((
					(IF(apple_profile_uuid IS NULL, 0, 1) +
					 IF(windows_profile_uuid IS NULL, 0, 1) +
					 IF(apple_declaration_uuid IS NULL, 0, 1) +
					 IF(android_profile_uuid IS NULL, 0, 1) +
					 IF(certificate_template_id IS NULL, 0, 1) +
					 IF(android_app_configuration_id IS NULL, 0, 1) +
					 IF(apple_ddm_activation_uuid IS NULL, 0, 1)) = 1
				))
	`)
	if err != nil {
		return fmt.Errorf("extending mdm_configuration_profile_variables for DDM activations: %w", err)
	}

	// Tracks when a host's activation last changed so the declaration's
	// effective token can be regenerated, mirroring variables_updated_at and
	// assets_updated_at. DATETIME(6) rather than TIMESTAMP because
	// EffectiveDDMToken formats these values into the token string and
	// TIMESTAMP would apply session timezone conversion on read.
	_, err = tx.Exec(`ALTER TABLE host_mdm_apple_declarations ADD COLUMN activation_updated_at DATETIME(6) DEFAULT NULL`)
	if err != nil {
		return fmt.Errorf("adding activation_updated_at to host_mdm_apple_declarations: %w", err)
	}

	return nil
}

func Down_20260729115013(tx *sql.Tx) error {
	return nil
}
