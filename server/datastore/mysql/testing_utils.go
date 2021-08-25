package mysql

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
)

const (
	testUsername = "root"
	testPassword = "toor"
	testAddress  = "localhost:3307"
)

func connectMySQL(t *testing.T, testName string, opts *DatastoreTestOptions) *Datastore {
	config := config.MysqlConfig{
		Username: testUsername,
		Password: testPassword,
		Database: testName,
		Address:  testAddress,
	}

	// Create datastore client
	var replicaOpt DBOption
	if opts.Replica {
		replicaOpt = Replica(&config)
	}
	ds, err := New(config, clock.NewMockClock(), Logger(log.NewNopLogger()), LimitAttempts(1), replicaOpt)
	require.Nil(t, err)
	return ds
}

// initializeDatabase loads the dumped schema into a newly created database in
// MySQL. This is much faster than running the full set of migrations on each
// test.
func initializeDatabase(t *testing.T, testName string, opts *DatastoreTestOptions) *Datastore {
	_, filename, _, _ := runtime.Caller(0)
	base := path.Dir(filename)
	schema, err := ioutil.ReadFile(path.Join(base, "schema.sql"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	// TODO(mna): Some thoughts about how to test replication with potential lag:
	// * When opts.Replica is true, create 2 temp DBs, the primary and the replica,
	//   load the schema file in both.
	// * Start a goroutine that select{}s on a timer and a context, with the timer
	//   based on opts.ReplicaLag (a time.Duration).
	// * On each tick, read (from the primary) the list of tables updated since the
	//   last tick, and re-create those tables (using CREATE TABLE ... SELECT ...).
	// * Register a t.Cleanup() to close/cancel the context so that the "replication"
	//   goroutine is terminated on test exit.
	//
	// I believe that would work - no index/constraint would be created in the replica,
	// but that is fine for testing. Only concern is whether this is fast enough for
	// the "happy path" (when we don't want to simulate replication lag issues).
	// Replicating lag issues (where written data is not visible in the read replica
	// yet) will most definitely work.
	//
	// Another option would be to actually setup a primary-replica in docker, but I
	// think the lightweight version is simpler for tests.

	// Load schema from dumpfile
	if out, err := exec.Command(
		"docker-compose", "exec", "-T", "mysql_test",
		// Command run inside container
		"mysql",
		"-u"+testUsername, "-p"+testPassword,
		"-e",
		fmt.Sprintf(
			"DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s; SET FOREIGN_KEY_CHECKS=0; %s;",
			testName, testName, testName, schema,
		),
	).CombinedOutput(); err != nil {
		t.Error(err)
		t.Error(string(out))
		t.FailNow()
	}
	return connectMySQL(t, testName, opts)
}

// DatastoreTestOptions configures how the test datastore is created
// by CreateMySQLDSWithOptions.
type DatastoreTestOptions struct {
	Replica bool
}

func createMySQLDSWithOptions(t *testing.T, opts *DatastoreTestOptions) *Datastore {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.Skip("MySQL tests are disabled")
	}

	t.Parallel()

	if opts == nil {
		// so it is never nil in internal helper functions
		opts = new(DatastoreTestOptions)
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
	return initializeDatabase(t, cleanName, opts)
}

func CreateMySQLDSWithOptions(t *testing.T, opts *DatastoreTestOptions) *Datastore {
	return createMySQLDSWithOptions(t, opts)
}

func CreateMySQLDS(t *testing.T) *Datastore {
	return createMySQLDSWithOptions(t, nil)
}
