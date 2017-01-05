package tables

import "github.com/kolide/goose"

var (
	MigrationClient = goose.New("migration_status_tables", goose.MySqlDialect{})
)
