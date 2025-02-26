package common_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/go-kit/log"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/ngrok/sqlmw"
)

type DBOptions struct {
	// MaxAttempts configures the number of retries to connect to the DB
	MaxAttempts         int
	Logger              log.Logger
	ReplicaConfig       *config.MysqlConfig
	Interceptor         sqlmw.Interceptor
	TracingConfig       *config.LoggingConfig
	MinLastOpenedAtDiff time.Duration
	SqlMode             string
	PrivateKey          string
}

func NewDB(conf *config.MysqlConfig, opts *DBOptions, otelDriverName string) (*sqlx.DB, error) {
	driverName := "mysql"
	if opts.TracingConfig != nil && opts.TracingConfig.TracingEnabled {
		if opts.TracingConfig.TracingType == "opentelemetry" {
			driverName = otelDriverName
		} else {
			driverName = "apm/mysql"
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
	db, err := sqlx.Open(driverName, dsn)
	if err != nil {
		return nil, err
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
func generateMysqlConnectionString(conf config.MysqlConfig) string {
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
	if conf.TLSConfig != "" {
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
