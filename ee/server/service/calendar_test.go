package service

import (
	"context"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	authz_ctx "github.com/fleetdm/fleet/v4/server/contexts/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/service/calendar"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Implement fleet.Lock interface
type mockLock struct {
	AcquireLockFn func(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error)
	GetFn         func(ctx context.Context, key string) (*string, error)
	AddToSetFn    func(ctx context.Context, key string, value string) error
	ReleaseLockFn func(ctx context.Context, key string, value string) (ok bool, err error)
}

func (m *mockLock) SetIfNotExist(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error) {
	if m.AcquireLockFn != nil {
		return m.AcquireLockFn(ctx, key, value, expireMs)
	}
	return false, nil
}

func (m *mockLock) ReleaseLock(ctx context.Context, key string, value string) (ok bool, err error) {
	if m.ReleaseLockFn != nil {
		return m.ReleaseLockFn(ctx, key, value)
	}
	return true, nil
}

func (m *mockLock) Get(ctx context.Context, key string) (*string, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, key)
	}
	return nil, nil
}

func (m *mockLock) GetAndDelete(ctx context.Context, key string) (*string, error) {
	return nil, nil
}

func (m *mockLock) AddToSet(ctx context.Context, key string, value string) error {
	if m.AddToSetFn != nil {
		return m.AddToSetFn(ctx, key, value)
	}
	return nil
}

func (m *mockLock) RemoveFromSet(ctx context.Context, key string, value string) error {
	return nil
}

func (m *mockLock) GetSet(ctx context.Context, key string) ([]string, error) {
	return nil, nil
}

var calendarTestSetup = func(t *testing.T) (*mockLock, *Service) {
	lock := &mockLock{}
	svc := &Service{
		distributedLock: lock,
	}
	return lock, svc
}

func TestGetCalendarLock(t *testing.T) {
	lock, svc := calendarTestSetup(t)
	ctx := context.Background()
	eventUUID := "testUUID"
	lock.AcquireLockFn = func(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error) {
		return true, nil
	}
	lock.GetFn = func(ctx context.Context, key string) (*string, error) {
		// not reserved
		return nil, nil
	}
	lockValue, reserved, err := svc.getCalendarLock(ctx, eventUUID, false)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.NotEmpty(t, lockValue)

	// Make sure lock value is empty if we don't acquire the lock.
	lock.AcquireLockFn = func(ctx context.Context, key string, value string, expireMs uint64) (ok bool, err error) {
		return false, nil
	}
	lock.GetFn = func(ctx context.Context, key string) (*string, error) {
		value := "value"
		return &value, nil
	}
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, false)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Empty(t, lockValue)

	addedToSet := false
	lock.AddToSetFn = func(ctx context.Context, key string, value string) error {
		addedToSet = true
		return nil
	}
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, true)
	require.NoError(t, err)
	assert.True(t, reserved)
	assert.Empty(t, lockValue)
	assert.True(t, addedToSet)

	addedToSet = false
	lock.GetFn = func(ctx context.Context, key string) (*string, error) {
		// not reserved
		return nil, nil
	}
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, false)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Empty(t, lockValue)
	assert.False(t, addedToSet)

	addedToSet = false
	lockValue, reserved, err = svc.getCalendarLock(ctx, eventUUID, true)
	require.NoError(t, err)
	assert.False(t, reserved)
	assert.Empty(t, lockValue)
	assert.True(t, addedToSet)

}

