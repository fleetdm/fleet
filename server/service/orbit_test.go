package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/require"
)

func TestGetOrbitConfigLinuxEscrow(t *testing.T) {
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
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal, onlyActive bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint, active bool) ([]string, error) {
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
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal, onlyActive bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint, active bool) ([]string, error) {
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
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		host := &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
		}
		ctx = test.HostContext(ctx, host)
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
			encryptedBase64Salt string, keySlotToPersist uint) error {
			require.Equal(t, host.ID, incomingHost.ID)
			key := config.TestConfig().Server.PrivateKey

			decryptedPassphrase, err := mdm.DecodeAndDecrypt(encryptedBase64Passphrase, key)
			require.NoError(t, err)
			require.Equal(t, passphrase, decryptedPassphrase)

			decryptedSalt, err := mdm.DecodeAndDecrypt(encryptedBase64Salt, key)
			require.NoError(t, err)
			require.Equal(t, salt, decryptedSalt)

			require.Equal(t, *keySlot, keySlotToPersist)

			return nil
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
	})

	t.Run("fail when no/invalid private key is set", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		host := &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			ID:            1,
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
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal, onlyActive bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint, active bool) ([]string, error) {
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
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint, active bool) ([]string, error) {
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
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal, onlyActive bool) ([]*fleet.HostScriptResult, error) {
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
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal, onlyActive bool) ([]*fleet.HostScriptResult, error) {
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
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint, active bool) ([]string, error) {
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
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal, onlyActive bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListPendingSoftwareInstallsFunc = func(ctx context.Context, hostID uint, active bool) ([]string, error) {
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
		ds.ListPendingHostScriptExecutionsFunc = func(ctx context.Context, hostID uint, onlyShowInternal, onlyActive bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
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
