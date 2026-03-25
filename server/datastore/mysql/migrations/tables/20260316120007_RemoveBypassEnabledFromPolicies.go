package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20260316120007, Down_20260316120007)
}

func Up_20260316120007(tx *sql.Tx) error {
	// Promote bypass=true policies to critical for teams that have Okta conditional
	// access enabled, but only when Okta is globally configured.
	_, err := tx.Exec(`
		UPDATE policies
		SET critical = TRUE
		WHERE conditional_access_bypass_enabled = TRUE
		  AND team_id IN (
		      SELECT id FROM teams
		      WHERE JSON_UNQUOTE(JSON_EXTRACT(config, '$.integrations.conditional_access_enabled')) = 'true'
		  )
		  AND EXISTS (
		      SELECT 1 FROM app_config_json
		      WHERE IFNULL(JSON_UNQUOTE(JSON_EXTRACT(json_value, '$.conditional_access.okta_idp_id')), '') != ''
		        AND IFNULL(JSON_UNQUOTE(JSON_EXTRACT(json_value, '$.conditional_access.okta_assertion_consumer_service_url')), '') != ''
		        AND IFNULL(JSON_UNQUOTE(JSON_EXTRACT(json_value, '$.conditional_access.okta_audience_uri')), '') != ''
		        AND IFNULL(JSON_UNQUOTE(JSON_EXTRACT(json_value, '$.conditional_access.okta_certificate')), '') != ''
		  )
	`)
	if err != nil {
		return fmt.Errorf("migrate policy bypass enabled field: %w", err)
	}

	if _, err := tx.Exec("ALTER TABLE policies DROP COLUMN conditional_access_bypass_enabled"); err != nil {
		return fmt.Errorf("migrate policies drop bypass enabled field: %w", err)
	}

	return nil
}

func Down_20260316120007(tx *sql.Tx) error {
	return nil
}
