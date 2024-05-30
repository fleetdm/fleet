package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestGetOrbitConfigNudge(t *testing.T) {
	t.Run("missing values in AppConfig", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		os := &fleet.OperatingSystem{
			Platform: "darwin",
			Version:  "12.2",
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return os, nil
		}
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ctx = test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
			MDMInfo: &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}})

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.AppConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false

		appCfg.MDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.AppConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false

		appCfg.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("2022-04-01")
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, cfg.NudgeConfig)
		require.True(t, ds.AppConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false
	})

	t.Run("missing values in TeamConfig", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
		appCfg.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("2022-04-01")
		appCfg.MDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		os := &fleet.OperatingSystem{
			Platform: "darwin",
			Version:  "12.2",
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return os, nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		team := fleet.Team{ID: 1}
		teamMDM := fleet.TeamMDM{}
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			require.Equal(t, team.ID, teamID)
			return &teamMDM, nil
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return ptr.RawMessage(json.RawMessage(`{}`)), nil
		}
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}

		ctx = test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
			TeamID:        ptr.Uint(team.ID),
			MDMInfo: &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}})

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamMDMConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false
		ds.TeamMDMConfigFuncInvoked = false

		teamMDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamMDMConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false
		ds.TeamMDMConfigFuncInvoked = false

		teamMDM.MacOSUpdates.MinimumVersion = optjson.SetString("2022-04-01")
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, cfg.NudgeConfig)
		require.True(t, ds.AppConfigFuncInvoked)
		require.True(t, ds.TeamMDMConfigFuncInvoked)
		ds.AppConfigFuncInvoked = false
		ds.TeamMDMConfigFuncInvoked = false
	})

	t.Run("non-elegible MDM status", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		os := &fleet.OperatingSystem{
			Platform: "darwin",
			Version:  "12.2",
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return os, nil
		}
		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
		appCfg.MDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		appCfg.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("2022-04-01")
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}

		team := fleet.Team{ID: 1}
		teamMDM := fleet.TeamMDM{}
		teamMDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		teamMDM.MacOSUpdates.MinimumVersion = optjson.SetString("12.1")
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			require.Equal(t, team.ID, teamID)
			return &teamMDM, nil
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return ptr.RawMessage(json.RawMessage(`{}`)), nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		checkEmptyNudgeConfig := func(h *fleet.Host) {
			ctx := test.HostContext(ctx, h)
			cfg, err := svc.GetOrbitConfig(ctx)
			require.NoError(t, err)
			require.Empty(t, cfg.NudgeConfig)
			require.True(t, ds.AppConfigFuncInvoked)
			ds.AppConfigFuncInvoked = false
		}

		checkHostVariations := func(h *fleet.Host) {
			// host uses another MDM
			h.MDMInfo.Name = fleet.WellKnownMDMIntune
			checkEmptyNudgeConfig(h)

			// host has MDM turned off
			h.MDMInfo.Name = fleet.WellKnownMDMFleet
			h.MDMInfo.Enrolled = false
			checkEmptyNudgeConfig(h)

			// host has MDM turned on but is not enrolled
			h.MDMInfo.Enrolled = true
			h.OsqueryHostID = nil
			checkEmptyNudgeConfig(h)

			// mdminfo is nil
			h.MDMInfo = nil
			checkEmptyNudgeConfig(h)
		}

		// global host
		checkHostVariations(&fleet.Host{
			OsqueryHostID: ptr.String("test"),
			MDMInfo: &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}})

		// team host
		checkHostVariations(&fleet.Host{
			OsqueryHostID: ptr.String("test"),
			TeamID:        ptr.Uint(team.ID),
			MDMInfo: &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}})

	})

	t.Run("no-nudge on macos versions greater than 14", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		os := &fleet.OperatingSystem{
			Platform: "darwin",
			Version:  "12.2",
		}
		host := &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
			MDMInfo: &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}}

		team := fleet.Team{ID: 1}
		teamMDM := fleet.TeamMDM{}
		teamMDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		teamMDM.MacOSUpdates.MinimumVersion = optjson.SetString("12.1")
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			require.Equal(t, team.ID, teamID)
			return &teamMDM, nil
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return ptr.RawMessage(json.RawMessage(`{}`)), nil
		}
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
		appCfg.MDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		appCfg.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("12.3")
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return os, nil
		}
		ctx = test.HostContext(ctx, host)

		// Version < 14 gets nudge
		host.ID = 1
		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, cfg.NudgeConfig)
		require.True(t, ds.GetHostOperatingSystemFuncInvoked)

		// Version > 14 gets no nudge
		os.Version = "14.1"
		ds.GetHostOperatingSystemFuncInvoked = false
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.GetHostOperatingSystemFuncInvoked)

		// windows gets no nudge
		os.Platform = "windows"
		ds.GetHostOperatingSystemFuncInvoked = false
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.GetHostOperatingSystemFuncInvoked)

		//// team section below
		host.TeamID = ptr.Uint(team.ID)
		os.Platform = "darwin"
		os.Version = "12.1"

		// Version < 14 gets nudge
		host.ID = 1
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.NotEmpty(t, cfg.NudgeConfig)
		require.True(t, ds.GetHostOperatingSystemFuncInvoked)

		// Version > 14 gets no nudge
		os.Version = "14.1"
		ds.GetHostOperatingSystemFuncInvoked = false
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.GetHostOperatingSystemFuncInvoked)

		// windows gets no nudge
		os.Platform = "windows"
		ds.GetHostOperatingSystemFuncInvoked = false
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Empty(t, cfg.NudgeConfig)
		require.True(t, ds.GetHostOperatingSystemFuncInvoked)
	})
}

func TestGetSoftwareInstallDetails(t *testing.T) {
	t.Run("hosts can't get each others installers", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

		ds.GetSoftwareInstallDetailsFunc = func(ctx context.Context, executionId string) (*fleet.SoftwareInstallDetails, error) {
			return &fleet.SoftwareInstallDetails{
				HostID: 1,
			}, nil
		}

		goodCtx := test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
			MDMInfo: &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}})

		badCtx := test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            2,
			MDMInfo: &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}})

		d1, err := svc.GetSoftwareInstallDetails(goodCtx, "")
		require.NoError(t, err)
		require.Equal(t, uint(1), d1.HostID)

		d2, err := svc.GetSoftwareInstallDetails(badCtx, "")
		require.Error(t, err)
		require.Nil(t, d2)
	})
}
