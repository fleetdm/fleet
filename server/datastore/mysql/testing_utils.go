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
	_, filename, _, _ := runtime.Caller(0)
	base := path.Dir(filename)
	schema, err := ioutil.ReadFile(path.Join(base, "schema.sql"))
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
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
	cleanName = strings.ReplaceAll(cleanName, "(*", "")
	cleanName = strings.ReplaceAll(cleanName, ")", "")
	cleanName = strings.ReplaceAll(cleanName, ".", "_")
	return initializeDatabase(t, cleanName)
}
