package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query/live_query_mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestIntegrationLiveQueriesTestSuite(t *testing.T) {
	testingSuite := new(liveQueriesTestSuite)
	testingSuite.s = &testingSuite.Suite
	suite.Run(t, testingSuite)
}

type liveQueriesTestSuite struct {
	withServer
	suite.Suite

	lq    *live_query_mock.MockLiveQuery
	hosts []*fleet.Host
}

// SetupTest partially implements suite.SetupTestSuite.
func (s *liveQueriesTestSuite) SetupTest() {
	s.lq.Mock.Test(s.T())
}

// SetupSuite partially implements suite.SetupAllSuite.
func (s *liveQueriesTestSuite) SetupSuite() {
	s.T().Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "5s")

	s.withDS.SetupSuite("liveQueriesTestSuite")

	rs := pubsub.NewInmemQueryResults()
	lq := live_query_mock.New(s.T())
	s.lq = lq

	users, server := RunServerForTestsWithDS(s.T(), s.ds, &TestServerOpts{Lq: lq, Rs: rs})
	s.server = server
	s.users = users
	s.token = getTestAdminToken(s.T(), s.server)

	t := s.T()
	for i := 0; i < 3; i++ {
		host, err := s.ds.NewHost(context.Background(), &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-time.Duration(i) * time.Minute),
			OsqueryHostID:   ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			NodeKey:         ptr.String(fmt.Sprintf("%s%d", t.Name(), i)),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
		})
		require.NoError(s.T(), err)
		s.hosts = append(s.hosts, host)
	}
}

// TearDownTest partially implements suite.TearDownTestSuite.
func (s *liveQueriesTestSuite) TearDownTest() {
	// reset the mock
	s.lq.Mock = mock.Mock{}
}

func (s *liveQueriesTestSuite) TestLiveQueriesRestOneHostOneQuery() {
	t := s.T()

	host := s.hosts[0]

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Query:       "select 1 from osquery;",
		Description: "desc1",
		Name:        t.Name() + "query1",
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", uint(1)).Return(map[string]string{fmt.Sprint(q1.ID): "select 1 from osquery;"}, nil)
	s.lq.On("QueryCompletedByHost", mock.Anything, mock.Anything).Return(nil)
	s.lq.On("RunQuery", mock.Anything, "select 1 from osquery;", []uint{host.ID}).Return(nil)
	s.lq.On("StopQuery", mock.Anything).Return(nil)

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{q1.ID},
		HostIDs:  []uint{host.ID},
	}
	liveQueryResp := runLiveQueryResponse{}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)

	cid := getCIDForQ(s, q1)

	distributedReq := submitDistributedQueryResultsRequestShim{
		NodeKey: *host.NodeKey,
		Results: map[string]json.RawMessage{
			hostDistributedQueryPrefix + cid:          json.RawMessage(`[{"col1": "a", "col2": "b"}]`),
			hostDistributedQueryPrefix + "invalidcid": json.RawMessage(`""`), // empty string is sometimes sent for no results
			hostDistributedQueryPrefix + "9999":       json.RawMessage(`""`),
		},
		Statuses: map[string]interface{}{
			hostDistributedQueryPrefix + cid:    0,
			hostDistributedQueryPrefix + "9999": "0",
		},
		Messages: map[string]string{
			hostDistributedQueryPrefix + cid: "some msg",
		},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	wg.Wait()

	require.Len(t, liveQueryResp.Results, 1)
	assert.Equal(t, 1, liveQueryResp.Summary.RespondedHostCount)
	assert.Equal(t, q1.ID, liveQueryResp.Results[0].QueryID)
	require.Len(t, liveQueryResp.Results[0].Results[0].Rows, 1)
	assert.Equal(t, "a", liveQueryResp.Results[0].Results[0].Rows[0]["col1"])
	assert.Equal(t, "b", liveQueryResp.Results[0].Results[0].Rows[0]["col2"])
}

