// Package mysql implements the MySQL datastore for the activity bounded context.
// This package is internal to the activity bounded context and should not be
// imported by other bounded contexts.
package mysql

import (
	"context"

	"github.com/jmoiron/sqlx"
)

type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
}

// NewDatastore creates a new MySQL store for activities.
// It accepts the same database connections used by the main datastore,
// allowing the activity bounded context to share connections.
func NewDatastore(primary *sqlx.DB, replica *sqlx.DB) *Datastore {
	return &Datastore{
		primary: primary,
		replica: replica,
	}
}

// Ping verifies database connectivity by querying the activities table.
func (s *Datastore) Ping(ctx context.Context) error {
	var result int
	return s.replica.QueryRowxContext(ctx, "SELECT 1 FROM activities LIMIT 1").Scan(&result)
}
