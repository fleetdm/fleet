package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	logging "github.com/fleetdm/fleet/v4/server/platform/logging"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestGetOrbitConfigLinuxEscrow(t *testing.T) {
	setupEscrowContext := func() (*mock.Store, fleet.Service, context.Context, *fleet.Host, fleet.Team) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		os := &fleet.OperatingSystem{
			Platform: "ubuntu",
			Version:  "20.04",
		}
		host := &fleet.Host{
			OsqueryHostID:         ptr.String("test"),
			ID:                    1,
			OSVersion:             "Ubuntu 20.04",
			Platform:              "ubuntu",
			DiskEncryptionEnabled: ptr.Bool(true),
		}

		team := fleet.Team{ID: 1}
		teamMDM := fleet.TeamMDM{EnableDiskEncryption: true}
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			require.Equal(t, team.ID, teamID)
			return &teamMDM, nil
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return ptr.RawMessage(json.RawMessage(`{}`)), nil
		}
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return true, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return nil, nil
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return true
		}
		ds.ClearPendingEscrowFunc = func(ctx context.Context, hostID uint) error {
			return nil
		}

		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnableDiskEncryption: optjson.SetBool(true)}}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return os, nil
		}

		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}

		ctx = test.HostContext(ctx, host)

		return ds, svc, ctx, host, team
	}

	t.Run("don't check for pending escrow if unsupported platform or encryption is not enabled", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		os := &fleet.OperatingSystem{
			Platform: "rhel",
			Version:  "9.0",
		}
		host := &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
			OSVersion:     "Red Hat Enterprise Linux 9.0",
			Platform:      "rhel",
		}

		team := fleet.Team{ID: 1}
		teamMDM := fleet.TeamMDM{EnableDiskEncryption: true}
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			require.Equal(t, team.ID, teamID)
			return &teamMDM, nil
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return ptr.RawMessage(json.RawMessage(`{}`)), nil
		}
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return true, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return nil, nil
		}

		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnableDiskEncryption: optjson.SetBool(true)}}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return os, nil
		}

		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}

		ctx = test.HostContext(ctx, host)

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.False(t, cfg.Notifications.RunDiskEncryptionEscrow)

		host.OSVersion = "Fedora 38.0"
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.False(t, cfg.Notifications.RunDiskEncryptionEscrow)
	})

	t.Run("pending escrow sets config flag and clears in DB", func(t *testing.T) {
		ds, svc, ctx, host, team := setupEscrowContext()

		// no-team
		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.True(t, cfg.Notifications.RunDiskEncryptionEscrow)
		require.True(t, ds.ClearPendingEscrowFuncInvoked)

		// with team
		ds.ClearPendingEscrowFuncInvoked = false
		host.TeamID = ptr.Uint(team.ID)
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.True(t, cfg.Notifications.RunDiskEncryptionEscrow)
		require.True(t, ds.ClearPendingEscrowFuncInvoked)

		// ignore clear escrow errors
		ds.ClearPendingEscrowFuncInvoked = false
		ds.ClearPendingEscrowFunc = func(ctx context.Context, hostID uint) error {
			return errors.New("clear pending escrow")
		}
		cfg, err = svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.True(t, cfg.Notifications.RunDiskEncryptionEscrow)
		require.True(t, ds.ClearPendingEscrowFuncInvoked)
	})
}

