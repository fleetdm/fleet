package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/pubsub"
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

	lq    *live_query.MockLiveQuery
	hosts []*fleet.Host
}

func (s *liveQueriesTestSuite) SetupSuite() {
	require.NoError(s.T(), os.Setenv("FLEET_LIVE_QUERY_REST_PERIOD", "5s"))

	s.withDS.SetupSuite("liveQueriesTestSuite")

	rs := pubsub.NewInmemQueryResults()
	lq := new(live_query.MockLiveQuery)
	s.lq = lq

	users, server := RunServerForTestsWithDS(s.T(), s.ds, TestServerOpts{Lq: lq, Rs: rs})
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
			OsqueryHostID:   fmt.Sprintf("%s%d", t.Name(), i),
			NodeKey:         fmt.Sprintf("%s%d", t.Name(), i),
			UUID:            fmt.Sprintf("%s%d", t.Name(), i),
			Hostname:        fmt.Sprintf("%sfoo.local%d", t.Name(), i),
		})
		require.NoError(s.T(), err)
		s.hosts = append(s.hosts, host)
	}
}

func (s *liveQueriesTestSuite) TestLiveQueriesRestOneHostOneQuery() {
	t := s.T()

	host := s.hosts[0]

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{Query: "select 1 from osquery;", Description: "desc1", Name: t.Name() + "query1"})
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
		s.DoJSON("GET", "/api/v1/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)

	cid := getCIDForQ(s, q1)

	distributedReq := SubmitDistributedQueryResultsRequest{
		NodeKey: host.NodeKey,
		Results: map[string][]map[string]string{
			hostDistributedQueryPrefix + cid: {{"col1": "a", "col2": "b"}},
		},
		Statuses: map[string]fleet.OsqueryStatus{
			hostDistributedQueryPrefix + cid: 0,
		},
		Messages: map[string]string{
			hostDistributedQueryPrefix + cid: "some msg",
		},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/v1/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

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

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{Query: "select 1 from osquery;", Description: "desc1", Name: t.Name() + "query1"})
	require.NoError(t, err)

	q2, err := s.ds.NewQuery(context.Background(), &fleet.Query{Query: "select 2 from osquery;", Description: "desc2", Name: t.Name() + "query2"})
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
		s.DoJSON("GET", "/api/v1/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)

	cid1 := getCIDForQ(s, q1)
	cid2 := getCIDForQ(s, q2)

	distributedReq := SubmitDistributedQueryResultsRequest{
		NodeKey: host.NodeKey,
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
	s.DoJSON("POST", "/api/v1/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

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

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{Query: "select 1 from osquery;", Description: "desc1", Name: t.Name() + "query1"})
	require.NoError(t, err)

	q2, err := s.ds.NewQuery(context.Background(), &fleet.Query{Query: "select 2 from osquery;", Description: "desc2", Name: t.Name() + "query2"})
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
		s.DoJSON("GET", "/api/v1/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)
	cid1 := getCIDForQ(s, q1)
	cid2 := getCIDForQ(s, q2)
	for i, h := range []*fleet.Host{h1, h2} {
		distributedReq := SubmitDistributedQueryResultsRequest{
			NodeKey: h.NodeKey,
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
		s.DoJSON("POST", "/api/v1/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)
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

func (s *liveQueriesTestSuite) TestLiveQueriesRestFailsToCreateCampaign() {
	t := s.T()

	liveQueryRequest := runLiveQueryRequest{
		QueryIDs: []uint{999},
		HostIDs:  []uint{888},
	}
	liveQueryResp := runLiveQueryResponse{}

	s.DoJSON("GET", "/api/v1/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)

	require.Len(t, liveQueryResp.Results, 1)
	assert.Equal(t, 0, liveQueryResp.Summary.RespondedHostCount)
	require.NotNil(t, liveQueryResp.Results[0].Error)
	assert.Contains(t, *liveQueryResp.Results[0].Error, "selecting query: ")
	assert.Contains(t, *liveQueryResp.Results[0].Error, "sql: no rows in result set")
}

func (s *liveQueriesTestSuite) TestLiveQueriesRestFailsOnSomeHost() {
	t := s.T()

	h1 := s.hosts[0]
	h2 := s.hosts[1]

	q1, err := s.ds.NewQuery(context.Background(), &fleet.Query{Query: "select 1 from osquery;", Description: "desc1", Name: t.Name() + "query1"})
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
		s.DoJSON("GET", "/api/v1/fleet/queries/run", liveQueryRequest, http.StatusOK, &liveQueryResp)
	}()

	// Give the above call a couple of seconds to create the campaign
	time.Sleep(2 * time.Second)
	cid1 := getCIDForQ(s, q1)
	distributedReq := SubmitDistributedQueryResultsRequest{
		NodeKey: h1.NodeKey,
		Results: map[string][]map[string]string{
			hostDistributedQueryPrefix + cid1: {{"col1": "a", "col2": "b"}},
		},
		Statuses: map[string]fleet.OsqueryStatus{
			hostDistributedQueryPrefix + cid1: 0,
		},
		Messages: map[string]string{
			hostDistributedQueryPrefix + cid1: "some msg",
		},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/v1/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	distributedReq = SubmitDistributedQueryResultsRequest{
		NodeKey: h2.NodeKey,
		Results: map[string][]map[string]string{
			hostDistributedQueryPrefix + cid1: {},
		},
		Statuses: map[string]fleet.OsqueryStatus{
			hostDistributedQueryPrefix + cid1: 123,
		},
		Messages: map[string]string{
			hostDistributedQueryPrefix + cid1: "some error!",
		},
	}
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/v1/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

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
