package mysql

import (
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/common_mysql/testing_utils"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

// Android MySQL testing utilities. This file should contain VERY LITTLE code since it is also compiled into the production binary.
// Whenever possible, new code should go into a dedicated testing package (e.g. mdm/android/mysql/tests/testing_utils.go).
// These utilities are used to create a MySQL Datastore for testing the Android MDM MySQL implementation.
// They are located in the same package as the implementation to prevent a circular dependency. If put it in a different package,
// the circular dependency would be: mysql -> testing_utils -> mysql

func CreateMySQLDS(t testing.TB) *Datastore {
	return createMySQLDSWithOptions(t, nil)
}

func createMySQLDSWithOptions(t testing.TB, opts *testing_utils.DatastoreTestOptions) *Datastore {
	cleanTestName, opts := testing_utils.ProcessOptions(t, opts)
	ds := InitializeDatabase(t, cleanTestName, opts)
	t.Cleanup(func() { Close(ds) })
	return ds
}

// InitializeDatabase loads the dumped schema into a newly created database in MySQL.
// This is much faster than running the full set of migrations on each test.
func InitializeDatabase(t testing.TB, testName string, opts *testing_utils.DatastoreTestOptions) *Datastore {
	_, filename, _, _ := runtime.Caller(0)
	schemaPath := path.Join(path.Dir(filename), "schema.sql")
	testing_utils.LoadSchema(t, testName, opts, schemaPath)
	return connectMySQL(t, testName)
}

func connectMySQL(t testing.TB, testName string) *Datastore {
	// Import TestSQLMode from main MySQL testing utils to ensure consistent SQL modes across all tests
	// This ensures Android tests catch the same data integrity issues as other MySQL tests
	dbWriter, err := common_mysql.NewDB(testing_utils.MysqlTestConfig(testName), &common_mysql.DBOptions{
		SqlMode: common_mysql.TestSQLMode,
	}, "")
	require.NoError(t, err)
	ds := New(log.NewLogfmtLogger(os.Stdout), dbWriter, dbWriter)
	return ds.(*Datastore)
}

func Close(ds *Datastore) {
	_ = ds.primary.Close()
}
