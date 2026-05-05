// Package testutils provides shared test utilities for the chart bounded context.
package testutils

import (
	"log/slog"
	"testing"
	"time"

	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestDB holds the database connection for tests.
type TestDB struct {
	DB     *sqlx.DB
	Logger *slog.Logger
}

// SetupTestDB creates a test database with the Fleet schema loaded. Tests are
// skipped automatically when MYSQL_TEST is not set.
func SetupTestDB(t *testing.T, testNamePrefix string) *TestDB {
	t.Helper()

	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: testNamePrefix + "_" + t.Name(),
	})

	mysql_testing_utils.LoadDefaultSchema(t, testName, opts)
	config := mysql_testing_utils.MysqlTestConfig(testName)
	db, err := common_mysql.NewDB(config, &common_mysql.DBOptions{}, "")
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })

	return &TestDB{
		DB:     db,
		Logger: slog.New(slog.DiscardHandler),
	}
}

// Conns returns DBConnections for creating a datastore.
func (tdb *TestDB) Conns() *common_mysql.DBConnections {
	return &common_mysql.DBConnections{Primary: tdb.DB, Replica: tdb.DB}
}

// TruncateTables clears the tables used by the chart bounded context.
func (tdb *TestDB) TruncateTables(t *testing.T) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, tdb.DB, tdb.Logger, nil, "host_scd_data")
}

// InsertSCDRow inserts a single host_scd_data row for tests. host_bitmap is
// stored as an empty blob since cleanup tests don't care about its contents.
func (tdb *TestDB) InsertSCDRow(t *testing.T, dataset, entityID string, validFrom, validTo time.Time) {
	t.Helper()
	ctx := t.Context()

	_, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from, valid_to)
		VALUES (?, ?, ?, ?, ?)
	`, dataset, entityID, []byte{}, validFrom, validTo)
	require.NoError(t, err)
}

// InsertSCDRowWithBitmap inserts a host_scd_data row with a caller-supplied
// host_bitmap and returns the auto-assigned id.
func (tdb *TestDB) InsertSCDRowWithBitmap(t *testing.T, dataset, entityID string, bitmap []byte, validFrom, validTo time.Time) uint {
	t.Helper()
	ctx := t.Context()

	res, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, valid_from, valid_to)
		VALUES (?, ?, ?, ?, ?)
	`, dataset, entityID, bitmap, validFrom, validTo)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	require.GreaterOrEqual(t, id, int64(0), "AUTO_INCREMENT should never produce a negative id")
	return uint(id) //nolint:gosec // G115: id is a positive AUTO_INCREMENT primary key
}

// SCDBitmap returns the host_bitmap column for the given row id.
func (tdb *TestDB) SCDBitmap(t *testing.T, id uint) []byte {
	t.Helper()
	ctx := t.Context()

	var b []byte
	err := tdb.DB.GetContext(ctx, &b, `SELECT host_bitmap FROM host_scd_data WHERE id = ?`, id)
	require.NoError(t, err)
	return b
}

// CountSCDRows returns the total number of rows in host_scd_data.
func (tdb *TestDB) CountSCDRows(t *testing.T) int {
	t.Helper()
	ctx := t.Context()

	var n int
	err := tdb.DB.GetContext(ctx, &n, `SELECT COUNT(*) FROM host_scd_data`)
	require.NoError(t, err)
	return n
}
