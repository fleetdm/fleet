package mysql

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/testutils"
	activityapi "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	mysql_testing_utils "github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListActivities(t *testing.T) {
	ds := setupTestDatastore(t)

	cases := []struct {
		name string
		fn   func(t *testing.T, ds *Datastore)
	}{
		{"Basic", testListActivitiesBasic},
		{"Streamed", testListActivitiesStreamed},
		{"PaginationMetadata", testListActivitiesPaginationMetadata},
		{"ActivityTypeFilter", testListActivitiesActivityTypeFilter},
		{"DateRangeFilter", testListActivitiesDateRangeFilter},
		{"MatchQuery", testListActivitiesMatchQuery},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer truncateTables(t, ds)
			c.fn(t, ds)
		})
	}
}

func testListActivitiesBasic(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	// Create 3 activities
	for i := 1; i <= 3; i++ {
		insertTestActivity(t, ds, userID, fmt.Sprintf("test_activity_%d", i), map[string]any{"detail": i})
	}

	activities, meta, err := ds.ListActivities(ctx, types.ListOptions{
		ListOptions:     activityapi.ListOptions{PerPage: 100},
		IncludeMetadata: true,
	})
	require.NoError(t, err)
	assert.Len(t, activities, 3)
	assert.NotNil(t, meta)

	// Verify activities have expected fields
	for _, a := range activities {
		assert.NotZero(t, a.ID)
		assert.NotEmpty(t, a.Type)
		assert.NotNil(t, a.ActorID)
		assert.Equal(t, userID, *a.ActorID)
		assert.NotNil(t, a.Details)
	}
}

func testListActivitiesStreamed(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	// Create 3 activities
	var activityIDs []uint
	for i := 1; i <= 3; i++ {
		id := insertTestActivity(t, ds, userID, "test_activity", map[string]any{"detail": i})
		activityIDs = append(activityIDs, id)
	}

	// Mark first activity as streamed
	_, err := ds.primary.ExecContext(ctx, "UPDATE activities SET streamed = true WHERE id = ?", activityIDs[0])
	require.NoError(t, err)

	// List all activities
	activities, _, err := ds.ListActivities(ctx, types.ListOptions{ListOptions: activityapi.ListOptions{PerPage: 100}})
	require.NoError(t, err)
	assert.Len(t, activities, 3)

	// List non-streamed activities
	nonStreamed, _, err := ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, Streamed: ptr.Bool(false)},
	})
	require.NoError(t, err)
	assert.Len(t, nonStreamed, 2)

	// List streamed activities
	streamed, _, err := ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, Streamed: ptr.Bool(true)},
	})
	require.NoError(t, err)
	assert.Len(t, streamed, 1)
	assert.Equal(t, activityIDs[0], streamed[0].ID)
}

func testListActivitiesPaginationMetadata(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	// Create 3 activities
	for i := range 3 {
		insertTestActivity(t, ds, userID, fmt.Sprintf("test_%d", i), map[string]any{})
	}

	// Test HasNextResults
	activities, meta, err := ds.ListActivities(ctx, types.ListOptions{ListOptions: activityapi.ListOptions{PerPage: 2}, IncludeMetadata: true})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	require.NotNil(t, meta)
	assert.True(t, meta.HasNextResults)
	assert.False(t, meta.HasPreviousResults)

	// Test HasPreviousResults
	activities, meta, err = ds.ListActivities(ctx, types.ListOptions{ListOptions: activityapi.ListOptions{PerPage: 2, Page: 1}, IncludeMetadata: true})
	require.NoError(t, err)
	assert.Len(t, activities, 1)
	require.NotNil(t, meta)
	assert.False(t, meta.HasNextResults)
	assert.True(t, meta.HasPreviousResults)

	// Test no extra results
	activities, meta, err = ds.ListActivities(ctx, types.ListOptions{ListOptions: activityapi.ListOptions{PerPage: 100}, IncludeMetadata: true})
	require.NoError(t, err)
	assert.Len(t, activities, 3)
	require.NotNil(t, meta)
	assert.False(t, meta.HasNextResults)
	assert.False(t, meta.HasPreviousResults)
}

func testListActivitiesActivityTypeFilter(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	// Create activities with different types
	insertTestActivity(t, ds, userID, "edited_script", map[string]any{})
	insertTestActivity(t, ds, userID, "edited_script", map[string]any{})
	insertTestActivity(t, ds, userID, "mdm_enrolled", map[string]any{})

	// Filter by type - should find 2
	activities, _, err := ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, ActivityType: "edited_script"},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 2)
	for _, a := range activities {
		assert.Equal(t, "edited_script", a.Type)
	}

	// Filter by different type - should find 1
	activities, _, err = ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, ActivityType: "mdm_enrolled"},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 1)

	// Filter by non-existent type - should find 0
	activities, _, err = ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, ActivityType: "non_existent"},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 0)
}

