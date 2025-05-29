// Package mysql is a MySQL implementation of the android.Datastore interface.
package mysql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// Datastore is an implementation of android.Datastore interface backed by MySQL
type Datastore struct {
	logger  log.Logger
	primary *sqlx.DB
	replica fleet.DBReader // so it cannot be used to perform writes
}

// New creates a new Datastore
func New(logger log.Logger, primary *sqlx.DB, replica fleet.DBReader) android.Datastore {
	return &Datastore{
		logger:  logger,
		primary: primary,
		replica: replica,
	}
}

// reader returns the DB instance to use for read-only statements, which is the
// replica unless the primary has been explicitly required via
// ctxdb.RequirePrimary.
func (ds *Datastore) reader(ctx context.Context) fleet.DBReader {
	if ctxdb.IsPrimaryRequired(ctx) {
		return ds.primary
	}
	return ds.replica
}

// Writer returns the DB instance to use for write statements, which is always
// the primary.
func (ds *Datastore) Writer(_ context.Context) *sqlx.DB {
	return ds.primary
}

func (ds *Datastore) WithRetryTxx(ctx context.Context, fn common_mysql.TxFn) (err error) {
	return common_mysql.WithRetryTxx(ctx, ds.Writer(ctx), fn, ErrorWrapper{}, ds.logger)
}

func ExecAdhocSQL(tb testing.TB, ds *Datastore, fn func(q sqlx.ExtContext) error) {
	tb.Helper()
	err := fn(ds.primary)
	require.NoError(tb, err)
}

// ErrorWrapper implements the Wrap interface
type ErrorWrapper struct{}

// Wrap wraps an error.
// We are not using the standard ctxerr wrapper because it is dependent on the fleet package, which has many other dependencies of its own.
// In an effort to decouple this package from the rest of the fleet codebase, we are using a custom wrapper.
func (w ErrorWrapper) Wrap(_ context.Context, cause error, msgs ...string) error {
	return fmt.Errorf("%s: %w", strings.Join(msgs, ", "), cause)
}
