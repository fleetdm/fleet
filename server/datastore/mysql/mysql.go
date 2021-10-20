// Package mysql is a MySQL implementation of the Datastore interface.
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/VividCortex/mysqlerr"
	"github.com/WatchBeam/clock"
	"github.com/cenkalti/backoff/v4"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/data"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
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

// dbReader is an interface that defines the methods required for reads.
type dbReader interface {
	sqlx.QueryerContext

	Close() error
	Rebind(string) string
}

// Datastore is an implementation of fleet.Datastore interface backed by
// MySQL
type Datastore struct {
	reader dbReader // so it cannot be used to perform writes
	writer *sqlx.DB

	logger log.Logger
	clock  clock.Clock
	config config.MysqlConfig

	// nil if no read replica
	readReplicaConfig *config.MysqlConfig
}

type txFn func(sqlx.ExtContext) error

type entity struct {
	name string
}

var (
	hostsTable            = entity{"hosts"}
	invitesTable          = entity{"invites"}
	labelsTable           = entity{"labels"}
	packsTable            = entity{"packs"}
	queriesTable          = entity{"queries"}
	scheduledQueriesTable = entity{"scheduled_queries"}
	sessionsTable         = entity{"sessions"}
	teamsTable            = entity{"teams"}
	usersTable            = entity{"users"}
)

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
func (d *Datastore) withRetryTxx(ctx context.Context, fn txFn) (err error) {
	operation := func() error {
		tx, err := d.writer.BeginTxx(ctx, nil)
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

// withTx provides a common way to commit/rollback a txFn
func (d *Datastore) withTx(ctx context.Context, fn txFn) (err error) {
	tx, err := d.writer.BeginTxx(ctx, nil)
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
			return errors.Wrapf(err, "got err '%s' rolling back after err", rbErr.Error())
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "commit transaction")
	}

	return nil
}

// New creates an MySQL datastore.
func New(config config.MysqlConfig, c clock.Clock, opts ...DBOption) (*Datastore, error) {
	options := &dbOptions{
		maxAttempts: defaultMaxAttempts,
		logger:      log.NewNopLogger(),
	}

	for _, setOpt := range opts {
		if setOpt != nil {
			setOpt(options)
		}
	}

	if err := checkConfig(&config); err != nil {
		return nil, err
	}
	if options.replicaConfig != nil {
		if err := checkConfig(options.replicaConfig); err != nil {
			return nil, errors.Wrap(err, "replica")
		}
	}

	dbWriter, err := newDB(&config, options)
	if err != nil {
		return nil, err
	}
	dbReader := dbWriter
	if options.replicaConfig != nil {
		dbReader, err = newDB(options.replicaConfig, options)
		if err != nil {
			return nil, err
		}
	}

	ds := &Datastore{
		writer:            dbWriter,
		reader:            dbReader,
		logger:            options.logger,
		clock:             c,
		config:            config,
		readReplicaConfig: options.replicaConfig,
	}

	return ds, nil
}

func newDB(conf *config.MysqlConfig, opts *dbOptions) (*sqlx.DB, error) {
	dsn := generateMysqlConnectionString(*conf)
	db, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(conf.MaxIdleConns)
	db.SetMaxOpenConns(conf.MaxOpenConns)
	db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifetime))

	var dbError error
	for attempt := 0; attempt < opts.maxAttempts; attempt++ {
		dbError = db.Ping()
		if dbError == nil {
			// we're connected!
			break
		}
		interval := time.Duration(attempt) * time.Second
		opts.logger.Log("mysql", fmt.Sprintf(
			"could not connect to db: %v, sleeping %v", dbError, interval))
		time.Sleep(interval)
	}

	if dbError != nil {
		return nil, dbError
	}
	return db, nil
}

func checkConfig(conf *config.MysqlConfig) error {
	if conf.PasswordPath != "" && conf.Password != "" {
		return errors.New("A MySQL password and a MySQL password file were provided - please specify only one")
	}

	// Check to see if the flag is populated
	// Check if file exists on disk
	// If file exists read contents
	if conf.PasswordPath != "" {
		fileContents, err := ioutil.ReadFile(conf.PasswordPath)
		if err != nil {
			return err
		}
		conf.Password = strings.TrimSpace(string(fileContents))
	}

	if conf.TLSCA != "" {
		conf.TLSConfig = "custom"
		err := registerTLS(*conf)
		if err != nil {
			return errors.Wrap(err, "register TLS config for mysql")
		}
	}
	return nil
}

func (d *Datastore) MigrateTables(ctx context.Context) error {
	return tables.MigrationClient.Up(d.writer.DB, "")
}

func (d *Datastore) MigrateData(ctx context.Context) error {
	return data.MigrationClient.Up(d.writer.DB, "")
}

func (d *Datastore) MigrationStatus(ctx context.Context) (fleet.MigrationStatus, error) {
	if tables.MigrationClient.Migrations == nil || data.MigrationClient.Migrations == nil {
		return 0, errors.New("unexpected nil migrations list")
	}

	lastTablesMigration, err := tables.MigrationClient.Migrations.Last()
	if err != nil {
		return 0, errors.Wrap(err, "missing tables migrations")
	}

	currentTablesVersion, err := tables.MigrationClient.GetDBVersion(d.writer.DB)
	if err != nil {
		return 0, errors.Wrap(err, "cannot get table migration status")
	}

	lastDataMigration, err := data.MigrationClient.Migrations.Last()
	if err != nil {
		return 0, errors.Wrap(err, "missing data migrations")
	}

	currentDataVersion, err := data.MigrationClient.GetDBVersion(d.writer.DB)
	if err != nil {
		return 0, errors.Wrap(err, "cannot get data migration status")
	}

	switch {
	case currentDataVersion == 0 && currentTablesVersion == 0:
		return fleet.NoMigrationsCompleted, nil

	case currentTablesVersion != lastTablesMigration.Version ||
		currentDataVersion != lastDataMigration.Version:
		return fleet.SomeMigrationsCompleted, nil

	default:
		return fleet.AllMigrationsCompleted, nil
	}
}

