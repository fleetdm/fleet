package mysql

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"text/tabwriter"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	nanodep_client "github.com/fleetdm/fleet/v4/server/mdm/nanodep/client"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/smallstep/pkcs7"
	"github.com/stretchr/testify/require"
)

const (
	testUsername              = "root"
	testPassword              = "toor"
	testAddress               = "localhost:3307"
	testReplicaDatabaseSuffix = "_replica"
	testReplicaAddress        = "localhost:3310"
)

func connectMySQL(t testing.TB, testName string, opts *DatastoreTestOptions) *Datastore {
	cfg := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Database: testName,
		Address:  testAddress,
	}

	// Create datastore client
	var replicaOpt DBOption
	if opts.DummyReplica {
		replicaConf := cfg
		replicaConf.Database += testReplicaDatabaseSuffix
		replicaOpt = Replica(&replicaConf)
	}

	// For use with WithFleetConfig. Note that since we're setting up the DB in a different way
	// than in production, we have to reset the MinSoftwareLastOpenedAtDiff field to its default so
	// it's not overwritten here.
	tc := config.TestConfig()
	tc.Osquery.MinSoftwareLastOpenedAtDiff = defaultMinLastOpenedAtDiff

	// TODO: for some reason we never log datastore messages when running integration tests, why?
	//
	// Changes below assume that we want to follows the same pattern as the rest of the codebase.
	dslogger := log.NewLogfmtLogger(os.Stdout)
	if os.Getenv("FLEET_INTEGRATION_TESTS_DISABLE_LOG") != "" {
		dslogger = log.NewNopLogger()
	}

	// set SQL mode to ANSI, as it's a special mode equivalent to:
	// REAL_AS_FLOAT, PIPES_AS_CONCAT, ANSI_QUOTES, IGNORE_SPACE, and
	// ONLY_FULL_GROUP_BY
	//
	// Per the docs:
	// > This mode changes syntax and behavior to conform more closely to
	// standard SQL.
	//
	// https://dev.mysql.com/doc/refman/8.0/en/sql-mode.html#sqlmode_ansi
	ds, err := New(cfg, clock.NewMockClock(), Logger(dslogger), LimitAttempts(1), replicaOpt, SQLMode("ANSI"), WithFleetConfig(&tc))
	require.Nil(t, err)

	if opts.DummyReplica {
		setupDummyReplica(t, testName, ds, opts)
	}
	if opts.RealReplica {
		replicaOpts := &dbOptions{
			minLastOpenedAtDiff: defaultMinLastOpenedAtDiff,
			maxAttempts:         1,
			logger:              log.NewNopLogger(),
			sqlMode:             "ANSI",
		}
		setupRealReplica(t, testName, ds, replicaOpts)
	}

	return ds
}