func TestOrbitLUKSDataSave(t *testing.T) {
	t.Run("when private key is set", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		opts := &TestServerOpts{License: license, SkipCreateTestUsers: true}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		host := &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
		}
		ctx = test.HostContext(ctx, host)

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
			}, nil
		}

		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
			require.Equal(t, activity.ActivityName(), fleet.ActivityTypeEscrowedDiskEncryptionKey{}.ActivityName())
			return nil
		}

		expectedErrorMessage := "There was an error."
		ds.ReportEscrowErrorFunc = func(ctx context.Context, hostID uint, err string) error {
			require.Equal(t, expectedErrorMessage, err)
			return nil
		}

		// test reporting client errors
		err := svc.EscrowLUKSData(ctx, "foo", "bar", nil, expectedErrorMessage)
		require.NoError(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)

		// blank passphrase
		ds.ReportEscrowErrorFuncInvoked = false
		expectedErrorMessage = "passphrase, salt, and key_slot must be provided to escrow LUKS data"
		err = svc.EscrowLUKSData(ctx, "", "bar", ptr.Uint(0), "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)

		ds.ReportEscrowErrorFuncInvoked = false
		passphrase, salt := "foo", ""
		var keySlot *uint
		ds.SaveLUKSDataFunc = func(ctx context.Context, incomingHost *fleet.Host, encryptedBase64Passphrase string,
			encryptedBase64Salt string, keySlotToPersist uint) (bool, error) {
			require.Equal(t, host.ID, incomingHost.ID)
			key := config.TestConfig().Server.PrivateKey

			decryptedPassphrase, err := mdm.DecodeAndDecrypt(encryptedBase64Passphrase, key)
			require.NoError(t, err)
			require.Equal(t, passphrase, decryptedPassphrase)

			decryptedSalt, err := mdm.DecodeAndDecrypt(encryptedBase64Salt, key)
			require.NoError(t, err)
			require.Equal(t, salt, decryptedSalt)

			require.Equal(t, *keySlot, keySlotToPersist)

			return true, nil
		}

		// with no salt
		err = svc.EscrowLUKSData(ctx, passphrase, salt, keySlot, "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)
		require.False(t, ds.SaveLUKSDataFuncInvoked)

		// with no key slot
		ds.ReportEscrowErrorFuncInvoked = false
		salt = "baz"
		err = svc.EscrowLUKSData(ctx, passphrase, salt, keySlot, "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)
		require.False(t, ds.SaveLUKSDataFuncInvoked)

		// with salt and key slot
		keySlot = ptr.Uint(0)
		ds.ReportEscrowErrorFuncInvoked = false
		err = svc.EscrowLUKSData(ctx, passphrase, salt, keySlot, "")
		require.NoError(t, err)
		require.False(t, ds.ReportEscrowErrorFuncInvoked)
		require.True(t, ds.SaveLUKSDataFuncInvoked)
		require.True(t, opts.ActivityMock.NewActivityFuncInvoked)
	})

	t.Run("fail when no/invalid private key is set", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		host := &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
			}, nil
		}

		expectedErrorMessage := "internal error: missing server private key"
		ds.ReportEscrowErrorFunc = func(ctx context.Context, hostID uint, err string) error {
			require.Equal(t, expectedErrorMessage, err)
			return nil
		}

		cfg := config.TestConfig()
		cfg.Server.PrivateKey = ""
		svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		ctx = test.HostContext(ctx, host)
		err := svc.EscrowLUKSData(ctx, "foo", "bar", ptr.Uint(0), "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)

		expectedErrorMessage = "internal error: could not encrypt LUKS data: create new cipher: crypto/aes: invalid key size 7"
		ds.ReportEscrowErrorFuncInvoked = false
		cfg.Server.PrivateKey = "invalid"
		svc, ctx = newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		ctx = test.HostContext(ctx, host)
		err = svc.EscrowLUKSData(ctx, "foo", "bar", ptr.Uint(0), "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)
	})
}

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
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return true, nil
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}

		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}, nil
		}

		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}

		ctx = test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
		})

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
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
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
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return true, nil
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}

		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}, nil
		}

		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}

		ctx = test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
			TeamID:        ptr.Uint(team.ID),
		})

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

	t.Run("non-eligible MDM status", func(t *testing.T) {
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
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
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
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return nil, sql.ErrNoRows
		}
		var isHostConnectedToFleet bool
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, h *fleet.Host) (bool, error) {
			return isHostConnectedToFleet, nil
		}

		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
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
			// host is not connected to fleet
			isHostConnectedToFleet = false
			checkEmptyNudgeConfig(h)

			// host has MDM turned on but is not enrolled
			isHostConnectedToFleet = true
			h.OsqueryHostID = nil
			checkEmptyNudgeConfig(h)
		}

		// global host
		checkHostVariations(&fleet.Host{
			OsqueryHostID: ptr.String("test"),
			Platform:      "darwin",
		})

		// team host
		checkHostVariations(&fleet.Host{
			OsqueryHostID: ptr.String("test"),
			TeamID:        ptr.Uint(team.ID),
			Platform:      "darwin",
		})
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
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return true, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}, nil
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}

		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true}}
		appCfg.MDM.MacOSUpdates.Deadline = optjson.SetString("2022-04-01")
		appCfg.MDM.MacOSUpdates.MinimumVersion = optjson.SetString("12.3")
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return os, nil
		}

		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
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
		require.False(t, cfg.Notifications.RunDiskEncryptionEscrow)
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

		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{
				IsServer:         false,
				InstalledFromDep: true,
				Enrolled:         true,
				Name:             fleet.WellKnownMDMFleet,
			}, nil
		}

		goodCtx := test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
		})

		badCtx := test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            2,
		})

		d1, err := svc.GetSoftwareInstallDetails(goodCtx, "")
		require.NoError(t, err)
		require.Equal(t, uint(1), d1.HostID)

		d2, err := svc.GetSoftwareInstallDetails(badCtx, "")
		require.Error(t, err)
		require.Nil(t, d2)
	})
}

