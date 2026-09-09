package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
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

// hostActivity is a host-linked activity (implements types.ActivityHosts).
type hostActivity struct {
	simpleActivity
	hostIDs []uint
}

func (a hostActivity) HostIDs() []uint { return a.hostIDs }

// hostActivityWithID mirrors the single-host activity types (ran_script,
// locked_host, ...) whose details already carry host_id.
type hostActivityWithID struct {
	hostActivity
	HostID uint `json:"host_id"`
}

// hostActivityNestedID has a host_id key only inside a nested object; the
// top-level details carry no host identifier, so injection must still happen.
type hostActivityNestedID struct {
	hostActivity
	Nested map[string]uint `json:"nested"`
}

// hostActivityStoredEmptyIDs serializes an empty host_ids of its own (e.g. a
// future batch type whose HostIDList loses the json:"-" tag).
type hostActivityStoredEmptyIDs struct {
	hostActivity
	StoredHostIDs []uint `json:"host_ids"`
}

// hostActivityMalformedIDs serializes a non-array host_ids of its own.
type hostActivityMalformedIDs struct {
	hostActivity
	StoredHostIDs string `json:"host_ids"`
}

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
	return NewService(&mockAuthorizer{}, ds, providers, slog.New(slog.DiscardHandler))
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

// newTestServiceWithWebhook creates a service configured for webhook delivery tests.
func newTestServiceWithWebhook(ds types.Datastore, providers activity.DataProviders) *Service {
	return NewService(&mockAuthorizer{}, ds, providers, slog.New(slog.DiscardHandler))
}

func TestNewActivityWebhook(t *testing.T) {
	t.Parallel()

	webhookChannel := make(chan struct{}, 1)
	var webhookBody webhookPayload
	fail429 := false

	startMockServer := func(t *testing.T) string {
		srv := httptest.NewServer(
			http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					webhookBody = webhookPayload{}
					if r.Method != "POST" {
						w.WriteHeader(http.StatusMethodNotAllowed)
						return
					}
					switch r.URL.Path {
					case "/ok":
						err := json.NewDecoder(r.Body).Decode(&webhookBody)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
						}
					case "/error":
						webhookBody.Type = "error"
						w.WriteHeader(http.StatusTeapot)
					case "/429":
						fail429 = !fail429
						if fail429 {
							w.WriteHeader(http.StatusTooManyRequests)
							return
						}
						err := json.NewDecoder(r.Body).Decode(&webhookBody)
						if err != nil {
							t.Log(err)
							w.WriteHeader(http.StatusBadRequest)
						}
					default:
						w.WriteHeader(http.StatusNotFound)
						return
					}
					webhookChannel <- struct{}{}
				},
			),
		)
		t.Cleanup(srv.Close)
		return srv.URL
	}

	mockURL := startMockServer(t)
	testURL := mockURL

	ds := &newActivityMockDatastore{}
	providers := &newActivityMockProviders{
		mockDataProviders: mockDataProviders{
			mockUserProvider: &mockUserProvider{},
			mockHostProvider: &mockHostProvider{},
			webhookConfig: &activity.ActivitiesWebhookSettings{
				Enable:         true,
				DestinationURL: testURL,
			},
		},
	}

	svc := newTestServiceWithWebhook(ds, providers)

	tests := []struct {
		name    string
		user    *api.User
		url     string
		doError bool
	}{
		{
			name: "nil user",
			url:  mockURL + "/ok",
			user: nil,
		},
		{
			name: "real user",
			url:  mockURL + "/ok",
			user: &api.User{
				ID:    1,
				Name:  "testUser",
				Email: "testUser@example.com",
			},
		},
		{
			name:    "error",
			url:     mockURL + "/error",
			doError: true,
		},
		{
			name: "429",
			url:  mockURL + "/429",
			user: &api.User{
				ID:    2,
				Name:  "testUserRetry",
				Email: "testUserRetry@example.com",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ds.newActivityCalled = false
			providers.webhookConfig.DestinationURL = tt.url
			startTime := time.Now()
			act := simpleActivity{Name: tt.name}
			err := svc.NewActivity(t.Context(), tt.user, act)
			require.NoError(t, err)
			select {
			case <-time.After(3 * time.Second):
				t.Error("timeout waiting for webhook")
			case <-webhookChannel:
				if tt.doError {
					assert.Equal(t, "error", webhookBody.Type)
				} else {
					endTime := time.Now()
					assert.False(
						t, webhookBody.Timestamp.Before(startTime), "timestamp %s is before start time %s",
						webhookBody.Timestamp.String(), startTime.String(),
					)
					assert.False(t, webhookBody.Timestamp.After(endTime))
					if tt.user == nil {
						assert.Nil(t, webhookBody.ActorFullName)
						assert.Nil(t, webhookBody.ActorID)
						assert.Nil(t, webhookBody.ActorEmail)
					} else {
						require.NotNil(t, webhookBody.ActorFullName)
						assert.Equal(t, tt.user.Name, *webhookBody.ActorFullName)
						require.NotNil(t, webhookBody.ActorID)
						assert.Equal(t, tt.user.ID, *webhookBody.ActorID)
						require.NotNil(t, webhookBody.ActorEmail)
						assert.Equal(t, tt.user.Email, *webhookBody.ActorEmail)
					}
					assert.Equal(t, act.ActivityName(), webhookBody.Type)
					var details map[string]string
					require.NoError(t, json.Unmarshal(*webhookBody.Details, &details))
					assert.Len(t, details, 1)
					assert.Equal(t, tt.name, details["name"])
				}
			}
			require.True(t, ds.newActivityCalled)
		})
	}
}