func setupDummyReplica(t testing.TB, testName string, ds *Datastore, opts *DatastoreTestOptions) {
	t.Helper()

	// create the context that will cancel the replication goroutine on test exit
	var cancel func()
	ctx := context.Background()
	if tt, ok := t.(*testing.T); ok {
		if dl, ok := tt.Deadline(); ok {
			ctx, cancel = context.WithDeadline(ctx, dl)
		} else {
			ctx, cancel = context.WithCancel(ctx)
		}
	}
	t.Cleanup(cancel)

	type replicationRun struct {
		forceTables     []string
		replicationDone chan struct{}
	}

	// start the replication goroutine that runs when signalled through a
	// channel, the replication runs in lock-step - the test is in control of
	// when the replication happens, by calling opts.RunReplication(), and when
	// that call returns, the replication is guaranteed to be done. This supports
	// simulating all kinds of replica lag.
	ch := make(chan replicationRun)
	go func() {
		// if it exits because of a panic/failed replication, cancel the context
		// immediately so that RunReplication is unblocked too.
		defer cancel()

		primary := ds.primary
		replica := ds.replica.(*sqlx.DB)
		replicaDB := testName + testReplicaDatabaseSuffix
		last := time.Now().Add(-time.Minute)

		// drop all foreign keys in the replica, as that causes issues even with
		// FOREIGN_KEY_CHECKS=0
		var fks []struct {
			TableName      string `db:"TABLE_NAME"`
			ConstraintName string `db:"CONSTRAINT_NAME"`
		}
		err := primary.SelectContext(ctx, &fks, `
          SELECT
            TABLE_NAME, CONSTRAINT_NAME
          FROM
            INFORMATION_SCHEMA.KEY_COLUMN_USAGE
          WHERE
            TABLE_SCHEMA = ? AND
            REFERENCED_TABLE_NAME IS NOT NULL`, testName)
		require.NoError(t, err)
		for _, fk := range fks {
			stmt := fmt.Sprintf(`ALTER TABLE %s.%s DROP FOREIGN KEY %s`, replicaDB, fk.TableName, fk.ConstraintName)
			_, err := replica.ExecContext(ctx, stmt)
			// If the FK was already removed do nothing
			if err != nil && strings.Contains(err.Error(), "check that column/key exists") {
				continue
			}

			require.NoError(t, err)
		}

		for {
			select {
			case out := <-ch:
				// identify tables with changes since the last call
				var tables []string
				err := primary.SelectContext(ctx, &tables, `
          SELECT
            table_name
          FROM
            information_schema.tables
          WHERE
            table_schema = ? AND
            table_type = 'BASE TABLE' AND
            update_time >= ?`, testName, last)
				require.NoError(t, err)

				// dedupe and add forced tables
				tableSet := make(map[string]bool, len(tables)+len(out.forceTables))
				for _, tbl := range tables {
					tableSet[tbl] = true
				}
				for _, tbl := range out.forceTables {
					tableSet[tbl] = true
				}
				tables = tables[:0]
				for tbl := range tableSet {
					tables = append(tables, tbl)
				}
				t.Logf("changed tables since %v: %v", last, tables)

				err = primary.GetContext(ctx, &last, `
          SELECT
            MAX(update_time)
          FROM
            information_schema.tables
          WHERE
            table_schema = ? AND
            table_type = 'BASE TABLE'`, testName)
				require.NoError(t, err)
				t.Logf("last update time of primary is now %v", last)

				// replicate by dropping the existing table and re-creating it from
				// the primary.
				for _, tbl := range tables {
					stmt := fmt.Sprintf(`DROP TABLE IF EXISTS %s.%s`, replicaDB, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					require.NoError(t, err)
					stmt = fmt.Sprintf(`CREATE TABLE %s.%s LIKE %s.%s`, replicaDB, tbl, testName, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					require.NoError(t, err)
					stmt = fmt.Sprintf(`INSERT INTO %s.%s SELECT * FROM %s.%s`, replicaDB, tbl, testName, tbl)
					t.Log(stmt)
					_, err = replica.ExecContext(ctx, stmt)
					require.NoError(t, err)
				}

				out.replicationDone <- struct{}{}
				t.Logf("replication step executed, next will consider updates since %s", last)

			case <-ctx.Done():
				return
			}
		}
	}()

	// set RunReplication to a function that triggers the replication and waits
	// for it to complete.
	opts.RunReplication = func(forceTables ...string) {
		done := make(chan struct{})
		ch <- replicationRun{forceTables, done}
		select {
		case <-done:
		case <-ctx.Done():
		}
	}
}

// we need to keep track of the databases that need replication in order to
// configure the replica to only track those, otherwise the replica worker
// might fail/stop trying to execute statements on databases that don't exist.
//
// this happens because we create a database and import our test dump on the
// leader each time `connectMySQL` is called, but we only do the same on the
// replica when it's enabled via options.
var (
	mu                   sync.Mutex
	databasesToReplicate string
)

func setupRealReplica(t testing.TB, testName string, ds *Datastore, options *dbOptions) {
	t.Helper()
	const replicaUser = "replicator"
	const replicaPassword = "rotacilper"

	t.Cleanup(
		func() {
			// Stop replica
			if out, err := exec.Command(
				"docker", "compose", "exec", "-T", "mysql_replica_test",
				// Command run inside container
				"mysql",
				"-u"+testUsername, "-p"+testPassword,
				"-e",
				"STOP REPLICA; RESET REPLICA ALL;",
			).CombinedOutput(); err != nil {
				t.Log(err)
				t.Log(string(out))
			}
		},
	)

	ctx := context.Background()

	// Create replication user
	_, err := ds.primary.ExecContext(ctx, fmt.Sprintf("DROP USER IF EXISTS '%s'", replicaUser))
	require.NoError(t, err)
	_, err = ds.primary.ExecContext(ctx, fmt.Sprintf("CREATE USER '%s'@'%%' IDENTIFIED BY '%s'", replicaUser, replicaPassword))
	require.NoError(t, err)
	_, err = ds.primary.ExecContext(ctx, fmt.Sprintf("GRANT REPLICATION SLAVE ON *.* TO '%s'@'%%'", replicaUser))
	require.NoError(t, err)
	_, err = ds.primary.ExecContext(ctx, "FLUSH PRIVILEGES")
	require.NoError(t, err)

	var version string
	err = ds.primary.GetContext(ctx, &version, "SELECT VERSION()")
	require.NoError(t, err)

	// Retrieve master binary log coordinates
	ms, err := ds.MasterStatus(ctx, version)
	require.NoError(t, err)

	mu.Lock()
	databasesToReplicate = strings.TrimPrefix(databasesToReplicate+fmt.Sprintf(", `%s`", testName), ",")
	mu.Unlock()

	setSourceStmt := fmt.Sprintf(`
			CHANGE REPLICATION SOURCE TO
				GET_SOURCE_PUBLIC_KEY=1,
				SOURCE_HOST='mysql_test',
				SOURCE_USER='%s',
				SOURCE_PASSWORD='%s',
				SOURCE_LOG_FILE='%s',
				SOURCE_LOG_POS=%d
		`, replicaUser, replicaPassword, ms.File, ms.Position)
	if strings.HasPrefix(version, "8.0") {
		setSourceStmt = fmt.Sprintf(`
			CHANGE MASTER TO
				GET_MASTER_PUBLIC_KEY=1,
				MASTER_HOST='mysql_test',
				MASTER_USER='%s',
				MASTER_PASSWORD='%s',
				MASTER_LOG_FILE='%s',
				MASTER_LOG_POS=%d
		`, replicaUser, replicaPassword, ms.File, ms.Position)
	}

	// Configure replica and start replication
	if out, err := exec.Command(
		"docker", "compose", "exec", "-T", "mysql_replica_test",
		// Command run inside container
		"mysql",
		"-u"+testUsername, "-p"+testPassword,
		"-e",
		fmt.Sprintf(
			`
			STOP REPLICA;
			RESET REPLICA ALL;
			CHANGE REPLICATION FILTER REPLICATE_DO_DB = ( %s );
			%s;
			START REPLICA;
			`, databasesToReplicate, setSourceStmt,
		),
	).CombinedOutput(); err != nil {
		t.Error(err)
		t.Error(string(out))
		t.FailNow()
	}

	// Connect to the replica
	replicaConfig := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Database: testName,
		Address:  testReplicaAddress,
	}
	require.NoError(t, checkConfig(&replicaConfig))
	replica, err := newDB(&replicaConfig, options)
	require.NoError(t, err)
	ds.replica = replica
	ds.readReplicaConfig = &replicaConfig
}

// initializeDatabase loads the dumped schema into a newly created database in
// MySQL. This is much faster than running the full set of migrations on each
// test.
func initializeDatabase(t testing.TB, testName string, opts *DatastoreTestOptions) *Datastore {
	_, filename, _, _ := runtime.Caller(0)
	base := path.Dir(filename)
	schema, err := os.ReadFile(path.Join(base, "schema.sql"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// execute the schema for the test db, and once more for the replica db if
	// that option is set.
	dbs := []string{testName}
	if opts.DummyReplica {
		dbs = append(dbs, testName+testReplicaDatabaseSuffix)
	}
	for _, dbName := range dbs {
		// Load schema from dumpfile
		sqlCommands := fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; SET FOREIGN_KEY_CHECKS=0; %s;",
			dbName, dbName, dbName, schema,
		)

		cmd := exec.Command(
			"docker", "compose", "exec", "-T", "mysql_test",
			// Command run inside container
			"mysql",
			"-u"+testUsername, "-p"+testPassword,
		)
		cmd.Stdin = strings.NewReader(sqlCommands)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Error(err)
			t.Error(string(out))
			t.FailNow()
		}
	}
	if opts.RealReplica {
		// Load schema from dumpfile
		sqlCommands := fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; SET FOREIGN_KEY_CHECKS=0; %s;",
			testName, testName, testName, schema,
		)

		cmd := exec.Command(
			"docker", "compose", "exec", "-T", "mysql_replica_test",
			// Command run inside container
			"mysql",
			"-u"+testUsername, "-p"+testPassword,
		)
		cmd.Stdin = strings.NewReader(sqlCommands)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Error(err)
			t.Error(string(out))
			t.FailNow()
		}
	}

	return connectMySQL(t, testName, opts)
}

