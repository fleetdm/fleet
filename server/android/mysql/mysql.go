// Package mysql is a MySQL implementation of the android.Datastore interface.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/android"
	android_migrations "github.com/fleetdm/fleet/v4/server/android/mysql/migrations"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/goose"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
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
	return common_mysql.WithRetryTxx(ctx, ds.Writer(ctx), fn, ds.logger)
}

func (ds *Datastore) MigrateTables(ctx context.Context) error {
	return android_migrations.MigrationClient.Up(ds.Writer(ctx).DB, "")
}

// loadMigrations manually loads the applied migrations in ascending
// order (goose doesn't provide such functionality).
func (ds *Datastore) loadMigrations(
	ctx context.Context,
	writer *sql.DB,
	reader fleet.DBReader,
) (tableRecs []int64, err error) {
	// We need to run the following to trigger the creation of the migration status tables.
	_, err = android_migrations.MigrationClient.GetDBVersion(writer)
	if err != nil {
		return nil, err
	}
	// version_id > 0 to skip the bootstrap migration that creates the migration tables.
	if err := sqlx.SelectContext(ctx, reader, &tableRecs,
		"SELECT version_id FROM "+android_migrations.MigrationClient.TableName+" WHERE version_id > 0 AND is_applied ORDER BY id ASC",
	); err != nil {
		return nil, err
	}
	return tableRecs, nil
}

// MigrationStatus will return the current status of the migrations
// comparing the known migrations in code and the applied migrations in the database.
//
// It assumes some deployments may have performed migrations out of order.
func (ds *Datastore) MigrationStatus(ctx context.Context) (*android.MigrationStatus, error) {
	if android_migrations.MigrationClient.Migrations == nil {
		return nil, errors.New("unexpected nil android_migrations list")
	}
	appliedFeatureTables, err := ds.loadMigrations(ctx, ds.primary.DB, ds.replica)
	if err != nil {
		return nil, fmt.Errorf("cannot load feature migrations: %w", err)
	}
	return compareTableMigrations(
		android_migrations.MigrationClient.Migrations,
		appliedFeatureTables,
	), nil
}

func compareTableMigrations(knownTable goose.Migrations, appliedTable []int64) *android.MigrationStatus {
	if len(appliedTable) == 0 {
		return &android.MigrationStatus{
			StatusCode: android.NoMigrationsCompleted,
		}
	}

	missingTable, unknownTable, equalTable := common_mysql.CompareVersions(
		common_mysql.GetVersionsFromMigrations(knownTable),
		appliedTable,
		knownUnknownTableMigrations,
	)

	if equalTable {
		return &android.MigrationStatus{
			StatusCode: android.AllMigrationsCompleted,
		}
	}

	// Check for missing migrations first, as these are more important
	// to detect than the unknown migrations.
	if len(missingTable) > 0 {
		return &android.MigrationStatus{
			StatusCode:   android.SomeMigrationsCompleted,
			MissingTable: missingTable,
		}
	}

	// len(unknownTable) > 0 || len(unknownData) > 0
	return &android.MigrationStatus{
		StatusCode:   android.UnknownMigrations,
		UnknownTable: unknownTable,
	}
}

var (
	knownUnknownTableMigrations = map[int64]struct{}{}
)
