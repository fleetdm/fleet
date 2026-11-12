package service

// File carving tests for the core (no-license) suite.
//
// Belongs here: beginning a carve, uploading carve blocks, listing and fetching
// carve metadata, and the unauthenticated carve path used by osquery.

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationTestSuite) TestListGetCarves() {
	t := s.T()

	ctx := context.Background()

	hosts := s.createHosts(t)
	c1, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[0].ID,
		Name:      t.Name() + "_1",
		SessionId: "ssn1",
	})
	require.NoError(t, err)
	c2, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[1].ID,
		Name:      t.Name() + "_2",
		SessionId: "ssn2",
	})
	require.NoError(t, err)
	c3, err := s.ds.NewCarve(ctx, &fleet.CarveMetadata{
		CreatedAt: time.Now(),
		HostId:    hosts[2].ID, // nolint:nilaway // createHosts always returns at least one host
		Name:      t.Name() + "_3",
		SessionId: "ssn3",
	})
	require.NoError(t, err)

	// set c1 max block
	c1.MaxBlock = 3
	require.NoError(t, s.ds.UpdateCarve(ctx, c1))
	// make c2 expired, set max block
	c2.Expired = true
	c2.MaxBlock = 3
	require.NoError(t, s.ds.UpdateCarve(ctx, c2))

	var listResp fleet.ListCarvesResponse
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c3.ID, listResp.Carves[1].ID)

	// with 'after' param
	s.DoJSON(
		"GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "after",
		strconv.FormatInt(c1.ID, 10),
	)
	require.Len(t, listResp.Carves, 1)
	assert.Equal(t, c3.ID, listResp.Carves[0].ID)

	// include expired
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "per_page", "2", "order_key", "id", "expired", "1")
	require.Len(t, listResp.Carves, 2)
	assert.Equal(t, c1.ID, listResp.Carves[0].ID)
	assert.Equal(t, c2.ID, listResp.Carves[1].ID)

	// empty page
	s.DoJSON("GET", "/api/latest/fleet/carves", nil, http.StatusOK, &listResp, "page", "3", "per_page", "2", "order_key", "id", "expired", "1")
	require.Empty(t, listResp.Carves)

	// get specific carve
	var getResp fleet.GetCarveResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", c2.ID), nil, http.StatusOK, &getResp)
	require.Equal(t, c2.ID, getResp.Carve.ID)
	require.True(t, getResp.Carve.Expired)

	// get non-existing carve
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", c3.ID+1), nil, http.StatusNotFound, &getResp)

	// get expired carve block
	var blkResp fleet.GetCarveBlockResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c2.ID, 1), nil, http.StatusInternalServerError, &blkResp)

	// get valid carve block, but block not inserted yet
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusNotFound, &blkResp)

	require.NoError(t, s.ds.NewBlock(ctx, c1, 1, []byte("block1")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 2, []byte("block2")))
	require.NoError(t, s.ds.NewBlock(ctx, c1, 3, []byte("block3")))

	// get valid carve block
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d/block/%d", c1.ID, 1), nil, http.StatusOK, &blkResp)
	require.Equal(t, "block1", string(blkResp.Data))
}

