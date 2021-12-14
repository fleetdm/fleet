// Package mysql is a MySQL implementation of the Datastore interface.
package mysql

import (
	"context"
	"database/sql"
	"errors"
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
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/data"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/goose"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

const (
	defaultSelectLimit   = 1000000
	mySQLTimestampFormat = "2006-01-02 15:04:05" // %Y/%m/%d %H:%M:%S
)

// Matches all non-word and '-' characters for replacement
var columnCharsRegexp = regexp.MustCompile(`[^\w-.]`)

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

	writeCh chan itemToWrite
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
	base := ctxerr.Cause(err)
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
			return ctxerr.Wrap(ctx, err, "create transaction")
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
				return backoff.Permanent(ctxerr.Wrapf(ctx, err, "got err '%s' rolling back after err", rbErr.Error()))
			}

			if retryableError(err) {
				return err
			}

			// Consider any other errors to be non-retryable
			return backoff.Permanent(err)
		}

		if err := tx.Commit(); err != nil {
			err = ctxerr.Wrap(ctx, err, "commit transaction")

			if retryableError(err) {
				return err
			}

			return backoff.Permanent(err)
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
		return ctxerr.Wrap(ctx, err, "create transaction")
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
			return ctxerr.Wrapf(ctx, err, "got err '%s' rolling back after err", rbErr.Error())
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return ctxerr.Wrap(ctx, err, "commit transaction")
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
			return nil, fmt.Errorf("replica: %w", err)
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
		writeCh:           make(chan itemToWrite),
	}

	go ds.writeChanLoop()

	return ds, nil
}

type itemToWrite struct {
	ctx   context.Context
	errCh chan error
	item  interface{}
}

type hostXUpdatedAt struct {
	hostID    uint
	updatedAt time.Time
	what      string
}

func (d *Datastore) writeChanLoop() {
	for item := range d.writeCh {
		switch actualItem := item.item.(type) {
		case *fleet.Host:
			item.errCh <- d.SaveHost(item.ctx, actualItem)
		case hostXUpdatedAt:
			query := fmt.Sprintf(`UPDATE hosts SET %s = ? WHERE id=?`, actualItem.what)
			_, err := d.writer.ExecContext(item.ctx, query, actualItem.updatedAt, actualItem.hostID)
			item.errCh <- ctxerr.Wrap(item.ctx, err, "updating hosts label updated at")
		}
	}
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
			return fmt.Errorf("register TLS config for mysql: %w", err)
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

// loadMigrations manually loads the applied migrations in ascending
// order (goose doesn't provide such functionality).
//
// Returns two lists of version IDs (one for "table" and one for "data").
func (d *Datastore) loadMigrations(
	ctx context.Context,
) (tableRecs []int64, dataRecs []int64, err error) {
	// We need to run the following to trigger the creation of the migration status tables.
	tables.MigrationClient.GetDBVersion(d.writer.DB)
	data.MigrationClient.GetDBVersion(d.writer.DB)
	// version_id > 0 to skip the bootstrap migration that creates the migration tables.
	if err := sqlx.SelectContext(ctx, d.reader, &tableRecs,
		"SELECT version_id FROM "+tables.MigrationClient.TableName+" WHERE version_id > 0 AND is_applied ORDER BY id ASC",
	); err != nil {
		return nil, nil, err
	}
	if err := sqlx.SelectContext(ctx, d.reader, &dataRecs,
		"SELECT version_id FROM "+data.MigrationClient.TableName+" WHERE version_id > 0 AND is_applied ORDER BY id ASC",
	); err != nil {
		return nil, nil, err
	}
	return tableRecs, dataRecs, nil
}

// MigrationStatus will return the current status of the migrations
// comparing the known migrations in code and the applied migrations in the database.
//
// It assumes some deployments may perform migrations out of order.
func (d *Datastore) MigrationStatus(ctx context.Context) (*fleet.MigrationStatus, error) {
	if tables.MigrationClient.Migrations == nil || data.MigrationClient.Migrations == nil {
		return nil, errors.New("unexpected nil migrations list")
	}
	appliedTable, appliedData, err := d.loadMigrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load migrations: %w", err)
	}
	if len(appliedTable) == 0 && len(appliedData) == 0 {
		return &fleet.MigrationStatus{
			StatusCode: fleet.NoMigrationsCompleted,
		}, nil
	}

	knownTable := tables.MigrationClient.Migrations
	missingTable, unknownTable, equalTable := compareVersions(
		getVersionsFromMigrations(knownTable),
		appliedTable,
		knownUnknownTableMigrations,
	)

	knownData := data.MigrationClient.Migrations
	missingData, unknownData, equalData := compareVersions(
		getVersionsFromMigrations(knownData),
		appliedData,
		knownUnknownDataMigrations,
	)

	if equalData && equalTable {
		return &fleet.MigrationStatus{
			StatusCode: fleet.AllMigrationsCompleted,
		}, nil
	}

	// The following code assumes there cannot be migrations missing on
	// "table" and database being ahead on "data" (and vice-versa).
	if len(unknownTable) > 0 || len(unknownData) > 0 {
		return &fleet.MigrationStatus{
			StatusCode:   fleet.UnknownMigrations,
			UnknownTable: unknownTable,
			UnknownData:  unknownData,
		}, nil
	}

	// len(missingTable) > 0 || len(missingData) > 0
	return &fleet.MigrationStatus{
		StatusCode:   fleet.SomeMigrationsCompleted,
		MissingTable: missingTable,
		MissingData:  missingData,
	}, nil
}

