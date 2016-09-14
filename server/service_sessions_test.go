package server

import (
	"testing"

	kitlog "github.com/go-kit/kit/log"
	"github.com/kolide/kolide-ose/config"
	"github.com/kolide/kolide-ose/datastore"
	"github.com/kolide/kolide-ose/kolide"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

const bcryptCost = 6

func TestAuthenticate(t *testing.T) {
	ds, err := datastore.New("gorm-sqlite3", ":memory:")
	assert.Nil(t, err)

	svc, err := NewService(ds, kitlog.NewNopLogger(), config.TestConfig())
	assert.Nil(t, err)

	ctx := context.Background()

	user, err := kolide.NewUser("foo", "bar", "foo@kolide.co", false, false, bcryptCost)
	assert.Nil(t, err)
	user, err = ds.NewUser(user)
	assert.Nil(t, err)
	assert.NotZero(t, user.ID)

	loggedIn, token, err := svc.Login(ctx, "foo", "bar")
	assert.Nil(t, err)
	assert.Equal(t, user.ID, loggedIn.ID)
	assert.NotEmpty(t, token)
}
