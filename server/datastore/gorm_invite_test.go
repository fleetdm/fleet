package datastore

import (
	"os"
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func TestCreateInvite(t *testing.T) {
	var ds kolide.Datastore
	address := os.Getenv("MYSQL_ADDR")
	if address == "" {
		ds = setup(t)
	} else {
		ds = setupMySQLGORM(t)
		defer teardownMySQLGORM(t, ds)
	}

	invite := &kolide.Invite{}

	invite, err := ds.NewInvite(invite)
	assert.Nil(t, err)

	verify, err := ds.InviteByEmail(invite.Email)
	assert.Nil(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	assert.Equal(t, invite.Email, verify.Email)
}
