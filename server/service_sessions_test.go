package server

import (
	"testing"

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
			loggedIn, token, err := svc.Login(context.Background(), tt.username, tt.password)
			require.Nil(st, err, "login unsuccesful")
			assert.Equal(st, user.ID, loggedIn.ID)
			assert.NotEmpty(st, token)
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
