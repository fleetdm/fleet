package testing_utils

// TODO(26218): Refactor this to remove duplication.

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mysql"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func connectMySQL(t testing.TB, testName string) *mysql.Datastore {
	cfg := config.MysqlConfig{
		Username: testing_utils.TestUsername,
		Password: testing_utils.TestPassword,
		Database: testName,
		Address:  testing_utils.TestAddress,
	}

	dbWriter, err := common_mysql.NewDB(&cfg, &common_mysql.DBOptions{}, "")
	require.NoError(t, err)
	ds := mysql.New(log.NewLogfmtLogger(os.Stdout), dbWriter, dbWriter)
	return ds.(*mysql.Datastore)
}

// initializeDatabase loads the dumped schema into a newly created database in
// MySQL. This is much faster than running the full set of migrations on each
// test.
func initializeDatabase(t testing.TB, testName string, opts *DatastoreTestOptions) *mysql.Datastore {
	_, filename, _, _ := runtime.Caller(0)
	base := path.Dir(filename)
	schema, err := os.ReadFile(path.Join(base, "../schema.sql"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// execute the schema for the test db, and once more for the replica db if
	// that option is set.
	dbs := []string{testName}
	if opts.DummyReplica {
		dbs = append(dbs, testName+testing_utils.TestReplicaDatabaseSuffix)
	}
	for _, dbName := range dbs {
		// Load schema from dumpfile
		sqlCommands := fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; SET FOREIGN_KEY_CHECKS=0; %s;",
			dbName, dbName, dbName, schema,
		)

		cmd := exec.Command( // nolint:gosec // Waive G204 since this is a test file
			"docker", "compose", "exec", "-T", "mysql_test",
			// Command run inside container
			"mysql",
			"-u"+testing_utils.TestUsername, "-p"+testing_utils.TestPassword,
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

		cmd := exec.Command( // nolint:gosec // Waive G204 since this is a test file
			"docker", "compose", "exec", "-T", "mysql_replica_test",
			// Command run inside container
			"mysql",
			"-u"+testing_utils.TestUsername, "-p"+testing_utils.TestPassword,
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

func createMySQLDSWithOptions(t testing.TB, opts *DatastoreTestOptions) *mysql.Datastore {
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
	t.Cleanup(func() { Close(ds) })
	return ds
}

func Close(ds *mysql.Datastore) {
	_ = ds.Writer(context.Background()).Close()
}

func CreateMySQLDS(t testing.TB) *mysql.Datastore {
	return createMySQLDSWithOptions(t, nil)
}

func ExecAdhocSQL(tb testing.TB, ds *mysql.Datastore, fn func(q sqlx.ExtContext) error) {
	tb.Helper()
	err := fn(ds.Writer(context.Background()))
	require.NoError(tb, err)
}
