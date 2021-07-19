package mysql

import (
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/ptr"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnicode(t *testing.T) {
	ds := CreateMySQLDS(t)
	defer ds.Close()

	if ds.Name() == "inmem" {
		t.Skip("inmem is being deprecated, test skipped")
	}

	l1 := fleet.LabelSpec{
		ID:    1,
		Name:  "測試",
		Query: "query foo",
	}
	err := ds.ApplyLabelSpecs([]*fleet.LabelSpec{&l1})
	require.Nil(t, err)

	label, err := ds.Label(l1.ID)
	require.Nil(t, err)
	assert.Equal(t, "測試", label.Name)

	host, err := ds.NewHost(&fleet.Host{
		Hostname:        "🍌",
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		SeenTime:        time.Now(),
	})
	require.Nil(t, err)

	host, err = ds.Host(host.ID)
	require.Nil(t, err)
	assert.Equal(t, "🍌", host.Hostname)

	user, err := ds.NewUser(&fleet.User{
		Name:       "🍱",
		Email:      "test@example.com",
		Password:   []byte{},
		GlobalRole: ptr.String(fleet.RoleObserver),
	})
	require.Nil(t, err)

	user, err = ds.UserByID(user.ID)
	require.Nil(t, err)
	assert.Equal(t, "🍱", user.Name)

	pack := test.NewPack(t, ds, "👨🏾‍🚒")

	pack, err = ds.Pack(pack.ID)
	require.Nil(t, err)
	assert.Equal(t, "👨🏾‍🚒", pack.Name)
}
