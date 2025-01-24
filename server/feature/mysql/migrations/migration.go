package feature_migrations

import (
	"github.com/fleetdm/fleet/v4/server/goose"
	_ "github.com/go-sql-driver/mysql"
)

var MigrationClient = goose.New("feature_migration_status", goose.MySqlDialect{})
