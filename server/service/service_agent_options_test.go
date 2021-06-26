package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentOptionsForHost(t *testing.T) {
	ds := new(mock.Store)
	svc := newTestService(ds, nil, nil)

	teamID := uint(1)
	ds.TeamFunc = func(tid uint) (*fleet.Team, error) {
		assert.Equal(t, teamID, tid)
		opt := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
		return &fleet.Team{AgentOptions: &opt}, nil
	}
	ds.AppConfigFunc = func() (*fleet.AppConfig, error) {
		return &fleet.AppConfig{AgentOptions: ptr.RawMessage(json.RawMessage(`{"config":{"baz":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override2"}}}}`))}, nil
	}

	host := &fleet.Host{
		TeamID:   &teamID,
		Platform: "darwin",
	}

	opt, err := svc.AgentOptionsForHost(context.Background(), host)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"override"}`, string(opt))

	host.Platform = "windows"
	opt, err = svc.AgentOptionsForHost(context.Background(), host)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"bar"}`, string(opt))

	// Should take gobal option with no team
	host.TeamID = nil
	opt, err = svc.AgentOptionsForHost(context.Background(), host)
	require.NoError(t, err)
	assert.JSONEq(t, `{"baz":"bar"}`, string(opt))

	host.Platform = "darwin"
	opt, err = svc.AgentOptionsForHost(context.Background(), host)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"override2"}`, string(opt))
}