func TestShouldRetrySoftwareInstall(t *testing.T) {
	svc := &Service{
		logger: logging.NewNopLogger(),
	}
	ctx := context.Background()

	t.Run("nil attempt number returns false", func(t *testing.T) {
		hsi := &fleet.HostSoftwareInstallerResult{
			AttemptNumber: nil,
		}
		shouldRetry, err := svc.shouldRetrySoftwareInstall(ctx, hsi)
		require.NoError(t, err)
		require.False(t, shouldRetry)
	})

	t.Run("attempt below max returns true", func(t *testing.T) {
		for _, attempt := range []int{1, 2} {
			hsi := &fleet.HostSoftwareInstallerResult{
				AttemptNumber: ptr.Int(attempt),
			}
			shouldRetry, err := svc.shouldRetrySoftwareInstall(ctx, hsi)
			require.NoError(t, err)
			require.True(t, shouldRetry, "attempt %d should retry", attempt)
		}
	})

	t.Run("attempt at max returns false", func(t *testing.T) {
		hsi := &fleet.HostSoftwareInstallerResult{
			AttemptNumber: ptr.Int(fleet.MaxSoftwareInstallAttempts),
		}
		shouldRetry, err := svc.shouldRetrySoftwareInstall(ctx, hsi)
		require.NoError(t, err)
		require.False(t, shouldRetry)
	})

	t.Run("attempt above max returns false", func(t *testing.T) {
		hsi := &fleet.HostSoftwareInstallerResult{
			AttemptNumber: ptr.Int(fleet.MaxSoftwareInstallAttempts + 1),
		}
		shouldRetry, err := svc.shouldRetrySoftwareInstall(ctx, hsi)
		require.NoError(t, err)
		require.False(t, shouldRetry)
	})
}

func TestRetrySoftwareInstall(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{
		ds:     ds,
		logger: logging.NewNopLogger(),
	}
	ctx := context.Background()

	installerID := uint(42)
	userID := uint(7)
	host := &fleet.Host{ID: 1}
	hsi := &fleet.HostSoftwareInstallerResult{
		SoftwareInstallerID: &installerID,
		SelfService:         true,
		UserID:              &userID,
		AttemptNumber:       ptr.Int(1),
	}

	var capturedOpts fleet.HostSoftwareInstallOptions
	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		require.Equal(t, host.ID, hostID)
		require.Equal(t, installerID, softwareInstallerID)
		capturedOpts = opts
		return "new-uuid", nil
	}

	t.Run("preserves self-service and user ID", func(t *testing.T) {
		err := svc.retrySoftwareInstall(ctx, host, hsi, false)
		require.NoError(t, err)
		require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.True(t, capturedOpts.SelfService)
		require.NotNil(t, capturedOpts.UserID)
		require.Equal(t, userID, *capturedOpts.UserID)
		require.False(t, capturedOpts.ForSetupExperience)
		require.True(t, capturedOpts.WithRetries)
	})

	t.Run("passes setup experience flag", func(t *testing.T) {
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		err := svc.retrySoftwareInstall(ctx, host, hsi, true)
		require.NoError(t, err)
		require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.True(t, capturedOpts.ForSetupExperience)
	})
}