func TestNewActivityWebhookDisabled(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("webhook server should not be called when webhook is disabled")
		}),
	)
	t.Cleanup(srv.Close)

	ds := &newActivityMockDatastore{}
	providers := &newActivityMockProviders{
		mockDataProviders: mockDataProviders{
			mockUserProvider: &mockUserProvider{},
			mockHostProvider: &mockHostProvider{},
			webhookConfig: &activity.ActivitiesWebhookSettings{
				Enable:         false,
				DestinationURL: srv.URL,
			},
		},
	}

	svc := newTestServiceWithWebhook(ds, providers)
	err := svc.NewActivity(t.Context(), &api.User{ID: 1}, simpleActivity{Name: "no webhook"})
	require.NoError(t, err)
	require.True(t, ds.newActivityCalled)
}

func TestNewActivityHostWebhook(t *testing.T) {
	t.Parallel()

	// One channel-fed server so payloads can be asserted; each fleet webhook
	// destination is a distinct path on it.
	type receivedPayload struct {
		path string
		body webhookPayload
	}
	received := make(chan receivedPayload, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body webhookPayload
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Log(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		received <- receivedPayload{path: r.URL.Path, body: body}
	}))
	t.Cleanup(srv.Close)

	user := &api.User{ID: 1, Name: "testUser", Email: "testUser@example.com"}

	t.Run("fires one webhook per fleet destination", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/fleet-a", HostIDs: []uint{42}},
					{DestinationURL: srv.URL + "/fleet-b", HostIDs: []uint{43}},
				},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		act := hostActivity{simpleActivity: simpleActivity{Name: "host act"}, hostIDs: []uint{42, 43}}
		err := svc.NewActivity(t.Context(), user, act)
		require.NoError(t, err)
		require.True(t, ds.newActivityCalled)
		require.True(t, providers.hostWebhooksCalled)
		assert.Equal(t, []uint{42, 43}, providers.hostWebhookHostIDs)

		paths := make(map[string]webhookPayload, 2)
		for range 2 {
			select {
			case <-time.After(3 * time.Second):
				t.Fatal("timeout waiting for host activities webhooks")
			case p := <-received:
				paths[p.path] = p.body
			}
		}
		require.Len(t, paths, 2)
		// Each destination's payload carries only its own fleet's host IDs.
		wantHostIDs := map[string][]any{
			"/fleet-a": {float64(42)},
			"/fleet-b": {float64(43)},
		}
		for _, path := range []string{"/fleet-a", "/fleet-b"} {
			body, ok := paths[path]
			require.True(t, ok, "expected a webhook POST to %s", path)
			// Same format as the global activities webhook.
			assert.Equal(t, act.ActivityName(), body.Type)
			require.NotNil(t, body.ActorFullName)
			assert.Equal(t, user.Name, *body.ActorFullName)
			require.NotNil(t, body.ActorID)
			assert.Equal(t, user.ID, *body.ActorID)
			require.NotNil(t, body.ActorEmail)
			assert.Equal(t, user.Email, *body.ActorEmail)
			var details map[string]any
			require.NoError(t, json.Unmarshal(*body.Details, &details))
			assert.Equal(t, "host act", details["name"])
			// host_ids is injected into the webhook payload at fire time...
			assert.Equal(t, wantHostIDs[path], details["host_ids"])
		}
		// ...but not into the stored details, which stay lean for API/feed
		// responses.
		var stored map[string]any
		require.NoError(t, json.Unmarshal(ds.lastDetails, &stored))
		_, hasHostIDs := stored["host_ids"]
		assert.False(t, hasHostIDs, "host_ids must not be stored in details")
	})

	t.Run("details with their own host_id get no host_ids injected", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/single-host", HostIDs: []uint{7}},
				},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		act := hostActivityWithID{
			hostActivity: hostActivity{simpleActivity: simpleActivity{Name: "single"}, hostIDs: []uint{7}},
			HostID:       7,
		}
		err := svc.NewActivity(t.Context(), user, act)
		require.NoError(t, err)

		select {
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for host activities webhook")
		case p := <-received:
			require.Equal(t, "/single-host", p.path)
			var details map[string]any
			require.NoError(t, json.Unmarshal(*p.body.Details, &details))
			assert.EqualValues(t, 7, details["host_id"])
			_, hasHostIDs := details["host_ids"]
			assert.False(t, hasHostIDs, "host_ids must not be injected when host_id is present")
		}
	})

	t.Run("nested host_id keys do not suppress injection", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/nested", HostIDs: []uint{5}},
				},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		act := hostActivityNestedID{
			hostActivity: hostActivity{simpleActivity: simpleActivity{Name: "nested"}, hostIDs: []uint{5}},
			Nested:       map[string]uint{"host_id": 5},
		}
		require.NoError(t, svc.NewActivity(t.Context(), user, act))

		select {
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for host activities webhook")
		case p := <-received:
			require.Equal(t, "/nested", p.path)
			var details map[string]any
			require.NoError(t, json.Unmarshal(*p.body.Details, &details))
			// Only top-level keys count: the nested host_id must not
			// suppress the injection.
			assert.Equal(t, []any{float64(5)}, details["host_ids"])
		}
	})

	t.Run("stored empty host_ids is replaced with the delivery's list", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/stored-empty", HostIDs: []uint{11}},
				},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		act := hostActivityStoredEmptyIDs{
			hostActivity:  hostActivity{simpleActivity: simpleActivity{Name: "stored empty"}, hostIDs: []uint{11}},
			StoredHostIDs: []uint{},
		}
		require.NoError(t, svc.NewActivity(t.Context(), user, act))

		select {
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for host activities webhook")
		case p := <-received:
			require.Equal(t, "/stored-empty", p.path)
			var details map[string]any
			require.NoError(t, json.Unmarshal(*p.body.Details, &details))
			// The stored empty list identifies nothing, so the delivery's
			// list wins.
			assert.Equal(t, []any{float64(11)}, details["host_ids"])
		}
	})

	t.Run("malformed stored host_ids is replaced with the delivery's list", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/stored-malformed", HostIDs: []uint{13}},
				},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		act := hostActivityMalformedIDs{
			hostActivity:  hostActivity{simpleActivity: simpleActivity{Name: "stored malformed"}, hostIDs: []uint{13}},
			StoredHostIDs: "not-a-list",
		}
		require.NoError(t, svc.NewActivity(t.Context(), user, act))

		select {
		case <-time.After(3 * time.Second):
			t.Fatal("timeout waiting for host activities webhook")
		case p := <-received:
			require.Equal(t, "/stored-malformed", p.path)
			var details map[string]any
			require.NoError(t, json.Unmarshal(*p.body.Details, &details))
			assert.Equal(t, []any{float64(13)}, details["host_ids"])
		}
	})

	t.Run("global activity does not look up fleet webhooks", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/never"},
				},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		err := svc.NewActivity(t.Context(), user, simpleActivity{Name: "global act"})
		require.NoError(t, err)
		require.True(t, ds.newActivityCalled)
		assert.False(t, providers.hostWebhooksCalled)
	})

	// e.g. mdm_enrolled/mdm_unenrolled return nil HostIDs when the host record
	// is unknown (Azure automatic enrollments) or already deleted.
	t.Run("host activity with no host IDs does not look up fleet webhooks", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		err := svc.NewActivity(t.Context(), user, hostActivity{simpleActivity: simpleActivity{Name: "no hosts"}})
		require.NoError(t, err)
		require.True(t, ds.newActivityCalled)
		assert.False(t, providers.hostWebhooksCalled)
	})

	t.Run("no enabled fleet webhooks fires nothing and stores the activity", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks:     nil,
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		err := svc.NewActivity(t.Context(), user, hostActivity{simpleActivity: simpleActivity{Name: "quiet"}, hostIDs: []uint{7}})
		require.NoError(t, err)
		require.True(t, ds.newActivityCalled)
		require.True(t, providers.hostWebhooksCalled)
		// Nothing should be delivered anywhere. Prior subtests drained their own
		// deliveries, so any message here would be a stray fire from this one.
		select {
		case p := <-received:
			t.Fatalf("unexpected webhook POST to %s", p.path)
		case <-time.After(200 * time.Millisecond):
		}
	})

	t.Run("provider error is best-effort: activity is still stored, nothing fires", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/partial", HostIDs: []uint{7}},
				},
				hostWebhooksErr: errors.New("boom"),
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		err := svc.NewActivity(t.Context(), user, hostActivity{simpleActivity: simpleActivity{Name: "err"}, hostIDs: []uint{7}})
		require.NoError(t, err)
		assert.True(t, ds.newActivityCalled)

		select {
		case p := <-received:
			t.Fatalf("unexpected webhook POST to %s after lookup error", p.path)
		case <-time.After(200 * time.Millisecond):
		}
	})

	t.Run("global and fleet webhooks both fire for a host activity", func(t *testing.T) {
		ds := &newActivityMockDatastore{}
		providers := &newActivityMockProviders{
			mockDataProviders: mockDataProviders{
				mockUserProvider: &mockUserProvider{},
				mockHostProvider: &mockHostProvider{},
				webhookConfig: &activity.ActivitiesWebhookSettings{
					Enable:         true,
					DestinationURL: srv.URL + "/global",
				},
				hostWebhooks: []activity.HostActivitiesWebhook{
					{DestinationURL: srv.URL + "/fleet-c", HostIDs: []uint{9}},
				},
			},
		}
		svc := newTestServiceWithWebhook(ds, providers)

		act := hostActivity{simpleActivity: simpleActivity{Name: "both"}, hostIDs: []uint{9}}
		err := svc.NewActivity(t.Context(), user, act)
		require.NoError(t, err)

		paths := make(map[string]webhookPayload, 2)
		for range 2 {
			select {
			case <-time.After(3 * time.Second):
				t.Fatal("timeout waiting for global + fleet webhooks")
			case p := <-received:
				paths[p.path] = p.body
			}
		}
		require.Len(t, paths, 2)
		assert.Contains(t, paths, "/global")
		assert.Contains(t, paths, "/fleet-c")
		// Same envelope on both destinations, but host_ids is injected into
		// the per-fleet payload only — the global payload keeps its
		// pre-existing format (stored details, no host_ids).
		var globalDetails, fleetDetails map[string]any
		require.NoError(t, json.Unmarshal(*paths["/global"].Details, &globalDetails))
		require.NoError(t, json.Unmarshal(*paths["/fleet-c"].Details, &fleetDetails))
		assert.Equal(t, "both", globalDetails["name"])
		assert.Equal(t, "both", fleetDetails["name"])
		_, globalHasHostIDs := globalDetails["host_ids"]
		assert.False(t, globalHasHostIDs, "global payload must not carry injected host_ids")
		assert.Equal(t, []any{float64(9)}, fleetDetails["host_ids"])
	})
}
