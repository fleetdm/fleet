// Package tables holds fleet table migrations.
//
// Migrations can be tested with tests following the following format:
//
//	$ cat 20220208144831_AddSoftwareReleaseArchVendorColumns_test.go
//
//	[...]
//	func TestUp_20220208144831(t *testing.T) {
//		db := newDBConnForTests(t)
//
//		// Apply all migrations up to 20220208144831 (name of test), not included.
//		applyUpToPrev(t, db)
//
//		// insert testing data, etc.
//
//		// The following will apply migration 20220208144831.
//		applyNext(t, db)
//
//		// insert testing data, verify migration.
//	}
package tables

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TODO(lucas): I'm copy pasting some of the mysql functionality methods here
// otherwise we have import cycle errors.
//
// We need to decouple the server/datastore/mysql package,
// it contains both the implementation of the fleet.Datastore and
// MySQL functionality, and MySQL test functionality.
const (
	testUsername = "root"
	testPassword = "toor"
	testAddress  = "localhost:3307"
)

func newDBConnForTests(t *testing.T) *sqlx.DB {
	db, err := sqlx.Open(
		"mysql",
		fmt.Sprintf("%s:%s@tcp(%s)/?multiStatements=true", testUsername, testPassword, testAddress),
	)
	require.NoError(t, err)
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s; CREATE DATABASE %s; USE %s;", t.Name(), t.Name(), t.Name()))
	require.NoError(t, err)
	return db
}

func getMigrationVersion(t *testing.T) int64 {
	v, err := strconv.Atoi(strings.TrimPrefix(t.Name(), "TestUp_"))
	require.NoError(t, err)
	return int64(v)
}

// gooseNoDir is the value to not parse local files and instead use
// the migrations that were added manually via Add().
const gooseNoDir = ""

func applyUpToPrev(t *testing.T, db *sqlx.DB) {
	v := getMigrationVersion(t)
	for {
		current, err := MigrationClient.GetDBVersion(db.DB)
		require.NoError(t, err)
		next, err := MigrationClient.Migrations.Next(current)
		require.NoError(t, err)
		if next.Version == v {
			return
		}
		applyNext(t, db)
	}
}

func applyNext(t *testing.T, db *sqlx.DB) {
	err := MigrationClient.UpByOne(db.DB, gooseNoDir)
	require.NoError(t, err)
}
