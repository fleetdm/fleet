package main

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/fleetdm/fleet/v4/server/service"
	kitlog "github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSavedLiveQuery(t *testing.T) {
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)

	logger := kitlog.NewJSONLogger(os.Stdout)
	logger = level.NewFilter(logger, level.AllowDebug())

	_, ds := runServerWithMockedDS(t, &service.TestServerOpts{
		Rs:     rs,
		Lq:     lq,
		Logger: logger,
	})

	users, err := ds.ListUsersFunc(context.Background(), fleet.UserListOptions{})
	require.NoError(t, err)
	var admin *fleet.User
	for _, user := range users {
		if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin {
			admin = user
		}
	}

	const queryName = "saved-query"
	const queryString = "select 42, * from time"
	query := fleet.Query{
		ID:    42,
		Name:  queryName,
		Query: queryString,
		Saved: true,
	}

	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostIdentifiers []string) ([]uint, error) {
		if len(hostIdentifiers) == 1 && hostIdentifiers[0] == "1234" {
			return []uint{1234}, nil
		}
		return nil, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		return nil, nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.ListQueriesFunc = func(ctx context.Context, opt fleet.ListQueryOptions) ([]*fleet.Query, error) {
		if opt.MatchQuery == queryName {
			return []*fleet.Query{&query}, nil
		}
		return []*fleet.Query{}, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		camp.ID = 321
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	noHostsTargeted := false
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		if noHostsTargeted {
			return nil, nil
		}
		return []uint{1}, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{TotalHosts: 1, OnlineHosts: 1}, nil
	}

	lq.On("QueriesForHost", uint(1)).Return(
		map[string]string{
			"42": queryString,
		},
		nil,
	)
	lq.On("QueryCompletedByHost", "42", 99).Return(nil)
	lq.On("RunQuery", "321", queryString, []uint{1}).Return(nil)

	ds.DistributedQueryCampaignTargetIDsFunc = func(ctx context.Context, id uint) (targets *fleet.HostTargets, err error) {
		return &fleet.HostTargets{HostIDs: []uint{99}}, nil
	}
	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return &fleet.DistributedQueryCampaign{
			ID:     321,
			UserID: admin.ID,
		}, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &query, nil
	}
	ds.IsSavedQueryFunc = func(ctx context.Context, queryID uint) (bool, error) {
		return true, nil
	}
	var GetLiveQueryStatsFuncWg sync.WaitGroup
	GetLiveQueryStatsFuncWg.Add(2)
	ds.GetLiveQueryStatsFunc = func(ctx context.Context, queryID uint, hostIDs []uint) ([]*fleet.LiveQueryStats, error) {
		stats := []*fleet.LiveQueryStats{
			{
				LastExecuted: time.Now(),
			},
		}
		GetLiveQueryStatsFuncWg.Done()
		return stats, nil
	}
	var UpdateLiveQueryStatsFuncWg sync.WaitGroup
	UpdateLiveQueryStatsFuncWg.Add(1)
	ds.UpdateLiveQueryStatsFunc = func(ctx context.Context, queryID uint, stats []*fleet.LiveQueryStats) error {
		UpdateLiveQueryStatsFuncWg.Done()
		return nil
	}
	var CalculateAggregatedPerfStatsPercentilesFuncWg sync.WaitGroup
	CalculateAggregatedPerfStatsPercentilesFuncWg.Add(1)
	ds.CalculateAggregatedPerfStatsPercentilesFunc = func(ctx context.Context, aggregate fleet.AggregatedStatsType, queryID uint) error {
		CalculateAggregatedPerfStatsPercentilesFuncWg.Done()
		return nil
	}

	go func() {
		time.Sleep(2 * time.Second)
		require.NoError(t, rs.WriteResult(
			fleet.DistributedQueryResult{
				DistributedQueryCampaignID: 321,
				Rows:                       []map[string]string{{"bing": "fds"}},
				Host: fleet.ResultHostData{
					ID:          99,
					Hostname:    "somehostname",
					DisplayName: "somehostname",
				},
				Stats: &fleet.Stats{
					WallTimeMs: 10,
					UserTime:   20,
					SystemTime: 30,
					Memory:     40,
				},
			},
		))
	}()

	// errors before requesting live query
	_, err = runAppNoChecks([]string{"query", "--hosts", "", "--query-name", queryName})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No hosts or labels targeted")

	expected := `{"host":"somehostname","rows":[{"bing":"fds","host_display_name":"somehostname","host_hostname":"somehostname"}]}
`
	// Note: runAppForTest never closes the WebSocket connection and does not exit,
	// so we are unable to see the activity data that is written after WebSocket disconnects.
	assert.Equal(t, expected, runAppForTest(t, []string{"query", "--hosts", "1234", "--query-name", queryName}))

	// We need to use waitGroups to detect whether Database functions were called because this is an asynchronous test which will flag data races otherwise.
	c := make(chan struct{})
	go func() {
		defer close(c)
		GetLiveQueryStatsFuncWg.Wait()
		UpdateLiveQueryStatsFuncWg.Wait()
		CalculateAggregatedPerfStatsPercentilesFuncWg.Wait()
	}()
	select {
	case <-time.After(time.Second):
		require.Fail(
			t,
			"Expected invocation of one of these Database functions did not happen: GetLiveQueryStats, UpdateLiveQueryStats, or CalculateAggregatedPerfStatsPercentiles",
		)
	case <-c: // All good
	}

	// Test targeting no hosts (e.g. host does exist)
	noHostsTargeted = true
	_, err = runAppNoChecks([]string{"query", "--hosts", "foobar", "--query-name", queryName})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "No hosts targeted")
}

