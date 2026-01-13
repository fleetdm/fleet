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
		{"Ordering", testListActivitiesOrdering},
		{"CursorPagination", testListActivitiesCursorPagination},
		{"HostOnlyExcluded", testListActivitiesHostOnlyExcluded},
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

	for i := 1; i <= 3; i++ {
		insertTestActivity(t, ds, userID, fmt.Sprintf("test_activity_%d", i), map[string]any{"detail": i})
	}

	activities, meta, err := ds.ListActivities(ctx, listOpts(withMetadata()))
	require.NoError(t, err)
	assert.Len(t, activities, 3)
	assert.NotNil(t, meta)

	for _, a := range activities {
		assert.NotZero(t, a.ID)
		assert.NotEmpty(t, a.Type)
		require.NotNil(t, a.ActorID)
		assert.Equal(t, userID, *a.ActorID)
		assert.NotNil(t, a.Details)
	}
}

func testListActivitiesStreamed(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	var activityIDs []uint
	for i := 1; i <= 3; i++ {
		id := insertTestActivity(t, ds, userID, "test_activity", map[string]any{"detail": i})
		activityIDs = append(activityIDs, id)
	}

	// Mark first activity as streamed
	_, err := ds.primary.ExecContext(ctx, "UPDATE activities SET streamed = true WHERE id = ?", activityIDs[0])
	require.NoError(t, err)

	cases := []struct {
		name            string
		streamed        *bool
		expectedCount   int
		expectedFirstID uint
	}{
		{"all", nil, 3, 0},
		{"non-streamed only", ptr.Bool(false), 2, 0},
		{"streamed only", ptr.Bool(true), 1, activityIDs[0]},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activities, _, err := ds.ListActivities(ctx, listOpts(withStreamed(tc.streamed)))
			require.NoError(t, err)
			assert.Len(t, activities, tc.expectedCount)
			if tc.expectedFirstID != 0 {
				assert.Equal(t, tc.expectedFirstID, activities[0].ID)
			}
		})
	}
}

func testListActivitiesPaginationMetadata(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	for i := range 3 {
		insertTestActivity(t, ds, userID, fmt.Sprintf("test_%d", i), map[string]any{})
	}

	cases := []struct {
		name      string
		perPage   uint
		page      uint
		wantCount int
		wantNext  bool
		wantPrev  bool
	}{
		{"first page with more", 2, 0, 2, true, false},
		{"second page partial", 2, 1, 1, false, true},
		{"all results", 100, 0, 3, false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activities, meta, err := ds.ListActivities(ctx, listOpts(withPerPage(tc.perPage), withPage(tc.page), withMetadata()))
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
			require.NotNil(t, meta)
			assert.Equal(t, tc.wantNext, meta.HasNextResults)
			assert.Equal(t, tc.wantPrev, meta.HasPreviousResults)
		})
	}
}

func testListActivitiesActivityTypeFilter(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	insertTestActivity(t, ds, userID, "edited_script", map[string]any{})
	insertTestActivity(t, ds, userID, "edited_script", map[string]any{})
	insertTestActivity(t, ds, userID, "mdm_enrolled", map[string]any{})

	cases := []struct {
		activityType string
		wantCount    int
	}{
		{"edited_script", 2},
		{"mdm_enrolled", 1},
		{"non_existent", 0},
	}

	for _, tc := range cases {
		t.Run(tc.activityType, func(t *testing.T) {
			activities, _, err := ds.ListActivities(ctx, listOpts(withActivityType(tc.activityType)))
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
			for _, a := range activities {
				assert.Equal(t, tc.activityType, a.Type)
			}
		})
	}
}

func testListActivitiesDateRangeFilter(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")
	now := time.Now().UTC().Truncate(time.Second)

	// Only create activities in the past/present (activities can't have future creation dates)
	dates := []time.Time{
		now.Add(-48 * time.Hour),
		now.Add(-24 * time.Hour),
		now,
	}
	for _, dt := range dates {
		insertTestActivityWithTime(t, ds, userID, "test_activity", map[string]any{}, dt)
	}

	cases := []struct {
		name      string
		start     string
		end       string
		wantCount int
	}{
		{"start only", now.Add(-72 * time.Hour).Format(time.RFC3339), "", 3},
		{"start and end", now.Add(-72 * time.Hour).Format(time.RFC3339), now.Add(-12 * time.Hour).Format(time.RFC3339), 2},
		{"end only", "", now.Add(-30 * time.Hour).Format(time.RFC3339), 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activities, _, err := ds.ListActivities(ctx, listOpts(withDateRange(tc.start, tc.end)))
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
		})
	}
}

func testListActivitiesMatchQuery(t *testing.T, ds *Datastore) {
	ctx := t.Context()

	johnUserID := insertTestUser(t, ds, "john_doe", "john@example.com")
	janeUserID := insertTestUser(t, ds, "jane_smith", "jane@example.com")

	insertTestActivity(t, ds, johnUserID, "test_activity", map[string]any{})
	insertTestActivity(t, ds, janeUserID, "test_activity", map[string]any{})

	cases := []struct {
		name            string
		query           string
		matchingUserIDs []uint
		wantCount       int
	}{
		{"by username prefix", "john", nil, 1},
		{"by email prefix", "jane@", nil, 1},
		{"no match", "nomatch", nil, 0},
		{"via matching user IDs", "nomatch", []uint{johnUserID}, 1},
		{"via multiple matching user IDs", "nomatch", []uint{johnUserID, janeUserID}, 2},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := listOpts(withMatchQuery(tc.query))
			opts.MatchingUserIDs = tc.matchingUserIDs
			activities, _, err := ds.ListActivities(ctx, opts)
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
		})
	}
}

