// Package testutils provides shared test utilities for the activity bounded context.
package testutils

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// TestDB holds the database connection for tests.
type TestDB struct {
	DB     *sqlx.DB
	Logger log.Logger
}

// SetupTestDB creates a test database with the Fleet schema loaded.
func SetupTestDB(t *testing.T, testNamePrefix string) *TestDB {
	t.Helper()

	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: testNamePrefix + "_" + t.Name(),
	})

	_, thisFile, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(thisFile), "../../../../server/datastore/mysql/schema.sql")
	mysql_testing_utils.LoadSchema(t, testName, opts, schemaPath)

	config := mysql_testing_utils.MysqlTestConfig(testName)
	db, err := common_mysql.NewDB(config, &common_mysql.DBOptions{}, "")
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })

	return &TestDB{
		DB:     db,
		Logger: log.NewNopLogger(),
	}
}

// Conns returns DBConnections for creating a datastore.
func (tdb *TestDB) Conns() *common_mysql.DBConnections {
	return &common_mysql.DBConnections{Primary: tdb.DB, Replica: tdb.DB}
}

// TruncateTables clears the activities, users, hosts, and host_activities tables.
func (tdb *TestDB) TruncateTables(t *testing.T) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, tdb.DB, tdb.Logger, nil, "host_activities", "activities", "hosts", "users")
}

// InsertUser creates a user in the database and returns the user ID.
func (tdb *TestDB) InsertUser(t *testing.T, name, email string) uint {
	t.Helper()
	ctx := t.Context()

	result, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO users (name, email, password, salt, created_at, updated_at)
		VALUES (?, ?, 'password', 'salt', NOW(), NOW())
	`, name, email)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}

// InsertActivity creates an activity in the database and returns the activity ID.
// Pass nil for userID to create a system activity (no associated user).
func (tdb *TestDB) InsertActivity(t *testing.T, userID *uint, activityType string, details map[string]any) uint {
	t.Helper()
	return tdb.InsertActivityWithTime(t, userID, activityType, details, time.Now().UTC())
}

// InsertActivityWithTime creates an activity with a specific timestamp.
// Pass nil for userID to create a system activity (no associated user).
func (tdb *TestDB) InsertActivityWithTime(t *testing.T, userID *uint, activityType string, details map[string]any, createdAt time.Time) uint {
	t.Helper()
	ctx := t.Context()

	detailsJSON, err := json.Marshal(details)
	require.NoError(t, err)

	var userName *string
	var userEmail string // NOT NULL DEFAULT '' in schema
	if userID != nil {
		var user struct {
			Name  string `db:"name"`
			Email string `db:"email"`
		}
		err = sqlx.GetContext(ctx, tdb.DB, &user, "SELECT name, email FROM users WHERE id = ?", *userID)
		require.NoError(t, err)
		userName = &user.Name
		userEmail = user.Email
	}

	result, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
		VALUES (?, ?, ?, ?, ?, ?, false, false)
	`, userID, userName, userEmail, activityType, detailsJSON, createdAt)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}

// InsertHost creates a host in the database and returns the host ID.
func (tdb *TestDB) InsertHost(t *testing.T, hostname string, teamID *uint) uint {
	t.Helper()
	ctx := t.Context()

	result, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO hosts (hostname, team_id, created_at, updated_at)
		VALUES (?, ?, NOW(), NOW())
	`, hostname, teamID)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}

// InsertHostActivity creates a link between a host and an activity in the host_activities junction table.
func (tdb *TestDB) InsertHostActivity(t *testing.T, hostID, activityID uint) {
	t.Helper()
	ctx := t.Context()

	_, err := tdb.DB.ExecContext(ctx, `
		INSERT INTO host_activities (host_id, activity_id) VALUES (?, ?)
	`, hostID, activityID)
	require.NoError(t, err)
}
