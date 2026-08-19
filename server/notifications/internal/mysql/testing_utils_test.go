package mysql

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

// testEnv is a datastore wired to a test database, plus enough raw SQL to set
// up rows the datastore has no method for.
type testEnv struct {
	db *sqlx.DB
	ds *Datastore
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "notifications_mysql_" + t.Name(),
	})
	mysql_testing_utils.LoadDefaultSchema(t, testName, opts)

	db, err := common_mysql.NewDB(mysql_testing_utils.MysqlTestConfig(testName), &common_mysql.DBOptions{}, "")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	logger := slog.New(slog.DiscardHandler)
	return &testEnv{
		db: db,
		ds: NewDatastore(&common_mysql.DBConnections{Primary: db, Replica: db}, logger),
	}
}

func (env *testEnv) TruncateTables(t testing.TB) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, env.db, slog.New(slog.DiscardHandler), nil,
		"notifications_end_user", "upcoming_activities", "host_orbit_info", "hosts")
}

func (env *testEnv) InsertHost(t testing.TB, hostname string, platform string) uint {
	t.Helper()
	res, err := env.db.ExecContext(context.Background(), `
		INSERT INTO hosts (hostname, platform, created_at, updated_at) VALUES (?, ?, NOW(), NOW())`,
		hostname, platform)
	require.NoError(t, err)
	id, err := res.LastInsertId()
	require.NoError(t, err)
	return uint(id) //nolint:gosec // dismiss G115
}

// InsertHostOrbitInfo makes a host look fleetd-managed, which the dispatch
// query requires.
func (env *testEnv) InsertHostOrbitInfo(t testing.TB, hostID uint) {
	t.Helper()
	_, err := env.db.ExecContext(context.Background(),
		`INSERT INTO host_orbit_info (host_id, version) VALUES (?, '1.40.0')`, hostID)
	require.NoError(t, err)
}

func (env *testEnv) InsertNotification(t testing.TB, hostID uint, kind string, nextAttemptAt, expiresAt *time.Time) string {
	t.Helper()
	notificationUUID := uuid.NewString()
	_, err := env.db.ExecContext(context.Background(), `
		INSERT INTO notifications_end_user (uuid, host_id, status, kind, payload, next_attempt_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, COALESCE(?, NOW(6) + INTERVAL 24 HOUR))`,
		notificationUUID, hostID, api.EndUserNotificationPending, kind,
		json.RawMessage(`{"title": "hello"}`), nextAttemptAt, expiresAt)
	require.NoError(t, err)
	return notificationUUID
}

func (env *testEnv) DeleteHost(t testing.TB, hostID uint) {
	t.Helper()
	_, err := env.db.ExecContext(context.Background(), `DELETE FROM hosts WHERE id = ?`, hostID)
	require.NoError(t, err)
}