func (s *liveQueriesTestSuite) TestLiveQueriesRestOneHostMultipleQuery() {
	t := s.T()

	host := s.hosts[0]

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Query:       "select 1 from osquery;",
		Description: "desc1",
		Name:        t.Name() + "query1",
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	q2, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Query:       "select 2 from osquery;",
		Description: "desc2",
		Name:        t.Name() + "query2",
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{
		fmt.Sprint(q1.ID): "select 1 from osquery;",
		fmt.Sprint(q2.ID): "select 2 from osquery;",
	}, nil)
	s.lq.On("QueryCompletedByHost", mock.Anything, mock.Anything).Return(nil)
	s.lq.On("RunQuery", mock.Anything, "select 1 from osquery;", []uint{host.ID}).Return(nil)
	s.lq.On("RunQuery", mock.Anything, "select 2 from osquery;", []uint{host.ID}).Return(nil)
	s.lq.On("StopQuery", mock.Anything).Return(nil)

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{q1.ID, q2.ID},
		HostIDs:  []uint{host.ID},
	}
	liveQueryResp := runLiveQueryResponse{}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)

	cid1 := getCIDForQ(s, q1)
	cid2 := getCIDForQ(s, q2)

	distributedReq := SubmitDistributedQueryResultsRequest{
		NodeKey: *host.NodeKey,
		Results: map[string][]map[string]string{
			hostDistributedQueryPrefix + cid1: {{"col1": "a", "col2": "b"}},
			hostDistributedQueryPrefix + cid2: {{"col3": "c", "col4": "d"}, {"col3": "e", "col4": "f"}},
		},
		Statuses: map[string]fleet.OsqueryStatus{
			hostDistributedQueryPrefix + cid1: 0,
			hostDistributedQueryPrefix + cid2: 0,
		},
		Messages: map[string]string{
			hostDistributedQueryPrefix + cid1: "some msg",
			hostDistributedQueryPrefix + cid2: "some other msg",
		},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	wg.Wait()

	require.Len(t, liveQueryResp.Results, 2)
	assert.Equal(t, 1, liveQueryResp.Summary.RespondedHostCount)

	sort.Slice(liveQueryResp.Results, func(i, j int) bool {
		return liveQueryResp.Results[i].QueryID < liveQueryResp.Results[j].QueryID
	})

	require.True(t, q1.ID < q2.ID)

	assert.Equal(t, q1.ID, liveQueryResp.Results[0].QueryID)
	require.Len(t, liveQueryResp.Results[0].Results, 1)
	q1Results := liveQueryResp.Results[0].Results[0]
	require.Len(t, q1Results.Rows, 1)
	assert.Equal(t, "a", q1Results.Rows[0]["col1"])
	assert.Equal(t, "b", q1Results.Rows[0]["col2"])

	assert.Equal(t, q2.ID, liveQueryResp.Results[1].QueryID)
	require.Len(t, liveQueryResp.Results[1].Results, 1)
	q2Results := liveQueryResp.Results[1].Results[0]
	require.Len(t, q2Results.Rows, 2)
	assert.Equal(t, "c", q2Results.Rows[0]["col3"])
	assert.Equal(t, "d", q2Results.Rows[0]["col4"])
	assert.Equal(t, "e", q2Results.Rows[1]["col3"])
	assert.Equal(t, "f", q2Results.Rows[1]["col4"])
}

func getCIDForQ(s *liveQueriesTestSuite, q1 *fleet.Query) string {
	t := s.T()
	campaigns, err := s.ds.DistributedQueryCampaignsForQuery(context.Background(), q1.ID)
	require.NoError(t, err)
	require.Len(t, campaigns, 1)
	cid1 := fmt.Sprint(campaigns[0].ID)
	return cid1
}

