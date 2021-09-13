package main

import (
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
	server, ds := runServerWithMockedDS(t, service.TestServerOpts{Rs: rs, Lq: lq})
	defer server.Close()

	ds.HostIDsByNameFunc = func(filter fleet.TeamFilter, hostnames []string) ([]uint, error) {
		return []uint{1234}, nil
	}
	ds.LabelIDsByNameFunc = func(labels []string) ([]uint, error) {
		return nil, nil
	}
	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewQueryFunc = func(query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		query.ID = 42
		return query, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		camp.ID = 321
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.HostIDsInTargetsFunc = func(filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1}, nil
	}
	ds.CountHostsInTargetsFunc = func(filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{TotalHosts: 1, OnlineHosts: 1}, nil
	}
	ds.NewActivityFunc = func(user *fleet.User, activityType string, details *map[string]interface{}) error {
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

	ds.DistributedQueryCampaignTargetIDsFunc = func(id uint) (targets *fleet.HostTargets, err error) {
		return &fleet.HostTargets{HostIDs: []uint{99}}, nil
	}
	ds.DistributedQueryCampaignFunc = func(id uint) (*fleet.DistributedQueryCampaign, error) {
		return &fleet.DistributedQueryCampaign{ID: 321}, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(camp *fleet.DistributedQueryCampaign) error {
		return nil
	}
	ds.QueryFunc = func(id uint) (*fleet.Query, error) {
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
