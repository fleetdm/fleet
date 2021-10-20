package service

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/WatchBeam/clock"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/live_query"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/pubsub"
	ws "github.com/fleetdm/fleet/v4/server/websocket"
	kitlog "github.com/go-kit/kit/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestStreamCampaignResultsClosesReditOnWSClose(t *testing.T) {
	t.Skip("Seems to be a bit problematic in CI")

	store := pubsub.SetupRedisForTest(t, false, false)

	mockClock := clock.NewMockClock()
	ds := new(mock.Store)
	lq := new(live_query.MockLiveQuery)
	svc := newTestServiceWithClock(ds, store, lq, mockClock)

	campaign := &fleet.DistributedQueryCampaign{ID: 42}

	ds.LabelQueriesForHostFunc = func(ctx context.Context, host *fleet.Host) (map[string]string, error) {
		return map[string]string{}, nil
	}
	ds.SaveHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{}, nil
	}
	ds.NewQueryFunc = func(ctx context.Context, query *fleet.Query, opts ...fleet.OptionalArg) (*fleet.Query, error) {
		return query, nil
	}
	ds.NewDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) (*fleet.DistributedQueryCampaign, error) {
		return camp, nil
	}
	ds.NewDistributedQueryCampaignTargetFunc = func(ctx context.Context, target *fleet.DistributedQueryCampaignTarget) (*fleet.DistributedQueryCampaignTarget, error) {
		return target, nil
	}
	ds.HostIDsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets) ([]uint, error) {
		return []uint{1}, nil
	}
	ds.CountHostsInTargetsFunc = func(ctx context.Context, filter fleet.TeamFilter, targets fleet.HostTargets, now time.Time) (fleet.TargetMetrics, error) {
		return fleet.TargetMetrics{TotalHosts: 1}, nil
	}
	ds.NewActivityFunc = func(ctx context.Context, user *fleet.User, activityType string, details *map[string]interface{}) error {
		return nil
	}
	ds.SessionByKeyFunc = func(ctx context.Context, key string) (*fleet.Session, error) {
		return &fleet.Session{
			CreateTimestamp: fleet.CreateTimestamp{CreatedAt: time.Now()},
			ID:              42,
			AccessedAt:      time.Now(),
			UserID:          999,
			Key:             "asd",
		}, nil
	}

	host := &fleet.Host{ID: 1, Platform: "windows"}

	lq.On("QueriesForHost", uint(1)).Return(
		map[string]string{
			strconv.Itoa(int(campaign.ID)): "select * from time",
		},
		nil,
	)
	lq.On("QueryCompletedByHost", strconv.Itoa(int(campaign.ID)), host.ID).Return(nil)
	lq.On("RunQuery", "0", "select year, month, day, hour, minutes, seconds from time", []uint{1}).Return(nil)
	viewerCtx := viewer.NewContext(context.Background(), viewer.Viewer{
		User: &fleet.User{
			ID:         0,
			GlobalRole: ptr.String(fleet.RoleAdmin),
		},
	})
	q := "select year, month, day, hour, minutes, seconds from time"
	_, err := svc.NewDistributedQueryCampaign(viewerCtx, q, nil, fleet.HostTargets{HostIDs: []uint{2}, LabelIDs: []uint{1}})
	require.NoError(t, err)

	s := httptest.NewServer(makeStreamDistributedQueryCampaignResultsHandler(svc, kitlog.NewNopLogger()))
	defer s.Close()
	// Convert http://127.0.0.1 to ws://127.0.0.1
	u := "ws" + strings.TrimPrefix(s.URL, "http") + "/api/v1/fleet/results/websocket"

	// Connect to the server
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: true},
	}

	conn, _, err := dialer.Dial(u, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer conn.Close()

	err = conn.WriteJSON(ws.JSONMessage{
		Type: "auth",
		Data: map[string]interface{}{"token": "asd"},
	})
	require.NoError(t, err)

	err = conn.WriteJSON(ws.JSONMessage{
		Type: "select_campaign",
		Data: map[string]interface{}{"campaign_id": campaign.ID},
	})
	require.NoError(t, err)

	ds.MarkSessionAccessedFunc = func(context.Context, *fleet.Session) error {
		return nil
	}
	ds.UserByIDFunc = func(ctx context.Context, id uint) (*fleet.User, error) {
		return &fleet.User{GlobalRole: ptr.String(fleet.RoleAdmin)}, nil
	}
	ds.DistributedQueryCampaignFunc = func(ctx context.Context, id uint) (*fleet.DistributedQueryCampaign, error) {
		return campaign, nil
	}
	ds.SaveDistributedQueryCampaignFunc = func(ctx context.Context, camp *fleet.DistributedQueryCampaign) error {
		return nil
	}
	ds.DistributedQueryCampaignTargetIDsFunc = func(ctx context.Context, id uint) (targets *fleet.HostTargets, err error) {
		return &fleet.HostTargets{HostIDs: []uint{1}}, nil
	}
	ds.QueryFunc = func(ctx context.Context, id uint) (*fleet.Query, error) {
		return &fleet.Query{}, nil
	}

	/*****************************************************************************************/
	/* THE ACTUAL TEST BEGINS HERE                                                           */
	/*****************************************************************************************/
	prevActiveConn := 0
	for prevActiveConn < 3 {
		time.Sleep(2 * time.Second)

		for _, s := range store.Pool().Stats() {
			prevActiveConn = s.ActiveCount
		}
	}

	conn.Close()
	time.Sleep(10 * time.Second)

	newActiveConn := prevActiveConn
	for _, s := range store.Pool().Stats() {
		newActiveConn = s.ActiveCount
	}
	require.Equal(t, prevActiveConn-1, newActiveConn)
}