func TestCalendarWebhookErrorCases(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		eventUUID     string
		channelID     string
		resourceState string
		setupMocks    func(*mock.Store, *mockLock)
		expectedError string
		expectNoError bool
	}{
		{
			name:          "app config load error",
			eventUUID:     "test-uuid-1",
			channelID:     "channel-1",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, _ *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return nil, errors.New("database error")
				}
			},
			expectedError: "load app config: database error",
		},
		{
			name:          "no google calendar integration configured",
			eventUUID:     "test-uuid-2",
			channelID:     "channel-2",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, _ *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{},
						},
					}, nil
				}
			},
			expectNoError: true,
		},
		{
			name:          "sync resource state",
			eventUUID:     "test-uuid-3",
			channelID:     "channel-3",
			resourceState: "sync",
			setupMocks: func(ds *mock.Store, _ *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
			},
			expectNoError: true,
		},
		{
			name:          "recent update lock get error",
			eventUUID:     "test-uuid-4",
			channelID:     "channel-4",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, lock *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
				lock.GetFn = func(_ context.Context, key string) (*string, error) {
					if key == calendar.RecentUpdateKeyPrefix+"test-uuid-4" {
						return nil, errors.New("redis error")
					}
					return nil, nil
				}
			},
			expectedError: "redis error",
		},
		{
			name:          "event recently updated",
			eventUUID:     "test-uuid-5",
			channelID:     "channel-5",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, lock *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
				lock.GetFn = func(_ context.Context, key string) (*string, error) {
					if key == calendar.RecentUpdateKeyPrefix+"test-uuid-5" {
						value := calendar.RecentCalendarUpdateValue
						return &value, nil
					}
					return nil, nil
				}
			},
			expectNoError: true,
		},
		{
			name:          "calendar lock acquisition error",
			eventUUID:     "test-uuid-6",
			channelID:     "channel-6",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, lock *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
				lock.GetFn = func(_ context.Context, key string) (*string, error) {
					if key == calendar.ReservedLockKeyPrefix+"test-uuid-6" {
						return nil, errors.New("lock error")
					}
					return nil, nil
				}
			},
			expectedError: "get calendar reserved lock: lock error",
		},
		{
			name:          "event not found in database",
			eventUUID:     "test-uuid-7",
			channelID:     "channel-7",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, lock *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
				ds.GetCalendarEventDetailsByUUIDFunc = func(_ context.Context, _ string) (*fleet.CalendarEventDetails, error) {
					return nil, &testNotFoundError{}
				}
			},
			expectNoError: true,
		},
		{
			name:          "event has no host ID (deleted host)",
			eventUUID:     "test-uuid-8",
			channelID:     "channel-8",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, lock *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
				ds.GetCalendarEventDetailsByUUIDFunc = func(_ context.Context, _ string) (*fleet.CalendarEventDetails, error) {
					return &fleet.CalendarEventDetails{
						CalendarEvent: fleet.CalendarEvent{},
						HostID:        nil, // No host ID
						TeamID:        ptr.Uint(1),
					}, nil
				}
			},
			expectNoError: true,
		},
		{
			name:          "event has no team ID",
			eventUUID:     "test-uuid-9",
			channelID:     "channel-9",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, lock *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
				ds.GetCalendarEventDetailsByUUIDFunc = func(_ context.Context, _ string) (*fleet.CalendarEventDetails, error) {
					return &fleet.CalendarEventDetails{
						CalendarEvent: fleet.CalendarEvent{},
						HostID:        ptr.Uint(1),
						TeamID:        nil, // No team ID
					}, nil
				}
			},
			expectedError: "calendar event test-uuid-9 has no team ID",
		},
		{
			name:          "database error when getting event details",
			eventUUID:     "test-uuid-10",
			channelID:     "channel-10",
			resourceState: "exists",
			setupMocks: func(ds *mock.Store, lock *mockLock) {
				ds.AppConfigFunc = func(_ context.Context) (*fleet.AppConfig, error) {
					return &fleet.AppConfig{
						Integrations: fleet.Integrations{
							GoogleCalendar: []*fleet.GoogleCalendarIntegration{
								{Domain: "example.com"},
							},
						},
					}, nil
				}
				ds.GetCalendarEventDetailsByUUIDFunc = func(ctx context.Context, uuid string) (*fleet.CalendarEventDetails, error) {
					return nil, errors.New("database connection error")
				}
			},
			expectedError: "database connection error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			ds := &mock.Store{}
			lock := &mockLock{}

			// Setup default mock functions if not provided by test case
			if lock.GetFn == nil {
				lock.GetFn = func(_ context.Context, _ string) (*string, error) {
					return nil, nil
				}
			}
			if lock.AcquireLockFn == nil {
				lock.AcquireLockFn = func(_ context.Context, _ string, _ string, _ uint64) (ok bool, err error) {
					return false, nil
				}
			}
			if lock.AddToSetFn == nil {
				lock.AddToSetFn = func(_ context.Context, _ string, _ string) error {
					return nil
				}
			}

			// Create a real authorizer (required for the Service)
			auth, authErr := authz.NewAuthorizer()
			require.NoError(t, authErr)
			authzctx := &authz_ctx.AuthorizationContext{}
			ctx = authz_ctx.NewContext(ctx, authzctx)

			svc := &Service{
				ds:              ds,
				distributedLock: lock,
				authz:           auth,
				logger:          log.NewNopLogger(),
			}

			// Apply test-specific mocks
			tc.setupMocks(ds, lock)

			err := svc.CalendarWebhook(ctx, tc.eventUUID, tc.channelID, tc.resourceState)

			if tc.expectNoError {
				require.NoError(t, err)
			} else if tc.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedError)
			}

			require.True(t, authzctx.Checked(), "Make sure we either checked or explicitly skipped authorization")
		})
	}
}

// testNotFoundError is a simple implementation of fleet.NotFoundError for testing
type testNotFoundError struct{}

func (e *testNotFoundError) Error() string {
	return "not found"
}

func (e *testNotFoundError) IsNotFound() bool {
	return true
}