// DatastoreTestOptions configures how the test datastore is created
// by CreateMySQLDSWithOptions.
type DatastoreTestOptions struct {
	// DummyReplica indicates that a read replica test database should be created.
	DummyReplica bool

	// RunReplication is the function to call to execute the replication of all
	// missing changes from the primary to the replica. The function is created
	// and set automatically by CreateMySQLDSWithOptions. The test is in full
	// control of when the replication is executed. Only applies to DummyReplica.
	// Note that not all changes to data show up in the information_schema
	// update_time timestamp, so to work around that limitation, explicit table
	// names can be provided to force their replication.
	RunReplication func(forceTables ...string)

	// RealReplica indicates that the replica should be a real DB replica, with a dedicated connection.
	RealReplica bool
}

func createMySQLDSWithOptions(t testing.TB, opts *DatastoreTestOptions) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	if opts == nil {
		// so it is never nil in internal helper functions
		opts = new(DatastoreTestOptions)
	}

	if tt, ok := t.(*testing.T); ok && !opts.RealReplica {
		tt.Parallel()
	}

	if opts.RealReplica {
		if _, ok := os.LookupEnv("MYSQL_REPLICA_TEST"); !ok {
			t.Skip("MySQL replica tests are disabled. Set env var MYSQL_REPLICA_TEST=1 to enable.")
		}
	}

	pc, _, _, ok := runtime.Caller(2)
	details := runtime.FuncForPC(pc)
	if !ok || details == nil {
		t.FailNow()
	}

	cleanName := strings.ReplaceAll(
		strings.TrimPrefix(details.Name(), "github.com/fleetdm/fleet/v4/"), "/", "_",
	)
	cleanName = strings.ReplaceAll(cleanName, ".", "_")
	if len(cleanName) > 60 {
		// the later parts are more unique than the start, with the package names,
		// so trim from the start.
		cleanName = cleanName[len(cleanName)-60:]
	}
	ds := initializeDatabase(t, cleanName, opts)
	t.Cleanup(func() { ds.Close() })
	return ds
}

