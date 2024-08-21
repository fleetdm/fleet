package tables

import (
	"database/sql"
	"encoding/json"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/goose"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

var MigrationClient = goose.New("migration_status_tables", goose.MySqlDialect{})

func fkExists(tx *sql.Tx, table, name string) bool {
	var count int
	err := tx.QueryRow(`
SELECT COUNT(1)
FROM information_schema.REFERENTIAL_CONSTRAINTS
WHERE CONSTRAINT_SCHEMA = DATABASE() 
AND TABLE_NAME = ?
AND CONSTRAINT_NAME = ? 
	`, table, name).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

func columnExists(tx *sql.Tx, table, column string) bool {
	var count int
	err := tx.QueryRow(
		`
SELECT
    count(*)
FROM
    information_schema.columns
WHERE
    TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = ?
    AND COLUMN_NAME = ?
`,
		table, column,
	).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

func indexExists(tx *sqlx.DB, table, index string) bool {
	var count int
	err := tx.QueryRow(`
SELECT COUNT(1)
FROM INFORMATION_SCHEMA.STATISTICS
WHERE table_schema = DATABASE()
AND table_name = ?
AND index_name = ?
`, table, index).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

func indexExistsTx(tx *sql.Tx, table, index string) bool {
	var count int
	err := tx.QueryRow(`
SELECT COUNT(1)
FROM INFORMATION_SCHEMA.STATISTICS
WHERE table_schema = DATABASE()
AND table_name = ?
AND index_name = ?
`, table, index).Scan(&count)
	if err != nil {
		return false
	}

	return count > 0
}

// updateAppConfigJSON updates the `json_value` stored in the `app_config_json` after applying the
// supplied callback to the current config object.
func updateAppConfigJSON(tx *sql.Tx, fn func(config *fleet.AppConfig) error) error {
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
		return errors.Wrap(err, "unmarshal app_config_json")
	}

	if err := fn(&config); err != nil {
		return errors.Wrap(err, "callback app_config_json")
	}

	b, err := json.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "marshal updated app_config_json")
	}

	const updateStmt = `UPDATE app_config_json SET json_value = ? WHERE id = 1`
	if _, err := tx.Exec(updateStmt, b); err != nil {
		return errors.Wrap(err, "update app_config_json")
	}

	return nil
}
