package tables

import (
	"bytes"
	"database/sql"
	"encoding/json"

	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20220404091216, Down_20220404091216)
}

func Up_20220404091216(tx *sql.Tx) error {
	const selectStmt = `SELECT json_value FROM app_config_json LIMIT 1`

	var raw json.RawMessage
	var config map[string]*json.RawMessage

	row := tx.QueryRow(selectStmt)
	if err := row.Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return errors.Wrap(err, "select app_config_json")
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		return errors.Wrap(err, "unmarshal appconfig")
	}

	var (
		oldPath = []byte(`"/api/v1/osquery/log"`)
		newPath = []byte(`"/api/osquery/log"`)
		updated = false
	)
	if opts := config["agent_options"]; opts != nil {
		oldOpts := []byte(*opts)
		newOpts := json.RawMessage(bytes.ReplaceAll(oldOpts, oldPath, newPath))
		config["agent_options"] = &newOpts
		updated = !bytes.Equal(oldOpts, newOpts)
	}
	if !updated {
		return nil
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

func Down_20220404091216(tx *sql.Tx) error {
	return nil
}
