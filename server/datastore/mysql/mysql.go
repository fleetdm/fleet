// Package mysql is a MySQL implementation of the Datastore interface.
package mysql

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/WatchBeam/clock"
	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore/mysql/migrations/data"
	"github.com/fleetdm/fleet/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/go-kit/kit/log"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

const (
	defaultSelectLimit   = 1000000
	mySQLTimestampFormat = "2006-01-02 15:04:05" // %Y/%m/%d %H:%M:%S
)

var (
	// Matches all non-word and '-' characters for replacement
	columnCharsRegexp = regexp.MustCompile(`[^\w-]`)
)

// Datastore is an implementation of kolide.Datastore interface backed by
// MySQL
type Datastore struct {
	db     *sqlx.DB
	logger log.Logger
	clock  clock.Clock
	config config.MysqlConfig
}

type dbfunctions interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}

func (d *Datastore) getTransaction(opts []kolide.OptionalArg) dbfunctions {
	var result dbfunctions = d.db
	for _, opt := range opts {
		switch t := opt().(type) {
		case dbfunctions:
			result = t
		}
	}
	return result
}

type txFn func(*sqlx.Tx) error

// retryableError determines whether a MySQL error can be retried. By default
// errors are considered non-retryable. Only errors that we know have a
// possibility of succeeding on a retry should return true in this function.
func retryableError(err error) bool {
	base := errors.Cause(err)
	if b, ok := base.(*mysql.MySQLError); ok {
		switch b.Number {
		// Consider lock related errors to be retryable
		case mysqlerr.ER_LOCK_DEADLOCK, mysqlerr.ER_LOCK_WAIT_TIMEOUT:
			return true
		}
	}

	return false
}

