// Package bootstrap wires the activity bounded context dependencies for production use.
// This package bridges the internal mysql implementation with the public service interface.
package bootstrap

import (
	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/internal/mysql"
	"github.com/fleetdm/fleet/v4/server/activity/service"
	"github.com/jmoiron/sqlx"
)

// NewService creates a new activity service with a MySQL datastore.
// This is the production constructor used for dependency injection.
func NewService(authz activity.Authorizer, primaryDB, replicaDB *sqlx.DB) *service.Service {
	store := mysql.NewDatastore(primaryDB, replicaDB)
	return service.NewService(authz, store)
}
