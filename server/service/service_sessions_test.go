package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/server/config"
	"github.com/fleetdm/fleet/server/contexts/token"
	"github.com/fleetdm/fleet/server/datastore/inmem"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthenticate(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	require.Nil(t, err)
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)
	users := createTestUsers(t, ds)

	var loginTests = []struct {
		username string
		password string
		user     kolide.User
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
			ctx := context.Background()
			loggedIn, token, err := svc.Login(ctx, tt.username, tt.password)
			require.Nil(st, err, "login unsuccessful")
			assert.Equal(st, user.ID, loggedIn.ID)
			assert.NotEmpty(st, token)

			sessions, err := svc.GetInfoAboutSessionsForUser(ctx, user.ID)
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

	svc := authViewerService{}
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
	kolide.Service
}

func (authViewerService) GetSessionByKey(ctx context.Context, key string) (*kolide.Session, error) {
	return &kolide.Session{}, nil
}

func (authViewerService) User(ctx context.Context, uid uint) (*kolide.User, error) {
	return &kolide.User{}, nil
}
