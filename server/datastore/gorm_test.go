package datastore

import (
	"fmt"
	"os"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func setupGorm(t *testing.T) kolide.Datastore {
	user := "kolide"
	password := "kolide"
	dbName := "kolide"

	// try container first
	host := os.Getenv("MYSQL_PORT_3306_TCP_ADDR")
	if host == "" {
		host = "127.0.0.1"
	}
	host = fmt.Sprintf("%s:3306", host)

	connString := fmt.Sprintf("%s:%s@(%s)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, dbName)
	ds, err := New("gorm-mysql", connString)

	err = ds.Migrate()
	assert.Nil(t, err)
	return ds
}

func teardownGorm(t *testing.T, ds kolide.Datastore) {
	err := ds.Drop()
	assert.Nil(t, err)
}

func TestGorm(t *testing.T) {
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		t.SkipNow()
	}
	for _, f := range testFunctions {
		t.Run(functionName(f), func(t *testing.T) {
			ds := setupGorm(t)
			defer teardownGorm(t, ds)
			f(t, ds)
		})
	}
}