func (s *integrationTestSuite) TestCarve() {
	t := s.T()
	hosts := s.createHosts(t)

	// begin a carve with an invalid node key
	var errRes map[string]any
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey + "zzz",
		BlockCount: 1,
		BlockSize:  1,
		CarveSize:  1,
		CarveId:    "c1",
	}, http.StatusUnauthorized, &errRes)
	assert.Contains(t, errRes["error"], "invalid node key")

	// invalid carve size
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  0,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size must be greater")

	// invalid block size too big
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  maxBlockSize + 1,
		CarveSize:  maxCarveSize,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "block_size exceeds max")

	// invalid carve size too big
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  maxBlockSize,
		CarveSize:  maxCarveSize + 1,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size exceeds max")

	// invalid carve size, does not match blocks
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  1,
		CarveId:    "c1",
	}, http.StatusInternalServerError, &errRes) // TODO: should be 4xx, see #4406
	assert.Contains(t, errRes["error"], "carve_size does not match")

	// valid carve begin
	var beginResp fleet.CarveBeginResponse
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *hosts[0].NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  8,
		CarveId:    "c1",
		RequestId:  "r1",
	}, http.StatusOK, &beginResp)
	require.NotEmpty(t, beginResp.SessionId)
	sid := beginResp.SessionId

	// sending a block with invalid session id
	var blockResp fleet.CarveBlockResponse
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid + "zz",
		RequestId: "??",
		Data:      []byte("p1."),
	}, http.StatusUnauthorized, &blockResp)

	// sending a block with valid session id but invalid request id
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "??",
		Data:      []byte("p1."),
	}, http.StatusUnauthorized, &blockResp)

	checkCarveError := func(id uint, err string) {
		var getResp fleet.GetCarveResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", id), nil, http.StatusOK, &getResp)
		require.Equal(t, err, *getResp.Carve.Error)
	}

	// sending a block with unexpected block id (expects 0, got 1)
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p1."),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "block_id does not match expected block (0): 1")

	// sending a block with valid payload, block 0
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   0,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p1."),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending next block
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p2."),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending already-sent block again
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   1,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p2."),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "block_id does not match expected block (2): 1")

	// sending final block with too many bytes
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   2,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p3extra"),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "exceeded declared block size 3: 7")

	// sending actual final block
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   2,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p3"),
	}, http.StatusOK, &blockResp)
	require.True(t, blockResp.Success)

	// sending unexpected block
	blockResp = fleet.CarveBlockResponse{}
	s.DoJSON("POST", "/api/osquery/carve/block", fleet.CarveBlockRequest{
		BlockId:   3,
		SessionId: sid,
		RequestId: "r1",
		Data:      []byte("p4."),
	}, http.StatusBadRequest, &blockResp)
	checkCarveError(1, "block_id exceeds expected max (2): 3")
}

func (s *integrationTestSuite) TestCarveUnauthenticated() {
	t := s.T()

	verifyAuthError := func(t *testing.T, res *http.Response) {
		var errs validationErrResp
		err := json.NewDecoder(res.Body).Decode(&errs)
		require.NoError(t, err)
		res.Body.Close()
		assert.Equal(t, "Authentication failed", errs.Message)
		require.Len(t, errs.Errors, 1)
		assert.Equal(t, "Authentication failed", errs.Errors[0].Reason)
	}

	// Sending invalid format for data on purpose on purpose to check that the error is a HTTP 401 error
	// vs a decoding/parsing error (this way we check it never gets to parse "data").
	for _, tc := range []struct {
		testName       string
		rawJSONRequest string
	}{
		{
			testName:       "empty-json",
			rawJSONRequest: `{}`,
		},
		{
			testName: "with-spaces", // osquery does not send spaces in the JSON
			rawJSONRequest: `{
				"block_id":   1,
				"request_id": "invalid",
				"data":      9999999999
			}`,
		},
		{
			testName:       "without-session-id",
			rawJSONRequest: `{"block_id":1,"request_id":"invalid","data":9999999999}`,
		},
		{
			testName:       "invalid-session-id-format",
			rawJSONRequest: `{"block_id":1,"session_id":2,"request_id": "invalid","data":9999999999}`,
		},
		{
			testName:       "invalid-session-id",
			rawJSONRequest: `{"block_id":1,"session_id":"invalid","request_id":"invalid","data":9999999999}`,
		},
		{
			testName:       "invalid-JSON",
			rawJSONRequest: `{"block_ASDASDASDASDASDASDASDASDASDASDASDASDASD":1}`,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			res := s.DoRaw("POST", "/api/osquery/carve/block", []byte(tc.rawJSONRequest), http.StatusUnauthorized)
			verifyAuthError(t, res)
		})
	}
}
