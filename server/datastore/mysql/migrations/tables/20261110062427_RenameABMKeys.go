package tables

import (
	"database/sql"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261110062427, Down_20261110062427)
}

var abmConfigKeyRenames = []struct{ Old, New string }{
	{Old: "apple_business_manager", New: "apple_business"},
	{Old: "apple_bm_enabled_and_configured", New: "apple_business_enabled_and_configured"},
	{Old: "apple_bm_terms_expired", New: "apple_business_terms_expired"},
}

// Up_20261110062427 moves the stored Apple Business settings to their new names and drops apple_bm_default_team.
func Up_20261110062427(tx *sql.Tx) error {
	for _, r := range abmConfigKeyRenames {
		oldPath, newPath := "$.mdm."+r.Old, "$.mdm."+r.New
		//nolint:gosec // built from abmConfigKeyRenames, no user input
		stmt := fmt.Sprintf(
			`UPDATE app_config_json SET json_value = JSON_REMOVE(JSON_SET(json_value, '%[2]s', JSON_EXTRACT(json_value, '%[1]s')), '%[1]s')
			WHERE JSON_CONTAINS_PATH(json_value, 'one', '%[1]s')`,
			oldPath, newPath,
		)
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("rename %s in app_config_json: %w", oldPath, err)
		}
	}

	// We are not renaming *_team to *_fleet inside the appconfig, as that would break until the equivalent team->fleet renaming has happened in code.

	// Already copied into each token's default fleets by 20240829165448_SupportMultipleABMTokens.
	if _, err := tx.Exec(`
		UPDATE app_config_json
		SET json_value = JSON_REMOVE(json_value, '$.mdm.apple_bm_default_team')
		WHERE JSON_CONTAINS_PATH(json_value, 'one', '$.mdm.apple_bm_default_team')`); err != nil {
		return fmt.Errorf("remove apple_bm_default_team from app_config_json: %w", err)
	}
	return nil
}

func Down_20261110062427(tx *sql.Tx) error {
	return nil
}
