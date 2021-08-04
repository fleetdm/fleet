package mysql

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

const (
	schemaDbName = "schemadb"
	dumpfile     = "/tmpfs/dump.sql"
	testUsername = "root"
	testPassword = "toor"
	testAddress  = "localhost:3307"
)

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

func CreateMySQLDS(t *testing.T) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	t.Parallel()

	pc, _, _, ok := runtime.Caller(1)
	details := runtime.FuncForPC(pc)
	if !ok || details == nil {
		t.FailNow()
	}

	cleanName := strings.ReplaceAll(
		strings.TrimPrefix(details.Name(), "github.com/fleetdm/fleet/v4/"), "/", "_",
	)
	cleanName = strings.ReplaceAll(cleanName, ".", "_")
	return initializeDatabase(t, cleanName)
}