func (s *liveQueriesTestSuite) TestLiveQueriesRestMultipleHostMultipleQuery() {
	t := s.T()

	h1 := s.hosts[0]
	h2 := s.hosts[1]

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Query:       "select 1 from osquery;",
		Description: "desc1",
		Name:        t.Name() + "query1",
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	q2, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Query:       "select 2 from osquery;",
		Description: "desc2",
		Name:        t.Name() + "query2",
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", h1.ID).Return(map[string]string{
		fmt.Sprint(q1.ID): "select 1 from osquery;",
		fmt.Sprint(q2.ID): "select 2 from osquery;",
	}, nil)
	s.lq.On("QueriesForHost", h2.ID).Return(map[string]string{
		fmt.Sprint(q1.ID): "select 1 from osquery;",
		fmt.Sprint(q2.ID): "select 2 from osquery;",
	}, nil)
	s.lq.On("QueryCompletedByHost", mock.Anything, mock.Anything).Return(nil)
	s.lq.On("RunQuery", mock.Anything, "select 1 from osquery;", []uint{h1.ID, h2.ID}).Return(nil)
	s.lq.On("RunQuery", mock.Anything, "select 2 from osquery;", []uint{h1.ID, h2.ID}).Return(nil)
	s.lq.On("StopQuery", mock.Anything).Return(nil)

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{q1.ID, q2.ID},
		HostIDs:  []uint{h1.ID, h2.ID},
	}
	liveQueryResp := runLiveQueryResponse{}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)
	cid1 := getCIDForQ(s, q1)
	cid2 := getCIDForQ(s, q2)
	for i, h := range []*fleet.Host{h1, h2} {
		distributedReq := SubmitDistributedQueryResultsRequest{
			NodeKey: *h.NodeKey,
			Results: map[string][]map[string]string{
				hostDistributedQueryPrefix + cid1: {{"col1": fmt.Sprintf("a%d", i), "col2": fmt.Sprintf("b%d", i)}},
				hostDistributedQueryPrefix + cid2: {{"col3": fmt.Sprintf("c%d", i), "col4": fmt.Sprintf("d%d", i)}, {"col3": fmt.Sprintf("e%d", i), "col4": fmt.Sprintf("f%d", i)}},
			},
			Statuses: map[string]fleet.OsqueryStatus{
				hostDistributedQueryPrefix + cid1: 0,
				hostDistributedQueryPrefix + cid2: 0,
			},
			Messages: map[string]string{
				hostDistributedQueryPrefix + cid1: "some msg",
				hostDistributedQueryPrefix + cid2: "some other msg",
			},
		}
		distributedResp := submitDistributedQueryResultsResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
	}

	wg.Wait()

	require.Len(t, liveQueryResp.Results, 2) // 2 queries
	assert.Equal(t, 2, liveQueryResp.Summary.RespondedHostCount)

	sort.Slice(liveQueryResp.Results, func(i, j int) bool {
		return liveQueryResp.Results[i].QueryID < liveQueryResp.Results[j].QueryID
	})

	require.Equal(t, q1.ID, liveQueryResp.Results[0].QueryID)
	require.Len(t, liveQueryResp.Results[0].Results, 2)
	for i, r := range liveQueryResp.Results[0].Results {
		require.Len(t, r.Rows, 1)
		assert.Equal(t, fmt.Sprintf("a%d", i), r.Rows[0]["col1"])
		assert.Equal(t, fmt.Sprintf("b%d", i), r.Rows[0]["col2"])
	}

	require.Equal(t, q2.ID, liveQueryResp.Results[1].QueryID)
	require.Len(t, liveQueryResp.Results[1].Results, 2)
	for i, r := range liveQueryResp.Results[1].Results {
		require.Len(t, r.Rows, 2)
		assert.Equal(t, fmt.Sprintf("c%d", i), r.Rows[0]["col3"])
		assert.Equal(t, fmt.Sprintf("d%d", i), r.Rows[0]["col4"])
		assert.Equal(t, fmt.Sprintf("e%d", i), r.Rows[1]["col3"])
		assert.Equal(t, fmt.Sprintf("f%d", i), r.Rows[1]["col4"])
	}
}

