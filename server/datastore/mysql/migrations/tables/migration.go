package tables

import "github.com/fleetdm/goose"

var (
	MigrationClient = goose.New("migration_status_tables", goose.MySqlDialect{})
)