func TestGetSoftwareInstallerAttemptNumber(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{
		ds:     ds,
		logger: logging.NewNopLogger(),
	}
	ctx := context.Background()
	host := &fleet.Host{ID: 1}

	t.Run("returns nil when install not found", func(t *testing.T) {
		ds.GetSoftwareInstallResultsFunc = func(ctx context.Context, installUUID string) (*fleet.HostSoftwareInstallerResult, error) {
			return nil, newNotFoundError()
		}
		result, err := svc.getSoftwareInstallerAttemptNumber(ctx, host, "uuid-1")
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("returns nil when software installer ID is nil", func(t *testing.T) {
		ds.GetSoftwareInstallResultsFunc = func(ctx context.Context, installUUID string) (*fleet.HostSoftwareInstallerResult, error) {
			return &fleet.HostSoftwareInstallerResult{SoftwareInstallerID: nil}, nil
		}
		result, err := svc.getSoftwareInstallerAttemptNumber(ctx, host, "uuid-1")
		require.NoError(t, err)
		require.Nil(t, result)
	})

	t.Run("counts policy install attempts", func(t *testing.T) {
		policyID := uint(10)
		installerID := uint(20)
		ds.GetSoftwareInstallResultsFunc = func(ctx context.Context, installUUID string) (*fleet.HostSoftwareInstallerResult, error) {
			return &fleet.HostSoftwareInstallerResult{
				SoftwareInstallerID: &installerID,
				PolicyID:            &policyID,
			}, nil
		}
		ds.CountHostSoftwareInstallAttemptsFunc = func(ctx context.Context, hostID, siID, polID uint) (int, error) {
			require.Equal(t, host.ID, hostID)
			require.Equal(t, installerID, siID)
			require.Equal(t, policyID, polID)
			return 2, nil
		}
		result, err := svc.getSoftwareInstallerAttemptNumber(ctx, host, "uuid-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 2, *result)
		require.True(t, ds.CountHostSoftwareInstallAttemptsFuncInvoked)
	})

	t.Run("returns attempt number from install for non-policy retry-eligible install", func(t *testing.T) {
		installerID := uint(20)
		attemptNum := 2
		ds.GetSoftwareInstallResultsFunc = func(ctx context.Context, installUUID string) (*fleet.HostSoftwareInstallerResult, error) {
			return &fleet.HostSoftwareInstallerResult{
				SoftwareInstallerID: &installerID,
				PolicyID:            nil, // non-policy install
				AttemptNumber:       &attemptNum,
			}, nil
		}
		ds.CountHostSoftwareInstallAttemptsFuncInvoked = false
		result, err := svc.getSoftwareInstallerAttemptNumber(ctx, host, "uuid-1")
		require.NoError(t, err)
		require.NotNil(t, result)
		require.Equal(t, 2, *result)
		require.False(t, ds.CountHostSoftwareInstallAttemptsFuncInvoked)
	})

	t.Run("returns nil for non-policy install without retry support", func(t *testing.T) {
		installerID := uint(20)
		ds.GetSoftwareInstallResultsFunc = func(ctx context.Context, installUUID string) (*fleet.HostSoftwareInstallerResult, error) {
			return &fleet.HostSoftwareInstallerResult{
				SoftwareInstallerID: &installerID,
				PolicyID:            nil, // non-policy install
				AttemptNumber:       nil, // not created with WithRetries
			}, nil
		}
		ds.CountHostSoftwareInstallAttemptsFuncInvoked = false
		result, err := svc.getSoftwareInstallerAttemptNumber(ctx, host, "uuid-1")
		require.NoError(t, err)
		require.Nil(t, result)
		require.False(t, ds.CountHostSoftwareInstallAttemptsFuncInvoked)
	})
}