var (
	knownUnknownTableMigrations = map[int64]struct{}{
		// This migration was introduced incorrectly in fleet-v4.4.0 and its
		// timestamp was changed in fleet-v4.4.1.
		20210924114500: {},
	}
	knownUnknownDataMigrations = map[int64]struct{}{}
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

// appendListOptionsToSelect will apply the given list options to ds and
// return the new select dataset.
//
// NOTE: This is a copy of appendListOptionsToSQL that uses the goqu package.
func appendListOptionsToSelect(ds *goqu.SelectDataset, opts fleet.ListOptions) *goqu.SelectDataset {
	if opts.OrderKey != "" {
		ordersKeys := strings.Split(opts.OrderKey, ",")
		var orderedExps []exp.OrderedExpression
		for _, key := range ordersKeys {
			var orderedExp exp.OrderedExpression
			ident := goqu.I(sanitizeColumn(key))
			if opts.OrderDirection == fleet.OrderDescending {
				orderedExp = ident.Desc()
			} else {
				orderedExp = ident.Asc()
			}
			orderedExps = append(orderedExps, orderedExp)
		}
		ds = ds.Order(orderedExps...)
	}

	perPage := opts.PerPage
	// If caller doesn't supply a limit apply a default limit of 1000
	// to insure that an unbounded query with many results doesn't consume too
	// much memory or hang
	if perPage == 0 {
		perPage = defaultSelectLimit
	}
	ds = ds.Limit(perPage)

	offset := perPage * opts.Page
	if offset > 0 {
		ds = ds.Offset(offset)
	}
	return ds
}

func appendListOptionsToSQL(sql string, opts fleet.ListOptions) string {
	sql, _ = appendListOptionsWithCursorToSQL(sql, nil, opts)
	return sql
}

func appendListOptionsWithCursorToSQL(sql string, params []interface{}, opts fleet.ListOptions) (string, []interface{}) {
	orderKey := sanitizeColumn(opts.OrderKey)

	if opts.After != "" && orderKey != "" {
		afterSql := " WHERE "
		if strings.Contains(strings.ToLower(sql), "where") {
			afterSql = " AND "
		}
		if strings.HasSuffix(orderKey, "id") {
			i, _ := strconv.Atoi(opts.After)
			params = append(params, i)
		} else {
			params = append(params, opts.After)
		}
		direction := ">" // ASC
		if opts.OrderDirection == fleet.OrderDescending {
			direction = "<" // DESC
		}
		sql = fmt.Sprintf("%s %s %s %s ?", sql, afterSql, orderKey, direction)

		// After existing supersedes Page, so we disable it
		opts.Page = 0
	}

	if orderKey != "" {
		direction := "ASC"
		if opts.OrderDirection == fleet.OrderDescending {
			direction = "DESC"
		}

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

	return sql, params
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

	defaultAllowClause := "TRUE"
	if filter.TeamID != nil {
		defaultAllowClause = fmt.Sprintf("%s.team_id = %d", hostKey, *filter.TeamID)
	}

	if filter.User.GlobalRole != nil {
		switch *filter.User.GlobalRole {
		case fleet.RoleAdmin, fleet.RoleMaintainer:
			return defaultAllowClause

		case fleet.RoleObserver:
			if filter.IncludeObserver {
				return defaultAllowClause
			}
			return "FALSE"

		default:
			// Fall through to specific teams
		}
	}

	// Collect matching teams
	var idStrs []string
	var teamIDSeen bool
	for _, team := range filter.User.Teams {
		if team.Role == fleet.RoleAdmin || team.Role == fleet.RoleMaintainer ||
			(team.Role == fleet.RoleObserver && filter.IncludeObserver) {
			idStrs = append(idStrs, strconv.Itoa(int(team.ID)))
			if filter.TeamID != nil && *filter.TeamID == team.ID {
				teamIDSeen = true
			}
		}
	}

	if len(idStrs) == 0 {
		// User has no global role and no teams allowed by includeObserver.
		return "FALSE"
	}

	if filter.TeamID != nil {
		if teamIDSeen {
			// all good, this user has the right to see the requested team
			return defaultAllowClause
		}
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
		return fmt.Errorf("register mysql tls config: %w", err)
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
	err = ctxerr.Cause(err)
	mysqlErr, ok := err.(*mysql.MySQLError)
	if !ok {
		return false
	}

	// https://dev.mysql.com/doc/refman/5.7/en/error-messages-server.html#error_er_no_referenced_row_2
	const ER_NO_REFERENCED_ROW_2 = 1452
	return mysqlErr.Number == ER_NO_REFERENCED_ROW_2
}

// likePattern returns a pattern to match m with LIKE.
func likePattern(m string) string {
	m = strings.Replace(m, "_", "\\_", -1)
	m = strings.Replace(m, "%", "\\%", -1)
	return "%" + m + "%"
}

// searchLike adds SQL and parameters for a "search" using LIKE syntax.
//
// The input columns must be sanitized if they are provided by the user.
func searchLike(sql string, params []interface{}, match string, columns ...string) (string, []interface{}) {
	if len(columns) == 0 || len(match) == 0 {
		return sql, params
	}

	pattern := likePattern(match)
	ors := make([]string, 0, len(columns))
	for _, column := range columns {
		ors = append(ors, column+" LIKE ?")
		params = append(params, pattern)
	}

	sql += " AND (" + strings.Join(ors, " OR ") + ")"
	return sql, params
}
