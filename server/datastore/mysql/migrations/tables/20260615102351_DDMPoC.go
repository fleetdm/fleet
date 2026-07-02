package tables

import (
	"database/sql"
)

func init() {
	MigrationClient.AddMigration(Up_20260615102351, Down_20260615102351)
}

func Up_20260615102351(tx *sql.Tx) error {
	_, err := tx.Exec(`CREATE TABLE mdm_apple_declaration_assets (
	asset_uuid VARCHAR(36) NOT NULL PRIMARY KEY,
	name VARCHAR(255) NOT NULL,
	identifier VARCHAR(255) NOT NULL,
	raw_json MEDIUMBLOB NOT NULL, -- We encrypt the payload since it's often credentials
	created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
	uploaded_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
	token BINARY(16) GENERATED ALWAYS AS (unhex(md5(raw_json))) STORED, -- Add secrets_updated_at if we want to support that like the DDM token
	auto_increment BIGINT NOT NULL AUTO_INCREMENT UNIQUE,
	UNIQUE KEY unique_identifier (identifier)
	)`)

	// TODO: Should we introduce this into the mdm_configuration_profile_variables, or keep it locally to this table?
	return err
}

func Down_20260615102351(tx *sql.Tx) error {
	return nil
}
