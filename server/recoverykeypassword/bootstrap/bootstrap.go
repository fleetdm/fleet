// Package bootstrap provides the entry point for the recoverykeypassword bounded context.
package bootstrap

import (
	"log/slog"

	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword"
	"github.com/fleetdm/fleet/v4/server/recoverykeypassword/internal/mysql"
)

// New creates a new recovery key password service.
// The commander parameter is used to send MDM commands to devices.
func New(
	dbConns *platform_mysql.DBConnections,
	commander recoverykeypassword.MDMCommander,
	logger *slog.Logger,
) recoverykeypassword.Service {
	ds := mysql.NewDatastore(dbConns, logger)
	return recoverykeypassword.NewService(ds, commander, logger)
}