// TestLiveQueriesInvalidInput without query/host IDs
func (s *liveQueriesTestSuite) TestLiveQueriesInvalidInputs() {
	t := s.T()

	host := s.hosts[0]

	q1, err := s.ds.NewQuery(
		context.Background(), &fleet.Query{
			Query:       "select 1 from osquery;",
			Description: "desc1",
			Name:        t.Name() + "query1",
			Logging:     fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{},
		HostIDs:  []uint{host.ID},
	}
	liveQueryResp := runLiveQueryResponse{}
	s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusBadRequest, &liveQueryResp)

	liveQueryRequest = runLiveQueryRequest{
		QueryIDs: []uint{q1.ID},
		HostIDs:  []uint{},
	}
	s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusBadRequest, &liveQueryResp)

	liveQueryRequest = runLiveQueryRequest{
		QueryIDs: []uint{},
		HostIDs:  []uint{},
	}
	s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusBadRequest, &liveQueryResp)
}

// TestLiveQueriesFailsToAuthorize when an observer tries to run a live query
func (s *liveQueriesTestSuite) TestLiveQueriesFailsToAuthorize() {
	t := s.T()

	host := s.hosts[0]

	q1, err := s.ds.NewQuery(
		context.Background(), &fleet.Query{
			Query:       "select 1 from osquery;",
			Description: "desc1",
			Name:        t.Name() + "query1",
			Logging:     fleet.LoggingSnapshot,
		},
	)
	require.NoError(t, err)

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{q1.ID},
		HostIDs:  []uint{host.ID},
	}
	liveQueryResp := runLiveQueryResponse{}

	// Switch to observer user.
	originalToken := s.token
	s.token = getTestUserToken(t, s.server, "user2")
	defer func() {
		s.token = originalToken
	}()
	s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusForbidden, &liveQueryResp)
}

func (s *liveQueriesTestSuite) TestLiveQueriesRestFailsToCreateCampaign() {
	t := s.T()

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{999},
		HostIDs:  []uint{888},
	}
	liveQueryResp := runLiveQueryResponse{}

	s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)

	require.Len(t, liveQueryResp.Results, 1)
	assert.Equal(t, 0, liveQueryResp.Summary.RespondedHostCount)
	require.NotNil(t, liveQueryResp.Results[0].Error)
	assert.Contains(t, *liveQueryResp.Results[0].Error, "Query 999 was not found in the datastore")
}

