package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20240430111727, Down_20240430111727)
}

func Up_20240430111727(tx *sql.Tx) error {
	// This cleanup correspond to the following bug: https://github.com/fleetdm/fleet/issues/18079.
	// The following deletes "team query results" that do not match the host's team.
	_, err := tx.Exec(`
		DELETE qr
		FROM query_results qr
		JOIN queries q ON (q.id=qr.query_id)
		JOIN hosts h ON (h.id=qr.host_id)
		WHERE q.team_id IS NOT NULL AND q.team_id != COALESCE(h.team_id, 0);
	`)
	if err != nil {
		return fmt.Errorf("failed to delete query_results %w", err)
	}

	//
	// The following "fix" was introduced after this migration was released in 4.50.0.
	// We are adding it here to disable the AI features (set ai_features_disabled=true)
	// for non-new installations that are upgrading from < 4.50.0 using the new version
	// of this migration to be released in 4.51.X.
	//
	if err := fixDisableAIForNonNewInstallation(tx); err != nil {
		return fmt.Errorf("failed to update ai_features_disabled: %w", err)
	}

	return nil
}

func fixDisableAIForNonNewInstallation(tx *sql.Tx) error {
	var usersCount int
	row := tx.QueryRow(`SELECT COUNT(*) FROM users;`)
	if err := row.Scan(&usersCount); err != nil {
		return fmt.Errorf("select count users: %w", err)
	}
	if usersCount == 0 {
		return nil
	}

	//
	// At least a "setup" user was configured,
	// thus we assume this is not a new installation.
	//

	var raw json.RawMessage
	row = tx.QueryRow(`SELECT json_value FROM app_config_json LIMIT 1;`)
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("select app_config_json: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(raw, &config); err != nil {
		return fmt.Errorf("unmarshal appconfig: %w", err)
	}

	ss, ok := config["server_settings"]
	if !ok {
		return errors.New("missing server_settings")
	}
	serverSettings, ok := ss.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid type for server_settings: %T", ss)
	}
	serverSettings["ai_features_disabled"] = true

	b, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal updated appconfig: %w", err)
	}
	if _, err := tx.Exec(`UPDATE app_config_json SET json_value = ? WHERE id = 1;`, b); err != nil {
		return fmt.Errorf("update app_config_json: %w", err)
	}

	return nil
}

func Down_20240430111727(tx *sql.Tx) error {
	return nil
}
