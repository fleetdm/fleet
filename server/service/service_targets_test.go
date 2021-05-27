package service

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/server/contexts/viewer"
	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/fleetdm/fleet/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSearchTargets(t *testing.T) {
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	user := &kolide.User{GlobalRole: ptr.String(kolide.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})

	hosts := []*kolide.Host{
		{HostName: "foo.local"},
	}
	labels := []*kolide.Label{
		{
			Name:  "label foo",
			Query: "query foo",
		},
	}
	teams := []*kolide.Team{
		{Name: "team1"},
	}

	ds.SearchHostsFunc = func(filter kolide.TeamFilter, query string, omit ...uint) ([]*kolide.Host, error) {
		assert.Equal(t, user, filter.User)
		return hosts, nil
	}
	ds.SearchLabelsFunc = func(filter kolide.TeamFilter, query string, omit ...uint) ([]*kolide.Label, error) {
		assert.Equal(t, user, filter.User)
		return labels, nil
	}
	ds.SearchTeamsFunc = func(filter kolide.TeamFilter, query string, omit ...uint) ([]*kolide.Team, error) {
		assert.Equal(t, user, filter.User)
		return teams, nil
	}

	results, err := svc.SearchTargets(ctx, "foo", nil, kolide.HostTargets{})
	require.NoError(t, err)
	assert.Equal(t, hosts[0], results.Hosts[0])
	assert.Equal(t, labels[0], results.Labels[0])
	assert.Equal(t, teams[0], results.Teams[0])
}

func TestSearchWithOmit(t *testing.T) {
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.Nil(t, err)

	user := &kolide.User{GlobalRole: ptr.String(kolide.RoleAdmin)}
	ctx := viewer.NewContext(context.Background(), viewer.Viewer{User: user})

	ds.SearchHostsFunc = func(filter kolide.TeamFilter, query string, omit ...uint) ([]*kolide.Host, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{1, 2}, omit)
		return nil, nil
	}
	ds.SearchLabelsFunc = func(filter kolide.TeamFilter, query string, omit ...uint) ([]*kolide.Label, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{3, 4}, omit)
		return nil, nil
	}
	ds.SearchTeamsFunc = func(filter kolide.TeamFilter, query string, omit ...uint) ([]*kolide.Team, error) {
		assert.Equal(t, user, filter.User)
		assert.Equal(t, []uint{5, 6}, omit)
		return nil, nil
	}

	_, err = svc.SearchTargets(ctx, "foo", nil, kolide.HostTargets{HostIDs: []uint{1, 2}, LabelIDs: []uint{3, 4}, TeamIDs: []uint{5, 6}})
	require.Nil(t, err)
}
