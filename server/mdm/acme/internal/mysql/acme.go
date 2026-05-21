// Package mysql provides the MySQL datastore implementation for the ACME service module.
package mysql

import (
	"context"
	"log/slog"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/mdm/acme/internal/types"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"go.opentelemetry.io/otel"
)

// tracer is an OTEL tracer. It has no-op behavior when OTEL is not enabled.
var tracer = otel.Tracer("github.com/fleetdm/fleet/v4/server/mdm/acme/internal/mysql")

// Datastore is the MySQL implementation of the ACME datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

// NewDatastore creates a new MySQL datastore for activities.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger}
}

func (ds *Datastore) reader(ctx context.Context) sqlx.QueryerContext {
	if ctxdb.IsPrimaryRequired(ctx) {
		return ds.primary
	}
	return ds.replica
}

func (ds *Datastore) writer(_ context.Context) *sqlx.DB {
	return ds.primary
}

// Ensure Datastore implements types.Datastore
var _ types.Datastore = (*Datastore)(nil)
