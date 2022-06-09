package tables

import (
	"database/sql"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220608113128, Down_20220608113128)
}

func Up_20220608113128(tx *sql.Tx) error {
	var raw []byte
	row := tx.QueryRow(`SELECT json_value FROM app_config_json LIMIT 1`)
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return errors.Wrap(err, "select app_config_json")
	}

	var config fleet.AppConfig
	if err := json.Unmarshal(raw, &config); err != nil {
		return errors.Wrap(err, "unmarshal appconfig")
	}

	if config.FleetDesktop.TransparencyURL != "" {
		return errors.New("unexpected transparency_url value in app_config_json")
	}

	b, err := json.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshal updated appconfig")
	}

	const updateStmt = `UPDATE app_config_json SET json_value = ? WHERE id = 1`
	if _, err := tx.Exec(updateStmt, b); err != nil {
		return errors.Wrap(err, "update app_config_json")
	}

	return nil
}

func Down_20220608113128(tx *sql.Tx) error {
	return nil
}
