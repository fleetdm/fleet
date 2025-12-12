// Package mysql is a MySQL implementation of the android.Datastore interface.
package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
)

// AndroidDatastore is an implementation of android.Datastore interface backed by MySQL
type AndroidDatastore struct {
	logger  log.Logger
	primary *sqlx.DB
	replica fleet.DBReader // so it cannot be used to perform writes
}

// NewAndroidDatastore creates a new Android Datastore
func NewAndroidDatastore(logger log.Logger, primary *sqlx.DB, replica fleet.DBReader) android.Datastore {
	return &AndroidDatastore{
		logger:  logger,
		primary: primary,
		replica: replica,
	}
}

// reader returns the DB instance to use for read-only statements, which is the
// replica unless the primary has been explicitly required via
// ctxdb.RequirePrimary.
func (ds *AndroidDatastore) reader(ctx context.Context) fleet.DBReader {
	if ctxdb.IsPrimaryRequired(ctx) {
		return ds.primary
	}
	return ds.replica
}

// Writer returns the DB instance to use for write statements, which is always
// the primary.
func (ds *AndroidDatastore) Writer(_ context.Context) *sqlx.DB {
	return ds.primary
}

func (ds *AndroidDatastore) WithRetryTxx(ctx context.Context, fn common_mysql.TxFn) (err error) {
	return common_mysql.WithRetryTxx(ctx, ds.Writer(ctx), fn, ds.logger)
}

func (ds *AndroidDatastore) WithTxx(ctx context.Context, fn common_mysql.TxFn) (err error) {
	return common_mysql.WithTxx(ctx, ds.Writer(ctx), fn, ds.logger)
}
