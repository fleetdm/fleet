package main

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiveQuery(t *testing.T) {
	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	_, ds := runServerWithMockedDS(t, service.TestServerOpts{Rs: rs, Lq: lq})

	ds.HostIDsByNameFunc = func(ctx context.Context, filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
		return []uint{1234}, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) ([]uint, error) {
		return nil, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		query.ID = 42
		return query, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		camp.ID = 321
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1}, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{TotalHosts: 1, OnlineHosts: 1}, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}

	lq.On("QueriesForHost", uint(1)).Return(
		map[string]string{
			"42": "select 42, * from time",
		},
		nil,
	)
	lq.On("QueryCompletedByHost", "42", 99).Return(nil)
	lq.On("RunQuery", "321", "select 42, * from time", []uint{1}).Return(nil)

	ds.DistributedQueryCampaignTargetIDsFunc = func(ctx context.Context, id uint) (targets *fleet.HostTargets, err error) {
		return &fleet.HostTargets{HostIDs: []uint{99}}, nil
	}
	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return &fleet.DistributedQueryCampaign{ID: 321}, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}

	go func() {
		time.Sleep(2 * time.Second)
		require.NoError(t, rs.WriteResult(
			fleet.DistributedQueryResult{
				DistributedQueryCampaignID: 321,
				Rows:                       []map[string]string{{"bing": "fds"}},
				Host: fleet.Host{
					ID: 99,
					UpdateCreateTimestamps: fleet.UpdateCreateTimestamps{
						UpdateTimestamp: fleet.UpdateTimestamp{
							UpdatedAt: time.Now().UTC(),
						},
					},
					DetailUpdatedAt: time.Now().UTC(),
					Hostname:        "somehostname",
				},
			},
		))
	}()

	expected := `{"host":"somehostname","rows":[{"bing":"fds","host_hostname":"somehostname"}]}
`
	assert.Equal(t, expected, runAppForTest(t, []string{"query", "--hosts", "1234", "--query", "select 42, * from time"}))
}
