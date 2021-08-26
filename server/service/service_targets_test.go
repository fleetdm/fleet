package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchTargets(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})

	hosts := []*fleet.Host{
		{Hostname: "foo.local"},
	}
	labels := []*fleet.Label{
		{
			Name:  "label foo",
			Query: "query foo",
		},
	}
	teams := []*fleet.Team{
		{Name: "team1"},
	}

	ds.SearchHostsFunc = func(filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Host, error) {
		assert.Equal(t, user, filter.User)
		return hosts, nil
	}
	ds.SearchLabelsFunc = func(filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Label, error) {
		assert.Equal(t, user, filter.User)
		return labels, nil
	}
	ds.SearchTeamsFunc = func(filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Team, error) {
		assert.Equal(t, user, filter.User)
		return teams, nil
	}

	results, err := svc.SearchTargets(ctx, "foo", nil, fleet.HostTargets{})
	require.NoError(t, err)
	assert.Equal(t, hosts[0], results.Hosts[0])
	assert.Equal(t, labels[0], results.Labels[0])
	assert.Equal(t, teams[0], results.Teams[0])
}

func TestSearchWithOmit(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	user := &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})

	ds.SearchHostsFunc = func(filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Host, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{1, 2}, omit)
		return nil, nil
	}
	ds.SearchLabelsFunc = func(filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Label, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{3, 4}, omit)
		return nil, nil
	}
	ds.SearchTeamsFunc = func(filter fleet.TeamFilter, query string, omit ...uint) ([]*fleet.Team, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{5, 6}, omit)
		return nil, nil
	}

	_, err := svc.SearchTargets(ctx, "foo", nil, fleet.HostTargets{HostIDs: []uint{1, 2}, LabelIDs: []uint{3, 4}, TeamIDs: []uint{5, 6}})
	require.Nil(t, err)
}
