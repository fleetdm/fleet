// Package mysql is a MySQL implementation of the Datastore interface.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/android"
	android_migrations "github.com/fleetdm/fleet/v4/server/android/mysql/migrations"
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
// func (ds *Datastore) reader(ctx context.Context) fleet.DBReader {
// 	if ctxdb.IsPrimaryRequired(ctx) {
// 		return ds.primary
// 	}
// 	return ds.replica
// }

// writer returns the DB instance to use for write statements, which is always
// the primary.
func (ds *Datastore) writer(_ context.Context) *sqlx.DB {
	return ds.primary
}

// func (ds *Datastore) withRetryTxx(ctx context.Context, fn common_mysql.TxFn) (err error) {
// 	return common_mysql.WithRetryTxx(ctx, ds.writer(ctx), fn, ds.logger)
// }

func (ds *Datastore) MigrateTables(ctx context.Context) error {
	return android_migrations.MigrationClient.Up(ds.writer(ctx).DB, "")
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

	missingTable, unknownTable, equalTable := compareVersions(
		getVersionsFromMigrations(knownTable),
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

func unknownUnknowns(in []int64, knownUnknowns map[int64]struct{}) []int64 {
	var result []int64
	for _, t := range in {
		if _, ok := knownUnknowns[t]; !ok {
			result = append(result, t)
		}
	}
	return result
}

// compareVersions returns any missing or extra elements in v2 with respect to v1
// (v1 or v2 need not be ordered).
func compareVersions(v1, v2 []int64, knownUnknowns map[int64]struct{}) (missing []int64, unknown []int64, equal bool) {
	v1s := make(map[int64]struct{})
	for _, m := range v1 {
		v1s[m] = struct{}{}
	}
	v2s := make(map[int64]struct{})
	for _, m := range v2 {
		v2s[m] = struct{}{}
	}
	for _, m := range v1 {
		if _, ok := v2s[m]; !ok {
			missing = append(missing, m)
		}
	}
	for _, m := range v2 {
		if _, ok := v1s[m]; !ok {
			unknown = append(unknown, m)
		}
	}
	unknown = unknownUnknowns(unknown, knownUnknowns)
	if len(missing) == 0 && len(unknown) == 0 {
		return nil, nil, true
	}
	return missing, unknown, false
}

func getVersionsFromMigrations(migrations goose.Migrations) []int64 {
	versions := make([]int64, len(migrations))
	for i := range migrations {
		versions[i] = migrations[i].Version
	}
	return versions
}
