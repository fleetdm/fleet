package mysql

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

const (
	schemaDbName = "schemadb"
	dumpfile     = "dump.sql"
	testUsername = "root"
	testPassword = "toor"
	testAddress  = "localhost:3307"
)

func TestMain(m *testing.M) {
	if _, ok := os.LookupEnv("MYSQL_TEST"); ok {
		// Initialize the schema once for the entire test run.
		initializeSchemaOrPanic()
	}
	os.Exit(m.Run())
}

func panicif(err error) {
	if err != nil {
		panic(err)
	}
}

// initializeSchemaOrPanic initializes a database schema using the normal Fleet
// migrations, then outputs the schema with mysqldump within the MySQL Docker
// container.
func initializeSchemaOrPanic() {
	// Create the database (must use raw MySQL client to do this)
	db, err := sql.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true", testUsername, testPassword, testAddress),
	)
	panicif(err)
	defer db.Close()
	_, err = db.Exec("DROP DATABASE IF EXISTS schemadb; CREATE DATABASE schemadb;")
	panicif(err)

	// Create a datastore client in order to run migrations as usual
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Address:  testAddress,
		Database: schemaDbName,
	}
	ds, err := New(config, clock.NewMockClock(), Logger(log.NewNopLogger()), LimitAttempts(1))
	panicif(err)
	defer ds.Close()
	panicif(ds.MigrateTables())

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
		fmt.Println(string(out))
		panicif(err)
	}
}

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

// initializeDatabase loads the dumped schema into a newly created database in
// MySQL. This is much faster than running the full set of migrations on each
// test.
func initializeDatabase(t *testing.T, testName string) *Datastore {
	// Load schema from dumpfile
	if out, err := exec.Command(
		"docker-compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysql",
		"-u"+testUsername, "-p"+testPassword,
		"-e",
		fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; SET FOREIGN_KEY_CHECKS=0; SOURCE %s;",
			testName, testName, testName, dumpfile,
		),
	).CombinedOutput(); err != nil {
		t.Error(err)
		t.Error(string(out))
		t.FailNow()
	}
	return connectMySQL(t, testName)
}

func runTest(t *testing.T, testFunc func(*testing.T, fleet.Datastore)) {
	t.Run(test.FunctionName(testFunc), func(t *testing.T) {
		t.Parallel()

		// Create a new database and load the schema for each test
		ds := initializeDatabase(t, test.FunctionName(testFunc))
		defer ds.Close()

		testFunc(t, ds)
	})
}

func RunTestsAgainstMySQL(t *testing.T, tests []func(*testing.T, fleet.Datastore)) {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	for _, f := range tests {
		runTest(t, f)
	}
}

func CreateMySQLDS(t *testing.T) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	t.Parallel()

	return initializeDatabase(t, t.Name())
}
