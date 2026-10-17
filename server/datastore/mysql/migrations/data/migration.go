package data

import "github.com/fleetdm/fleet/v4/server/goose"

// MinOutOfOrderVersion is intentionally left at 0 (out-of-order application
// disabled): data migrations are deprecated and frozen, so no new one can
// ship in a patch release with an older timestamp.
var MigrationClient = goose.New("migration_status_data", goose.MySqlDialect{})
