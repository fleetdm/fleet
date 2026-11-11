package tables

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

func init() {
	MigrationClient.AddMigration(Up_20261111205515, Down_20261111205515)
}

// abmVPPConfigKeyRenames are the keys, by the config list they are in, that move from "team" to "fleet" naming.
var abmVPPConfigKeyRenames = map[string]map[string]string{
	"apple_business_manager": {
		"macos_team":  "macos_fleet",
		"ios_team":    "ios_fleet",
		"ipados_team": "ipados_fleet",
		"byod_team":   "byod_fleet",
	},
	"volume_purchasing_program": {
		"teams": "fleets",
	},
}

// Up_20261111205515 moves the stored ABM and VPP fleet assignments to the new keys. Without it an upgraded server reads the
// old keys as empty, so GET /config and generate-gitops lose the assignments. The values are only a mirror of abm_tokens and
// vpp_token_teams, but a GitOps run built from the empty mirror would clear every assignment. Other keys are kept as they are.
func Up_20261111205515(tx *sql.Tx) error {
	var raw []byte
	if err := tx.QueryRow(`SELECT json_value FROM app_config_json WHERE id = 1`).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("select app_config_json: %w", err)
	}

	var config map[string]json.RawMessage
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("unmarshal app_config_json: %w", err)
	}
	var mdm map[string]json.RawMessage
	if err := json.Unmarshal(config["mdm"], &mdm); err != nil || mdm == nil {
		return nil //nolint:nilerr // no mdm settings to migrate
	}

	var changed bool
	for list, renames := range abmVPPConfigKeyRenames {
		var entries []map[string]json.RawMessage
		if err := json.Unmarshal(mdm[list], &entries); err != nil || entries == nil {
			continue
		}
		for _, entry := range entries {
			for oldKey, newKey := range renames {
				value, ok := entry[oldKey]
				if !ok {
					continue
				}
				if _, hasNew := entry[newKey]; !hasNew {
					entry[newKey] = value
				}
				delete(entry, oldKey)
				changed = true
			}
		}
		out, err := json.Marshal(entries)
		if err != nil {
			return fmt.Errorf("marshal %s: %w", list, err)
		}
		mdm[list] = out
	}
	if !changed {
		return nil
	}

	mdmOut, err := json.Marshal(mdm)
	if err != nil {
		return fmt.Errorf("marshal mdm: %w", err)
	}
	config["mdm"] = mdmOut
	out, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal app_config_json: %w", err)
	}
	if _, err := tx.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1`, out); err != nil {
		return fmt.Errorf("update app_config_json: %w", err)
	}
	return nil
}

func Down_20261111205515(tx *sql.Tx) error {
	return nil
}
