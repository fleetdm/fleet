package mysql

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/testutils"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActivity(t *testing.T) {
	tdb := testutils.SetupTestDB(t, "activity_new")
	ds := NewDatastore(tdb.Conns(), tdb.Logger)
	env := &testEnv{TestDB: tdb, ds: ds}

	cases := []struct {
		name string
		fn   func(t *testing.T, env *testEnv)
	}{
		{"WebhookContextKeyRequired", testNewActivityWebhookContextKeyRequired},
		{"BasicWithUser", testNewActivityBasicWithUser},
		{"NilUser", testNewActivityNilUser},
		{"AutomationActivity", testNewActivityAutomation},
		{"HostAssociation", testNewActivityHostAssociation},
		{"HostOnly", testNewActivityHostOnly},
		{"DeletedUser", testNewActivityDeletedUser},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer env.TruncateTables(t)
			c.fn(t, env)
		})
	}
}

// dummyActivity is a minimal ActivityDetails implementation for testing.
type dummyActivity struct {
	name    string
	details map[string]any
}

func (d dummyActivity) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.details)
}

func (d dummyActivity) ActivityName() string {
	return d.name
}

// automatableActivity is a test activity that satisfies types.AutomatableActivity.
type automatableActivity struct {
	dummyActivity
}

func (a automatableActivity) WasFromAutomation() bool {
	return true
}

// hostActivity is a test activity that satisfies types.ActivityHosts.
type hostActivity struct {
	dummyActivity
	hostIDs []uint
}

func (h hostActivity) HostIDs() []uint {
	return h.hostIDs
}

// hostOnlyActivity is a test activity that satisfies types.ActivityHostOnly.
type hostOnlyActivity struct {
	dummyActivity
}

func (h hostOnlyActivity) HostOnly() bool {
	return true
}

// webhookCtx returns a context with the webhook key set, as required by NewActivity.
func webhookCtx(t *testing.T) context.Context {
	return context.WithValue(t.Context(), types.ActivityWebhookContextKey, true)
}

func testNewActivityWebhookContextKeyRequired(t *testing.T, env *testEnv) {
	ctx := t.Context()
	userID := env.InsertUser(t, "test", "test@example.com")
	user := &api.User{ID: userID, Name: "test", Email: "test@example.com"}
	activity := dummyActivity{name: "test", details: map[string]any{"key": "val"}}
	detailsJSON, err := json.Marshal(activity)
	require.NoError(t, err)

	// No webhook context key set; should fail
	assert.Error(t, env.ds.NewActivity(ctx, user, activity, detailsJSON, time.Now()))

	// Wrong context value type; should fail
	badCtx := context.WithValue(ctx, types.ActivityWebhookContextKey, "wrong")
	assert.Error(t, env.ds.NewActivity(badCtx, user, activity, detailsJSON, time.Now()))

	// Correct context key; should succeed
	assert.NoError(t, env.ds.NewActivity(webhookCtx(t), user, activity, detailsJSON, time.Now()))
}

func testNewActivityBasicWithUser(t *testing.T, env *testEnv) {
	ctx := webhookCtx(t)
	userID := env.InsertUser(t, "fullname", "email@example.com")
	user := &api.User{ID: userID, Name: "fullname", Email: "email@example.com"}

	details := map[string]any{"detail": 1, "sometext": "aaa"}
	detailsJSON, err := json.Marshal(details)
	require.NoError(t, err)

	require.NoError(t, env.ds.NewActivity(ctx, user, dummyActivity{name: "test_one", details: details}, detailsJSON, time.Now()))
	require.NoError(t, env.ds.NewActivity(ctx, user, dummyActivity{name: "test_two", details: map[string]any{"detail": 2}}, mustJSON(t, map[string]any{"detail": 2}), time.Now()))

	// Verify via listing
	activities, _, err := env.ds.ListActivities(t.Context(), listOpts(withPerPage(1)))
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Equal(t, "fullname", *activities[0].ActorFullName)
	assert.Equal(t, "test_one", activities[0].Type)

	// Second page
	activities, _, err = env.ds.ListActivities(t.Context(), listOpts(withPerPage(1), withPage(1)))
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Equal(t, "test_two", activities[0].Type)

	// All results
	activities, _, err = env.ds.ListActivities(t.Context(), listOpts())
	require.NoError(t, err)
	assert.Len(t, activities, 2)
}

