package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/server/authz"
	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/contexts/token"
	"github.com/fleetdm/fleet/server/datastore/inmem"
	"github.com/fleetdm/fleet/server/fleet"
	"github.com/fleetdm/fleet/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	svc := newTestService(ds, nil, nil)
	users := createTestUsers(t, ds)

	var loginTests = []struct {
		username string
		password string
		user     fleet.User
		wantErr  error
	}{
		{
			user:     users["admin1"],
			username: testUsers["admin1"].Username,
			password: testUsers["admin1"].PlaintextPassword,
		},
		{
			user:     users["user1"],
			username: testUsers["user1"].Email,
			password: testUsers["user1"].PlaintextPassword,
		},
	}

	for _, tt := range loginTests {
		t.Run(tt.username, func(st *testing.T) {
			user := tt.user
			loggedIn, token, err := svc.Login(test.UserContext(test.UserAdmin), tt.username, tt.password)
			require.Nil(st, err, "login unsuccessful")
			assert.Equal(st, user.ID, loggedIn.ID)
			assert.NotEmpty(st, token)

			sessions, err := svc.GetInfoAboutSessionsForUser(test.UserContext(test.UserAdmin), user.ID)
			require.Nil(st, err)
			require.Len(st, sessions, 1, "user should have one session")
			session := sessions[0]
			assert.Equal(st, user.ID, session.UserID)
			assert.WithinDuration(st, time.Now(), session.AccessedAt, 3*time.Second,
				"access time should be set with current time at session creation")
		})
	}
}

func TestGenerateJWT(t *testing.T) {
	jwtKey := ""
	tokenString, err := generateJWT("4", jwtKey)
	require.Nil(t, err)

	svc := authViewerService{Service: &Service{authz: authz.Must()}}
	viewer, err := authViewer(
		context.Background(),
		jwtKey,
		token.Token(tokenString),
		svc,
	)
	require.Nil(t, err)
	require.NotNil(t, viewer)
}

type authViewerService struct {
	fleet.Service
}

func (authViewerService) GetSessionByKey(ctx context.Context, key string) (*fleet.Session, error) {
	return &fleet.Session{}, nil
}

func (authViewerService) UserUnauthorized(ctx context.Context, uid uint) (*fleet.User, error) {
	return &fleet.User{}, nil
}
