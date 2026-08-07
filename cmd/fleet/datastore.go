package main

import (
	"log/slog"
	"os"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/s3"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
)

// buildMySQLOpts assembles the DBOptions for the primary datastore connection
// from the Fleet config: the base logger and config, plus the optional read
// replica, dev SQL interceptor, and tracing.
func buildMySQLOpts(config config.FleetConfig, logger *slog.Logger) []mysql.DBOption {
	opts := []mysql.DBOption{mysql.Logger(logger), mysql.WithFleetConfig(&config)}
	if config.MysqlReadReplica.Address != "" {
		opts = append(opts, mysql.Replica(&config.MysqlReadReplica))
	}
	// NOTE this will disable OTEL/APM interceptor
	if dev_mode.Env("FLEET_DEV_ENABLE_SQL_INTERCEPTOR") != "" {
		opts = append(opts, mysql.WithInterceptor(&devSQLInterceptor{
			logger: logger.With("component", "sql-interceptor"),
		}))
	}
	if config.Logging.TracingEnabled {
		opts = append(opts, mysql.TracingEnabled(&config.Logging))
	}
	return opts
}

// initDatastore brings up the MySQL datastore: shared DB connections, the
// datastore itself, and the carve store (S3-backed when configured, otherwise
// the datastore). Failures go through initFatal. Returns nil values on the
// failure path so the function is safe when initFatal does not terminate
// (e.g., tests using a recorder).
func initDatastore(config config.FleetConfig, logger *slog.Logger, c clock.Clock, initFatal func(err error, msg string)) (
	*mysql.Datastore,
	*common_mysql.DBConnections,
	fleet.CarveStore,
) {
	opts := buildMySQLOpts(config, logger)

	// Create database connections that can be shared across datastores
	dbConns, err := mysql.NewDBConnections(config.Mysql, opts...)
	if err != nil {
		initFatal(err, "initializing database connections")
		return nil, nil, nil
	}

	mds, err := mysql.NewDatastore(dbConns, config.Mysql, c)
	if err != nil {
		initFatal(err, "initializing datastore")
		return nil, nil, nil
	}

	var carveStore fleet.CarveStore = mds
	if config.S3.CarvesBucket != "" || config.S3.Bucket != "" {
		carveStore, err = s3.NewCarveStore(config.S3, mds)
		if err != nil {
			initFatal(err, "initializing S3 carvestore")
			return nil, nil, nil
		}
	}

	return mds, dbConns, carveStore
}

// evalMigrationStatus prints any operator guidance for the current migration
// status and reports whether runServeCmd should exit instead of starting. It
// encodes the boot/refuse-to-boot decision: unknown migrations are only fatal
// in dev mode; the v4.73.2 and partial-migration states are fatal unless
// missing migrations are explicitly allowed; an uninitialized database is
// always fatal.
func evalMigrationStatus(status *fleet.MigrationStatus, devMode, allowMissing bool) (shouldExit bool) {
	switch status.StatusCode {
	case fleet.AllMigrationsCompleted:
		// OK
		return false
	case fleet.UnknownMigrations:
		printUnknownMigrationsMessage(status.UnknownTable, status.UnknownData)
		return devMode
	case fleet.NeedsFleetv4732Fix:
		printFleetv4732FixNeededMessage()
		return !allowMissing
	case fleet.UnknownFleetv4732State:
		printFleetv4732UnknownStateMessage(status.StatusCode)
		return !allowMissing
	case fleet.SomeMigrationsCompleted:
		printMissingMigrationsWarning(os.Stdout, status.MissingTable, status.MissingData)
		return !allowMissing
	case fleet.NoMigrationsCompleted:
		printDatabaseNotInitializedError()
		return true
	}
	return false
}
