package wstransport

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/client"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeDistributedClient struct {
	readResp   *client.DistributedReadResponse
	readCalls  int
	writeReq   *client.DistributedWriteRequest
	writeCalls int
}

func (f *fakeDistributedClient) DistributedRead() (*client.DistributedReadResponse, error) {
	f.readCalls++
	return f.readResp, nil
}

func (f *fakeDistributedClient) DistributedWrite(req *client.DistributedWriteRequest) error {
	f.writeCalls++
	f.writeReq = req
	return nil
}

// newPluginManager returns a Manager suitable for exercising the distributed
// plugin's state interactions without running the connection loops.
func newPluginManager(fc distributedClient) *Manager {
	return NewManager(Options{Client: fc, Cache: NewQueryCache()})
}

func TestDistributedPluginGetQueries(t *testing.T) {
	fc := &fakeDistributedClient{}
	m := newPluginManager(fc)
	plugin := NewDistributedPlugin(m, fc)

	assert.Equal(t, DistributedPluginName, plugin.Name())
	assert.Equal(t, "distributed", plugin.RegistryName())

	// The plugin answers from the cache; an empty cache yields no queries and
	// no network calls.
	resp := plugin.Call(context.Background(), map[string]string{"action": "getQueries"})
	require.Equal(t, int32(0), resp.Status.Code, resp.Status.Message)
	assert.JSONEq(t, `{"queries":{}}`, resp.Response[0]["results"])
	assert.Zero(t, fc.readCalls)

	m.opts.Cache.Set(&client.DistributedReadResponse{
		Queries:    map[string]string{"q1": "SELECT 1"},
		Discovery:  map[string]string{"q1": "SELECT 1"},
		Accelerate: 10,
	})
	resp = plugin.Call(context.Background(), map[string]string{"action": "getQueries"})
	require.Equal(t, int32(0), resp.Status.Code, resp.Status.Message)
	assert.JSONEq(t,
		`{"queries":{"q1":"SELECT 1"},"discovery":{"q1":"SELECT 1"},"accelerate":10}`,
		resp.Response[0]["results"])
}

func TestDistributedPluginWriteResults(t *testing.T) {
	fc := &fakeDistributedClient{}
	m := newPluginManager(fc)
	plugin := NewDistributedPlugin(m, fc)

	// Walk an iteration into awaiting-write so the write closes it.
	m.mu.Lock()
	m.state = iterAwaitingWrite
	m.mu.Unlock()

	results := `{"queries":{"q1":[{"col":"val"}],"q2":[]},"statuses":{"q1":"0","q2":"1"},"messages":{"q2":"query failed"},"stats":{"q1":{"wall_time_ms":5,"user_time":2,"system_time":1,"memory":1024}}}`
	resp := plugin.Call(context.Background(), map[string]string{"action": "writeResults", "results": results})
	require.Equal(t, int32(0), resp.Status.Code, resp.Status.Message)

	state, _ := m.stateSnapshot()
	assert.Equal(t, iterIdle, state, "completed write must close the iteration")

	require.Equal(t, 1, fc.writeCalls)
	req := fc.writeReq
	assert.Equal(t, []map[string]string{{"col": "val"}}, req.Results["q1"])
	assert.Equal(t, []map[string]string{}, req.Results["q2"])
	assert.Equal(t, fleet.OsqueryStatus(0), req.Statuses["q1"])
	assert.Equal(t, fleet.OsqueryStatus(1), req.Statuses["q2"])
	assert.Equal(t, "query failed", req.Messages["q2"])
	require.NotNil(t, req.Stats["q1"])
	assert.Equal(t, uint64(5), req.Stats["q1"].WallTimeMs)
	assert.Equal(t, uint64(1024), req.Stats["q1"].Memory)
}

func TestDistributedPluginWriteResultsEmpty(t *testing.T) {
	// Results can legitimately be empty (everything discovery-filtered); no
	// HTTP write happens, but the pass — and its iteration — still ends.
	fc := &fakeDistributedClient{}
	m := newPluginManager(fc)
	plugin := NewDistributedPlugin(m, fc)

	m.mu.Lock()
	m.state = iterAwaitingWrite
	m.mu.Unlock()

	resp := plugin.Call(context.Background(), map[string]string{"action": "writeResults", "results": `{"queries":{},"statuses":{}}`})
	require.Equal(t, int32(0), resp.Status.Code, resp.Status.Message)
	assert.Zero(t, fc.writeCalls)

	state, _ := m.stateSnapshot()
	assert.Equal(t, iterIdle, state, "zero-result write must still close the iteration")
}
