package tables

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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

func constraintExists(tx *sql.Tx, table, name string) bool {
	var count int
	err := tx.QueryRow(`
SELECT COUNT(1)
FROM information_schema.TABLE_CONSTRAINTS
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
	return columnsExists(tx, table, column)
}

func columnsExists(tx *sql.Tx, table string, columns ...string) bool {
	if len(columns) == 0 {
		return false
	}
	inColumns := strings.TrimRight(strings.Repeat("?,", len(columns)), ",")
	args := make([]interface{}, 0, len(columns)+1)
	args = append(args, table)
	for _, column := range columns {
		args = append(args, column)
	}

	var count int
	err := tx.QueryRow(
		fmt.Sprintf(`
SELECT
    count(*)
FROM
    information_schema.columns
WHERE
    TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = ?
    AND COLUMN_NAME IN (%s)
`, inColumns), args...,
	).Scan(&count)
	if err != nil {
		return false
	}

	return count == len(columns)
}

func tableExists(tx *sql.Tx, table string) bool {
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
`,
		table,
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
