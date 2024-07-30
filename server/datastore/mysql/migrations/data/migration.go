package data

import "github.com/fleetdm/fleet/v4/server/goose"

var MigrationClient = goose.New("migration_status_data", goose.MySqlDialect{})
