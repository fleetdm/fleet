package datastore

import (
	"testing"

	"github.com/kolide/kolide-ose/server/kolide"
	"github.com/stretchr/testify/assert"
)

func testCreateInvite(t *testing.T, ds kolide.Datastore) {
	invite := &kolide.Invite{}

	invite, err := ds.NewInvite(invite)
	assert.Nil(t, err)

	verify, err := ds.InviteByEmail(invite.Email)
	assert.Nil(t, err)
	assert.Equal(t, invite.ID, verify.ID)
	assert.Equal(t, invite.Email, verify.Email)
}
