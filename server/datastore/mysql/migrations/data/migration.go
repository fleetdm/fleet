package data

import "github.com/kolide/goose"

var (
	MigrationClient = goose.New("migration_status_data", goose.MySqlDialect{})
)
