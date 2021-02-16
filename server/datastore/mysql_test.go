package datastore

import (
	"os"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/datastore/mysql"
	"github.com/fleetdm/fleet/server/test"
	"github.com/go-kit/kit/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

func setupMySQL(t *testing.T) (ds *mysql.Datastore, teardown func()) {
	config := config.MysqlConfig{
		Username: "fleet",
		Password: "insecure",
		Database: "fleet",
		// When using docker-compose.yml for local testing
		Address: "127.0.0.1:3307",
	}

	// When using Docker link on CI
	if h, ok := os.LookupEnv("MYSQL_PORT_3306_TCP_ADDR"); ok {
		config.Address = h + ":3306"
	}

	ds, err := mysql.New(config, clock.NewMockClock(), mysql.Logger(log.NewNopLogger()), mysql.LimitAttempts(1))
	require.Nil(t, err)
	teardown = func() {
		ds.Close()
	}

	return ds, teardown
}

func TestMySQL(t *testing.T) {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.SkipNow()
	}

	ds, teardown := setupMySQL(t)
	defer teardown()
	// get rid of database if it is hanging around
	err := ds.Drop()
	require.Nil(t, err)

	for _, f := range testFunctions {

		t.Run(test.FunctionName(f), func(t *testing.T) {
			defer func() { require.Nil(t, ds.Drop()) }()
			require.Nil(t, ds.MigrateTables())
			f(t, ds)
		})
	}

}
