package mysql

import (
	"fmt"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupExpiredActivities(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "activity_cleanup")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	env := &testEnv{TestDB: tdb, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"NothingToDelete", testCleanupExpiredActivitiesNoop},
		{"DeletesExpiredNonHostActivities", testCleanupExpiredActivitiesBasic},
		{"RespectsMaxCount", testCleanupExpiredActivitiesBatch},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

func testCleanupExpiredActivitiesNoop(t *testing.T, env *testEnv) {
	ctx := t.Context()

	// No activities exist -should be a no-op.
	err := env.ds.CleanupExpiredActivities(ctx, 500, 1)
	require.NoError(t, err)

	// Create a recent activity -should not be deleted.
	userID := env.InsertUser(t, "user", "user@example.com")
	env.InsertActivity(t, ptr.Uint(userID), "recent_activity", map[string]any{})

	err = env.ds.CleanupExpiredActivities(ctx, 500, 1)
	require.NoError(t, err)

	activities, _, err := env.ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 1)
}

func testCleanupExpiredActivitiesBasic(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "user", "user@example.com")
	hostID := env.InsertHost(t, "h1.local", nil)

	expiredTime := time.Now().Add(-48 * time.Hour)
	recentTime := time.Now()

	// Create activities with different states:
	// 1. Expired, no host link → should be deleted
	expiredNoHost := env.InsertActivityWithTime(t, ptr.Uint(userID), "expired_no_host", map[string]any{}, expiredTime)
	// 2. Expired, linked to host → should be preserved
	expiredWithHost := env.InsertActivityWithTime(t, ptr.Uint(userID), "expired_with_host", map[string]any{}, expiredTime)
	env.InsertHostActivity(t, hostID, expiredWithHost)
	// 3. Recent, no host link → should be preserved
	recentNoHost := env.InsertActivityWithTime(t, ptr.Uint(userID), "recent_no_host", map[string]any{}, recentTime)

	err := env.ds.CleanupExpiredActivities(ctx, 500, 1)
	require.NoError(t, err)

	activities, _, err := env.ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	require.Len(t, activities, 2)

	activityIDs := make([]uint, len(activities))
	for i, a := range activities {
		activityIDs[i] = a.ID
	}
	assert.NotContains(t, activityIDs, expiredNoHost, "expired non-host activity should be deleted")
	assert.Contains(t, activityIDs, expiredWithHost, "expired host-linked activity should be preserved")
	assert.Contains(t, activityIDs, recentNoHost, "recent activity should be preserved")

	// Verify host_activities link still exists for the preserved activity.
	var hostActivityCount int
	err = env.DB.GetContext(ctx, &hostActivityCount, "SELECT COUNT(*) FROM host_activities WHERE activity_id = ?", expiredWithHost)
	require.NoError(t, err)
	assert.Equal(t, 1, hostActivityCount)
}

func testCleanupExpiredActivitiesBatch(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "user", "user@example.com")
	expiredTime := time.Now().Add(-48 * time.Hour)

	// Create 10 expired activities (no host links).
	for i := range 10 {
		env.InsertActivityWithTime(t, ptr.Uint(userID), fmt.Sprintf("expired_%d", i), map[string]any{}, expiredTime)
	}

	// Cleanup with maxCount=3 -only 3 should be deleted per call.
	err := env.ds.CleanupExpiredActivities(ctx, 3, 1)
	require.NoError(t, err)

	activities, _, err := env.ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 7, "only 3 of 10 expired activities should be deleted")

	// Run again -another 3 deleted.
	err = env.ds.CleanupExpiredActivities(ctx, 3, 1)
	require.NoError(t, err)

	activities, _, err = env.ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 4)

	// Run again -another 3 deleted.
	err = env.ds.CleanupExpiredActivities(ctx, 3, 1)
	require.NoError(t, err)

	activities, _, err = env.ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 1)

	// Run again -last one deleted.
	err = env.ds.CleanupExpiredActivities(ctx, 3, 1)
	require.NoError(t, err)

	activities, _, err = env.ds.ListActivities(ctx, listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 0)
}