// withRetryTxx provides a common way to commit/rollback a txFn wrapped in a retry with exponential backoff
func (d *Datastore) withRetryTxx(fn txFn) (err error) {
	operation := func() error {
		tx, err := d.db.Beginx()
		if err != nil {
			return errors.Wrap(err, "create transaction")
		}

		defer func() {
			if p := recover(); p != nil {
				if err := tx.Rollback(); err != nil {
					d.logger.Log("err", err, "msg", "error encountered during transaction panic rollback")
				}
				panic(p)
			}
		}()

		if err := fn(tx); err != nil {
			rbErr := tx.Rollback()
			if rbErr != nil && rbErr != sql.ErrTxDone {
				// Consider rollback errors to be non-retryable
				return backoff.Permanent(errors.Wrapf(err, "got err '%s' rolling back after err", rbErr.Error()))
			}

			if retryableError(err) {
				return err
			}

			// Consider any other errors to be non-retryable
			return backoff.Permanent(err)
		}

		if err := tx.Commit(); err != nil {
			err = errors.Wrap(err, "commit transaction")

			if retryableError(err) {
				return err
			}

			return backoff.Permanent(errors.Wrap(err, "commit transaction"))
		}

		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 5 * time.Second
	return backoff.Retry(operation, bo)
}

// New creates an MySQL datastore.
func New(config config.MysqlConfig, c clock.Clock, opts ...DBOption) (*Datastore, error) {
	options := &dbOptions{
		maxAttempts: defaultMaxAttempts,
		logger:      log.NewNopLogger(),
	}

	for _, setOpt := range opts {
		setOpt(options)
	}

	if config.PasswordPath != "" && config.Password != "" {
		return nil, errors.New("A MySQL password and a MySQL password file were provided - please specify only one")
	}

	// Check to see if the flag is populated
	// Check if file exists on disk
	// If file exists read contents
	if config.PasswordPath != "" {
		fileContents, err := ioutil.ReadFile(config.PasswordPath)
		if err != nil {
			return nil, err
		}
		config.Password = strings.TrimSpace(string(fileContents))
	}

	if config.TLSConfig != "" {
		err := registerTLS(config)
		if err != nil {
			return nil, errors.Wrap(err, "register TLS config for mysql")
		}
	}

	dsn := generateMysqlConnectionString(config)
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetConnMaxLifetime(time.Second * time.Duration(config.ConnMaxLifetime))

	var dbError error
	for attempt := 0; attempt < options.maxAttempts; attempt++ {
		dbError = db.Ping()
		if dbError == nil {
			// we're connected!
			break
		}
		interval := time.Duration(attempt) * time.Second
		options.logger.Log("mysql", fmt.Sprintf(
			"could not connect to db: %v, sleeping %v", dbError, interval))
		time.Sleep(interval)
	}

	if dbError != nil {
		return nil, dbError
	}

	ds := &Datastore{
		db:     db,
		logger: options.logger,
		clock:  c,
		config: config,
	}

	return ds, nil

}

func (d *Datastore) Begin() (kolide.Transaction, error) {
	return d.db.Beginx()
}

func (d *Datastore) Name() string {
	return "mysql"
}

func (d *Datastore) MigrateTables() error {
	return tables.MigrationClient.Up(d.db.DB, "")
}

func (d *Datastore) MigrateData() error {
	return data.MigrationClient.Up(d.db.DB, "")
}

func (d *Datastore) MigrationStatus() (kolide.MigrationStatus, error) {
	if tables.MigrationClient.Migrations == nil || data.MigrationClient.Migrations == nil {
		return 0, errors.New("unexpected nil migrations list")
	}

	lastTablesMigration, err := tables.MigrationClient.Migrations.Last()
	if err != nil {
		return 0, errors.Wrap(err, "missing tables migrations")
	}

	currentTablesVersion, err := tables.MigrationClient.GetDBVersion(d.db.DB)
	if err != nil {
		return 0, errors.Wrap(err, "cannot get table migration status")
	}

	lastDataMigration, err := data.MigrationClient.Migrations.Last()
	if err != nil {
		return 0, errors.Wrap(err, "missing data migrations")
	}

	currentDataVersion, err := data.MigrationClient.GetDBVersion(d.db.DB)
	if err != nil {
		return 0, errors.Wrap(err, "cannot get data migration status")
	}

	switch {
	case currentDataVersion == 0 && currentTablesVersion == 0:
		return kolide.NoMigrationsCompleted, nil

	case currentTablesVersion != lastTablesMigration.Version ||
		currentDataVersion != lastDataMigration.Version:
		return kolide.SomeMigrationsCompleted, nil

	default:
		return kolide.AllMigrationsCompleted, nil
	}
}

// Drop removes database
func (d *Datastore) Drop() error {
	tables := []struct {
		Name string `db:"TABLE_NAME"`
	}{}

	sql := `
		SELECT TABLE_NAME
		FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_SCHEMA = ?;
	`

	if err := d.db.Select(&tables, sql, d.config.Database); err != nil {
		return err
	}

	tx, err := d.db.Begin()
	if err != nil {
		return err
	}

	_, err = tx.Exec("SET FOREIGN_KEY_CHECKS = 0")
	if err != nil {
		return tx.Rollback()
	}

	for _, table := range tables {
		_, err = tx.Exec(fmt.Sprintf("DROP TABLE %s;", table.Name))
		if err != nil {
			return tx.Rollback()
		}
	}
	_, err = tx.Exec("SET FOREIGN_KEY_CHECKS = 1")
	if err != nil {
		return tx.Rollback()
	}
	return tx.Commit()
}

// HealthCheck returns an error if the MySQL backend is not healthy.
func (d *Datastore) HealthCheck() error {
	_, err := d.db.Exec("select 1")
	return err
}

// Close frees resources associated with underlying mysql connection
func (d *Datastore) Close() error {
	return d.db.Close()
}

func sanitizeColumn(col string) string {
	return columnCharsRegexp.ReplaceAllString(col, "")
}

func appendListOptionsToSQL(sql string, opts kolide.ListOptions) string {
	if opts.OrderKey != "" {
		direction := "ASC"
		if opts.OrderDirection == kolide.OrderDescending {
			direction = "DESC"
		}
		orderKey := sanitizeColumn(opts.OrderKey)

		sql = fmt.Sprintf("%s ORDER BY %s %s", sql, orderKey, direction)
	}
	// REVIEW: If caller doesn't supply a limit apply a default limit of 1000
	// to insure that an unbounded query with many results doesn't consume too
	// much memory or hang
	if opts.PerPage == 0 {
		opts.PerPage = defaultSelectLimit
	}

	sql = fmt.Sprintf("%s LIMIT %d", sql, opts.PerPage)

	offset := opts.PerPage * opts.Page

	if offset > 0 {
		sql = fmt.Sprintf("%s OFFSET %d", sql, offset)
	}

	return sql
}

// registerTLS adds client certificate configuration to the mysql connection.
func registerTLS(config config.MysqlConfig) error {
	rootCertPool := x509.NewCertPool()
	pem, err := ioutil.ReadFile(config.TLSCA)
	if err != nil {
		return errors.Wrap(err, "read server-ca pem")
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return errors.New("failed to append PEM.")
	}
	clientCert := make([]tls.Certificate, 0, 1)
	certs, err := tls.LoadX509KeyPair(config.TLSCert, config.TLSKey)
	if err != nil {
		return errors.Wrap(err, "load mysql client cert and key")
	}
	clientCert = append(clientCert, certs)
	cfg := tls.Config{
		RootCAs:      rootCertPool,
		Certificates: clientCert,
	}
	if config.TLSServerName != "" {
		cfg.ServerName = config.TLSServerName
	}
	if err := mysql.RegisterTLSConfig(config.TLSConfig, &cfg); err != nil {
		return errors.Wrap(err, "register mysql tls config")
	}
	return nil
}

// generateMysqlConnectionString returns a MySQL connection string using the
// provided configuration.
func generateMysqlConnectionString(conf config.MysqlConfig) string {
	tz := url.QueryEscape("'-00:00'")
	dsn := fmt.Sprintf(
		"%s:%s@%s(%s)/%s?charset=utf8mb4&parseTime=true&loc=UTC&time_zone=%s&clientFoundRows=true&allowNativePasswords=true",
		conf.Username,
		conf.Password,
		conf.Protocol,
		conf.Address,
		conf.Database,
		tz,
	)

	if conf.TLSConfig != "" {
		dsn = fmt.Sprintf("%s&tls=%s", dsn, conf.TLSConfig)
	}

	return dsn
}

// isForeignKeyError checks if the provided error is a MySQL child foreign key
// error (Error #1452)
func isChildForeignKeyError(err error) bool {
	mysqlErr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	// https://dev.mysql.com/doc/refman/5.7/en/error-messages-server.html#error_er_no_referenced_row_2
	const ER_NO_REFERENCED_ROW_2 = 1452
	return mysqlErr.Number == ER_NO_REFERENCED_ROW_2
}

// searchLike adds SQL and parameters for a "search" using LIKE syntax.
//
// The input columns must be sanitized if they are provided by the user.
func searchLike(sql string, params []interface{}, match string, columns ...string) (string, []interface{}) {
	if len(columns) == 0 {
		return sql, params
	}

	match = strings.Replace(match, "_", "\\_", -1)
	match = strings.Replace(match, "%", "\\%", -1)
	pattern := "%" + match + "%"
	ors := make([]string, 0, len(columns))
	for _, column := range columns {
		ors = append(ors, column+" LIKE ?")
		params = append(params, pattern)
	}

	sql += " AND (" + strings.Join(ors, " OR ") + ")"
	return sql, params
}