// HealthCheck returns an error if the MySQL backend is not healthy.
func (d *Datastore) HealthCheck() error {
	// NOTE: does not receive a context as argument here, because the HealthCheck
	// interface potentially affects more than the datastore layer, and I'm not
	// sure we can safely identify and change them all at this moment.
	if _, err := d.writer.ExecContext(context.Background(), "select 1"); err != nil {
		return err
	}
	if d.readReplicaConfig != nil {
		var dst int
		if err := sqlx.GetContext(context.Background(), d.reader, &dst, "select 1"); err != nil {
			return err
		}
	}
	return nil
}

// Close frees resources associated with underlying mysql connection
func (d *Datastore) Close() error {
	err := d.writer.Close()
	if d.readReplicaConfig != nil {
		errRead := d.reader.Close()
		if err == nil {
			err = errRead
		}
	}
	return err
}

func sanitizeColumn(col string) string {
	return columnCharsRegexp.ReplaceAllString(col, "")
}

func appendListOptionsToSQL(sql string, opts fleet.ListOptions) string {
	if opts.OrderKey != "" {
		direction := "ASC"
		if opts.OrderDirection == fleet.OrderDescending {
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

// whereFilterHostsByTeams returns the appropriate condition to use in the WHERE
// clause to render only the appropriate teams.
//
// filter provides the filtering parameters that should be used. hostKey is the
// name/alias of the hosts table to use in generating the SQL.
func (d *Datastore) whereFilterHostsByTeams(filter fleet.TeamFilter, hostKey string) string {
	if filter.User == nil {
		// This is likely unintentional, however we would like to return no
		// results rather than panicking or returning some other error. At least
		// log.
		level.Info(d.logger).Log("err", "team filter missing user")
		return "FALSE"
	}

	if filter.User.GlobalRole != nil {
		switch *filter.User.GlobalRole {

		case fleet.RoleAdmin, fleet.RoleMaintainer:
			return "TRUE"

		case fleet.RoleObserver:
			if filter.IncludeObserver {
				return "TRUE"
			}
			return "FALSE"

		default:
			// Fall through to specific teams
		}
	}

	// Collect matching teams
	var idStrs []string
	for _, team := range filter.User.Teams {
		if team.Role == fleet.RoleAdmin || team.Role == fleet.RoleMaintainer ||
			(team.Role == fleet.RoleObserver && filter.IncludeObserver) {
			idStrs = append(idStrs, strconv.Itoa(int(team.ID)))
		}
	}

	if len(idStrs) == 0 {
		// User has no global role and no teams allowed by includeObserver.
		return "FALSE"
	}

	return fmt.Sprintf("%s.team_id IN (%s)", hostKey, strings.Join(idStrs, ","))
}

// whereFilterTeams returns the appropriate condition to use in the WHERE
// clause to render only the appropriate teams.
//
// filter provides the filtering parameters that should be used. hostKey is the
// name/alias of the teams table to use in generating the SQL.
func (d *Datastore) whereFilterTeams(filter fleet.TeamFilter, teamKey string) string {
	if filter.User == nil {
		// This is likely unintentional, however we would like to return no
		// results rather than panicking or returning some other error. At least
		// log.
		level.Info(d.logger).Log("err", "team filter missing user")
		return "FALSE"
	}

	if filter.User.GlobalRole != nil {
		switch *filter.User.GlobalRole {

		case fleet.RoleAdmin, fleet.RoleMaintainer:
			return "TRUE"

		case fleet.RoleObserver:
			if filter.IncludeObserver {
				return "TRUE"
			}
			return "FALSE"

		default:
			// Fall through to specific teams
		}
	}

	// Collect matching teams
	var idStrs []string
	for _, team := range filter.User.Teams {
		if team.Role == fleet.RoleAdmin || team.Role == fleet.RoleMaintainer ||
			(team.Role == fleet.RoleObserver && filter.IncludeObserver) {
			idStrs = append(idStrs, strconv.Itoa(int(team.ID)))
		}
	}

	if len(idStrs) == 0 {
		// User has no global role and no teams allowed by includeObserver.
		return "FALSE"
	}

	return fmt.Sprintf("%s.id IN (%s)", teamKey, strings.Join(idStrs, ","))
}

// whereOmitIDs returns the appropriate condition to use in the WHERE
// clause to omit the provided IDs from the selection.
func (d *Datastore) whereOmitIDs(colName string, omit []uint) string {
	if len(omit) == 0 {
		return "TRUE"
	}

	var idStrs []string
	for _, id := range omit {
		idStrs = append(idStrs, strconv.Itoa(int(id)))
	}

	return fmt.Sprintf("%s NOT IN (%s)", colName, strings.Join(idStrs, ","))
}

// registerTLS adds client certificate configuration to the mysql connection.
func registerTLS(conf config.MysqlConfig) error {
	tlsCfg := config.TLS{
		TLSCert:       conf.TLSCert,
		TLSKey:        conf.TLSKey,
		TLSCA:         conf.TLSCA,
		TLSServerName: conf.TLSServerName,
	}
	cfg, err := tlsCfg.ToTLSConfig()
	if err != nil {
		return err
	}
	if err := mysql.RegisterTLSConfig(conf.TLSConfig, cfg); err != nil {
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
	if len(columns) == 0 || len(match) == 0 {
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
