package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/activity"
	"github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/activity/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newActivityMockDatastore captures calls to NewActivity for assertions.
type newActivityMockDatastore struct {
	mockDatastore
	newActivityCalled bool
	lastUser          *api.User
	lastActivity      api.ActivityDetails
	lastDetails       []byte
	lastCreatedAt     time.Time
	lastCtx           context.Context
	newActivityErr    error
}

func (m *newActivityMockDatastore) NewActivity(ctx context.Context, user *api.User, act api.ActivityDetails, details []byte, createdAt time.Time) error {
	m.newActivityCalled = true
	m.lastUser = user
	m.lastActivity = act
	m.lastDetails = details
	m.lastCreatedAt = createdAt
	m.lastCtx = ctx
	return m.newActivityErr
}

// newActivityMockProviders extends mockDataProviders with call tracking for ActivateNextUpcomingActivity.
type newActivityMockProviders struct {
	mockDataProviders
	activateCalled bool
	lastHostID     uint
	lastCmdUUID    string
	activateErr    error
}

func (m *newActivityMockProviders) ActivateNextUpcomingActivity(ctx context.Context, hostID uint, cmdUUID string) error {
	m.activateCalled = true
	m.lastHostID = hostID
	m.lastCmdUUID = cmdUUID
	return m.activateErr
}

// Test activity types

type simpleActivity struct {
	Name string `json:"name"`
}

func (a simpleActivity) ActivityName() string { return "simple_test" }

type aliasedActivity struct {
	TeamID uint `json:"team_id" renameto:"fleet_id"`
}

func (a aliasedActivity) ActivityName() string { return "aliased_test" }

type activatorActivity struct {
	simpleActivity
	hostID  uint
	cmdUUID string
}

func (a activatorActivity) MustActivateNextUpcomingActivity() bool { return true }
func (a activatorActivity) ActivateNextUpcomingActivityArgs() (uint, string) {
	return a.hostID, a.cmdUUID
}

func newTestService(ds types.Datastore, providers activity.DataProviders) *Service {
	noopWebhookSend := func(_ context.Context, _ string, _ any) error { return nil }
	return NewService(&mockAuthorizer{}, ds, providers, noopWebhookSend, slog.New(slog.DiscardHandler))
}

func TestNewActivityStoresWithWebhookContextKey(t *testing.T) {
	t.Parallel()
	ds := &newActivityMockDatastore{}
	providers := &newActivityMockProviders{
		mockDataProviders: mockDataProviders{
			mockUserProvider: &mockUserProvider{},
			mockHostProvider: &mockHostProvider{},
		},
	}
	svc := newTestService(ds, providers)

	user := &api.User{ID: 1, Name: "test", Email: "test@example.com"}
	err := svc.NewActivity(t.Context(), user, simpleActivity{Name: "hello"})
	require.NoError(t, err)

	// Verify store was called
	require.True(t, ds.newActivityCalled)

	// Verify webhook context key was set
	processed, ok := ds.lastCtx.Value(types.ActivityWebhookContextKey).(bool)
	require.True(t, ok, "webhook context key should be set")
	assert.True(t, processed)

	// Verify user was passed through
	require.NotNil(t, ds.lastUser)
	assert.Equal(t, uint(1), ds.lastUser.ID)
	assert.Equal(t, "test", ds.lastUser.Name)
	assert.Equal(t, "test@example.com", ds.lastUser.Email)

	// Verify details were marshaled
	var details map[string]string
	require.NoError(t, json.Unmarshal(ds.lastDetails, &details))
	assert.Equal(t, "hello", details["name"])

	// Verify timestamp is reasonable
	assert.WithinDuration(t, time.Now(), ds.lastCreatedAt, 2*time.Second)
}

func TestNewActivityDuplicatesAliasedJSONKeys(t *testing.T) {
	t.Parallel()
	ds := &newActivityMockDatastore{}
	providers := &newActivityMockProviders{
		mockDataProviders: mockDataProviders{
			mockUserProvider: &mockUserProvider{},
			mockHostProvider: &mockHostProvider{},
		},
	}
	svc := newTestService(ds, providers)

	err := svc.NewActivity(t.Context(), nil, aliasedActivity{TeamID: 42})
	require.NoError(t, err)

	require.True(t, ds.newActivityCalled)

	// Details should contain both team_id and fleet_id
	var details map[string]any
	require.NoError(t, json.Unmarshal(ds.lastDetails, &details))
	assert.Equal(t, float64(42), details["team_id"])
	assert.Equal(t, float64(42), details["fleet_id"])
}

func TestNewActivityCallsActivator(t *testing.T) {
	t.Parallel()
	ds := &newActivityMockDatastore{}
	providers := &newActivityMockProviders{
		mockDataProviders: mockDataProviders{
			mockUserProvider: &mockUserProvider{},
			mockHostProvider: &mockHostProvider{},
		},
	}
	svc := newTestService(ds, providers)

	act := activatorActivity{
		simpleActivity: simpleActivity{Name: "install"},
		hostID:         99,
		cmdUUID:        "cmd-abc",
	}
	err := svc.NewActivity(t.Context(), nil, act)
	require.NoError(t, err)

	// Verify activator was called with correct args
	require.True(t, providers.activateCalled)
	assert.Equal(t, uint(99), providers.lastHostID)
	assert.Equal(t, "cmd-abc", providers.lastCmdUUID)

	// Verify store was also called
	require.True(t, ds.newActivityCalled)
}

func TestNewActivityActivatorErrorPreventsStore(t *testing.T) {
	t.Parallel()
	ds := &newActivityMockDatastore{}
	providers := &newActivityMockProviders{
		mockDataProviders: mockDataProviders{
			mockUserProvider: &mockUserProvider{},
			mockHostProvider: &mockHostProvider{},
		},
		activateErr: assert.AnError,
	}
	svc := newTestService(ds, providers)

	act := activatorActivity{
		simpleActivity: simpleActivity{Name: "install"},
		hostID:         99,
		cmdUUID:        "cmd-abc",
	}
	err := svc.NewActivity(t.Context(), nil, act)
	require.Error(t, err)

	// Activator was called but store should NOT have been called
	require.True(t, providers.activateCalled)
	assert.False(t, ds.newActivityCalled)
}

func TestNewActivityNilUser(t *testing.T) {
	t.Parallel()
	ds := &newActivityMockDatastore{}
	providers := &newActivityMockProviders{
		mockDataProviders: mockDataProviders{
			mockUserProvider: &mockUserProvider{},
			mockHostProvider: &mockHostProvider{},
		},
	}
	svc := newTestService(ds, providers)

	err := svc.NewActivity(t.Context(), nil, simpleActivity{Name: "system"})
	require.NoError(t, err)

	require.True(t, ds.newActivityCalled)
	assert.Nil(t, ds.lastUser)
}
