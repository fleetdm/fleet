// Package bootstrap wires the activity bounded context dependencies for production use.
// This package bridges the internal mysql implementation with the public service interface.
package bootstrap

import (
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/service"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

// NewService creates a new activity service with a MySQL datastore.
// This is the production constructor used for dependency injection.
func NewService(primaryDB, replicaDB *sqlx.DB, logger log.Logger) (*service.Service, error) {
	store := mysql.NewDatastore(primaryDB, replicaDB, logger)
	return service.NewService(store)
}
