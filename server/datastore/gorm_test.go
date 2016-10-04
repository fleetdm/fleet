package datastore

import (
	"testing"

	"github.com/jinzhu/gorm"
	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGorm(t *testing.T) kolide.Datastore {
	db, err := gorm.Open("sqlite3", ":memory:")
	require.Nil(t, err)

	ds := gormDB{DB: db, Driver: "sqlite3"}

	err = ds.Migrate()
	assert.Nil(t, err)
	return ds
}

func teardownGorm(t *testing.T, ds kolide.Datastore) {
	err := ds.Drop()
	assert.Nil(t, err)
}

func TestGorm(t *testing.T) {
	for _, f := range testFunctions {
		t.Run(functionName(f), func(t *testing.T) {
			ds := setupGorm(t)
			defer teardownGorm(t, ds)
			f(t, ds)
		})
	}
}