func testListActivitiesOrdering(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	now := time.Now().UTC().Truncate(time.Second)
	insertTestActivityWithTime(t, ds, userID, "activity_oldest", map[string]any{}, now.Add(-2*time.Hour))
	insertTestActivityWithTime(t, ds, userID, "activity_middle", map[string]any{}, now.Add(-1*time.Hour))
	insertTestActivityWithTime(t, ds, userID, "activity_newest", map[string]any{}, now)

	cases := []struct {
		name      string
		orderKey  string
		orderDir  activityapi.OrderDirection
		wantFirst string
		wantLast  string
	}{
		{"created_at desc", "created_at", activityapi.OrderDesc, "activity_newest", "activity_oldest"},
		{"created_at asc", "created_at", activityapi.OrderAsc, "activity_oldest", "activity_newest"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activities, _, err := ds.ListActivities(ctx, listOpts(withOrder(tc.orderKey, tc.orderDir)))
			require.NoError(t, err)
			require.Len(t, activities, 3)
			assert.Equal(t, tc.wantFirst, activities[0].Type)
			assert.Equal(t, tc.wantLast, activities[2].Type)
		})
	}

	// Verify ID ordering works
	activities, _, err := ds.ListActivities(ctx, listOpts(withOrder("id", activityapi.OrderAsc)))
	require.NoError(t, err)
	require.Len(t, activities, 3)
	assert.Less(t, activities[0].ID, activities[1].ID)
	assert.Less(t, activities[1].ID, activities[2].ID)
}

func testListActivitiesCursorPagination(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	for i := 1; i <= 5; i++ {
		insertTestActivity(t, ds, userID, fmt.Sprintf("activity_%d", i), map[string]any{})
	}

	// Get first page
	activities, _, err := ds.ListActivities(ctx, listOpts(withPerPage(2), withOrder("id", activityapi.OrderAsc)))
	require.NoError(t, err)
	require.Len(t, activities, 2)
	lastID := activities[1].ID

	// Get next page using cursor
	activities, _, err = ds.ListActivities(ctx, listOpts(withPerPage(2), withOrder("id", activityapi.OrderAsc), withAfter(fmt.Sprintf("%d", lastID))))
	require.NoError(t, err)
	require.Len(t, activities, 2)
	for _, a := range activities {
		assert.Greater(t, a.ID, lastID)
	}
}

func testListActivitiesHostOnlyExcluded(t *testing.T, ds *Datastore) {
	ctx := t.Context()
	userID := insertTestUser(t, ds, "testuser", "test@example.com")

	insertTestActivity(t, ds, userID, "regular_activity", map[string]any{})

	// Create host-only activity directly (should be excluded)
	_, err := ds.primary.ExecContext(ctx, `
		INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
		VALUES (?, 'testuser', 'test@example.com', 'host_only_activity', '{}', NOW(), true, false)
	`, userID)
	require.NoError(t, err)

	activities, _, err := ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "regular_activity", activities[0].Type)
}

// Test helpers

// listOptsFunc is a functional option for building ListOptions.
type listOptsFunc func(*types.ListOptions)

func listOpts(opts ...listOptsFunc) types.ListOptions {
	o := types.ListOptions{ListOptions: activityapi.ListOptions{PerPage: 100}}
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

func withPerPage(n uint) listOptsFunc {
	return func(o *types.ListOptions) { o.PerPage = n }
}

func withPage(n uint) listOptsFunc {
	return func(o *types.ListOptions) { o.Page = n }
}

func withMetadata() listOptsFunc {
	return func(o *types.ListOptions) { o.IncludeMetadata = true }
}

func withStreamed(s *bool) listOptsFunc {
	return func(o *types.ListOptions) { o.Streamed = s }
}

func withActivityType(t string) listOptsFunc {
	return func(o *types.ListOptions) { o.ActivityType = t }
}

func withMatchQuery(q string) listOptsFunc {
	return func(o *types.ListOptions) { o.MatchQuery = q }
}

func withDateRange(start, end string) listOptsFunc {
	return func(o *types.ListOptions) {
		o.StartCreatedAt = start
		o.EndCreatedAt = end
	}
}

func withOrder(key string, dir activityapi.OrderDirection) listOptsFunc {
	return func(o *types.ListOptions) {
		o.OrderKey = key
		o.OrderDirection = dir
	}
}

func withAfter(cursor string) listOptsFunc {
	return func(o *types.ListOptions) { o.After = cursor }
}

func setupTestDatastore(t *testing.T) *Datastore {
	t.Helper()

	testName, opts := mysql_testing_utils.ProcessOptions(t, &mysql_testing_utils.DatastoreTestOptions{
		UniqueTestName: "activity_mysql_" + t.Name(),
	})

	_, thisFile, _, _ := runtime.Caller(0)
	schemaPath := filepath.Join(filepath.Dir(thisFile), "../../../datastore/mysql/schema.sql")
	mysql_testing_utils.LoadSchema(t, testName, opts, schemaPath)

	config := mysql_testing_utils.MysqlTestConfig(testName)
	db, err := common_mysql.NewDB(config, &common_mysql.DBOptions{SqlMode: common_mysql.TestSQLMode}, "")
	require.NoError(t, err)

	t.Cleanup(func() { db.Close() })

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
	result, err := ds.primary.ExecContext(context.Background(), `
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
	ctx := t.Context()

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

	result, err := ds.primary.ExecContext(ctx, `
		INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
		VALUES (?, ?, ?, ?, ?, ?, false, false)
	`, userID, userName, userEmail, activityType, detailsJSON, createdAt)
	require.NoError(t, err)

	id, err := result.LastInsertId()
	require.NoError(t, err)
	return uint(id)
}
