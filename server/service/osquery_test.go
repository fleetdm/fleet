package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetClientConfig(t *testing.T) {
	ds := new(mock.Store)
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		return []*fleet.Pack{}, nil
	}
	ds.ListScheduledQueriesInPackFunc = func(ctx context.Context, pid uint) ([]*fleet.ScheduledQuery, error) {
		tru := true
		fals := false
		fortytwo := uint(42)
		switch pid {
		case 1:
			return []*fleet.ScheduledQuery{
				{Name: "time", Query: "select * from time", Interval: 30, Removed: &fals},
			}, nil
		case 4:
			return []*fleet.ScheduledQuery{
				{Name: "foobar", Query: "select 3", Interval: 20, Shard: &fortytwo},
				{Name: "froobing", Query: "select 'guacamole'", Interval: 60, Snapshot: &tru},
			}, nil
		default:
			return []*fleet.ScheduledQuery{}, nil
		}
	}
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":{"options":{"baz":"bar"}}}`))}, nil
	}
	ds.UpdateHostFunc = func(ctx context.Context, host *fleet.Host) error {
		return nil
	}
	ds.HostLiteFunc = func(ctx context.Context, id uint) (*fleet.Host, error) {
		if id != 1 && id != 2 {
			return nil, errors.New("not found")
		}
		return &fleet.Host{ID: id}, nil
	}

	svc := newTestService(ds, nil, nil)

	ctx1 := hostctx.NewContext(context.Background(), &fleet.Host{ID: 1})
	ctx2 := hostctx.NewContext(context.Background(), &fleet.Host{ID: 2})

	expectedOptions := map[string]interface{}{
		"baz": "bar",
	}

	expectedConfig := map[string]interface{}{
		"options": expectedOptions,
	}

	// No packs loaded yet
	conf, err := svc.GetClientConfig(ctx1)
	require.NoError(t, err)
	assert.Equal(t, expectedConfig, conf)

	conf, err = svc.GetClientConfig(ctx2)
	require.NoError(t, err)
	assert.Equal(t, expectedConfig, conf)

	// Now add packs
	ds.ListPacksForHostFunc = func(ctx context.Context, hid uint) ([]*fleet.Pack, error) {
		switch hid {
		case 1:
			return []*fleet.Pack{
				{ID: 1, Name: "pack_by_label"},
				{ID: 4, Name: "pack_by_other_label"},
			}, nil

		case 2:
			return []*fleet.Pack{
				{ID: 1, Name: "pack_by_label"},
			}, nil
		}
		return []*fleet.Pack{}, nil
	}

	conf, err = svc.GetClientConfig(ctx1)
	require.NoError(t, err)
	assert.Equal(t, expectedOptions, conf["options"])
	assert.JSONEq(t, `{
		"pack_by_other_label": {
			"queries": {
				"foobar":{"query":"select 3","interval":20,"shard":42},
				"froobing":{"query":"select 'guacamole'","interval":60,"snapshot":true}
			}
		},
		"pack_by_label": {
			"queries":{
				"time":{"query":"select * from time","interval":30,"removed":false}
			}
		}
	}`,
		string(conf["packs"].(json.RawMessage)),
	)

	conf, err = svc.GetClientConfig(ctx2)
	require.NoError(t, err)
	assert.Equal(t, expectedOptions, conf["options"])
	assert.JSONEq(t, `{
		"pack_by_label": {
			"queries":{
				"time":{"query":"select * from time","interval":30,"removed":false}
			}
		}
	}`,
		string(conf["packs"].(json.RawMessage)),
	)
}

func TestAgentOptionsForHost(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	teamID := uint(1)
	ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{
			AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":{"baz":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override2"}}}}`)),
		}, nil
	}
	ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
		return ptr.RawMessage(json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)), nil
	}

	host := &fleet.Host{
		TeamID:   &teamID,
		Platform: "darwin",
	}

	opt, err := svc.AgentOptionsForHost(context.Background(), host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"override"}`, string(opt))

	host.Platform = "windows"
	opt, err = svc.AgentOptionsForHost(context.Background(), host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"bar"}`, string(opt))

	// Should take gobal option with no team
	host.TeamID = nil
	opt, err = svc.AgentOptionsForHost(context.Background(), host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"baz":"bar"}`, string(opt))

	host.Platform = "darwin"
	opt, err = svc.AgentOptionsForHost(context.Background(), host.TeamID, host.Platform)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"override2"}`, string(opt))
}
