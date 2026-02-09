package mysql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/ngrok/sqlmw"
)

// ConnectorFactory creates a driver.Connector for custom database authentication.
// This allows injecting authentication mechanisms (like AWS IAM) without adding
// dependencies to this package.
type ConnectorFactory func(dsn string, logger log.Logger) (driver.Connector, error)

// TestSQLMode combines ANSI mode components with MySQL 8 default strict modes for testing
// ANSI mode includes: REAL_AS_FLOAT, PIPES_AS_CONCAT, ANSI_QUOTES, IGNORE_SPACE, ONLY_FULL_GROUP_BY
// We add all MySQL 8.0 default strict modes to match production behavior
// Note: The value needs to be wrapped in single quotes when passed to MySQL DSN due to comma separation
// Reference: https://dev.mysql.com/doc/refman/8.0/en/sql-mode.html
const TestSQLMode = "'REAL_AS_FLOAT,PIPES_AS_CONCAT,ANSI_QUOTES,IGNORE_SPACE,ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION'"

type DBOptions struct {
	// MaxAttempts configures the number of retries to connect to the DB
	MaxAttempts         int
	Logger              log.Logger
	ReplicaConfig       *MysqlConfig
	Interceptor         sqlmw.Interceptor
	TracingConfig       *LoggingConfig
	MinLastOpenedAtDiff time.Duration
	SqlMode             string
	PrivateKey          string
	// ConnectorFactory is an optional factory for creating custom database connectors.
	// When set, it's used instead of the standard connection method.
	ConnectorFactory ConnectorFactory
}

// DBConnections holds the database connections and options that can be shared across datastores.
// This allows bounded contexts to create their own datastores using the same
// underlying database connections and configuration.
type DBConnections struct {
	Primary *sqlx.DB
	Replica *sqlx.DB
	Options *DBOptions
}

// DBReadTx provides a minimal interface for read-only transactions that exposes
// only the methods required for reads.
type DBReadTx interface {
	sqlx.QueryerContext
	sqlx.PreparerContext

	Rebind(string) string
}

func NewDB(conf *MysqlConfig, opts *DBOptions, otelDriverName string) (*sqlx.DB, error) {
	driverName := "mysql"

	if opts.TracingConfig != nil && opts.TracingConfig.TracingEnabled {
		if opts.TracingConfig.TracingType == "elasticapm" {
			driverName = "apm/mysql"
		} else {
			driverName = otelDriverName
		}
	}
	if opts.Interceptor != nil {
		driverName = "mysql-mw"
		sql.Register(driverName, sqlmw.Driver(mysql.MySQLDriver{}, opts.Interceptor))
	}
	if opts.SqlMode != "" {
		conf.SQLMode = opts.SqlMode
	}

	dsn := generateMysqlConnectionString(*conf)

	var db *sqlx.DB
	if opts.ConnectorFactory != nil {
		connector, err := opts.ConnectorFactory(dsn, opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create connector: %w", err)
		}
		db = sqlx.NewDb(sql.OpenDB(connector), driverName)
	} else {
		var err error
		db, err = sqlx.Open(driverName, dsn)
		if err != nil {
			return nil, err
		}
	}

	db.SetMaxIdleConns(conf.MaxIdleConns)
	db.SetMaxOpenConns(conf.MaxOpenConns)
	db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifetime))

	var dbError error
	for attempt := 0; attempt < opts.MaxAttempts; attempt++ {
		dbError = db.Ping()
		if dbError == nil {
			// we're connected!
			break
		}
		interval := time.Duration(attempt) * time.Second
		opts.Logger.Log("mysql", fmt.Sprintf(
			"could not connect to db: %v, sleeping %v", dbError, interval))
		time.Sleep(interval)
	}

	if dbError != nil {
		return nil, dbError
	}
	return db, nil
}

// generateMysqlConnectionString returns a MySQL connection string using the
// provided configuration.
func generateMysqlConnectionString(conf MysqlConfig) string {
	params := url.Values{
		// using collation implicitly sets the charset too
		// and it's the recommended way to do it per the
		// driver documentation:
		// https://github.com/go-sql-driver/mysql#charset
		"collation":            []string{"utf8mb4_unicode_ci"},
		"parseTime":            []string{"true"},
		"loc":                  []string{"UTC"},
		"time_zone":            []string{"'-00:00'"},
		"clientFoundRows":      []string{"true"},
		"allowNativePasswords": []string{"true"},
		"group_concat_max_len": []string{"4194304"},
		"multiStatements":      []string{"true"},
	}
	if conf.Password == "" && conf.PasswordPath == "" && conf.Region != "" {
		params.Set("allowCleartextPasswords", "true")
		if conf.TLSConfig == "" {
			params.Set("tls", "rdsmysql")
		}
	} else if conf.TLSConfig != "" {
		params.Set("tls", conf.TLSConfig)
	}
	if conf.SQLMode != "" {
		params.Set("sql_mode", conf.SQLMode)
	}

	dsn := fmt.Sprintf(
		"%s:%s@%s(%s)/%s?%s",
		conf.Username,
		conf.Password,
		conf.Protocol,
		conf.Address,
		conf.Database,
		params.Encode(),
	)

	return dsn
}

func WithTxx(ctx context.Context, db *sqlx.DB, fn TxFn, logger log.Logger) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				logger.Log("err", err, "msg", "error encountered during transaction panic rollback")
			}
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil && rbErr != sql.ErrTxDone {
			return ctxerr.Wrapf(ctx, err, "got err '%s' rolling back after err", rbErr.Error())
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return ctxerr.Wrap(ctx, err, "commit transaction")
	}

	return nil
}

// WithReadOnlyTxx executes fn within an isolated, read-only transaction
func WithReadOnlyTxx(ctx context.Context, reader *sqlx.DB, fn ReadTxFn, logger log.Logger) error {
	tx, err := reader.BeginTxx(ctx, &sql.TxOptions{
		ReadOnly:  true,
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create read-only transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				logger.Log("err", err, "msg", "error encountered during read-only transaction panic rollback")
			}
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil && rbErr != sql.ErrTxDone {
			return ctxerr.Wrapf(ctx, err, "got err '%s' rolling back read-only transaction after err", rbErr.Error())
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return ctxerr.Wrap(ctx, err, "commit read-only transaction")
	}

	return nil
}

// MySQL is really particular about using zero values or old values for
// timestamps, so we set a default value that is plenty far in the past, but
// hopefully accepted by most MySQL configurations.
//
// NOTE: #3229 proposes a better fix that uses *time.Time for
// ScheduledQueryStats.LastExecuted.
var DefaultNonZeroTime = "2000-01-01T00:00:00Z"

func GetDefaultNonZeroTime() time.Time {
	return time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
}