func TestAdHocLiveQuery(t *testing.T) {
	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(t)

	logger := kitlog.NewJSONLogger(os.Stdout)
	logger = level.NewFilter(logger, level.AllowDebug())

	_, ds := runServerWithMockedDS(
		t, &service.TestServerOpts{
			Rs:     rs,
			Lq:     lq,
			Logger: logger,
		},
	)

	users, err := ds.ListUsersFunc(context.Background(), fleet.UserListOptions{})
	require.NoError(t, err)
	var admin *fleet.User
	for _, user := range users {
		if user.GlobalRole != nil && *user.GlobalRole == fleet.RoleAdmin {
			admin = user
		}
	}

	ds.HostIDsByIdentifierFunc = func(ctx context.Context, filter fleet.TeamFilter, hostIdentifiers []string) ([]uint, error) {
		return []uint{1234}, nil
	}
	ds.LabelIDsByNameFunc = func(ctx context.Context, labels []string) (map[string]uint, error) {
		return map[string]uint{"label1": uint(1)}, nil
	}

	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		query.ID = 42
		return query, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (
		*fleet.DistributedQueryCampaign, error,
	) {
		camp.ID = 321
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(
		ctx context.Context, target *fleet.DistributedQueryCampaignTarget,
	) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1}, nil
	}
	ds.CountHostsInTargetsFunc = func(
		ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time,
	) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{TotalHosts: 1, OnlineHosts: 1}, nil
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
		return &fleet.DistributedQueryCampaign{
			ID:     321,
			UserID: admin.ID,
		}, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) error {
		return nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}
	ds.IsSavedQueryFunc = func(ctx context.Context, queryID uint) (bool, error) {
		return false, nil
	}

	go func() {
		time.Sleep(2 * time.Second)
		require.NoError(
			t, rs.WriteResult(
				fleet.DistributedQueryResult{
					DistributedQueryCampaignID: 321,
					Rows:                       []map[string]string{{"bing": "fds"}},
					Host: fleet.ResultHostData{
						ID:          99,
						Hostname:    "somehostname",
						DisplayName: "somehostname",
					},
					Stats: &fleet.Stats{
						WallTimeMs: 10,
						UserTime:   20,
						SystemTime: 30,
						Memory:     40,
					},
				},
			),
		)
	}()

	// test label not found
	_, err = runAppNoChecks([]string{"query", "--hosts", "1234", "--labels", "iamnotalabel", "--query", "select 42, * from time"})
	assert.Error(t, err)
	assert.Equal(t, "[iamnotalabel] are not valid label names, remove to continue.", err.Error())

	// test if some labels were not found
	_, err = runAppNoChecks([]string{"query", "--labels", "label1, mac, windows", "--hosts", "1234", "--query", "select 42, * from time"})
	assert.Error(t, err)
	assert.Equal(t, "[mac, windows] are not valid label names, remove to continue.", err.Error())

	expected := `{"host":"somehostname","rows":[{"bing":"fds","host_display_name":"somehostname","host_hostname":"somehostname"}]}
`
	assert.Equal(t, expected, runAppForTest(t, []string{"query", "--labels", "", "--hosts", "1234", "--query", "select 42, * from time"}))
}
