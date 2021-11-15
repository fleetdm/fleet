package service

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestListHosts(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.ListHostsFunc = func(ctx context.Context, filter fleet.TeamFilter, opt fleet.HostListOptions) ([]*fleet.Host, error) {
		return []*fleet.Host{
			{ID: 1},
		}, nil
	}

	hosts, err := svc.ListHosts(test.UserContext(test.UserAdmin), fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	// anyone can list hosts
	hosts, err = svc.ListHosts(test.UserContext(test.UserNoRoles), fleet.HostListOptions{})
	require.NoError(t, err)
	require.Len(t, hosts, 1)

	// a user is required
	_, err = svc.ListHosts(context.Background(), fleet.HostListOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestGetHostSummary(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	ds.GenerateHostStatusStatisticsFunc = func(ctx context.Context, filter fleet.TeamFilter, now time.Time) (*fleet.HostSummary, error) {
		return &fleet.HostSummary{
			OnlineCount:      1,
			OfflineCount:     2,
			MIACount:         3,
			NewCount:         4,
			TotalsHostsCount: 5,
		}, nil
	}

	summary, err := svc.GetHostSummary(test.UserContext(test.UserAdmin), nil)
	require.NoError(t, err)
	require.Nil(t, summary.TeamID)
	require.Equal(t, uint(1), summary.OnlineCount)
	require.Equal(t, uint(2), summary.OfflineCount)
	require.Equal(t, uint(3), summary.MIACount)
	require.Equal(t, uint(4), summary.NewCount)
	require.Equal(t, uint(5), summary.TotalsHostsCount)

	_, err = svc.GetHostSummary(test.UserContext(test.UserNoRoles), nil)
	require.NoError(t, err)

	// a user is required
	_, err = svc.GetHostSummary(context.Background(), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}
