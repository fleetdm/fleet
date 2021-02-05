package data

import "github.com/fleetdm/goose"

var (
	MigrationClient = goose.New("migration_status_data", goose.MySqlDialect{})
)
