package server

import (
	"testing"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

const bcryptCost = 6

func TestAuthenticate(t *testing.T) {
	svc, payload, user := setupLoginTests(t)
	var loginTests = []struct {
		username string
		password string
		wantErr  error
	}{
		{
			username: *payload.Username,
			password: *payload.Password,
		},
		{
			username: *payload.Email,
			password: *payload.Password,
		},
	}

	for _, tt := range loginTests {
		t.Run(tt.username, func(st *testing.T) {
			svc, _, user = setupLoginTests(t)
			ctx := context.Background()
			loggedIn, token, err := svc.Login(ctx, tt.username, tt.password)
			require.Nil(st, err, "login unsuccesful")
			assert.Equal(st, user.ID, loggedIn.ID)
			assert.NotEmpty(st, token)

			sessions, err := svc.GetInfoAboutSessionsForUser(ctx, user.ID)
			require.Nil(st, err)
			require.Len(st, sessions, 1, "user should have one session")
			session := sessions[0]
			assert.Equal(st, user.ID, session.UserID)
			assert.WithinDuration(st, time.Now(), session.AccessedAt, 3*time.Second,
				"access time should be set with current time at session creation")
			oldAccessTime := session.AccessedAt

			session, err = svc.GetSessionByKey(ctx, session.Key)
			require.Nil(st, err)
			assert.True(st, session.AccessedAt.After(oldAccessTime), "session access time should be updated")

		})
	}

}

func setupLoginTests(t *testing.T) (kolide.Service, kolide.UserPayload, kolide.User) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(ds, kitlog.NewNopLogger(), config.TestConfig(), nil)
	assert.Nil(t, err)
	payload := kolide.UserPayload{
		Username: stringPtr("foo"),
		Password: stringPtr("bar"),
		Email:    stringPtr("foo@kolide.co"),
		Admin:    boolPtr(false),
	}
	ctx := context.Background()
	user, err := svc.NewUser(ctx, payload)
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)
	return svc, payload, *user
}