func testNewActivityNilUser(t *testing.T, env *testEnv) {
	ctx := webhookCtx(t)
	details := map[string]any{"detail": 1}
	detailsJSON := mustJSON(t, details)

	require.NoError(t, env.ds.NewActivity(ctx, nil, dummyActivity{name: "system_task", details: details}, detailsJSON, time.Now()))

	activities, _, err := env.ds.ListActivities(t.Context(), listOpts())
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Nil(t, activities[0].ActorID)
	assert.Nil(t, activities[0].ActorFullName)
	assert.Equal(t, "system_task", activities[0].Type)
}

func testNewActivityAutomation(t *testing.T, env *testEnv) {
	ctx := webhookCtx(t)
	activity := automatableActivity{
		dummyActivity: dummyActivity{name: "auto_task", details: map[string]any{"automated": true}},
	}
	detailsJSON := mustJSON(t, activity.details)

	require.NoError(t, env.ds.NewActivity(ctx, nil, activity, detailsJSON, time.Now()))

	activities, _, err := env.ds.ListActivities(t.Context(), listOpts())
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Nil(t, activities[0].ActorID)
	require.NotNil(t, activities[0].ActorFullName)
	assert.Equal(t, types.ActivityAutomationAuthor, *activities[0].ActorFullName)
	assert.True(t, activities[0].FleetInitiated)
}

func testNewActivityHostAssociation(t *testing.T, env *testEnv) {
	ctx := webhookCtx(t)
	userID := env.InsertUser(t, "testuser", "test@example.com")
	user := &api.User{ID: userID, Name: "testuser", Email: "test@example.com"}
	hostID := env.InsertHost(t, "h1.local", nil)

	activity := hostActivity{
		dummyActivity: dummyActivity{name: "ran_script", details: map[string]any{"host_id": float64(hostID)}},
		hostIDs:       []uint{hostID},
	}
	detailsJSON := mustJSON(t, activity.details)

	require.NoError(t, env.ds.NewActivity(ctx, user, activity, detailsJSON, time.Now()))

	// Verify the activity is linked to the host via host_activities
	acts, _, err := env.ds.ListHostPastActivities(t.Context(), hostID, listOpts())
	require.NoError(t, err)
	require.Len(t, acts, 1)
	assert.Equal(t, "ran_script", acts[0].Type)
	require.NotNil(t, acts[0].ActorFullName)
	assert.Equal(t, "testuser", *acts[0].ActorFullName)
}

func testNewActivityHostOnly(t *testing.T, env *testEnv) {
	ctx := webhookCtx(t)
	userID := env.InsertUser(t, "testuser", "test@example.com")
	user := &api.User{ID: userID, Name: "testuser", Email: "test@example.com"}

	// Create a regular activity and a host-only activity
	regularDetails := mustJSON(t, map[string]any{"regular": true})
	require.NoError(t, env.ds.NewActivity(ctx, user, dummyActivity{name: "regular", details: map[string]any{"regular": true}}, regularDetails, time.Now()))

	hostOnlyDetails := mustJSON(t, map[string]any{"host_only": true})
	require.NoError(t, env.ds.NewActivity(ctx, user, hostOnlyActivity{
		dummyActivity: dummyActivity{name: "host_scoped", details: map[string]any{"host_only": true}},
	}, hostOnlyDetails, time.Now()))

	// ListActivities excludes host-only activities
	activities, _, err := env.ds.ListActivities(t.Context(), listOpts())
	require.NoError(t, err)
	require.Len(t, activities, 1)
	assert.Equal(t, "regular", activities[0].Type)
}

func testNewActivityDeletedUser(t *testing.T, env *testEnv) {
	ctx := webhookCtx(t)
	// User with Deleted=true should have their name/email preserved but user_id set to NULL
	user := &api.User{ID: 42, Name: "deleted_user", Email: "deleted@example.com", Deleted: true}
	details := mustJSON(t, map[string]any{"detail": 1})

	require.NoError(t, env.ds.NewActivity(ctx, user, dummyActivity{name: "post_delete", details: map[string]any{"detail": 1}}, details, time.Now()))

	activities, _, err := env.ds.ListActivities(t.Context(), listOpts())
	require.NoError(t, err)
	require.Len(t, activities, 1)
	// user_id should be NULL (deleted user), but name is preserved
	assert.Nil(t, activities[0].ActorID)
	require.NotNil(t, activities[0].ActorFullName)
	assert.Equal(t, "deleted_user", *activities[0].ActorFullName)
}

// mustJSON marshals v and fails the test on error.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return b
}
