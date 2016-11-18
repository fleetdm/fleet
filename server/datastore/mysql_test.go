package datastore

import (
	"fmt"
	"os"
	"testing"

	"github.com/WatchBeam/clock"
	"github.com/go-kit/kit/log"
	_ "github.com/go-sql-driver/mysql"
	"github.com/kolide/kolide-ose/server/datastore/mysql"
	_ "github.com/kolide/kolide-ose/server/datastore/mysql/migrations"
	"github.com/stretchr/testify/require"
)

func setupMySQL(t *testing.T) (ds *mysql.Datastore, teardown func()) {
	var (
		user     = "kolide"
		password = "kolide"
		dbName   = "kolide"
		host     = "127.0.0.1"
	)

	if h, ok := os.LookupEnv("MYSQL_PORT_3306_TCP_ADDR"); ok {
		host = h
	}

	connString := fmt.Sprintf("%s:%s@(%s:3306)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, dbName)

	ds, err := mysql.New(connString, clock.NewMockClock(), mysql.Logger(log.NewNopLogger()), mysql.LimitAttempts(1))
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

		t.Run(functionName(f), func(t *testing.T) {
			require.Nil(t, ds.Migrate())
			defer func() { require.Nil(t, ds.Drop()) }()

			f(t, ds)
		})
	}

}
