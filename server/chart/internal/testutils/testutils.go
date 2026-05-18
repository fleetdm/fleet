// Package testutils provides shared test utilities for the chart bounded context.
package testutils

import (
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart"
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
	mysql_testing_utils.TruncateTables(t, tdb.DB, tdb.Logger, nil,
		"host_scd_data", "hosts", "host_seen_times", "nano_enrollments", "teams")
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

// InsertSCDRowWithBlob inserts a host_scd_data row with a caller-supplied
// chart.Blob (bytes + encoding) and returns the auto-assigned id.
func (tdb *TestDB) InsertSCDRowWithBlob(t *testing.T, dataset, entityID string, blob chart.Blob, validFrom, validTo time.Time) uint {
	t.Helper()
	ctx := t.Context()

	res, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO host_scd_data (dataset, entity_id, host_bitmap, encoding_type, valid_from, valid_to)
		VALUES (?, ?, ?, ?, ?, ?)
	`, dataset, entityID, blob.Bytes, blob.Encoding, validFrom, validTo)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	require.GreaterOrEqual(t, id, int64(0), "AUTO_INCREMENT should never produce a negative id")
	return uint(id) //nolint:gosec // G115: id is a positive AUTO_INCREMENT primary key
}

// InsertSCDRowWithHostIDs is a convenience wrapper for tests that just want to
// store a set of host IDs — produces a roaring-encoded row.
func (tdb *TestDB) InsertSCDRowWithHostIDs(t *testing.T, dataset, entityID string, hostIDs []uint, validFrom, validTo time.Time) uint {
	t.Helper()
	return tdb.InsertSCDRowWithBlob(t, dataset, entityID, chart.HostIDsToBlob(hostIDs), validFrom, validTo)
}

// SCDBlob returns the host_bitmap + encoding_type for the given row id.
func (tdb *TestDB) SCDBlob(t *testing.T, id uint) chart.Blob {
	t.Helper()
	ctx := t.Context()

	type row struct {
		HostBitmap   []byte `db:"host_bitmap"`
		EncodingType uint8  `db:"encoding_type"`
	}
	var r row
	err := tdb.DB.GetContext(ctx, &r, `SELECT host_bitmap, encoding_type FROM host_scd_data WHERE id = ?`, id)
	require.NoError(t, err)
	return chart.Blob{Bytes: r.HostBitmap, Encoding: r.EncodingType}
}

// SCDHostIDs returns the decoded host IDs for the given row id.
func (tdb *TestDB) SCDHostIDs(t *testing.T, id uint) []uint {
	t.Helper()
	rb, err := chart.DecodeBitmap(tdb.SCDBlob(t, id))
	require.NoError(t, err)
	return chart.BitmapToHostIDs(rb)
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
