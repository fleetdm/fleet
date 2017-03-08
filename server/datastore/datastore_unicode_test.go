package datastore

import (
	"testing"
	"time"

	"github.com/kolide/kolide/server/kolide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testUnicode(t *testing.T, ds kolide.Datastore) {
	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	label, err := ds.NewLabel(&kolide.Label{Name: "測試"})
	require.Nil(t, err)

	label, err = ds.Label(label.ID)
	require.Nil(t, err)
	assert.Equal(t, "測試", label.Name)

	host, err := ds.NewHost(&kolide.Host{
		HostName:         "🍌",
		DetailUpdateTime: time.Now(),
		SeenTime:         time.Now(),
	})
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	assert.Equal(t, "🍌", host.HostName)

	user, err := ds.NewUser(&kolide.User{Username: "🍱", Password: []byte{}})
	require.Nil(t, err)

	user, err = ds.User(user.Username)
	assert.Equal(t, "🍱", user.Username)

	pack, err := ds.NewPack(&kolide.Pack{Name: "👨🏾‍🚒"})
	require.Nil(t, err)

	pack, err = ds.Pack(pack.ID)
	assert.Equal(t, "👨🏾‍🚒", pack.Name)
}
