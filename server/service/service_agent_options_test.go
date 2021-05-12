package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/server/kolide"
	"github.com/fleetdm/fleet/server/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

func TestAgentOptionsForHost(t *testing.T) {
	ds := new(mock.Store)
	svc, err := newTestService(ds, nil, nil)
	require.NoError(t, err)

	teamID := uint(1)
	ds.TeamFunc = func(tid uint) (*kolide.Team, error) {
		assert.Equal(t, teamID, tid)
		opt := json.RawMessage(`{"config":{"foo":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override"}}}}`)
		return &kolide.Team{AgentOptions: &opt}, nil
	}
	ds.AppConfigFunc = func() (*kolide.AppConfig, error) {
		return &kolide.AppConfig{AgentOptions: json.RawMessage(`{"config":{"baz":"bar"},"overrides":{"platforms":{"darwin":{"foo":"override2"}}}}`)}, nil
	}

	host := &kolide.Host{
		TeamID:   null.IntFrom(int64(teamID)),
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
	host.TeamID.Valid = false
	opt, err = svc.AgentOptionsForHost(context.Background(), host)
	require.NoError(t, err)
	assert.JSONEq(t, `{"baz":"bar"}`, string(opt))

	host.Platform = "darwin"
	opt, err = svc.AgentOptionsForHost(context.Background(), host)
	require.NoError(t, err)
	assert.JSONEq(t, `{"foo":"override2"}`, string(opt))
}
