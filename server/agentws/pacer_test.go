package agentws

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnCoalescesPendingByType(t *testing.T) {
	c := &conn{
		pending:        make(map[string]string),
		wake:           make(chan struct{}, 1),
		notifiedByType: make(map[string]int64),
	}

	// More notifications than the old drop-oldest buffer held: none is lost,
	// and each type is delivered once with its latest reason.
	for range 20 {
		c.enqueue(fleet.AgentWSMessage{Type: fleet.AgentWSMessageTypeDistributedRead, Reason: fleet.AgentWSReasonLabel})
	}
	c.enqueue(fleet.AgentWSMessage{Type: fleet.AgentWSMessageTypeOrbitConfig, Reason: fleet.AgentWSReasonActivity})
	c.enqueue(fleet.AgentWSMessage{Type: fleet.AgentWSMessageTypeDistributedRead, Reason: fleet.AgentWSReasonDetail})

	assert.Equal(t, []fleet.AgentWSMessage{
		{Type: fleet.AgentWSMessageTypeDistributedRead, Reason: fleet.AgentWSReasonDetail},
		{Type: fleet.AgentWSMessageTypeOrbitConfig, Reason: fleet.AgentWSReasonActivity},
	}, c.takePending())
	assert.Empty(t, c.takePending())

	assert.Equal(t, int64(22), c.notified.Load())
	assert.Equal(t, int64(20), c.coalesced.Load())
	assert.Equal(t, map[string]int64{
		fleet.AgentWSMessageTypeDistributedRead: 21,
		fleet.AgentWSMessageTypeOrbitConfig:     1,
	}, c.notifiedByTypeSnapshot())
	assert.Len(t, c.wake, 1)
}

func TestPacerSpreadsAndMerges(t *testing.T) {
	p := newPacer(NewHub(discardLogger(), time.Minute, 30*time.Second), time.Second)

	ids := make([]uint, 100)
	for i := range ids {
		ids[i] = uint(i + 1)
	}
	require.Equal(t, 100, p.add(fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, ids))
	// Already queued hosts are merged, keeping the latest reason.
	require.Equal(t, 0, p.add(fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonResync, ids[:10]))
	require.Equal(t, 100, p.queueLen())

	// A tick releases its share of the remaining window, FIFO.
	start := p.deadline.Add(-time.Second)
	batch := p.next(start)
	require.Len(t, batch, 10)
	for _, id := range ids[:10] {
		assert.Equal(t, fleet.AgentWSReasonResync, batch[pacedKey{hostID: id, msgType: fleet.AgentWSMessageTypeOrbitConfig}])
	}
	// Halfway through the window, 90 targets left over 500ms: 18 per tick.
	require.Len(t, p.next(start.Add(500*time.Millisecond)), 18)
	// Once the deadline is reached, everything left is released.
	require.Len(t, p.next(start.Add(2*time.Second)), 72)
	require.Zero(t, p.queueLen())

	// A released host can be queued again.
	require.Equal(t, 1, p.add(fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, ids[:1]))
}

func TestPacerWithoutSpreadReleasesAtOnce(t *testing.T) {
	p := newPacer(NewHub(discardLogger(), time.Minute, 30*time.Second), 0)
	p.add(fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, []uint{1, 2, 3})
	require.Len(t, p.next(time.Now()), 3)
}

type fakeHostLister struct {
	teams map[uint]*uint // host ID → team ID
	err   error
	calls [][]uint
}

func (f *fakeHostLister) ListHostsLiteByIDs(ctx context.Context, ids []uint) ([]*fleet.Host, error) {
	f.calls = append(f.calls, ids)
	if f.err != nil {
		return nil, f.err
	}
	var hosts []*fleet.Host
	for _, id := range ids {
		if teamID, ok := f.teams[id]; ok {
			hosts = append(hosts, &fleet.Host{ID: id, TeamID: teamID})
		}
	}
	return hosts, nil
}

func readMsg(t *testing.T, ws *websocket.Conn, within time.Duration) (fleet.AgentWSMessage, error) {
	t.Helper()
	require.NoError(t, ws.SetReadDeadline(time.Now().Add(within)))
	var msg fleet.AgentWSMessage
	err := ws.ReadJSON(&msg)
	return msg, err
}

