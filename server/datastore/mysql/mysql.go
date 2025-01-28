// Package mysql is a MySQL implementation of the Datastore interface.
package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/XSAM/otelsql"
	"github.com/doug-martin/goqu/v9"
	"github.com/doug-martin/goqu/v9/exp"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/data"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/migrations/tables"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/goose"
	nano_push "github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	scep_depot "github.com/fleetdm/fleet/v4/server/mdm/scep/depot"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/go-sql-driver/mysql"
	"github.com/hashicorp/go-multierror"
	"github.com/jmoiron/sqlx"
	"github.com/ngrok/sqlmw"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

const (
	defaultSelectLimit   = 1000000
	mySQLTimestampFormat = "2006-01-02 15:04:05" // %Y/%m/%d %H:%M:%S
)

// Matches all non-word and '-' characters for replacement
var columnCharsRegexp = regexp.MustCompile(`[^\w-.]`)

// Datastore is an implementation of fleet.Datastore interface backed by
// MySQL
type Datastore struct {
	replica fleet.DBReader // so it cannot be used to perform writes
	primary *sqlx.DB

	logger log.Logger
	clock  clock.Clock
	config config.MysqlConfig
	pusher nano_push.Pusher

	// nil if no read replica
	readReplicaConfig *config.MysqlConfig

	// minimum interval between software last_opened_at timestamp to update the
	// database (see file software.go).
	minLastOpenedAtDiff time.Duration

	writeCh chan itemToWrite

	// stmtCacheMu protects access to stmtCache.
	stmtCacheMu sync.Mutex
	// stmtCache holds statements for queries.
	stmtCache map[string]*sqlx.Stmt

	// for tests, set to override the default batch size.
	testDeleteMDMProfilesBatchSize int
	// for tests, set to override the default batch size.
	testUpsertMDMDesiredProfilesBatchSize int
	// for tests set to override the default batch size.
	testSelectMDMProfilesBatchSize int

	// set this in tests to simulate an error at various stages in the
	// batchSetMDMAppleProfilesDB execution: if the string starts with "insert", it
	// will be in the insert/upsert stage, "delete" for deletion, "select" to load
	// existing ones, "reselect" to reload existing ones after insert, and "labels"
	// to simulate an error in batch setting the profile label associations.
	// "inselect", "inreselect", "indelete", etc. can also be used to fail the
	// sqlx.In before the corresponding statement.
	//
	//	e.g.: testBatchSetMDMAppleProfilesErr = "insert:fail"
	testBatchSetMDMAppleProfilesErr string

	// set this in tests to simulate an error at various stages in the
	// batchSetMDMWindowsProfilesDB execution: if the string starts with "insert",
	// it will be in the insert/upsert stage, "delete" for deletion, "select" to
	// load existing ones, "reselect" to reload existing ones after insert, and
	// "labels" to simulate an error in batch setting the profile label
	// associations. "inselect", "inreselect", "indelete", etc. can also be used to
	// fail the sqlx.In before the corresponding statement.
	//
	//	e.g.: testBatchSetMDMWindowsProfilesErr = "insert:fail"
	testBatchSetMDMWindowsProfilesErr string

	// This key is used to encrypt sensitive data stored in the Fleet DB, for example MDM
	// certificates and keys.
	serverPrivateKey string
}

