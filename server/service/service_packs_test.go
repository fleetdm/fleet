package service

import (
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/datastore/inmem"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestListPacks(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc := newTestService(ds, nil, nil)

	queries, err := svc.ListPacks(test.UserContext(test.UserAdmin), fleet.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 0)

	_, err = ds.NewPack(&fleet.Pack{
		Name: "foo",
	})
	assert.Nil(t, err)

	queries, err = svc.ListPacks(test.UserContext(test.UserAdmin), fleet.ListOptions{})
	assert.Nil(t, err)
	assert.Len(t, queries, 1)
}

func TestGetPack(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc := newTestService(ds, nil, nil)

	pack := &fleet.Pack{
		Name: "foo",
	}
	_, err = ds.NewPack(pack)
	assert.Nil(t, err)
	assert.NotZero(t, pack.ID)

	packVerify, err := svc.GetPack(test.UserContext(test.UserAdmin), pack.ID)
	assert.Nil(t, err)

	assert.Equal(t, pack.ID, packVerify.ID)
}

func TestNewSavesTargets(t *testing.T) {
	ds, err := inmem.New(config.TestConfig())
	assert.Nil(t, err)
	svc := newTestService(ds, nil, nil)

	host, err := ds.NewHost(&fleet.Host{
		ID:            42,
		OsqueryHostID: "42",
		NodeKey:       "42",
	})
	require.Nil(t, err)

	label := &fleet.Label{
		Name:  "foo",
		Query: "select * from foo;",
	}
	label, err = ds.NewLabel(label)
	require.NoError(t, err)
	assert.NotZero(t, label.ID)

	// TODO: allow teams to be tested with inmem or just move this to mysql
	//team, err := ds.NewTeam(&fleet.Team{Name: "team1"})
	//require.NoError(t, err)

	packPayload := fleet.PackPayload{
		Name:     ptr.String("foo"),
		HostIDs:  &[]uint{host.ID},
		LabelIDs: &[]uint{label.ID},
		//TeamIDs:  &[]uint{team.ID},
	}
	pack, err := svc.NewPack(test.UserContext(test.UserAdmin), packPayload)
	require.Nil(t, err)
	assert.NotZero(t, pack.ID)

	packVerify, err := svc.GetPack(test.UserContext(test.UserAdmin), pack.ID)
	require.Nil(t, err)

	require.Len(t, packVerify.HostIDs, 1)
	require.Len(t, packVerify.LabelIDs, 1)
	//require.Len(t, packVerify.TeamIDs, 1)
	assert.Equal(t, host.ID, packVerify.HostIDs[0])
	assert.Equal(t, label.ID, packVerify.LabelIDs[0])
	//assert.Equal(t, team.ID, packVerify.TeamIDs[0])
}