func CreateMySQLDSWithReplica(t *testing.T, opts *DatastoreTestOptions) *Datastore {
	if opts == nil {
		opts = new(DatastoreTestOptions)
	}
	opts.RealReplica = true
	const numberOfAttempts = 10
	var ds *Datastore
	for attempt := 0; attempt < numberOfAttempts; {
		attempt++
		ds = createMySQLDSWithOptions(t, opts)
		status, err := ds.ReplicaStatus(context.Background())
		require.NoError(t, err)
		if status["Replica_SQL_Running"] != "Yes" {
			t.Logf("create replica attempt: %d replica status: %+v", attempt, status)
			if lastErr, ok := status["Last_Error"]; ok && lastErr != "" {
				t.Logf("replica not running after attempt %d; Last_Error: %s", attempt, lastErr)
			}
			continue
		}
		break
	}
	require.NotNil(t, ds)
	return ds
}

func CreateMySQLDSWithOptions(t *testing.T, opts *DatastoreTestOptions) *Datastore {
	return createMySQLDSWithOptions(t, opts)
}

func CreateMySQLDS(t testing.TB) *Datastore {
	return createMySQLDSWithOptions(t, nil)
}

func CreateNamedMySQLDS(t *testing.T, name string) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	ds := initializeDatabase(t, name, new(DatastoreTestOptions))
	t.Cleanup(func() { ds.Close() })
	return ds
}

func ExecAdhocSQL(tb testing.TB, ds *Datastore, fn func(q sqlx.ExtContext) error) {
	tb.Helper()
	err := fn(ds.primary)
	require.NoError(tb, err)
}

func ExecAdhocSQLWithError(ds *Datastore, fn func(q sqlx.ExtContext) error) error {
	return fn(ds.primary)
}

