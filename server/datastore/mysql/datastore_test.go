package mysql

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/test"
	"github.com/go-kit/kit/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

const (
	schemaDbName = "schemadb"
	dumpfile     = "dump.sql"
	testUsername = "root"
	testPassword = "toor"
	testAddress  = "localhost:3307"
)

func connectMySQL(t *testing.T, testName string) *Datastore {
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Database: testName,
		Address:  testAddress,
	}

	// Create datastore client
	ds, err := New(config, clock.NewMockClock(), Logger(log.NewNopLogger()), LimitAttempts(1))
	require.Nil(t, err)
	return ds
}

// initializeSchema initializes a database schema using the normal Fleet
// migrations, then outputs the schema with mysqldump within the MySQL Docker
// container.
func initializeSchema(t *testing.T) {
	// Create the database (must use raw MySQL client to do this)
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true", testUsername, testPassword, testAddress),
	)
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec("DROP DATABASE IF EXISTS schemadb; CREATE DATABASE schemadb;")
	require.NoError(t, err)

	// Create a datastore client in order to run migrations as usual
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Address:  testAddress,
		Database: schemaDbName,
	}
	ds, err := New(config, clock.NewMockClock(), Logger(log.NewNopLogger()), LimitAttempts(1))
	require.Nil(t, err)
	defer ds.Close()
	require.Nil(t, ds.MigrateTables())

	// Dump schema to dumpfile
	if out, err := exec.Command(
		"docker-compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysqldump",
		"-u"+testUsername, "-p"+testPassword,
		"schemadb",
		"--compact", "--skip-comments",
		"--result-file="+dumpfile,
	).CombinedOutput(); err != nil {
		t.Error(err)
		t.Error(string(out))
		t.FailNow()
	}
}

// initializeDatabase loads the dumped schema into a newly created database in
// MySQL. This is much faster than running the full set of migrations on each
// test.
func initializeDatabase(t *testing.T, dbName string) {
	// Load schema from dumpfile
	if out, err := exec.Command(
		"docker-compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysql",
		"-u"+testUsername, "-p"+testPassword,
		"-e",
		fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; SET FOREIGN_KEY_CHECKS=0; SOURCE %s;",
			dbName, dbName, dbName, dumpfile,
		),
	).CombinedOutput(); err != nil {
		t.Error(err)
		t.Error(string(out))
		t.FailNow()
	}

}

func runTest(t *testing.T, testFunc func(*testing.T, kolide.Datastore)) {
	t.Run(test.FunctionName(testFunc), func(t *testing.T) {
		t.Parallel()

		// Create a new database and load the schema for each test
		initializeDatabase(t, test.FunctionName(testFunc))

		ds := connectMySQL(t, test.FunctionName(testFunc))
		defer ds.Close()

		testFunc(t, ds)
	})
}

func TestMySQL(t *testing.T) {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	// Initialize the schema once for the entire test run.
	initializeSchema(t)

	for _, f := range datastore.TestFunctions {
		runTest(t, f)
	}

}
