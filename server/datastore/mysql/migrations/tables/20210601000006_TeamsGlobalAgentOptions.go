package tables

import (
	"database/sql"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	"github.com/pkg/errors"
)

func init() {
	MigrationClient.AddMigration(Up_20210601000006, Down_20210601000006)
}

func Up_20210601000006(tx *sql.Tx) error {
	existingOptions, err := copyOptions(tx)
	if err != nil {
		return errors.Wrap(err, "get existing options")
	}

	sql := `
		ALTER TABLE app_configs
		ADD COLUMN agent_options JSON
	`
	if _, err := tx.Exec(sql); err != nil {
		return errors.Wrap(err, "add column agent_options")
	}

	sql = `UPDATE app_configs SET agent_options = ?`
	if _, err := tx.Exec(sql, existingOptions); err != nil {
		return errors.Wrap(err, "insert existing options")
	}

	return nil
}

// Below code copied and adapted from osquery options code removed in this commit.

func copyOptions(tx *sql.Tx) (json.RawMessage, error) {
	// Migrate pre teams osquery options to the new osquery option storage in app config.
	txx := sqlx.Tx{Tx: tx, Mapper: reflectx.NewMapperFunc("db", sqlx.NameMapper)}

	var rows []optionsRow
	if err := txx.Select(&rows, "SELECT * FROM osquery_options"); err != nil {
		return nil, errors.Wrap(err, "selecting options")
	}

	opt := &fleet.AgentOptions{
		Overrides: fleet.AgentOptionsOverrides{
			Platforms: make(map[string]json.RawMessage),
		},
	}
	for _, row := range rows {
		switch row.OverrideType {
		case 0: // was fleet.OptionOverrideTypeDefault
			opt.Config = json.RawMessage(row.Options)

		case 1: // was fleet.OptionOverrideTypePlatform
			opt.Overrides.Platforms[row.OverrideIdentifier] = json.RawMessage(row.Options)

		default:
			return nil, errors.Errorf("unknown override type: %d", row.OverrideType)
		}
	}

	jsonVal, err := json.Marshal(opt)
	if err != nil {
		return nil, errors.Wrap(err, "marshal options")
	}

	return jsonVal, nil
}

type optionsRow struct {
	ID                 int    `db:"id"`
	OverrideType       int    `db:"override_type"`
	OverrideIdentifier string `db:"override_identifier"`
	Options            string `db:"options"`
}

func Down_20210601000006(tx *sql.Tx) error {
	return nil
}
