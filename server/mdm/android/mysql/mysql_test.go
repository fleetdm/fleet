package mysql

// TODO(26218): Refactor this to remove duplication.

import (
	"context"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

// initializeDatabase loads the dumped schema into a newly created database in MySQL.
// This is much faster than running the full set of migrations on each test.
func initializeDatabase(t testing.TB, testName string, opts *testing_utils.DatastoreTestOptions) *Datastore {
	_, filename, _, _ := runtime.Caller(0)
	schemaPath := path.Join(path.Dir(filename), "schema.sql")
	testing_utils.LoadSchema(t, testName, opts, schemaPath)
	return connectMySQL(t, testName)
}

func connectMySQL(t testing.TB, testName string) *Datastore {
	dbWriter, err := common_mysql.NewDB(testing_utils.MysqlTestConfig(testName), &common_mysql.DBOptions{}, "")
	require.NoError(t, err)
	ds := New(log.NewLogfmtLogger(os.Stdout), dbWriter, dbWriter)
	return ds.(*Datastore)
}

func CreateMySQLDS(t testing.TB) *Datastore {
	return createMySQLDSWithOptions(t, nil)
}

func createMySQLDSWithOptions(t testing.TB, opts *testing_utils.DatastoreTestOptions) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	if opts == nil {
		// so it is never nil in internal helper functions
		opts = new(testing_utils.DatastoreTestOptions)
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

func Close(ds *Datastore) {
	_ = ds.Writer(context.Background()).Close()
}
