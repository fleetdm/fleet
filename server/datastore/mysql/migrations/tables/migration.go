package tables

import (
	"database/sql"

	"github.com/fleetdm/goose"
)

var (
	MigrationClient = goose.New("migration_status_tables", goose.MySqlDialect{})
)

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