func testListActivitiesDateRangeFilter(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")
	now := time.Now().UTC().Truncate(time.Second)

	// Create activities at different times
	dates := []time.Time{
		now.Add(-48 * time.Hour),
		now.Add(-24 * time.Hour),
		now,
		now.Add(24 * time.Hour),
	}
	for _, dt := range dates {
		insertTestActivityWithTime(t, ds, userID, "test_activity", map[string]any{}, dt)
	}

	// From 36 hours ago to now (should get 2 activities: -24h and now)
	activities, _, err := ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, StartCreatedAt: now.Add(-36 * time.Hour).Format(time.RFC3339)},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 2)

	// From 72 hours ago to 12 hours ago (should get 2 activities: -48h and -24h)
	activities, _, err = ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{
			PerPage:        100,
			StartCreatedAt: now.Add(-72 * time.Hour).Format(time.RFC3339),
			EndCreatedAt:   now.Add(-12 * time.Hour).Format(time.RFC3339),
		},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 2)

	// Only end date (should get 3 activities: -48h, -24h, now)
	activities, _, err = ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, EndCreatedAt: now.Add(1 * time.Hour).Format(time.RFC3339)},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 3)
}

func testListActivitiesMatchQuery(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	// Create users with different names/emails
	userID := insertTestUser(t, ds, "john_doe", "john@example.com")
	userIDTwo := insertTestUser(t, ds, "jane_smith", "jane@example.com")

	// Create activities for each user
	insertTestActivity(t, ds, userID, "test_activity", map[string]any{})
	insertTestActivity(t, ds, userIDTwo, "test_activity", map[string]any{})

	// Search by user_name in activity table (prefix match)
	activities, _, err := ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, MatchQuery: "john"},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 1)

	// Search by user_email in activity table (prefix match)
	activities, _, err = ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, MatchQuery: "jane@"},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 1)

	// Search with no match
	activities, _, err = ds.ListActivities(ctx, types.ListOptions{
		ListOptions: activityapi.ListOptions{PerPage: 100, MatchQuery: "nomatch"},
	})
	require.NoError(t, err)
	assert.Len(t, activities, 0)

	// Search with MatchingUserIDs (simulates user table search done by service layer)
	activities, _, err = ds.ListActivities(ctx, types.ListOptions{
		ListOptions:     activityapi.ListOptions{PerPage: 100, MatchQuery: "nomatch"}, // Won't match activity table
		MatchingUserIDs: []uint{userID},                                               // But matches via user IDs
	})
	require.NoError(t, err)
	assert.Len(t, activities, 1)
}

// Test helpers

func setupTestDatastore(t *testing.T) *Datastore {
	t.Helper()

	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "activity_mysql_" + t.Name(),
	})

	// Load schema
	_, thisFile, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(thisFile), "../../../datastore/mysql/schema.sql")
	mysql_testing_utils.LoadSchema(t, testName, opts, schemaPath)

	// Create DB connection
	config := mysql_testing_utils.MysqlTestConfig(testName)
	db, err := common_mysql.NewDB(config, &common_mysql.DBOptions{}, "")
	require.NoError(t, err)

	t.Cleanup(func() {
		db.Close()
	})

	logger := log.NewLogfmtLogger(&testutils.TestLogWriter{T: t})
	conns := &common_mysql.DBConnections{Primary: db, Replica: db}
	return NewDatastore(conns, logger)
}

func truncateTables(t *testing.T, ds *Datastore) {
	t.Helper()
	mysql_testing_utils.TruncateTables(t, ds.primary, log.NewNopLogger(), nil, "activities", "users")
}

func insertTestUser(t *testing.T, ds *Datastore, name, email string) uint {
	t.Helper()
	ctx := context.Background()

	result, err := ds.primary.ExecContext(ctx, `
		INSERT INTO users (name, email, password, salt, created_at, updated_at)
		VALUES (?, ?, 'password', 'salt', NOW(), NOW())
	`, name, email)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}

func insertTestActivity(t *testing.T, ds *Datastore, userID uint, activityType string, details map[string]any) uint {
	t.Helper()
	return insertTestActivityWithTime(t, ds, userID, activityType, details, time.Now().UTC())
}

func insertTestActivityWithTime(t *testing.T, ds *Datastore, userID uint, activityType string, details map[string]any, createdAt time.Time) uint {
	t.Helper()
	ctx := context.Background()

	detailsJSON, err := json.Marshal(details)
	require.NoError(t, err)

	var userName, userEmail *string
	if userID > 0 {
		var user struct {
			Name  string `db:"name"`
			Email string `db:"email"`
		}
		err = sqlx.GetContext(ctx, ds.primary, &user, "SELECT name, email FROM users WHERE id = ?", userID)
		require.NoError(t, err)
		userName = &user.Name
		userEmail = &user.Email
	}

	var result any
	if userID > 0 {
		result, err = ds.primary.ExecContext(ctx, `
			INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
			VALUES (?, ?, ?, ?, ?, ?, false, false)
		`, userID, userName, userEmail, activityType, detailsJSON, createdAt)
	} else {
		result, err = ds.primary.ExecContext(ctx, `
			INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
			VALUES (NULL, NULL, NULL, ?, ?, ?, false, false)
		`, activityType, detailsJSON, createdAt)
	}
	require.NoError(t, err)

	id, err := result.(interface{ LastInsertId() (int64, error) }).LastInsertId()
	require.NoError(t, err)
	return uint(id)
}
