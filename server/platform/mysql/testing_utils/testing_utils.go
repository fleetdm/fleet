package testing_utils

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	TestUsername              = "root"
	TestPassword              = "toor"
	TestReplicaDatabaseSuffix = "_replica"
)

var TestReplicaAddress = getTestReplicaAddress()

// getTestReplicaAddress returns the MySQL replica test server address from environment variable
// FLEET_MYSQL_REPLICA_TEST_PORT or defaults to localhost:3310
func getTestReplicaAddress() string {
	if port := os.Getenv("FLEET_MYSQL_REPLICA_TEST_PORT"); port != "" {
		return "localhost:" + port
	}
	return "localhost:3310"
}

var TestAddress = getTestAddress()

// getTestAddress returns the MySQL test server address from environment variable
// FLEET_MYSQL_TEST_PORT or defaults to localhost:3307
func getTestAddress() string {
	if port := os.Getenv("FLEET_MYSQL_TEST_PORT"); port != "" {
		return "localhost:" + port
	}
	return "localhost:3307"
}

// TruncateTables truncates the specified tables, in order, using ds.writer.
// Note that the order is typically not important because FK checks are
// disabled while truncating. If no table is provided, all tables (except
// those that are seeded by the SQL schema file) are truncated.
func TruncateTables(t testing.TB, db *sqlx.DB, logger *slog.Logger, nonEmptyTables map[string]bool, tables ...string) {
	// By setting DISABLE_TRUNCATE_TABLES a developer can troubleshoot tests
	// by inspecting mysql tables.
	if os.Getenv("DISABLE_TRUNCATE_TABLES") != "" {
		return
	}

	ctx := context.Background()

	require.NoError(t, common_mysql.WithTxx(ctx, db, func(tx sqlx.ExtContext) error {
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
	}, logger))
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

	UniqueTestName string
}

// MigrationVersions returns the versions of the Go migrations in dir, derived
// from their filenames the same way goose.NumericComponent does (goose isn't a
// platform package, so it can't be imported here).
func MigrationVersions(dir string) ([]int64, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}
	var versions []int64
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}
		// Non-migration files (migration.go, fleet4_migrations.go, ...) have no numeric prefix.
		prefix, _, found := strings.Cut(name, "_")
		if v, err := strconv.ParseInt(prefix, 10, 64); found && err == nil && v > 0 {
			versions = append(versions, v)
		}
	}
	slices.Sort(versions)
	return versions, nil
}

// LoadDefaultSchema loads the default database schema for testing.
func LoadDefaultSchema(t testing.TB, testName string, opts *DatastoreTestOptions) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Join(filepath.Dir(thisFile), "../../../datastore/mysql")
	schema, err := os.ReadFile(filepath.Join(dir, "schema.sql"))
	require.NoError(t, err)

	// schema.sql has no migration_status_tables rows, so that PRs adding
	// migrations don't conflict on it. Derive them from the migration files.
	versions, err := MigrationVersions(filepath.Join(dir, "migrations", "tables"))
	require.NoError(t, err)
	var sb strings.Builder
	sb.Write(schema)
	sb.WriteString("\nINSERT INTO `migration_status_tables` (`version_id`, `is_applied`) VALUES (0, 1)")
	for _, v := range versions {
		fmt.Fprintf(&sb, ", (%d, 1)", v)
	}
	sb.WriteString(";\n")
	loadSchemaSQL(t, testName, opts, sb.String())
}

// LoadSchema loads a database schema from the specified path for testing.
func LoadSchema(t testing.TB, testName string, opts *DatastoreTestOptions, schemaPath string) {
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	loadSchemaSQL(t, testName, opts, string(schema))
}

func loadSchemaSQL(t testing.TB, testName string, opts *DatastoreTestOptions, schema string) {
	// execute the schema for the test db, and once more for the replica db if
	// that option is set.
	dbs := []string{testName}
	if opts.DummyReplica {
		dbs = append(dbs, testName+TestReplicaDatabaseSuffix)
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
			"--default-character-set=utf8mb4",
			"-u"+TestUsername, "-p"+TestPassword,
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
			"--default-character-set=utf8mb4",
			"-u"+TestUsername, "-p"+TestPassword,
		)
		cmd.Stdin = strings.NewReader(sqlCommands)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Error(err)
			t.Error(string(out))
			t.FailNow()
		}
	}
}

func MysqlTestConfig(testName string) *common_mysql.MysqlConfig {
	return &common_mysql.MysqlConfig{
		Username:        TestUsername,
		Password:        TestPassword,
		Database:        testName,
		Address:         TestAddress,
		MaxIdleConns:    50,
		MaxOpenConns:    50,
		ConnMaxLifetime: 0,
	}
}

func ProcessOptions(t testing.TB, opts *DatastoreTestOptions) (string, *DatastoreTestOptions) {
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

	var cleanTestName string
	if opts.UniqueTestName != "" {
		cleanTestName = opts.UniqueTestName
	} else {
		const numberOfStackFramesFromTest = 3
		pc, _, _, ok := runtime.Caller(numberOfStackFramesFromTest)
		details := runtime.FuncForPC(pc)
		if !ok || details == nil {
			t.FailNow()
		}

		cleanTestName = strings.ReplaceAll(
			strings.TrimPrefix(details.Name(), "github.com/fleetdm/fleet/v4/"), "/", "_",
		)
	}

	cleanTestName = strings.ReplaceAll(cleanTestName, ".", "_")
	if len(cleanTestName) > 60 {
		// the later parts are more unique than the start, with the package names,
		// so trim from the start.
		cleanTestName = cleanTestName[len(cleanTestName)-60:]
	}
	return cleanTestName, opts
}
