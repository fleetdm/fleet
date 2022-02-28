package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	defer ds.Close()

	svc := newTestService(ds, nil, nil)
	createTestUsers(t, ds)

	var loginTests = []struct {
		name     string
		email    string
		password string
		wantErr  error
	}{
		{
			name:     "admin1",
			email:    testUsers["admin1"].Email,
			password: testUsers["admin1"].PlaintextPassword,
		},
		{
			name:     "user1",
			email:    testUsers["user1"].Email,
			password: testUsers["user1"].PlaintextPassword,
		},
	}

	for _, tt := range loginTests {
		t.Run(tt.email, func(st *testing.T) {
			loggedIn, token, err := svc.Login(test.UserContext(test.UserAdmin), tt.email, tt.password)
			require.Nil(st, err, "login unsuccessful")
			assert.Equal(st, tt.email, loggedIn.Email)
			assert.NotEmpty(st, token)

			sessions, err := svc.GetInfoAboutSessionsForUser(test.UserContext(test.UserAdmin), loggedIn.ID)
			require.Nil(st, err)
			require.Len(st, sessions, 1, "user should have one session")
			session := sessions[0]
			assert.NotZero(st, session.UserID)
			assert.WithinDuration(st, time.Now(), session.AccessedAt, 3*time.Second,
				"access time should be set with current time at session creation")
		})
	}
}

func TestGetSessionByKey(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)
	cfg := config.TestConfig()

	theSession := &fleet.Session{UserID: 123, Key: "abc"}

	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return theSession, nil
	}
	ds.DestroySessionFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}
	ds.MarkSessionAccessedFunc = func(ctx context.Context, ssn *fleet.Session) error {
		return nil
	}

	cases := []struct {
		desc     string
		accessed time.Duration
		apiOnly  bool
		fail     bool
	}{
		{"real user, accessed recently", -1 * time.Hour, false, false},
		{"real user, accessed too long ago", -(cfg.Session.Duration + time.Hour), false, true},
		{"api-only, accessed recently", -1 * time.Hour, true, false},
		{"api-only, accessed long ago", -(cfg.Session.Duration + time.Hour), true, false},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var authErr *fleet.AuthRequiredError
			ds.SessionByKeyFuncInvoked, ds.DestroySessionFuncInvoked, ds.MarkSessionAccessedFuncInvoked = false, false, false

			theSession.AccessedAt = time.Now().Add(tc.accessed)
			theSession.APIOnly = ptr.Bool(tc.apiOnly)
			_, err := svc.GetSessionByKey(context.Background(), theSession.Key)
			if tc.fail {
				require.Error(t, err)
				require.ErrorAs(t, err, &authErr)
				require.True(t, ds.SessionByKeyFuncInvoked)
				require.True(t, ds.DestroySessionFuncInvoked)
				require.False(t, ds.MarkSessionAccessedFuncInvoked)
			} else {
				require.NoError(t, err)
				require.True(t, ds.SessionByKeyFuncInvoked)
				require.False(t, ds.DestroySessionFuncInvoked)
				require.True(t, ds.MarkSessionAccessedFuncInvoked)
			}
		})
	}
}