func TestHubNotifyScope(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)

	_, err := hub.NotifyScope(t.Context(), fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, fleet.AgentNotificationScopeGlobal)
	require.Error(t, err, "scoped notifications not enabled")

	ws1 := dial(t, srv, "key-1")
	ws2 := dial(t, srv, "key-2")
	waitForConnCount(t, hub, 2)

	// Host 1 is in fleet 7, host 2 in no fleet.
	lister := &fakeHostLister{teams: map[uint]*uint{1: new(uint(7)), 2: nil}}
	hub.EnableScopedNotifications(t.Context(), lister, 1, 200*time.Millisecond)

	n, err := hub.NotifyScope(t.Context(), fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, fleet.AgentNotificationScopeTeam(7))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	// Held hosts are loaded in batches of 1.
	require.Len(t, lister.calls, 2)

	msg, err := readMsg(t, ws1, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, fleet.AgentWSMessage{Type: fleet.AgentWSMessageTypeOrbitConfig, Reason: fleet.AgentWSReasonSettings}, msg)
	// Host 2 is not in fleet 7. (A read timeout would break ws2, so check
	// the hub's counters instead.)
	time.Sleep(300 * time.Millisecond)
	notified := make(map[uint]int64)
	for _, info := range hub.Snapshot() {
		notified[info.HostID] = info.NotifiedCount
	}
	assert.Equal(t, map[uint]int64{1: 1, 2: 0}, notified)

	// Fleet 0 is "No fleet".
	n, err = hub.NotifyScope(t.Context(), fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, fleet.AgentNotificationScopeTeam(0))
	require.NoError(t, err)
	require.Equal(t, 1, n)
	msg, err = readMsg(t, ws2, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, fleet.AgentWSMessage{Type: fleet.AgentWSMessageTypeOrbitConfig, Reason: fleet.AgentWSReasonSettings}, msg)

	n, err = hub.NotifyScope(t.Context(), fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonResync, fleet.AgentNotificationScopeGlobal)
	require.NoError(t, err)
	require.Equal(t, 2, n)
	for _, ws := range []*websocket.Conn{ws1, ws2} {
		msg, err := readMsg(t, ws, 2*time.Second)
		require.NoError(t, err)
		assert.Equal(t, fleet.AgentWSReasonResync, msg.Reason)
	}
	require.Zero(t, hub.PacerQueueLen())
	assert.Equal(t, map[string]int64{fleet.AgentWSReasonSettings: 2, fleet.AgentWSReasonResync: 2},
		hub.NotifyCounts()[fleet.AgentWSMessageTypeOrbitConfig])

	lister.err = errors.New("db down")
	_, err = hub.NotifyScope(t.Context(), fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, fleet.AgentNotificationScopeTeam(7))
	require.Error(t, err)

	_, err = hub.NotifyScope(t.Context(), fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonSettings, "bogus")
	require.Error(t, err)
}

func TestAgentNotificationScopeTeamID(t *testing.T) {
	id, ok := fleet.AgentNotificationScopeTeam(42).TeamID()
	require.True(t, ok)
	require.Equal(t, uint(42), id)

	for _, s := range []fleet.AgentNotificationScope{fleet.AgentNotificationScopeGlobal, "team:", "team:x", "team:-1", ""} {
		_, ok := s.TeamID()
		require.False(t, ok, s)
	}
}

func TestHubScopeWorkerLoadsHostsOnce(t *testing.T) {
	hub := NewHub(discardLogger(), time.Minute, 30*time.Second)
	srv := newTestServer(t, hub)
	ws1 := dial(t, srv, "key-1")
	ws2 := dial(t, srv, "key-2")
	waitForConnCount(t, hub, 2)

	lister := &fakeHostLister{teams: map[uint]*uint{1: new(uint(7)), 2: new(uint(8))}}
	hub.EnableScopedNotifications(t.Context(), lister, 10, 0)

	// Fleet scopes resolved together load the held hosts once.
	n, err := hub.notifyScopes(t.Context(), []ScopedNotification{
		{MsgType: fleet.AgentWSMessageTypeOrbitConfig, Reason: fleet.AgentWSReasonSettings, Scope: fleet.AgentNotificationScopeTeam(7)},
		{MsgType: fleet.AgentWSMessageTypeOrbitConfig, Reason: fleet.AgentWSReasonSettings, Scope: fleet.AgentNotificationScopeTeam(8)},
	})
	require.NoError(t, err)
	require.Equal(t, 2, n)
	require.Len(t, lister.calls, 1)
	for _, ws := range []*websocket.Conn{ws1, ws2} {
		msg, err := readMsg(t, ws, 2*time.Second)
		require.NoError(t, err)
		assert.Equal(t, fleet.AgentWSMessageTypeOrbitConfig, msg.Type)
	}

	// QueueScope goes through the worker.
	hub.QueueScope(ScopedNotification{MsgType: fleet.AgentWSMessageTypeOrbitConfig, Reason: fleet.AgentWSReasonResync, Scope: fleet.AgentNotificationScopeGlobal})
	for _, ws := range []*websocket.Conn{ws1, ws2} {
		msg, err := readMsg(t, ws, 2*time.Second)
		require.NoError(t, err)
		assert.Equal(t, fleet.AgentWSReasonResync, msg.Reason)
	}

	// NotifyPaced only queues held hosts.
	hub.NotifyPaced(fleet.AgentWSMessageTypeOrbitConfig, fleet.AgentWSReasonActivity, []uint{2, 99})
	msg, err := readMsg(t, ws2, 2*time.Second)
	require.NoError(t, err)
	assert.Equal(t, fleet.AgentWSReasonActivity, msg.Reason)
}