// EncryptWithPrivateKey encrypts data with the server private key associated
// with the Datastore.
func EncryptWithPrivateKey(tb testing.TB, ds *Datastore, data []byte) ([]byte, error) {
	return encrypt(data, ds.serverPrivateKey)
}

// TruncateTables truncates the specified tables, in order, using ds.writer.
// Note that the order is typically not important because FK checks are
// disabled while truncating. If no table is provided, all tables (except
// those that are seeded by the SQL schema file) are truncated.
func TruncateTables(t testing.TB, ds *Datastore, tables ...string) {
	// By setting DISABLE_TRUNCATE_TABLES a developer can troubleshoot tests
	// by inspecting mysql tables.
	if os.Getenv("DISABLE_TRUNCATE_TABLES") != "" {
		return
	}

	// those tables are seeded with the schema.sql and as such must not
	// be truncated - a more precise approach must be used for those, e.g.
	// delete where id > max before test, or something like that.
	nonEmptyTables := map[string]bool{
		"app_config_json":                  true,
		"migration_status_tables":          true,
		"osquery_options":                  true,
		"mdm_delivery_status":              true,
		"mdm_operation_types":              true,
		"mdm_apple_declaration_categories": true,
	}
	ctx := context.Background()

	require.NoError(t, ds.withTx(ctx, func(tx sqlx.ExtContext) error {
		var skipSeeded bool

		if len(tables) == 0 {
			skipSeeded = true
			sql := `
      SELECT
        table_name
      FROM
        information_schema.tables
      WHERE
        table_schema = database() AND
        table_type = 'BASE TABLE'
    `
			if err := sqlx.SelectContext(ctx, tx, &tables, sql); err != nil {
				return err
			}
		}

		if _, err := tx.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS=0`); err != nil {
			return err
		}
		for _, tbl := range tables {
			if nonEmptyTables[tbl] {
				if skipSeeded {
					continue
				}
				return fmt.Errorf("cannot truncate table %s, it contains seed data from schema.sql", tbl)
			}
			if _, err := tx.ExecContext(ctx, "TRUNCATE TABLE "+tbl); err != nil {
				return err
			}
		}
		if _, err := tx.ExecContext(ctx, `SET FOREIGN_KEY_CHECKS=1`); err != nil {
			return err
		}
		return nil
	}))
}

// this is meant to be used for debugging/testing that statement uses an efficient
// plan (e.g. makes use of an index, avoids full scans, etc.) using the data already
// created for tests. Calls to this function should be temporary and removed when
// done investigating the plan, so it is expected that this function will be detected
// as unused.
func explainSQLStatement(w io.Writer, db sqlx.QueryerContext, stmt string, args ...interface{}) { //nolint:deadcode,unused
	var rows []struct {
		ID           int             `db:"id"`
		SelectType   string          `db:"select_type"`
		Table        sql.NullString  `db:"table"`
		Partitions   sql.NullString  `db:"partitions"`
		Type         sql.NullString  `db:"type"`
		PossibleKeys sql.NullString  `db:"possible_keys"`
		Key          sql.NullString  `db:"key"`
		KeyLen       sql.NullInt64   `db:"key_len"`
		Ref          sql.NullString  `db:"ref"`
		Rows         sql.NullInt64   `db:"rows"`
		Filtered     sql.NullFloat64 `db:"filtered"`
		Extra        sql.NullString  `db:"Extra"`
	}
	if err := sqlx.SelectContext(context.Background(), db, &rows, "EXPLAIN "+stmt, args...); err != nil {
		panic(err)
	}
	fmt.Fprint(w, "\n\n", strings.Repeat("-", 60), "\n", stmt, "\n", strings.Repeat("-", 60), "\n")
	tw := tabwriter.NewWriter(w, 0, 1, 1, ' ', tabwriter.Debug)

	fmt.Fprintln(tw, "id\tselect_type\ttable\tpartitions\ttype\tpossible_keys\tkey\tkey_len\tref\trows\tfiltered\textra")
	for _, row := range rows {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%d\t%f\t%s\n", row.ID, row.SelectType, row.Table.String, row.Partitions.String,
			row.Type.String, row.PossibleKeys.String, row.Key.String, row.KeyLen.Int64, row.Ref.String, row.Rows.Int64, row.Filtered.Float64, row.Extra.String)
	}
	if err := tw.Flush(); err != nil {
		panic(err)
	}
}

func DumpTable(t *testing.T, q sqlx.QueryerContext, tableName string) { //nolint: unused
	rows, err := q.QueryContext(context.Background(), fmt.Sprintf(`SELECT * FROM %s`, tableName))
	require.NoError(t, err)
	defer rows.Close()

	t.Logf(">> dumping table %s:", tableName)

	var anyDst []any
	var strDst []sql.NullString
	var sb strings.Builder
	for rows.Next() {
		if anyDst == nil {
			cols, err := rows.Columns()
			require.NoError(t, err)
			anyDst = make([]any, len(cols))
			strDst = make([]sql.NullString, len(cols))
			for i := 0; i < len(cols); i++ {
				anyDst[i] = &strDst[i]
			}
			t.Logf("%v", cols)
		}
		require.NoError(t, rows.Scan(anyDst...))

		sb.Reset()
		for _, v := range strDst {
			if v.Valid {
				sb.WriteString(v.String)
			} else {
				sb.WriteString("NULL")
			}
			sb.WriteString("\t")
		}
		t.Logf("%s", sb.String())
	}
	require.NoError(t, rows.Err())
	t.Logf("<< dumping table %s completed", tableName)
}

func generateDummyWindowsProfile(uuid string) []byte {
	return []byte(fmt.Sprintf(`<Replace><Target><LocUri>./Device/Foo/%s</LocUri></Target></Replace>`, uuid))
}

// TODO(roberto): update when we have datastore functions and API methods for this
func InsertWindowsProfileForTest(t *testing.T, ds *Datastore, teamID uint) string {
	profUUID := "w" + uuid.NewString()
	prof := generateDummyWindowsProfile(profUUID)
	ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_windows_configuration_profiles (profile_uuid, team_id, name, syncml, uploaded_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(context.Background(), stmt, profUUID, teamID, fmt.Sprintf("name-%s", profUUID), prof)
		return err
	})
	return profUUID
}

// GetAggregatedStats retrieves aggregated stats for the given query
func GetAggregatedStats(ctx context.Context, ds *Datastore, aggregate fleet.AggregatedStatsType, id uint) (fleet.AggregatedStats, error) {
	result := fleet.AggregatedStats{}
	stmt := `
	SELECT
		   JSON_EXTRACT(json_value, '$.user_time_p50') as user_time_p50,
		   JSON_EXTRACT(json_value, '$.user_time_p95') as user_time_p95,
		   JSON_EXTRACT(json_value, '$.system_time_p50') as system_time_p50,
		   JSON_EXTRACT(json_value, '$.system_time_p95') as system_time_p95,
		   JSON_EXTRACT(json_value, '$.total_executions') as total_executions
	FROM aggregated_stats WHERE id=? AND type=?
	`
	err := sqlx.GetContext(ctx, ds.reader(ctx), &result, stmt, id, aggregate)
	return result, err
}

// SetOrderedCreatedAtTimestamps enforces an ordered sequence of created_at
// timestamps in a database table. This can be useful in tests instead of
// adding time.Sleep calls to just force specific ordered timestamps for the
// test entries of interest, and it doesn't slow down the unit test.
//
// The first timestamp will be after afterTime, and each provided key will have
// a timestamp incremented by 1s.
func SetOrderedCreatedAtTimestamps(t testing.TB, ds *Datastore, afterTime time.Time, table, keyCol string, keys ...any) time.Time {
	now := afterTime
	for i := 0; i < len(keys); i++ {
		now = now.Add(time.Second)
		ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(context.Background(),
				fmt.Sprintf(`UPDATE %s SET created_at=? WHERE %s=?`, table, keyCol), now, keys[i])
			return err
		})
	}
	return now
}

func CreateABMKeyCertIfNotExists(t testing.TB, ds *Datastore) {
	certPEM, keyPEM, _, err := GenerateTestABMAssets(t)
	require.NoError(t, err)
	var assets []fleet.MDMConfigAsset
	_, err = ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetABMKey,
	}, nil)
	if err != nil {
		var nfe fleet.NotFoundError
		require.ErrorAs(t, err, &nfe)
		assets = append(assets, fleet.MDMConfigAsset{Name: fleet.MDMAssetABMKey, Value: keyPEM})
	}

	_, err = ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetABMCert,
	}, nil)
	if err != nil {
		var nfe fleet.NotFoundError
		require.ErrorAs(t, err, &nfe)
		assets = append(assets, fleet.MDMConfigAsset{Name: fleet.MDMAssetABMCert, Value: certPEM})
	}

	if len(assets) != 0 {
		err = ds.InsertMDMConfigAssets(context.Background(), assets, ds.writer(context.Background()))
		require.NoError(t, err)
	}
}

// CreateAndSetABMToken creates a new ABM token (using an existing ABM key/cert) and stores it in the DB.
func CreateAndSetABMToken(t testing.TB, ds *Datastore, orgName string) *fleet.ABMToken {
	assets, err := ds.GetAllMDMConfigAssetsByName(context.Background(), []fleet.MDMAssetName{
		fleet.MDMAssetABMKey,
		fleet.MDMAssetABMCert,
	}, nil)
	require.NoError(t, err)

	certPEM := assets[fleet.MDMAssetABMCert].Value

	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:       "test_consumer",
		ConsumerSecret:    "test_secret",
		AccessToken:       "test_access_token",
		AccessSecret:      "test_access_secret",
		AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rawToken, err := json.Marshal(testBMToken)
	require.NoError(t, err)

	smimeToken := fmt.Sprintf(
		"Content-Type: text/plain;charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 7bit\r\n"+
			"\r\n%s", rawToken,
	)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	encryptedToken, err := pkcs7.Encrypt([]byte(smimeToken), []*x509.Certificate{cert})
	require.NoError(t, err)

	tokenBytes := fmt.Sprintf(
		"Content-Type: application/pkcs7-mime; name=\"smime.p7m\"; smime-type=enveloped-data\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"Content-Disposition: attachment; filename=\"smime.p7m\"\r\n"+
			"Content-Description: S/MIME Encrypted Message\r\n"+
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken))

	tok, err := ds.InsertABMToken(context.Background(), &fleet.ABMToken{EncryptedToken: []byte(tokenBytes), OrganizationName: orgName})
	require.NoError(t, err)
	return tok
}

func SetTestABMAssets(t testing.TB, ds *Datastore, orgName string) *fleet.ABMToken {
	apnsCert, apnsKey, err := GenerateTestCertBytes()
	require.NoError(t, err)

	certPEM, keyPEM, tokenBytes, err := GenerateTestABMAssets(t)
	require.NoError(t, err)
	assets := []fleet.MDMConfigAsset{
		{Name: fleet.MDMAssetABMCert, Value: certPEM},
		{Name: fleet.MDMAssetABMKey, Value: keyPEM},
		{Name: fleet.MDMAssetAPNSCert, Value: apnsCert},
		{Name: fleet.MDMAssetAPNSKey, Value: apnsKey},
		{Name: fleet.MDMAssetCACert, Value: certPEM},
		{Name: fleet.MDMAssetCAKey, Value: keyPEM},
	}

	err = ds.InsertMDMConfigAssets(context.Background(), assets, nil)
	require.NoError(t, err)

	tok, err := ds.InsertABMToken(context.Background(), &fleet.ABMToken{EncryptedToken: tokenBytes, OrganizationName: orgName})
	require.NoError(t, err)

	appCfg, err := ds.AppConfig(context.Background())
	require.NoError(t, err)
	appCfg.MDM.EnabledAndConfigured = true
	appCfg.MDM.AppleBMEnabledAndConfigured = true
	err = ds.SaveAppConfig(context.Background(), appCfg)
	require.NoError(t, err)

	return tok
}

func GenerateTestABMAssets(t testing.TB) ([]byte, []byte, []byte, error) {
	certPEM, keyPEM, err := GenerateTestCertBytes()
	require.NoError(t, err)

	testBMToken := &nanodep_client.OAuth1Tokens{
		ConsumerKey:       "test_consumer",
		ConsumerSecret:    "test_secret",
		AccessToken:       "test_access_token",
		AccessSecret:      "test_access_secret",
		AccessTokenExpiry: time.Date(2999, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	rawToken, err := json.Marshal(testBMToken)
	require.NoError(t, err)

	smimeToken := fmt.Sprintf(
		"Content-Type: text/plain;charset=UTF-8\r\n"+
			"Content-Transfer-Encoding: 7bit\r\n"+
			"\r\n%s", rawToken,
	)

	block, _ := pem.Decode(certPEM)
	require.NotNil(t, block)
	require.Equal(t, "CERTIFICATE", block.Type)
	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	encryptedToken, err := pkcs7.Encrypt([]byte(smimeToken), []*x509.Certificate{cert})
	require.NoError(t, err)

	tokenBytes := fmt.Sprintf(
		"Content-Type: application/pkcs7-mime; name=\"smime.p7m\"; smime-type=enveloped-data\r\n"+
			"Content-Transfer-Encoding: base64\r\n"+
			"Content-Disposition: attachment; filename=\"smime.p7m\"\r\n"+
			"Content-Description: S/MIME Encrypted Message\r\n"+
			"\r\n%s", base64.StdEncoding.EncodeToString(encryptedToken))

	return certPEM, keyPEM, []byte(tokenBytes), nil
}

// TODO: move to mdmcrypto?
func GenerateTestCertBytes() ([]byte, []byte, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			ExtraNames: []pkix.AttributeTypeAndValue{
				{
					Type:  asn1.ObjectIdentifier{0, 9, 2342, 19200300, 100, 1, 1},
					Value: "com.apple.mgmt.Example",
				},
			},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})

	return certPEM, keyPEM, nil
}

// MasterStatus is a struct that holds the file and position of the master, retrieved by SHOW MASTER STATUS
type MasterStatus struct {
	File     string
	Position uint64
}

func (ds *Datastore) MasterStatus(ctx context.Context, mysqlVersion string) (MasterStatus, error) {
	stmt := "SHOW BINARY LOG STATUS"
	if strings.HasPrefix(mysqlVersion, "8.0") {
		stmt = "SHOW MASTER STATUS"
	}

	rows, err := ds.writer(ctx).Query(stmt)
	if err != nil {
		return MasterStatus{}, ctxerr.Wrap(ctx, err, stmt)
	}
	defer rows.Close()

	// Since we don't control the column names, and we want to be future compatible,
	// we only scan for the columns we care about.
	ms := MasterStatus{}
	// Get the column names from the query
	columns, err := rows.Columns()
	if err != nil {
		return ms, ctxerr.Wrap(ctx, err, "get columns")
	}
	numberOfColumns := len(columns)
	for rows.Next() {
		cols := make([]interface{}, numberOfColumns)
		for i := range cols {
			cols[i] = new(string)
		}
		err := rows.Scan(cols...)
		if err != nil {
			return ms, ctxerr.Wrap(ctx, err, "scan row")
		}
		for i, col := range cols {
			switch columns[i] {
			case "File":
				ms.File = *col.(*string)
			case "Position":
				ms.Position, err = strconv.ParseUint(*col.(*string), 10, 64)
				if err != nil {
					return ms, ctxerr.Wrap(ctx, err, "parse Position")
				}

			}
		}
	}
	if err := rows.Err(); err != nil {
		return ms, ctxerr.Wrap(ctx, err, "rows error")
	}
	if ms.File == "" || ms.Position == 0 {
		return ms, ctxerr.New(ctx, "missing required fields in master status")
	}
	return ms, nil
}

func (ds *Datastore) ReplicaStatus(ctx context.Context) (map[string]interface{}, error) {
	rows, err := ds.reader(ctx).QueryContext(ctx, "SHOW REPLICA STATUS")
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "show replica status")
	}
	defer rows.Close()

	// Get the column names from the query
	columns, err := rows.Columns()
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get columns")
	}
	numberOfColumns := len(columns)
	result := make(map[string]interface{}, numberOfColumns)
	for rows.Next() {
		cols := make([]interface{}, numberOfColumns)
		for i := range cols {
			cols[i] = &sql.NullString{}
		}
		err = rows.Scan(cols...)
		if err != nil {
			return result, ctxerr.Wrap(ctx, err, "scan row")
		}
		for i, col := range cols {
			colValue := col.(*sql.NullString)
			if colValue.Valid {
				result[columns[i]] = colValue.String
			} else {
				result[columns[i]] = nil
			}
		}
	}
	if err := rows.Err(); err != nil {
		return result, ctxerr.Wrap(ctx, err, "rows error")
	}
	return result, nil
}
