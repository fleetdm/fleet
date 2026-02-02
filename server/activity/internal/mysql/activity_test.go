package mysql

import (
	"fmt"
	"testing"
	"time"

	activityapi "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEnv holds test dependencies.
type testEnv struct {
	*testutils.TestDB
	ds *Datastore
}

func TestListActivities(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "activity_mysql")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	env := &testEnv{TestDB: tdb, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
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
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

func testListActivitiesBasic(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")

	// Create user activities and a system activity (nil user)
	for i := range 3 {
		env.InsertActivity(t, ptr.Uint(userID), fmt.Sprintf("test_activity_%d", i), map[string]any{"detail": i})
	}
	env.InsertActivity(t, nil, "system_activity", map[string]any{})

	activities, meta, err := env.ds.ListActivities(ctx, listOpts(withMetadata()))
	require.NoError(t, err)
	assert.Len(t, activities, 4)
	assert.NotNil(t, meta)

	// Verify user activities have actor info
	for _, a := range activities {
		assert.NotZero(t, a.ID)
		assert.NotEmpty(t, a.Type)
		assert.NotNil(t, a.Details)

		if a.Type == "system_activity" {
			// System activity has no actor
			assert.Nil(t, a.ActorID)
			assert.Nil(t, a.ActorFullName)
		} else {
			require.NotNil(t, a.ActorID)
			assert.Equal(t, userID, *a.ActorID)
		}
	}
}

func testListActivitiesStreamed(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")

	var activityIDs []uint
	for i := range 3 {
		id := env.InsertActivity(t, ptr.Uint(userID), "test_activity", map[string]any{"detail": i})
		activityIDs = append(activityIDs, id)
	}

	// Mark first activity as streamed
	_, err := env.DB.ExecContext(ctx, "UPDATE activities SET streamed = true WHERE id = ?", activityIDs[0])
	require.NoError(t, err)

	cases := []struct {
		name        string
		streamed    *bool
		expectedIDs []uint
	}{
		{"all", nil, activityIDs},
		{"non-streamed only", ptr.Bool(false), activityIDs[1:]},
		{"streamed only", ptr.Bool(true), activityIDs[:1]},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activities, _, err := env.ds.ListActivities(ctx, listOpts(withStreamed(tc.streamed)))
			require.NoError(t, err)
			gotIDs := make([]uint, len(activities))
			for i, a := range activities {
				gotIDs[i] = a.ID
			}
			assert.ElementsMatch(t, tc.expectedIDs, gotIDs)
		})
	}
}

func testListActivitiesPaginationMetadata(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")

	for i := range 3 {
		env.InsertActivity(t, ptr.Uint(userID), fmt.Sprintf("test_%d", i), map[string]any{})
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
			activities, meta, err := env.ds.ListActivities(ctx, listOpts(withPerPage(tc.perPage), withPage(tc.page), withMetadata()))
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
			require.NotNil(t, meta)
			assert.Equal(t, tc.wantNext, meta.HasNextResults)
			assert.Equal(t, tc.wantPrev, meta.HasPreviousResults)
		})
	}
}

func testListActivitiesActivityTypeFilter(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")

	env.InsertActivity(t, ptr.Uint(userID), "edited_script", map[string]any{})
	env.InsertActivity(t, ptr.Uint(userID), "edited_script", map[string]any{})
	env.InsertActivity(t, ptr.Uint(userID), "mdm_enrolled", map[string]any{})

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
			activities, _, err := env.ds.ListActivities(ctx, listOpts(withActivityType(tc.activityType)))
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
			for _, a := range activities {
				assert.Equal(t, tc.activityType, a.Type)
			}
		})
	}
}

func testListActivitiesDateRangeFilter(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")
	now := time.Now().UTC().Truncate(time.Second)

	// Only create activities in the past/present (activities can't have future creation dates)
	dates := []time.Time{
		now.Add(-48 * time.Hour),
		now.Add(-24 * time.Hour),
		now,
	}
	for _, dt := range dates {
		env.InsertActivityWithTime(t, ptr.Uint(userID), "test_activity", map[string]any{}, dt)
	}

	cases := []struct {
		name      string
		start     string
		end       string
		wantCount int
	}{
		{"no filter", "", "", 3},
		{"start only", now.Add(-72 * time.Hour).Format(time.RFC3339), "", 3},
		{"start and end", now.Add(-72 * time.Hour).Format(time.RFC3339), now.Add(-12 * time.Hour).Format(time.RFC3339), 2},
		{"end only", "", now.Add(-30 * time.Hour).Format(time.RFC3339), 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activities, _, err := env.ds.ListActivities(ctx, listOpts(withDateRange(tc.start, tc.end)))
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
		})
	}
}