func (s *liveQueriesTestSuite) TestLiveQueriesRestFailsOnSomeHost() {
	t := s.T()

	h1 := s.hosts[0]
	h2 := s.hosts[1]

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{
		Query:       "select 1 from osquery;",
		Description: "desc1",
		Name:        t.Name() + "query1",
		Logging:     fleet.LoggingSnapshot,
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", h1.ID).Return(map[string]string{fmt.Sprint(q1.ID): "select 1 from osquery;"}, nil)
	s.lq.On("QueriesForHost", h2.ID).Return(map[string]string{fmt.Sprint(q1.ID): "select 1 from osquery;"}, nil)
	s.lq.On("QueryCompletedByHost", mock.Anything, mock.Anything).Return(nil)
	s.lq.On("RunQuery", mock.Anything, "select 1 from osquery;", []uint{h1.ID, h2.ID}).Return(nil)
	s.lq.On("StopQuery", mock.Anything).Return(nil)

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{q1.ID},
		HostIDs:  []uint{h1.ID, h2.ID},
	}
	liveQueryResp := runLiveQueryResponse{}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.DoJSON("GET", "/api/latest/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)
	cid1 := getCIDForQ(s, q1)
	distributedReq := submitDistributedQueryResultsRequestShim{
		NodeKey: *h1.NodeKey,
		Results: map[string]json.RawMessage{
			hostDistributedQueryPrefix + cid1: json.RawMessage(`[{"col1": "a", "col2": "b"}]`),
		},
		Statuses: map[string]interface{}{
			hostDistributedQueryPrefix + cid1: "0",
		},
		Messages: map[string]string{
			hostDistributedQueryPrefix + cid1: "some msg",
		},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	distributedReq = submitDistributedQueryResultsRequestShim{
		NodeKey: *h2.NodeKey,
		Results: map[string]json.RawMessage{
			hostDistributedQueryPrefix + cid1: json.RawMessage(`""`),
		},
		Statuses: map[string]interface{}{
			hostDistributedQueryPrefix + cid1: 123,
		},
		Messages: map[string]string{
			hostDistributedQueryPrefix + cid1: "some error!",
		},
	}
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	wg.Wait()

	require.Len(t, liveQueryResp.Results, 1)
	assert.Equal(t, 2, liveQueryResp.Summary.RespondedHostCount)

	result := liveQueryResp.Results[0]
	require.Len(t, result.Results, 2)
	require.Len(t, result.Results[0].Rows, 1)
	assert.Equal(t, "a", result.Results[0].Rows[0]["col1"])
	assert.Equal(t, "b", result.Results[0].Rows[0]["col2"])
	require.Len(t, result.Results[1].Rows, 0)
	require.NotNil(t, result.Results[1].Error)
	assert.Equal(t, "some error!", *result.Results[1].Error)
}

func (s *liveQueriesTestSuite) TestCreateDistributedQueryCampaign() {
	t := s.T()

	// NOTE: this only tests creating the campaigns, as running them is tested
	// extensively in other test functions.

	h1 := s.hosts[0]
	h2 := s.hosts[1]
	s.lq.On("RunQuery", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	s.lq.On("StopQuery", mock.Anything).Return(nil)

	// create with no payload
	var createResp createDistributedQueryCampaignResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/run", nil, http.StatusUnprocessableEntity, &createResp)

	// create with unknown query
	req := createDistributedQueryCampaignRequest{
		QueryID: ptr.Uint(9999),
		Selected: fleet.HostTargets{
			HostIDs: []uint{1},
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusNotFound, &createResp)

	// create with no hosts
	req = createDistributedQueryCampaignRequest{
		QuerySQL: "SELECT 1",
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusBadRequest, &createResp)

	// wait a second to prevent duplicate name for new query
	time.Sleep(time.Second)

	// create with new query for specific hosts
	req = createDistributedQueryCampaignRequest{
		QuerySQL: "SELECT 2",
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID, h2.ID},
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusOK, &createResp)
	camp1 := *createResp.Campaign
	assert.Equal(t, uint(2), createResp.Campaign.Metrics.TotalHosts)

	// wait a second to prevent duplicate name for new query
	time.Sleep(time.Second)

	// create by host name
	req2 := createDistributedQueryCampaignByNamesRequest{
		QuerySQL: "SELECT 3",
		Selected: distributedQueryCampaignTargetsByNames{
			Hosts: []string{h1.Hostname},
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run_by_names", req2, http.StatusOK, &createResp)
	assert.NotEqual(t, camp1.ID, createResp.Campaign.ID)
	assert.Equal(t, uint(1), createResp.Campaign.Metrics.TotalHosts)

	// wait a second to prevent duplicate name for new query
	time.Sleep(time.Second)

	// create by unknown host name - it ignores the unknown names. Must have at least 1 valid host
	req2 = createDistributedQueryCampaignByNamesRequest{
		QuerySQL: "SELECT 3",
		Selected: distributedQueryCampaignTargetsByNames{
			Hosts: []string{h1.Hostname, h2.Hostname + "ZZZZZ"},
		},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run_by_names", req2, http.StatusOK, &createResp)
}

func (s *liveQueriesTestSuite) TestOsqueryDistributedRead() {
	t := s.T()

	hostID := s.hosts[1].ID
	s.lq.On("QueriesForHost", hostID).Return(map[string]string{fmt.Sprintf("%d", hostID): "select 1 from osquery;"}, nil)

	req := getDistributedQueriesRequest{NodeKey: *s.hosts[1].NodeKey}
	var resp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &resp)
	assert.Contains(t, resp.Queries, hostDistributedQueryPrefix+fmt.Sprintf("%d", hostID))

	// test with invalid node key
	var errRes map[string]interface{}
	req.NodeKey += "zzzz"
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusUnauthorized, &errRes)
	assert.Contains(t, errRes["error"], "invalid node key")
}

func (s *liveQueriesTestSuite) TestOsqueryDistributedReadWithFeatures() {
	t := s.T()

	spec := []byte(`
  features:
    additional_queries:
      time: SELECT * FROM time
    enable_host_users: true
    enable_software_inventory: true
`)
	s.applyConfig(spec)

	a := json.RawMessage(`{"time": "SELECT * FROM time"}`)
	team, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		ID:          42,
		Name:        "team1",
		Description: "desc team1",
		Config: fleet.TeamConfig{
			Features: fleet.Features{
				EnableHostUsers:         false,
				EnableSoftwareInventory: false,
				AdditionalQueries:       &a,
			},
		},
	})
	require.NoError(t, err)

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	s.lq.On("QueriesForHost", host.ID).Return(map[string]string{fmt.Sprintf("%d", host.ID): "select 1 from osquery;"}, nil)

	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)
	req := getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")

	err = s.ds.AddHostsToTeam(context.Background(), &team.ID, []uint{host.ID})
	require.NoError(t, err)
	err = s.ds.UpdateHostRefetchRequested(context.Background(), host.ID, true)
	require.NoError(t, err)
	req = getDistributedQueriesRequest{NodeKey: *host.NodeKey}
	dqResp = getDistributedQueriesResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_users")
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
}

func (s *liveQueriesTestSuite) TestCreateDistributedQueryCampaignBadRequest() {
	t := s.T()

	host, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   ptr.String(t.Name()),
		NodeKey:         ptr.String(t.Name()),
		UUID:            uuid.New().String(),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)

	// Query provided but no targets provided
	req := createDistributedQueryCampaignRequest{
		QuerySQL: "SELECT * FROM osquery_info;",
	}
	var createResp createDistributedQueryCampaignResponse
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusBadRequest, &createResp)

	// Target provided but no query provided.
	req = createDistributedQueryCampaignRequest{
		Selected: fleet.HostTargets{HostIDs: []uint{host.ID}},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusUnprocessableEntity, &createResp)

	// Query "provided" but empty.
	req = createDistributedQueryCampaignRequest{
		QuerySQL: " ",
		Selected: fleet.HostTargets{HostIDs: []uint{host.ID}},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusUnprocessableEntity, &createResp)

	s.lq.On("RunQuery", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Query and targets provided.
	req = createDistributedQueryCampaignRequest{
		QuerySQL: "select \"With automounting enabled anyone with physical access could attach a USB drive or disc and have its contents available in system even if they lacked permissions to mount it themselves.\" as Rationale;",
		Selected: fleet.HostTargets{HostIDs: []uint{host.ID}},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusOK, &createResp)

	// Disable live queries; should get 403, not 500.
	var acResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	appCfg := fleet.AppConfig{ServerSettings: fleet.ServerSettings{LiveQueryDisabled: true, ServerURL: acResp.ServerSettings.ServerURL}, OrgInfo: fleet.OrgInfo{OrgName: acResp.OrgInfo.OrgName}}
	s.DoRaw("PATCH", "/api/latest/fleet/config", jsonMustMarshal(t, appCfg), http.StatusOK)

	req = createDistributedQueryCampaignRequest{
		QuerySQL: "select \"With automounting enabled anyone with physical access could attach a USB drive or disc and have its contents available in system even if they lacked permissions to mount it themselves.\" as Rationale;",
		Selected: fleet.HostTargets{HostIDs: []uint{host.ID}},
	}
	s.DoJSON("POST", "/api/latest/fleet/queries/run", req, http.StatusForbidden, &createResp)

	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &acResp)
	appCfg = fleet.AppConfig{ServerSettings: fleet.ServerSettings{LiveQueryDisabled: false, ServerURL: acResp.ServerSettings.ServerURL}, OrgInfo: fleet.OrgInfo{OrgName: acResp.OrgInfo.OrgName}}
	s.DoRaw("PATCH", "/api/latest/fleet/config", jsonMustMarshal(t, appCfg), http.StatusOK)
}
