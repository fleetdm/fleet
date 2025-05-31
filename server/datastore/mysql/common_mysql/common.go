package common_mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
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

type ErrorWrapper interface {
	Wrap(ctx context.Context, cause error, msgs ...string) error
}

func WithTxx(ctx context.Context, db *sqlx.DB, fn TxFn, ew ErrorWrapper, logger log.Logger) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return ew.Wrap(ctx, err, "create transaction")
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
			return ew.Wrap(ctx, err, fmt.Sprintf("got err '%s' rolling back after err", rbErr.Error()))
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return ew.Wrap(ctx, err, "commit transaction")
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

func InsertOnDuplicateDidInsertOrUpdate(res sql.Result) bool {
	// From mysql's documentation:
	//
	// With ON DUPLICATE KEY UPDATE, the affected-rows value per row is 1 if
	// the row is inserted as a new row, 2 if an existing row is updated, and
	// 0 if an existing row is set to its current values. If you specify the
	// CLIENT_FOUND_ROWS flag to the mysql_real_connect() C API function when
	// connecting to mysqld, the affected-rows value is 1 (not 0) if an
	// existing row is set to its current values.
	//
	// If a table contains an AUTO_INCREMENT column and INSERT ... ON DUPLICATE KEY UPDATE
	// inserts or updates a row, the LAST_INSERT_ID() function returns the AUTO_INCREMENT value.
	//
	// https://dev.mysql.com/doc/refman/8.4/en/insert-on-duplicate.html
	//
	// Note that connection string sets CLIENT_FOUND_ROWS (see
	// generateMysqlConnectionString in this package), so it does return 1 when
	// an existing row is set to its current values, but with a last inserted id
	// of 0.
	//
	// Also note that with our mysql driver, Result.LastInsertId and
	// Result.RowsAffected can never return an error, they are retrieved at the
	// time of the Exec call, and the result simply returns the integers it
	// already holds:
	// https://github.com/go-sql-driver/mysql/blob/bcc459a906419e2890a50fc2c99ea6dd927a88f2/result.go

	lastID, _ := res.LastInsertId()
	aff, _ := res.RowsAffected()
	// something was updated (lastID != 0) AND row was found (aff == 1 or higher if more rows were found)
	return lastID != 0 && aff > 0
}