func testListActivitiesMatchQuery(t *testing.T, env *testEnv) {
	ctx := t.Context()

	johnUserID := env.InsertUser(t, "john_doe", "john@example.com")
	janeUserID := env.InsertUser(t, "jane_smith", "jane@example.com")

	env.InsertActivity(t, ptr.Uint(johnUserID), "test_activity", map[string]any{})
	env.InsertActivity(t, ptr.Uint(janeUserID), "test_activity", map[string]any{})

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
			activities, _, err := env.ds.ListActivities(ctx, opts)
			require.NoError(t, err)
			assert.Len(t, activities, tc.wantCount)
		})
	}
}

func testListActivitiesOrdering(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")

	now := time.Now().UTC().Truncate(time.Second)
	env.InsertActivityWithTime(t, ptr.Uint(userID), "activity_oldest", map[string]any{}, now.Add(-2*time.Hour))
	env.InsertActivityWithTime(t, ptr.Uint(userID), "activity_middle", map[string]any{}, now.Add(-1*time.Hour))
	env.InsertActivityWithTime(t, ptr.Uint(userID), "activity_newest", map[string]any{}, now)

	cases := []struct {
		name      string
		orderKey  string
		orderDir  activityapi.OrderDirection
		wantFirst string
		wantLast  string
	}{
		{"created_at desc", "created_at", activityapi.OrderDescending, "activity_newest", "activity_oldest"},
		{"created_at asc", "created_at", activityapi.OrderAscending, "activity_oldest", "activity_newest"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			activities, _, err := env.ds.ListActivities(ctx, listOpts(withOrder(tc.orderKey, tc.orderDir)))
			require.NoError(t, err)
			require.Len(t, activities, 3)
			assert.Equal(t, tc.wantFirst, activities[0].Type)
			assert.Equal(t, tc.wantLast, activities[2].Type)
		})
	}

	// Verify ID ordering works
	activities, _, err := env.ds.ListActivities(ctx, listOpts(withOrder("id", activityapi.OrderAscending)))
	require.NoError(t, err)
	require.Len(t, activities, 3)
	assert.Less(t, activities[0].ID, activities[1].ID)
	assert.Less(t, activities[1].ID, activities[2].ID)
}

func testListActivitiesCursorPagination(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")

	for i := range 5 {
		env.InsertActivity(t, ptr.Uint(userID), fmt.Sprintf("activity_%d", i), map[string]any{})
	}

	// Get first page
	activities, _, err := env.ds.ListActivities(ctx, listOpts(withPerPage(2), withOrder("id", activityapi.OrderAscending)))
	require.NoError(t, err)
	require.Len(t, activities, 2)
	lastID := activities[1].ID

	// Get next page using cursor
	activities, _, err = env.ds.ListActivities(ctx, listOpts(withPerPage(2), withOrder("id", activityapi.OrderAscending), withAfter(fmt.Sprintf("%d", lastID))))
	require.NoError(t, err)
	require.Len(t, activities, 2)
	for _, a := range activities {
		assert.Greater(t, a.ID, lastID)
	}
}

func testListActivitiesHostOnlyExcluded(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "testuser", "test@example.com")

	env.InsertActivity(t, ptr.Uint(userID), "regular_activity", map[string]any{})

	// Create host-only activity directly (should be excluded)
	_, err := env.DB.ExecContext(ctx, `
		INSERT INTO activities (user_id, user_name, user_email, activity_type, details, created_at, host_only, streamed)
		VALUES (?, 'testuser', 'test@example.com', 'host_only_activity', '{}', NOW(), true, false)
	`, userID)
	require.NoError(t, err)

	activities, _, err := env.ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 1)
	assert.Equal(t, "regular_activity", activities[0].Type)
}

// Test helpers for building ListOptions

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