// WithPusher sets an APNs pusher for the datastore, used when activating
// next activities that require MDM commands.
func (ds *Datastore) WithPusher(p nano_push.Pusher) {
	ds.pusher = p
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

// writer returns the DB instance to use for write statements, which is always
// the primary.
func (ds *Datastore) writer(ctx context.Context) *sqlx.DB {
	return ds.primary
}

// loadOrPrepareStmt will load a statement from the statements cache.
// If not available, it will attempt to prepare (create) it.
// Returns nil if it failed to prepare a statement.
//
// IMPORTANT: Adding prepare statements consumes MySQL server resources, and is limited by MySQL max_prepared_stmt_count
// system variable. This method may create 1 prepare statement for EACH database connection. Customers must be notified
// to update their MySQL configurations when additional prepare statements are added.
// For more detail, see: https://github.com/fleetdm/fleet/issues/15476
func (ds *Datastore) loadOrPrepareStmt(ctx context.Context, query string) *sqlx.Stmt {
	// the cache is only available on the replica
	if ctxdb.IsPrimaryRequired(ctx) {
		return nil
	}

	ds.stmtCacheMu.Lock()
	defer ds.stmtCacheMu.Unlock()

	stmt, ok := ds.stmtCache[query]
	if !ok {
		var err error
		stmt, err = sqlx.PreparexContext(ctx, ds.replica, query)
		if err != nil {
			level.Error(ds.logger).Log(
				"msg", "failed to prepare statement",
				"query", query,
				"err", err,
			)
			return nil
		}
		ds.stmtCache[query] = stmt
	}
	return stmt
}

func (ds *Datastore) deleteCachedStmt(query string) {
	ds.stmtCacheMu.Lock()
	defer ds.stmtCacheMu.Unlock()
	stmt, ok := ds.stmtCache[query]
	if ok {
		if err := stmt.Close(); err != nil {
			level.Error(ds.logger).Log(
				"msg", "failed to close prepared statement before deleting it",
				"query", query,
				"err", err,
			)
		}
		delete(ds.stmtCache, query)
	}
}

// NewMDMAppleSCEPDepot returns a scep_depot.Depot that uses the Datastore
// underlying MySQL writer *sql.DB.
func (ds *Datastore) NewSCEPDepot() (scep_depot.Depot, error) {
	return newSCEPDepot(ds.primary.DB, ds)
}

type entity struct {
	name string
}

var (
	hostsTable    = entity{"hosts"}
	invitesTable  = entity{"invites"}
	packsTable    = entity{"packs"}
	queriesTable  = entity{"queries"}
	sessionsTable = entity{"sessions"}
	usersTable    = entity{"users"}
)

func (ds *Datastore) withRetryTxx(ctx context.Context, fn common_mysql.TxFn) (err error) {
	return common_mysql.WithRetryTxx(ctx, ds.writer(ctx), fn, ds.logger)
}

// withTx provides a common way to commit/rollback a txFn
func (ds *Datastore) withTx(ctx context.Context, fn common_mysql.TxFn) (err error) {
	tx, err := ds.writer(ctx).BeginTxx(ctx, nil)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "create transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				ds.logger.Log("err", err, "msg", "error encountered during transaction panic rollback")
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
		minLastOpenedAtDiff: defaultMinLastOpenedAtDiff,
		maxAttempts:         defaultMaxAttempts,
		logger:              log.NewNopLogger(),
	}

	for _, setOpt := range opts {
		if setOpt != nil {
			if err := setOpt(options); err != nil {
				return nil, err
			}
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
		primary:             dbWriter,
		replica:             dbReader,
		logger:              options.logger,
		clock:               c,
		config:              config,
		readReplicaConfig:   options.replicaConfig,
		writeCh:             make(chan itemToWrite),
		stmtCache:           make(map[string]*sqlx.Stmt),
		minLastOpenedAtDiff: options.minLastOpenedAtDiff,
		serverPrivateKey:    options.privateKey,
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

func (ds *Datastore) writeChanLoop() {
	for item := range ds.writeCh {
		switch actualItem := item.item.(type) {
		case *fleet.Host:
			item.errCh <- ds.UpdateHost(item.ctx, actualItem)
		case hostXUpdatedAt:
			err := ds.withRetryTxx(
				item.ctx, func(tx sqlx.ExtContext) error {
					query := fmt.Sprintf(`UPDATE hosts SET %s = ? WHERE id=?`, actualItem.what)
					_, err := tx.ExecContext(item.ctx, query, actualItem.updatedAt, actualItem.hostID)
					return err
				},
			)
			item.errCh <- ctxerr.Wrap(item.ctx, err, "updating hosts label updated at")
		}
	}
}

var otelTracedDriverName string

func init() {
	var err error
	otelTracedDriverName, err = otelsql.Register("mysql",
		otelsql.WithAttributes(semconv.DBSystemMySQL),
		otelsql.WithSpanOptions(otelsql.SpanOptions{
			// DisableErrSkip ignores driver.ErrSkip errors which are frequently returned by the MySQL driver
			// when certain optional methods or paths are not implemented/taken.
			// For example: interpolateParams=false (the secure default) will not do a parametrized sql.conn.query directly without preparing it first, causing driver.ErrSkip
			DisableErrSkip: true,
			// Omitting span for sql.conn.reset_session since it takes ~1us and doesn't provide useful information
			OmitConnResetSession: true,
			// Omitting span for sql.rows since it is very quick and typically doesn't provide useful information beyond what's already reported by prepare/exec/query
			OmitRows: true,
		}),
		// WithSpanNameFormatter allows us to customize the span name, which is especially useful for SQL queries run outside an HTTPS transaction,
		// which do not belong to a parent span, show up as their own trace, and would otherwise be named "sql.conn.query" or "sql.conn.exec".
		otelsql.WithSpanNameFormatter(func(ctx context.Context, method otelsql.Method, query string) string {
			if query == "" {
				return string(method)
			}
			// Append query with extra whitespaces removed
			query = strings.Join(strings.Fields(query), " ")
			const maxQueryLen = 100
			if len(query) > maxQueryLen {
				query = query[:maxQueryLen] + "..."
			}
			return string(method) + ": " + query
		}),
	)
	if err != nil {
		panic(err)
	}
}

func newDB(conf *config.MysqlConfig, opts *dbOptions) (*sqlx.DB, error) {
	driverName := "mysql"
	if opts.tracingConfig != nil && opts.tracingConfig.TracingEnabled {
		if opts.tracingConfig.TracingType == "opentelemetry" {
			driverName = otelTracedDriverName
		} else {
			driverName = "apm/mysql"
		}
	}
	if opts.interceptor != nil {
		driverName = "mysql-mw"
		sql.Register(driverName, sqlmw.Driver(mysql.MySQLDriver{}, opts.interceptor))
	}
	if opts.sqlMode != "" {
		conf.SQLMode = opts.sqlMode
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
		fileContents, err := os.ReadFile(conf.PasswordPath)
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

func (ds *Datastore) MigrateTables(ctx context.Context) error {
	return tables.MigrationClient.Up(ds.writer(ctx).DB, "")
}

func (ds *Datastore) MigrateData(ctx context.Context) error {
	return data.MigrationClient.Up(ds.writer(ctx).DB, "")
}

// loadMigrations manually loads the applied migrations in ascending
// order (goose doesn't provide such functionality).
//
// Returns two lists of version IDs (one for "table" and one for "data").
func (ds *Datastore) loadMigrations(
	ctx context.Context,
	writer *sql.DB,
	reader fleet.DBReader,
) (tableRecs []int64, dataRecs []int64, err error) {
	// We need to run the following to trigger the creation of the migration status tables.
	_, err = tables.MigrationClient.GetDBVersion(writer)
	if err != nil {
		return nil, nil, err
	}
	_, err = data.MigrationClient.GetDBVersion(writer)
	if err != nil {
		return nil, nil, err
	}
	// version_id > 0 to skip the bootstrap migration that creates the migration tables.
	if err := sqlx.SelectContext(ctx, reader, &tableRecs,
		"SELECT version_id FROM "+tables.MigrationClient.TableName+" WHERE version_id > 0 AND is_applied ORDER BY id ASC",
	); err != nil {
		return nil, nil, err
	}
	if err := sqlx.SelectContext(ctx, reader, &dataRecs,
		"SELECT version_id FROM "+data.MigrationClient.TableName+" WHERE version_id > 0 AND is_applied ORDER BY id ASC",
	); err != nil {
		return nil, nil, err
	}
	return tableRecs, dataRecs, nil
}

// MigrationStatus will return the current status of the migrations
// comparing the known migrations in code and the applied migrations in the database.
//
// It assumes some deployments may have performed migrations out of order.
func (ds *Datastore) MigrationStatus(ctx context.Context) (*fleet.MigrationStatus, error) {
	if tables.MigrationClient.Migrations == nil || data.MigrationClient.Migrations == nil {
		return nil, errors.New("unexpected nil migrations list")
	}
	appliedTable, appliedData, err := ds.loadMigrations(ctx, ds.primary.DB, ds.replica)
	if err != nil {
		return nil, fmt.Errorf("cannot load migrations: %w", err)
	}
	return compareMigrations(
		tables.MigrationClient.Migrations,
		data.MigrationClient.Migrations,
		appliedTable,
		appliedData,
	), nil
}

// It assumes some deployments may have performed migrations out of order.
func compareMigrations(knownTable goose.Migrations, knownData goose.Migrations, appliedTable, appliedData []int64) *fleet.MigrationStatus {
	if len(appliedTable) == 0 && len(appliedData) == 0 {
		return &fleet.MigrationStatus{
			StatusCode: fleet.NoMigrationsCompleted,
		}
	}

	missingTable, unknownTable, equalTable := compareVersions(
		getVersionsFromMigrations(knownTable),
		appliedTable,
		knownUnknownTableMigrations,
	)

	missingData, unknownData, equalData := compareVersions(
		getVersionsFromMigrations(knownData),
		appliedData,
		knownUnknownDataMigrations,
	)

	if equalData && equalTable {
		return &fleet.MigrationStatus{
			StatusCode: fleet.AllMigrationsCompleted,
		}
	}

	//
	// The following code assumes there cannot be migrations missing on
	// "table" and database being ahead on "data" (and vice-versa).
	//

	// Check for missing migrations first, as these are more important
	// to detect than the unknown migrations.
	if len(missingTable) > 0 || len(missingData) > 0 {
		return &fleet.MigrationStatus{
			StatusCode:   fleet.SomeMigrationsCompleted,
			MissingTable: missingTable,
			MissingData:  missingData,
		}
	}

	// len(unknownTable) > 0 || len(unknownData) > 0
	return &fleet.MigrationStatus{
		StatusCode:   fleet.UnknownMigrations,
		UnknownTable: unknownTable,
		UnknownData:  unknownData,
	}
}

var (
	knownUnknownTableMigrations = map[int64]struct{}{
		// This migration was introduced incorrectly in fleet-v4.4.0 and its
		// timestamp was changed in fleet-v4.4.1.
		20210924114500: {},
	}
	knownUnknownDataMigrations = map[int64]struct{}{
		// This migration was present in 2.0.0, and was removed on a subsequent release.
		// Was basically running `DELETE FROM packs WHERE deleted = 1`, (such `deleted`
		// column doesn't exist anymore).
		20171212182459: {},
		// Deleted in
		// https://github.com/fleetdm/fleet/commit/fd61dcab67f341c9e47fb6cb968171650c19a681
		20161223115449: {},
		20170309091824: {},
		20171027173700: {},
		20171212182458: {},
	}
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
func (ds *Datastore) HealthCheck() error {
	// NOTE: does not receive a context as argument here, because the HealthCheck
	// interface potentially affects more than the datastore layer, and I'm not
	// sure we can safely identify and change them all at this moment.
	if _, err := ds.primary.ExecContext(context.Background(), "select 1"); err != nil {
		return err
	}
	if ds.readReplicaConfig != nil {
		var dst int
		if err := sqlx.GetContext(context.Background(), ds.replica, &dst, "select 1"); err != nil {
			return err
		}
	}
	return nil
}

func (ds *Datastore) closeStmts() error {
	ds.stmtCacheMu.Lock()
	defer ds.stmtCacheMu.Unlock()

	var err error
	for query, stmt := range ds.stmtCache {
		if errClose := stmt.Close(); errClose != nil {
			err = multierror.Append(err, errClose)
		}
		delete(ds.stmtCache, query)
	}
	return err
}

// Close frees resources associated with underlying mysql connection
func (ds *Datastore) Close() error {
	var err error
	if errStmt := ds.closeStmts(); errStmt != nil {
		err = multierror.Append(err, errStmt)
	}
	if errWriter := ds.primary.Close(); errWriter != nil {
		err = multierror.Append(err, errWriter)
	}
	if ds.readReplicaConfig != nil {
		if errRead := ds.replica.Close(); errRead != nil {
			err = multierror.Append(err, errRead)
		}
	}
	return err
}

// sanitizeColumn is used to sanitize column names which can't be passed as placeholders when executing sql queries
func sanitizeColumn(col string) string {
	col = columnCharsRegexp.ReplaceAllString(col, "")
	oldParts := strings.Split(col, ".")
	parts := oldParts[:0]
	for _, p := range oldParts {
		if len(p) != 0 {
			parts = append(parts, p)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	col = "`" + strings.Join(parts, "`.`") + "`"
	return col
}

// appendListOptionsToSelect will apply the given list options to ds and
// return the new select dataset.
//
// NOTE: This is a copy of appendListOptionsToSQL that uses the goqu package.
func appendListOptionsToSelect(ds *goqu.SelectDataset, opts fleet.ListOptions) *goqu.SelectDataset {
	ds = appendOrderByToSelect(ds, opts)
	ds = appendLimitOffsetToSelect(ds, opts)
	return ds
}

func appendOrderByToSelect(ds *goqu.SelectDataset, opts fleet.ListOptions) *goqu.SelectDataset {
	if opts.OrderKey != "" {
		ordersKeys := strings.Split(opts.OrderKey, ",")
		for _, key := range ordersKeys {
			ident := goqu.I(key)

			var orderedExpr exp.OrderedExpression
			if opts.OrderDirection == fleet.OrderDescending {
				orderedExpr = ident.Desc()
			} else {
				orderedExpr = ident.Asc()
			}

			ds = ds.OrderAppend(orderedExpr)
		}
	}

	return ds
}

func appendLimitOffsetToSelect(ds *goqu.SelectDataset, opts fleet.ListOptions) *goqu.SelectDataset {
	perPage := opts.PerPage
	// If caller doesn't supply a limit apply a reasonably large default limit
	// to insure that an unbounded query with many results doesn't consume too
	// much memory or hang
	if perPage == 0 {
		perPage = defaultSelectLimit
	}

	offset := perPage * opts.Page
	if offset > 0 {
		ds = ds.Offset(offset)
	}

	if opts.IncludeMetadata {
		perPage++
	}

	ds = ds.Limit(perPage)

	return ds
}

// Appends the list options SQL to the passed in SQL string. This appended
// SQL is determined by the passed in options.
//
// NOTE: this method will mutate the options argument if no explicit PerPage
// option is set (a default value will be provided) or if the cursor approach is used.
func appendListOptionsToSQL(sql string, opts *fleet.ListOptions) (string, []interface{}) {
	return appendListOptionsWithCursorToSQL(sql, nil, opts)
}

// Appends the list options SQL to the passed in SQL string. This appended
// SQL is determined by the passed in options. This supports cursor options
//
// NOTE: this method will mutate the options argument if no explicit PerPage option
// is set (a default value will be provided) or if the cursor approach is used.
func appendListOptionsWithCursorToSQL(sql string, params []interface{}, opts *fleet.ListOptions) (string, []interface{}) {
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
		if opts.TestSecondaryOrderKey != "" {
			direction := "ASC"
			if opts.TestSecondaryOrderDirection == fleet.OrderDescending {
				direction = "DESC"
			}
			sql += fmt.Sprintf(`, %s %s`, sanitizeColumn(opts.TestSecondaryOrderKey), direction)
		}
	}
	// REVIEW: If caller doesn't supply a limit apply a default limit to insure
	// that an unbounded query with many results doesn't consume too much memory
	// or hang
	if opts.PerPage == 0 {
		opts.PerPage = defaultSelectLimit
	}

	perPage := opts.PerPage
	if opts.IncludeMetadata {
		perPage++
	}
	sql = fmt.Sprintf("%s LIMIT %d", sql, perPage)

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
func (ds *Datastore) whereFilterHostsByTeams(filter fleet.TeamFilter, hostKey string) string {
	if filter.User == nil {
		// This is likely unintentional, however we would like to return no
		// results rather than panicking or returning some other error. At least
		// log.
		level.Info(ds.logger).Log("err", "team filter missing user")
		return "FALSE"
	}

	defaultAllowClause := "TRUE"
	if filter.TeamID != nil {
		defaultAllowClause = fmt.Sprintf("%s.team_id = %d", hostKey, *filter.TeamID)
	}

	if filter.User.GlobalRole != nil {
		switch *filter.User.GlobalRole {
		case fleet.RoleAdmin, fleet.RoleMaintainer, fleet.RoleObserverPlus:
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
		if team.Role == fleet.RoleAdmin ||
			team.Role == fleet.RoleMaintainer ||
			team.Role == fleet.RoleObserverPlus ||
			(team.Role == fleet.RoleObserver && filter.IncludeObserver) {
			idStrs = append(idStrs, fmt.Sprint(team.ID))
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

// whereFilterGlobalOrTeamIDByTeams is the same as whereFilterHostsByTeams, it
// returns the appropriate condition to use in the WHERE clause to render only
// the appropriate teams, but is to be used when the team_id column uses "0" to
// mean "all teams including no team". This is the case e.g. for
// software_title_host_counts.
//
// filter provides the filtering parameters that should be used.
// filterTableAlias is the name/alias of the table to use in generating the
// SQL.
func (ds *Datastore) whereFilterGlobalOrTeamIDByTeams(filter fleet.TeamFilter, filterTableAlias string) string {
	globalFilter := fmt.Sprintf("%s.team_id = 0 AND %[1]s.global_stats = 1", filterTableAlias)
	teamIDFilter := fmt.Sprintf("%s.team_id", filterTableAlias)
	return ds.whereFilterGlobalOrTeamIDByTeamsWithSqlFilter(filter, globalFilter, teamIDFilter)
}

func (ds *Datastore) whereFilterGlobalOrTeamIDByTeamsWithSqlFilter(
	filter fleet.TeamFilter, globalSqlFilter string, teamIDSqlFilter string,
) string {
	if filter.User == nil {
		// This is likely unintentional, however we would like to return no
		// results rather than panicking or returning some other error. At least
		// log.
		level.Info(ds.logger).Log("err", "team filter missing user")
		return "FALSE"
	}

	defaultAllowClause := globalSqlFilter
	if filter.TeamID != nil {
		defaultAllowClause = fmt.Sprintf("%s = %d", teamIDSqlFilter, *filter.TeamID)
	}

	if filter.User.GlobalRole != nil {
		switch *filter.User.GlobalRole {
		case fleet.RoleAdmin, fleet.RoleMaintainer, fleet.RoleObserverPlus:
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
		if team.Role == fleet.RoleAdmin ||
			team.Role == fleet.RoleMaintainer ||
			team.Role == fleet.RoleObserverPlus ||
			(team.Role == fleet.RoleObserver && filter.IncludeObserver) {
			idStrs = append(idStrs, fmt.Sprint(team.ID))
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

	return fmt.Sprintf("%s IN (%s)", teamIDSqlFilter, strings.Join(idStrs, ","))
}

// whereFilterTeams returns the appropriate condition to use in the WHERE
// clause to render only the appropriate teams.
//
// filter provides the filtering parameters that should be used. teamKey is the
// name/alias of the teams table to use in generating the SQL.
func (ds *Datastore) whereFilterTeams(filter fleet.TeamFilter, teamKey string) string {
	if filter.User == nil {
		// This is likely unintentional, however we would like to return no
		// results rather than panicking or returning some other error. At least
		// log.
		level.Info(ds.logger).Log("err", "team filter missing user")
		return "FALSE"
	}

	if filter.User.GlobalRole != nil {
		switch *filter.User.GlobalRole {
		case fleet.RoleAdmin, fleet.RoleMaintainer, fleet.RoleGitOps, fleet.RoleObserverPlus:
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
		if team.Role == fleet.RoleAdmin ||
			team.Role == fleet.RoleMaintainer ||
			team.Role == fleet.RoleGitOps ||
			team.Role == fleet.RoleObserverPlus ||
			(team.Role == fleet.RoleObserver && filter.IncludeObserver) {
			idStrs = append(idStrs, fmt.Sprint(team.ID))
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
func (ds *Datastore) whereOmitIDs(colName string, omit []uint) string {
	if len(omit) == 0 {
		return "TRUE"
	}

	var idStrs []string
	for _, id := range omit {
		idStrs = append(idStrs, fmt.Sprint(id))
	}

	return fmt.Sprintf("%s NOT IN (%s)", colName, strings.Join(idStrs, ","))
}

func (ds *Datastore) whereFilterHostsByIdentifier(identifier, stmt string, params []interface{}) (string, []interface{}) {
	if identifier == "" {
		return stmt, params
	}

	stmt += " AND ? IN (h.hostname, h.osquery_host_id, h.node_key, h.uuid, h.hardware_serial)"
	params = append(params, identifier)

	return stmt, params
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

type patternReplacer func(string) string

// likePattern returns a pattern to match m with LIKE.
func likePattern(m string) string {
	m = strings.ReplaceAll(m, "_", "\\_")
	m = strings.ReplaceAll(m, "%", "\\%")
	return "%" + m + "%"
}

// noneReplacer doesn't manipulate
func noneReplacer(m string) string {
	return m
}

// searchLike adds SQL and parameters for a "search" using LIKE syntax.
//
// The input columns must be sanitized if they are provided by the user.
func searchLike(sql string, params []interface{}, match string, columns ...string) (string, []interface{}) {
	return searchLikePattern(sql, params, match, likePattern, columns...)
}

func searchLikePattern(sql string, params []interface{}, match string, replacer patternReplacer, columns ...string) (string, []interface{}) {
	if len(columns) == 0 || len(match) == 0 {
		return sql, params
	}

	pattern := replacer(match)
	ors := make([]string, 0, len(columns))
	for _, column := range columns {
		ors = append(ors, column+" LIKE ?")
		params = append(params, pattern)
	}

	sql += " AND (" + strings.Join(ors, " OR ") + ")"
	return sql, params
}

/*
This regex matches any occurrence of a character from the ASCII character set followed by one or more characters that are not from the ASCII character set.
The first part `[[:ascii:]]` matches any character that is within the ASCII range (0 to 127 in the ASCII table),
while the second part `[^[:ascii:]]` matches any character that is not within the ASCII range.
So, when these two parts are combined with no space in between, the resulting regex matches any
sequence of characters where the first character is within the ASCII range and the following characters are not within the ASCII range.
*/
var (
	nonascii        = regexp.MustCompile(`(?P<ascii>[[:ascii:]])(?P<nonascii>[^[:ascii:]]+)`)
	nonacsiiReplace = regexp.MustCompile(`[^[:ascii:]]`)
)

// hostSearchLike searches hosts based on the given columns plus searching in hosts_emails. Note:
// the host from the `hosts` table must be aliased to `h` in `sql`.
func hostSearchLike(sql string, params []interface{}, match string, columns ...string) (string, []interface{}, bool) {
	var matchesEmail bool
	base, args := searchLike(sql, params, match, columns...)

	// special-case for hosts: if match looks like an email address, add searching
	// in host_emails table as an option, in addition to the provided columns.
	if fleet.IsLooseEmail(match) {
		matchesEmail = true
		// remove the closing paren and add the email condition to the list
		base = strings.TrimSuffix(base, ")") + " OR (" + ` EXISTS (SELECT 1 FROM host_emails he WHERE he.host_id = h.id AND he.email LIKE ?)))`
		args = append(args, likePattern(match))
	}
	return base, args, matchesEmail
}

func hostSearchLikeAny(sql string, params []interface{}, match string, columns ...string) (string, []interface{}) {
	return searchLikePattern(sql, params, buildWildcardMatchPhrase(match), noneReplacer, columns...)
}

func buildWildcardMatchPhrase(matchQuery string) string {
	return replaceMatchAny(likePattern(matchQuery))
}

func hasNonASCIIRegex(s string) bool {
	return nonascii.MatchString(s)
}

func replaceMatchAny(s string) string {
	return nonacsiiReplace.ReplaceAllString(s, "_")
}

func (ds *Datastore) InnoDBStatus(ctx context.Context) (string, error) {
	status := struct {
		Type   string `db:"Type"`
		Name   string `db:"Name"`
		Status string `db:"Status"`
	}{}
	// using the writer even when doing a read to get the data from the main db node
	err := ds.writer(ctx).GetContext(ctx, &status, "show engine innodb status")
	if err != nil {
		// To read innodb tables, DB user must have PROCESS privilege
		// This can be set by DB admin like: GRANT PROCESS,SELECT ON *.* TO 'fleet'@'%';
		if isMySQLAccessDenied(err) {
			return "", &accessDeniedError{
				Message:     "getting innodb status: DB user must have global PROCESS and SELECT privilege",
				InternalErr: err,
			}
		}
		return "", ctxerr.Wrap(ctx, err, "getting innodb status")
	}
	return status.Status, nil
}

func (ds *Datastore) ProcessList(ctx context.Context) ([]fleet.MySQLProcess, error) {
	var processList []fleet.MySQLProcess
	// using the writer even when doing a read to get the data from the main db node
	err := ds.writer(ctx).SelectContext(ctx, &processList, "show processlist")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "Getting process list")
	}
	return processList, nil
}

func insertOnDuplicateDidInsertOrUpdate(res sql.Result) bool {
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

type parameterizedStmt struct {
	Statement string
	Args      []interface{}
}

// optimisticGetOrInsert encodes an efficient pattern of looking up a row's ID
// for a unique key that is more likely to already exist (i.e. the insert
// should be infrequent, the read should succeed most of the time).
// It proceeds as follows:
//  1. Try to read the ID from the read replica.
//  2. If it does not exist, try to insert the row in the primary.
//  3. If it fails due to a duplicate key, try to read the ID again, this
//     time from the primary.
//
// The read statement must only SELECT the id column.
func (ds *Datastore) optimisticGetOrInsert(ctx context.Context, readStmt, insertStmt *parameterizedStmt) (id uint, err error) {
	return ds.optimisticGetOrInsertWithWriter(ctx, ds.writer(ctx), readStmt, insertStmt)
}

// optimisticGetOrInsertWithWriter is the same as optimisticGetOrInsert but it
// uses the provided writer to perform the insert or second read operations.
// This makes it possible to use this from inside a transaction.
func (ds *Datastore) optimisticGetOrInsertWithWriter(ctx context.Context, writer sqlx.ExtContext, readStmt, insertStmt *parameterizedStmt) (id uint, err error) { //nolint: gocritic // it's ok in this case to use ds.reader even if we receive an ExtContext
	readID := func(q sqlx.QueryerContext) (uint, error) {
		var id uint
		err := sqlx.GetContext(ctx, q, &id, readStmt.Statement, readStmt.Args...)
		return id, err
	}

	// 1. read from the read replica, as it is likely to already exist
	id, err = readID(ds.reader(ctx))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// this does not exist yet, try to insert it
			res, err := writer.ExecContext(ctx, insertStmt.Statement, insertStmt.Args...)
			if err != nil {
				if IsDuplicate(err) {
					// it might've been created between the select and the insert, read
					// again this time from the primary database connection.
					id, err := readID(writer)
					if err != nil {
						return 0, ctxerr.Wrap(ctx, err, "get id from writer")
					}
					return id, nil
				}
				return 0, ctxerr.Wrap(ctx, err, "insert")
			}
			id, _ := res.LastInsertId()
			return uint(id), nil //nolint:gosec // dismiss G115
		}
		return 0, ctxerr.Wrap(ctx, err, "get id from reader")
	}
	return id, nil
}

// batchProcessDB abstracts the batch processing logic, for a given payload:
//
// - generateValueArgs will get called for each item, the expected return values are:
//   - a string containing the placeholders for each item in the batch
//   - a slice of arguments containing one item for each placeholder
//
// - executeBatch will get called on each batch to perform the operation in the db
//
// TODO(roberto): use this function in all the functions where we do ad-hoc
// batch processing.
func batchProcessDB[T any](
	payload []T,
	batchSize int,
	generateValueArgs func(T) (string, []any),
	executeBatch func(string, []any) error,
) error {
	if len(payload) == 0 {
		return nil
	}

	var (
		args       []any
		sb         strings.Builder
		batchCount int
	)

	resetBatch := func() {
		batchCount = 0
		args = args[:0]
		sb.Reset()
	}

	for _, item := range payload {
		valuePart, itemArgs := generateValueArgs(item)
		args = append(args, itemArgs...)
		sb.WriteString(valuePart)
		batchCount++

		if batchCount >= batchSize {
			if err := executeBatch(sb.String(), args); err != nil {
				return err
			}
			resetBatch()
		}
	}

	if batchCount > 0 {
		if err := executeBatch(sb.String(), args); err != nil {
			return err
		}
	}
	return nil
}
