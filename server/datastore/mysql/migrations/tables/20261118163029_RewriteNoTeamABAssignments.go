package tables

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func init() {
	MigrationClient.AddMigration(Up_20261118163029, Down_20261118163029)
}

// Up_20261118163029 rewrites "No team", in any letter case, in the stored Apple Business assignments to the empty string that
// unassigned saves as. Fleet 5 rejects "No team", and GET /config and generate-gitops would hand it back. Old key names are covered too,
// in case the key renames have not run yet.
func Up_20261118163029(tx *sql.Tx) error {
	var raw []byte
	switch err := tx.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1 FOR UPDATE`).Scan(&raw); {
	case errors.Is(err, sql.ErrNoRows):
		return nil
	case err != nil:
		return fmt.Errorf("select app_config_json: %w", err)
	}
	var appConfig struct {
		MDM map[string]any `json:"mdm"`
	}
	if err := json.Unmarshal(raw, &appConfig); err != nil {
		return fmt.Errorf("unmarshal app_config_json: %w", err)
	}

	for _, listKey := range []string{"apple_business_manager", "apple_business"} {
		entries, _ := appConfig.MDM[listKey].([]any)
		for i, e := range entries {
			entry, _ := e.(map[string]any)
			for _, nameKey := range []string{"macos_team", "ios_team", "ipados_team", "byod_team", "macos_fleet", "ios_fleet", "ipados_fleet", "byod_fleet"} {
				if name, _ := entry[nameKey].(string); !strings.EqualFold(name, "No team") {
					continue
				}
				path := fmt.Sprintf("$.mdm.%s[%d].%s", listKey, i, nameKey)
				if _, err := tx.Exec(`UPDATE app_config_json SET json_value = JSON_SET(json_value, ?, '') WHERE id = 1`, path); err != nil {
					return fmt.Errorf("update %s in app_config_json: %w", path, err)
				}
			}
		}
	}
	return nil
}

func Down_20261118163029(tx *sql.Tx) error {
	return nil
}
