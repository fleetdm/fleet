// Package bootstrap provides the entry point for the recoverykeypassword bounded context.
package bootstrap

import (
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/recoverykeypassword"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword/internal/mysql"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
)

// New creates a new recovery key password datastore.
func New(
	dbConns *platform_mysql.DBConnections,
	logger *slog.Logger,
) recoverykeypassword.Datastore {
	return mysql.NewDatastore(dbConns, logger)
}
