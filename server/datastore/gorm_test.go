package datastore

import (
	"fmt"
	"os"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/require"
)

func setupGorm(t *testing.T) (ds kolide.Datastore, teardown func()) {
	var (
		user     = "kolide"
		password = "kolide"
		dbName   = "kolide"
		host     = "127.0.0.1"
	)

	// use linked container if available.
	if h, ok := os.LookupEnv("MYSQL_PORT_3306_TCP_ADDR"); ok {
		host = h
	}
	connString := fmt.Sprintf("%s:%s@(%s:3306)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, dbName)
	ds, err := New("gorm-mysql", connString)
	require.Nil(t, err)
	teardown = func() {
		db, ok := ds.(gormDB)
		if !ok {
			panic("expected gormDB datastore")
		}
		require.Nil(t, db.Drop())
		db.DB.Close()
	}
	return ds, teardown
}

func TestGorm(t *testing.T) {
	if _, ok := os.LookupEnv("MYSQL_TEST"); !ok {
		t.SkipNow()
	}
	for _, f := range testFunctions {
		t.Run(functionName(f), func(t *testing.T) {
			ds, teardown := setupGorm(t)
			defer teardown()
			f(t, ds)
		})
	}
}
