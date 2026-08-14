// Package testutils provides shared test utilities for the notifications
// bounded context.
package testutils

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/notifications/api"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestDB holds the database connection for tests.
type TestDB struct {
	DB     *sqlx.DB
	Logger *slog.Logger
}

// SetupTestDB creates a test database with the Fleet schema loaded.
func SetupTestDB(t testing.TB, testNamePrefix string) *TestDB {
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

// TruncateTables clears the tables used by the notifications bounded context.
func (tdb *TestDB) TruncateTables(t testing.TB) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, tdb.DB, tdb.Logger, nil,
		"end_user_notifications", "upcoming_activities", "host_orbit_info", "hosts")
}

// InsertHost creates a host with the given platform and returns its ID.
func (tdb *TestDB) InsertHost(t testing.TB, hostname string, platform string) uint {
	t.Helper()

	result, err := tdb.DB.ExecContext(context.Background(), `
		INSERT INTO hosts (hostname, platform, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
	`, hostname, platform)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id) //nolint:gosec // dismiss G115
}

// InsertHostOrbitInfo gives a host orbit info, which the dispatch query
// requires to consider it fleetd-managed.
func (tdb *TestDB) InsertHostOrbitInfo(t testing.TB, hostID uint) {
	t.Helper()

	_, err := tdb.DB.ExecContext(context.Background(), `
		INSERT INTO host_orbit_info (host_id, version) VALUES (?, '1.40.0')
	`, hostID)
	require.NoError(t, err)
}

// InsertNotification creates an end user notification and returns its UUID.
func (tdb *TestDB) InsertNotification(t testing.TB, hostID uint, kind string, nextAttemptAt, expiresAt *time.Time) string {
	t.Helper()

	notificationUUID := uuid.NewString()
	_, err := tdb.DB.ExecContext(context.Background(), `
		INSERT INTO end_user_notifications (uuid, host_id, status, kind, payload, next_attempt_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, notificationUUID, hostID, api.EndUserNotificationPending, kind, json.RawMessage(`{"title": "hello"}`), nextAttemptAt, expiresAt)
	require.NoError(t, err)
	return notificationUUID
}

// InsertUpcomingActivity gives a host a queued script activity under
// executionID, so SetEndUserNotificationsDispatched's join to
// upcoming_activities has something to match.
func (tdb *TestDB) InsertUpcomingActivity(t testing.TB, hostID uint, executionID string) {
	t.Helper()

	_, err := tdb.DB.ExecContext(context.Background(), `
		INSERT INTO upcoming_activities (host_id, activity_type, execution_id, payload)
		VALUES (?, 'script', ?, '{}')
	`, hostID, executionID)
	require.NoError(t, err)
}

// DeleteHost deletes a host, for exercising the notifications table's
// foreign key cascade.
func (tdb *TestDB) DeleteHost(t testing.TB, hostID uint) {
	t.Helper()

	_, err := tdb.DB.ExecContext(context.Background(), `DELETE FROM hosts WHERE id = ?`, hostID)
	require.NoError(t, err)
}
