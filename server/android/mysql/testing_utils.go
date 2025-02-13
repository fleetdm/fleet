package mysql

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/go-kit/log"
	"github.com/hashicorp/go-multierror"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	testUsername              = "root"
	testPassword              = "toor"
	testAddress               = "localhost:3307"
	testReplicaDatabaseSuffix = "_replica"
)

func connectMySQL(t testing.TB, testName string) *Datastore {
	cfg := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Database: testName,
		Address:  testAddress,
	}

	dbWriter, err := newDB(&cfg)
	require.NoError(t, err)
	ds := New(log.NewLogfmtLogger(os.Stdout), dbWriter, dbWriter)
	return ds.(*Datastore)
}

func newDB(conf *config.MysqlConfig) (*sqlx.DB, error) {
	driverName := "mysql"

	dsn := generateMysqlConnectionString(*conf)
	db, err := sqlx.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxIdleConns(conf.MaxIdleConns)
	db.SetMaxOpenConns(conf.MaxOpenConns)
	db.SetConnMaxLifetime(time.Second * time.Duration(conf.ConnMaxLifetime))

	var dbError error
	maxConnectionAttempts := 10
	for attempt := 0; attempt < maxConnectionAttempts; attempt++ {
		dbError = db.Ping()
		if dbError == nil {
			// we're connected!
			break
		}
		interval := time.Duration(attempt) * time.Second
		fmt.Printf("could not connect to db: %v, sleeping %v\n", dbError, interval)
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

	return connectMySQL(t, testName)
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

func (ds *Datastore) Close() error {
	var err error
	if errWriter := ds.primary.Close(); errWriter != nil {
		err = multierror.Append(err, errWriter)
	}
	return err
}

func CreateMySQLDS(t testing.TB) *Datastore {
	return createMySQLDSWithOptions(t, nil)
}

func ExecAdhocSQL(tb testing.TB, ds *Datastore, fn func(q sqlx.ExtContext) error) {
	tb.Helper()
	err := fn(ds.primary)
	require.NoError(tb, err)
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

	require.NoError(t, ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
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

func testCtx() context.Context {
	return context.Background()
}
