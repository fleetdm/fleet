package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	hostidentity_types "github.com/fleetdm/fleet/v4/ee/pkg/hostidentity/types"
	"github.com/fleetdm/fleet/v4/pkg/optjson"
	activity_api "github.com/fleetdm/fleet/v4/server/activity/api"
	"github.com/fleetdm/fleet/v4/server/config"
	"github.com/fleetdm/fleet/v4/server/contexts/capabilities"
	hostctx "github.com/fleetdm/fleet/v4/server/contexts/host"
	"github.com/fleetdm/fleet/v4/server/contexts/viewer"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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
		// the notification is gated on the host fleet's Linux escrow setting
		ds.GetConfigEnableDiskEncryptionFunc = func(ctx context.Context, teamID *uint) (fleet.DiskEncryptionConfig, error) {
			return fleet.DiskEncryptionConfig{LinuxEscrowEnabled: true}, nil
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

	t.Run("escrow turned off after the host went pending", func(t *testing.T) {
		ds, svc, ctx, _, _ := setupEscrowContext()
		// escrow can be disabled while a host is already pending; without this
		// gate the user is prompted for a passphrase that EscrowLUKSData would
		// then discard
		ds.GetConfigEnableDiskEncryptionFunc = func(ctx context.Context, teamID *uint) (fleet.DiskEncryptionConfig, error) {
			return fleet.DiskEncryptionConfig{LinuxEscrowEnabled: false}, nil
		}

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.False(t, cfg.Notifications.RunDiskEncryptionEscrow)
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
			Platform:      "ubuntu",
			ID:            1,
		}
		ctx = test.HostContext(ctx, host)

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(true),
					LinuxSettings:        fleet.LinuxSettings{EnableEscrowDiskEncryptionKey: optjson.SetBool(true)},
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
		err := svc.EscrowLUKSData(ctx, "foo", "bar", nil, expectedErrorMessage, "")
		require.NoError(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)

		// blank passphrase
		ds.ReportEscrowErrorFuncInvoked = false
		expectedErrorMessage = "passphrase, salt, and key_slot must be provided to escrow LUKS data"
		err = svc.EscrowLUKSData(ctx, "", "bar", new(uint(0)), "", "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)

		ds.ReportEscrowErrorFuncInvoked = false
		passphrase, salt := "foo", ""
		var keySlot *uint
		ds.SaveLUKSDataFunc = func(ctx context.Context, incomingHost *fleet.Host, encryptedBase64Passphrase string,
			encryptedBase64Salt string, keySlotToPersist *uint,
		) (bool, error) {
			require.Equal(t, host.ID, incomingHost.ID)
			key := config.TestConfig().Server.PrivateKey

			decryptedPassphrase, err := mdm.DecodeAndDecrypt(encryptedBase64Passphrase, key)
			require.NoError(t, err)
			require.Equal(t, passphrase, decryptedPassphrase)

			decryptedSalt, err := mdm.DecodeAndDecrypt(encryptedBase64Salt, key)
			require.NoError(t, err)
			require.Equal(t, salt, decryptedSalt)

			require.Equal(t, keySlot, keySlotToPersist)

			return true, nil
		}

		// with no salt
		err = svc.EscrowLUKSData(ctx, passphrase, salt, keySlot, "", "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)
		require.False(t, ds.SaveLUKSDataFuncInvoked)

		// with no key slot
		ds.ReportEscrowErrorFuncInvoked = false
		salt = "baz"
		err = svc.EscrowLUKSData(ctx, passphrase, salt, keySlot, "", "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)
		require.False(t, ds.SaveLUKSDataFuncInvoked)

		// with salt and key slot
		keySlot = ptr.Uint(0)
		ds.ReportEscrowErrorFuncInvoked = false
		err = svc.EscrowLUKSData(ctx, passphrase, salt, keySlot, "", "")
		require.NoError(t, err)
		require.False(t, ds.ReportEscrowErrorFuncInvoked)
		require.True(t, ds.SaveLUKSDataFuncInvoked)
		require.True(t, opts.ActivityMock.NewActivityFuncInvoked)
	})

	t.Run("recovery key escrow has no salt or key slot", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		opts := &TestServerOpts{License: license, SkipCreateTestUsers: true}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		host := &fleet.Host{
			OsqueryHostID: new("test"),
			Platform:      "ubuntu",
			ID:            1,
		}
		ctx = test.HostContext(ctx, host)

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(true),
					LinuxSettings:        fleet.LinuxSettings{EnableEscrowDiskEncryptionKey: optjson.SetBool(true)},
				},
			}, nil
		}

		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
			require.Equal(t, activity.ActivityName(), fleet.ActivityTypeEscrowedDiskEncryptionKey{}.ActivityName())
			return nil
		}

		ds.ReportEscrowErrorFunc = func(ctx context.Context, hostID uint, err string) error {
			return nil
		}

		recoveryKey := "55055-39320-64491-48436-47667-15525-36879-32875"
		ds.SaveLUKSDataFunc = func(ctx context.Context, incomingHost *fleet.Host, encryptedBase64Passphrase string,
			encryptedBase64Salt string, keySlotToPersist *uint,
		) (bool, error) {
			require.Equal(t, host.ID, incomingHost.ID)
			key := config.TestConfig().Server.PrivateKey

			decrypted, err := mdm.DecodeAndDecrypt(encryptedBase64Passphrase, key)
			require.NoError(t, err)
			require.Equal(t, recoveryKey, decrypted)

			// snapd owns the LUKS key slots, so a recovery key has no salt or
			// numeric key slot to escrow.
			require.Empty(t, encryptedBase64Salt)
			require.Nil(t, keySlotToPersist)

			return true, nil
		}

		// A recovery key requires no salt or key slot.
		err := svc.EscrowLUKSData(ctx, recoveryKey, "", nil, "", fleet.LUKSKeyTypeRecoveryKey)
		require.NoError(t, err)
		require.False(t, ds.ReportEscrowErrorFuncInvoked)
		require.True(t, ds.SaveLUKSDataFuncInvoked)
		require.True(t, opts.ActivityMock.NewActivityFuncInvoked)

		// A recovery key escrow with no key still fails validation.
		ds.SaveLUKSDataFuncInvoked = false
		err = svc.EscrowLUKSData(ctx, "", "", nil, "", fleet.LUKSKeyTypeRecoveryKey)
		require.Error(t, err)
		require.False(t, ds.SaveLUKSDataFuncInvoked)

		// Stray salt / key slot on the recovery-key path are rejected, not
		// silently discarded — those fields are meaningless when snapd owns the
		// LUKS key slots, and accepting them would hide client bugs.
		ds.SaveLUKSDataFuncInvoked = false
		err = svc.EscrowLUKSData(ctx, recoveryKey, "some-salt", nil, "", fleet.LUKSKeyTypeRecoveryKey)
		require.Error(t, err)
		require.False(t, ds.SaveLUKSDataFuncInvoked)

		ds.SaveLUKSDataFuncInvoked = false
		strayKeySlot := uint(0)
		err = svc.EscrowLUKSData(ctx, recoveryKey, "", &strayKeySlot, "", fleet.LUKSKeyTypeRecoveryKey)
		require.Error(t, err)
		require.False(t, ds.SaveLUKSDataFuncInvoked)
	})

	t.Run("fail when no/invalid private key is set", func(t *testing.T) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		host := &fleet.Host{
			OsqueryHostID: ptr.String("test"),
			Platform:      "ubuntu",
			ID:            1,
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(true),
					LinuxSettings:        fleet.LinuxSettings{EnableEscrowDiskEncryptionKey: optjson.SetBool(true)},
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
		err := svc.EscrowLUKSData(ctx, "foo", "bar", new(uint(0)), "", "")
		require.Error(t, err)
		require.True(t, ds.ReportEscrowErrorFuncInvoked)

		expectedErrorMessage = "internal error: could not encrypt LUKS data: create new cipher: crypto/aes: invalid key size 7"
		ds.ReportEscrowErrorFuncInvoked = false
		cfg.Server.PrivateKey = "invalid"
		svc, ctx = newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})
		ctx = test.HostContext(ctx, host)
		err = svc.EscrowLUKSData(ctx, "foo", "bar", new(uint(0)), "", "")
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
				ConnectedToFleet: true,
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
				ConnectedToFleet: true,
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
		// GetOrbitConfig derives the Fleet-MDM connection state from GetHostMDM
		// (ConnectedToFleet).
		var connectedToFleetMDM bool
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet, ConnectedToFleet: connectedToFleetMDM}, nil
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
			// host is osquery-enrolled but not connected to Fleet MDM
			connectedToFleetMDM = false
			checkEmptyNudgeConfig(h)

			// host is connected to Fleet MDM but not osquery-enrolled
			connectedToFleetMDM = true
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
				ConnectedToFleet: true,
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

func TestGetOrbitConfigWebSocketTransport(t *testing.T) {
	setupCtx := func(wsEnabled bool) (fleet.Service, context.Context) {
		ds := new(mock.Store)
		cfg := config.TestConfig()
		cfg.WebSocket.TransportEnabled = wsEnabled
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{}, nil
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return new(json.RawMessage(`{}`)), nil
		}
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return false, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return nil, newNotFoundError()
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}
		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{}, nil
		}

		ctx = test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: new("test"),
			ID:            1,
			Platform:      "ubuntu",
			TeamID:        new(uint(1)),
		})
		return svc, ctx
	}

	t.Run("enabled", func(t *testing.T) {
		svc, ctx := setupCtx(true)
		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.NotNil(t, cfg.WebSocketTransport)
		require.True(t, cfg.WebSocketTransport.Enabled)
	})

	t.Run("disabled omits the directive", func(t *testing.T) {
		svc, ctx := setupCtx(false)
		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Nil(t, cfg.WebSocketTransport)
	})
}

func TestGetOrbitConfigScriptTimeoutFallback(t *testing.T) {
	setupCtx := func(teamAgentOpts, globalAgentOpts *json.RawMessage) (fleet.Service, context.Context, *mock.Store) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

		team := fleet.Team{ID: 1}
		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{}, nil
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return teamAgentOpts, nil
		}
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, host *fleet.Host) (bool, error) {
			return false, nil
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return nil, newNotFoundError()
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}
		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}
		appCfg := &fleet.AppConfig{AgentOptions: globalAgentOpts}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}

		ctx = test.HostContext(ctx, &fleet.Host{
			OsqueryHostID: new("test"),
			ID:            1,
			Platform:      "ubuntu",
			TeamID:        &team.ID,
		})
		return svc, ctx, ds
	}

	t.Run("team timeout set wins over global", func(t *testing.T) {
		team := new(json.RawMessage(`{"script_execution_timeout": 600}`))
		global := new(json.RawMessage(`{"config": {}, "script_execution_timeout": 1200}`))
		svc, ctx, _ := setupCtx(team, global)

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, 600, cfg.ScriptExeTimeout)
	})

	t.Run("team timeout unset falls back to global", func(t *testing.T) {
		team := new(json.RawMessage(`{}`))
		global := new(json.RawMessage(`{"config": {}, "script_execution_timeout": 1200}`))
		svc, ctx, _ := setupCtx(team, global)

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, 1200, cfg.ScriptExeTimeout)
	})

	t.Run("team timeout zero falls back to global", func(t *testing.T) {
		team := new(json.RawMessage(`{"script_execution_timeout": 0}`))
		global := new(json.RawMessage(`{"config": {}, "script_execution_timeout": 900}`))
		svc, ctx, _ := setupCtx(team, global)

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, 900, cfg.ScriptExeTimeout)
	})

	t.Run("team and global both unset", func(t *testing.T) {
		team := new(json.RawMessage(`{}`))
		global := new(json.RawMessage(`{"config": {}}`))
		svc, ctx, _ := setupCtx(team, global)

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, cfg.ScriptExeTimeout)
	})

	t.Run("nil global agent options, team unset", func(t *testing.T) {
		team := new(json.RawMessage(`{}`))
		svc, ctx, _ := setupCtx(team, nil)

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		require.Equal(t, 0, cfg.ScriptExeTimeout)
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
				ConnectedToFleet: true,
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
		logger: slog.New(slog.DiscardHandler),
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
		logger: slog.New(slog.DiscardHandler),
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
	var capturedInstallerID uint
	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		require.Equal(t, host.ID, hostID)
		capturedInstallerID = softwareInstallerID
		capturedOpts = opts
		return "new-uuid", nil
	}
	// By default the frozen installer is still the active one for its title.
	ds.ResolveActiveInstallerForRetryFunc = func(ctx context.Context, installerID uint) (uint, error) {
		return installerID, nil
	}

	t.Run("preserves self-service and user ID", func(t *testing.T) {
		err := svc.retrySoftwareInstall(ctx, host, hsi, false)
		require.NoError(t, err)
		require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Equal(t, installerID, capturedInstallerID)
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

	t.Run("retries the active installer after a version change", func(t *testing.T) {
		const activeID = uint(99)
		ds.ResolveActiveInstallerForRetryFunc = func(ctx context.Context, gotID uint) (uint, error) {
			require.Equal(t, installerID, gotID)
			return activeID, nil
		}
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		err := svc.retrySoftwareInstall(ctx, host, hsi, false)
		require.NoError(t, err)
		require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Equal(t, activeID, capturedInstallerID, "retry targets the current active installer, not the frozen one")
	})
}

func TestRetryPolicyAutomationSoftwareInstall(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{ds: ds, logger: slog.New(slog.DiscardHandler)}
	ctx := context.Background()

	frozenID := uint(42)
	policyID := uint(5)
	host := &fleet.Host{ID: 1}
	hsi := &fleet.HostSoftwareInstallerResult{
		SoftwareInstallerID: &frozenID,
		PolicyID:            &policyID,
		AttemptNumber:       new(1),
	}

	var capturedInstallerID uint
	var capturedOpts fleet.HostSoftwareInstallOptions
	ds.InsertSoftwareInstallRequestFunc = func(ctx context.Context, hostID uint, softwareInstallerID uint, opts fleet.HostSoftwareInstallOptions) (string, error) {
		require.Equal(t, host.ID, hostID)
		capturedInstallerID = softwareInstallerID
		capturedOpts = opts
		return "new-uuid", nil
	}

	t.Run("retries the frozen installer when it is still active", func(t *testing.T) {
		ds.ResolveActiveInstallerForRetryFunc = func(ctx context.Context, id uint) (uint, error) { return id, nil }
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		require.NoError(t, svc.retryPolicyAutomationSoftwareInstall(ctx, host, hsi))
		require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Equal(t, frozenID, capturedInstallerID)
		require.Equal(t, &policyID, capturedOpts.PolicyID)
	})

	t.Run("retries the active installer after a version change", func(t *testing.T) {
		const activeID = uint(99)
		ds.ResolveActiveInstallerForRetryFunc = func(ctx context.Context, id uint) (uint, error) {
			require.Equal(t, frozenID, id)
			return activeID, nil
		}
		ds.InsertSoftwareInstallRequestFuncInvoked = false
		require.NoError(t, svc.retryPolicyAutomationSoftwareInstall(ctx, host, hsi))
		require.True(t, ds.InsertSoftwareInstallRequestFuncInvoked)
		require.Equal(t, activeID, capturedInstallerID, "policy retry targets the current active installer, not the frozen one")
		require.Equal(t, &policyID, capturedOpts.PolicyID)
	})
}

func TestGetSoftwareInstallerAttemptNumber(t *testing.T) {
	ds := new(mock.Store)
	svc := &Service{
		ds:     ds,
		logger: slog.New(slog.DiscardHandler),
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

func TestSoftwareInstallReplicaLag(t *testing.T) {
	// Create datastore with dummy replica to simulate replication lag
	opts := &testing_utils.DatastoreTestOptions{DummyReplica: true}
	ds := mysqltest.CreateMySQLDSWithOptions(t, opts)
	defer ds.Close()

	svc, ctx := newTestService(t, ds, nil, nil)

	// Create admin user
	user, err := ds.NewUser(ctx, &fleet.User{
		Name:       "Admin",
		Password:   []byte("p4ssw0rd.123"),
		Email:      "admin@example.com",
		GlobalRole: ptr.String(fleet.RoleAdmin),
	})
	require.NoError(t, err)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	// Create a host
	host := test.NewHost(t, ds, "host1", "10.0.0.1", "host1Key", "host1UUID", time.Now())
	opts.RunReplication("hosts")

	// Create a policy
	policy, err := ds.NewGlobalPolicy(ctx, &user.ID, fleet.PolicyPayload{
		Name:  "test policy",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	opts.RunReplication("policies")

	// Create software installer
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "echo 'installing'",
		Filename:        "test_installer.pkg",
		StorageID:       uuid.New().String(),
		Title:           "Test Software",
		Version:         "1.0.0",
		Source:          "apps",
		Platform:        "darwin",
		UserID:          user.ID,
		TeamID:          nil,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	}
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, payload)
	require.NoError(t, err)
	opts.RunReplication("software_installers", "software_titles")

	// Mark policy as failing for the host
	_, err = ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{policy.ID: new(false)}, time.Now(), false, nil)
	require.NoError(t, err)
	opts.RunReplication("policy_membership")

	// simulate Orbit picking up upcoming_activity and activating
	installUUID := uuid.New().String()
	var titleID uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, q, &titleID,
			`SELECT title_id FROM software_installers WHERE id = ?`, installerID)
		if err != nil {
			return err
		}

		_, err = q.ExecContext(ctx, `
			INSERT INTO host_software_installs (
				execution_id, host_id, software_installer_id, policy_id,
				installer_filename, version, software_title_id, software_title_name
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		`, installUUID, host.ID, installerID, policy.ID,
			payload.Filename, payload.Version, titleID, payload.Title)
		return err
	})

	var attemptNumberBeforeResult *int
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &attemptNumberBeforeResult,
			`SELECT attempt_number FROM host_software_installs WHERE execution_id = ?`,
			installUUID)
	})
	require.Nil(t, attemptNumberBeforeResult, "attempt_number should be NULL after activation")

	// Make the activated install available to replica
	opts.RunReplication("host_software_installs")

	result := &fleet.HostSoftwareInstallResultPayload{
		HostID:                host.ID,
		InstallUUID:           installUUID,
		InstallScriptExitCode: ptr.Int(1), // Failed
		InstallScriptOutput:   ptr.String("install failed"),
	}
	ctx = hostctx.NewContext(ctx, host)
	err = svc.SaveHostSoftwareInstallResult(ctx, result)
	require.NoError(t, err, "SaveHostSoftwareInstallResult should use primary DB to avoid replication lag")

	// Verify the attempt_number was set in the primary
	var attemptNumberInWriter *int
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &attemptNumberInWriter,
			`SELECT attempt_number FROM host_software_installs WHERE execution_id = ?`,
			installUUID)
	})
	require.NotNil(t, attemptNumberInWriter, "attempt_number should be set in primary after result is reported")
	require.Equal(t, 1, *attemptNumberInWriter, "first attempt should be 1")

	// verify retry was scheduled, and that we did not throw an error because of nil attempt_number
	var retryCount int
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &retryCount,
			`SELECT COUNT(*) FROM upcoming_activities
			WHERE activity_type = 'software_install'`,
		)
	})
	require.Equal(t, 1, retryCount, "should have scheduled a retry in upcoming_activities")
}

// TestSaveHostSoftwareInstallResultAppOpenSkip verifies that an app-open result on a patch-when-closed
// policy install is a skip (attempt_number=0, no retry, activity flagged), while an ordinary empty
// pre_install_query on a non-managed policy still fails, counts, and retries.
func TestSaveHostSoftwareInstallResultAppOpenSkip(t *testing.T) {
	ds := mysqltest.CreateMySQLDS(t)
	defer ds.Close()

	opts := &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}, SkipCreateTestUsers: true}
	svc, ctx := newTestService(t, ds, nil, nil, opts)

	// The test service mocks the activity service, so capture emitted activities by install UUID.
	installedActivities := make(map[string]fleet.ActivityTypeInstalledSoftware)
	opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, activity activity_api.ActivityDetails) error {
		if a, ok := activity.(fleet.ActivityTypeInstalledSoftware); ok {
			installedActivities[a.InstallUUID] = a
		}
		return nil
	}

	user, err := ds.NewUser(ctx, &fleet.User{
		Name:       "Admin",
		Password:   []byte("p4ssw0rd.123"),
		Email:      "admin@example.com",
		GlobalRole: new(fleet.RoleAdmin),
	})
	require.NoError(t, err)
	ctx = viewer.NewContext(ctx, viewer.Viewer{User: user})

	// patch_when_closed is only valid on a team policy, never global: it forces continuous
	// automations and a title-bound patch policy, both rejected on "All fleets".
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "patch-when-closed-team"})
	require.NoError(t, err)

	installerPayload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript:   "echo 'installing'",
		Filename:        "test_installer.pkg",
		StorageID:       uuid.New().String(),
		Title:           "Test Software",
		Version:         "1.0.0",
		Source:          "apps",
		Platform:        "darwin",
		UserID:          user.ID,
		TeamID:          &team.ID,
		ValidatedLabels: &fleet.LabelIdentsWithScope{},
	}
	installerID, _, err := ds.MatchOrCreateSoftwareInstaller(ctx, installerPayload)
	require.NoError(t, err)

	var titleID uint
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleID,
			`SELECT title_id FROM software_installers WHERE id = ?`, installerID)
	})

	// createFailingPolicy makes a failing team policy for the host (optionally patch-when-closed) so a
	// retry would be eligible. patch_when_closed isn't settable via the create path yet, so set it directly.
	createFailingPolicy := func(t *testing.T, host *fleet.Host, patchWhenClosed bool) uint {
		policy, err := ds.NewTeamPolicy(ctx, team.ID, &user.ID, fleet.PolicyPayload{
			Name:  "policy-" + uuid.NewString(),
			Query: "SELECT 1;",
		})
		require.NoError(t, err)
		if patchWhenClosed {
			mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(ctx, `UPDATE policies SET patch_when_closed = 1 WHERE id = ?`, policy.ID)
				return err
			})
		}
		_, err = ds.RecordPolicyQueryExecutions(ctx, host, map[uint]*bool{policy.ID: new(false)}, time.Now(), false, nil)
		require.NoError(t, err)
		return policy.ID
	}

	// insertPendingInstall queues a pending policy-automation install, returning its execution id.
	insertPendingInstall := func(t *testing.T, host *fleet.Host, policyID uint) string {
		installUUID := uuid.New().String()
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				INSERT INTO host_software_installs (
					execution_id, host_id, software_installer_id, policy_id,
					installer_filename, version, software_title_id, software_title_name
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, installUUID, host.ID, installerID, policyID,
				installerPayload.Filename, installerPayload.Version, titleID, installerPayload.Title)
			return err
		})
		return installUUID
	}

	getAttemptNumber := func(t *testing.T, installUUID string) *int {
		var attempt *int
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &attempt,
				`SELECT attempt_number FROM host_software_installs WHERE execution_id = ?`, installUUID)
		})
		return attempt
	}

	countPendingRetries := func(t *testing.T, hostID uint) int {
		var n int
		mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n,
				`SELECT COUNT(*) FROM upcoming_activities WHERE activity_type = 'software_install' AND host_id = ?`, hostID)
		})
		return n
	}

	t.Run("app open -> skip, no attempt consumed, no retry, activity flagged", func(t *testing.T) {
		host := test.NewHost(t, ds, "skip-host", "10.0.0.1", uuid.NewString(), uuid.NewString(), time.Now())
		require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
		installUUID := insertPendingInstall(t, host, createFailingPolicy(t, host, true))

		result := &fleet.HostSoftwareInstallResultPayload{
			HostID:                    host.ID,
			InstallUUID:               installUUID,
			PreInstallConditionOutput: new(""), // app open
		}
		hctx := hostctx.NewContext(ctx, host)
		require.NoError(t, svc.SaveHostSoftwareInstallResult(hctx, result))

		attempt := getAttemptNumber(t, installUUID)
		require.NotNil(t, attempt)
		require.Equal(t, 0, *attempt, "skip must not consume a retry attempt")

		require.Equal(t, 0, countPendingRetries(t, host.ID), "skip must not queue an immediate retry")

		act, ok := installedActivities[installUUID]
		require.True(t, ok, "an installed_software activity should have been emitted")
		require.Equal(t, string(fleet.SoftwareInstallFailed), act.Status)
		require.True(t, act.SkippedInstall, "activity should be flagged as an app-open skip")
	})

	t.Run("regression: ordinary empty pre_install_query fails, counts, and retries", func(t *testing.T) {
		host := test.NewHost(t, ds, "regress-host", "10.0.0.2", uuid.NewString(), uuid.NewString(), time.Now())
		require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
		installUUID := insertPendingInstall(t, host, createFailingPolicy(t, host, false))

		result := &fleet.HostSoftwareInstallResultPayload{
			HostID:                    host.ID,
			InstallUUID:               installUUID,
			PreInstallConditionOutput: new(""),
		}
		hctx := hostctx.NewContext(ctx, host)
		require.NoError(t, svc.SaveHostSoftwareInstallResult(hctx, result))

		attempt := getAttemptNumber(t, installUUID)
		require.NotNil(t, attempt)
		require.Equal(t, 1, *attempt, "ordinary pre-install failure must count toward the retry limit")

		require.Equal(t, 1, countPendingRetries(t, host.ID), "ordinary failure should queue a retry")

		act, ok := installedActivities[installUUID]
		require.True(t, ok, "an installed_software activity should have been emitted")
		require.Equal(t, string(fleet.SoftwareInstallFailed), act.Status)
		require.False(t, act.SkippedInstall, "non-managed failure must not be flagged as a skip")
	})

	t.Run("many consecutive app-open runs never hit the retry cap", func(t *testing.T) {
		host := test.NewHost(t, ds, "many-runs-host", "10.0.0.3", uuid.NewString(), uuid.NewString(), time.Now())
		require.NoError(t, ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&team.ID, []uint{host.ID})))
		policyID := createFailingPolicy(t, host, true)

		// More consecutive runs than the retry cap; each is a fresh install the app-open query skips.
		for range fleet.MaxPolicyAutomationRetries + 2 {
			installUUID := insertPendingInstall(t, host, policyID)
			hctx := hostctx.NewContext(ctx, host)
			require.NoError(t, svc.SaveHostSoftwareInstallResult(hctx, &fleet.HostSoftwareInstallResultPayload{
				HostID:                    host.ID,
				InstallUUID:               installUUID,
				PreInstallConditionOutput: new(""),
			}))
			attempt := getAttemptNumber(t, installUUID)
			require.NotNil(t, attempt)
			require.Equal(t, 0, *attempt, "every consecutive skip must store attempt_number=0")
		}

		// The count stays 0, so the cap is never reached and no retries queue.
		count, err := ds.CountHostSoftwareInstallAttempts(ctx, host.ID, installerID, policyID)
		require.NoError(t, err)
		require.Equal(t, 0, count, "skips never accumulate toward the retry cap")
		require.Equal(t, 0, countPendingRetries(t, host.ID), "skips never queue retries")
	})
}

// TestGetOrbitConfigWindowsSetupExperience verifies that GetOrbitConfig sets
// notifs.RunSetupExperience=true for Windows hosts whose MDM enrollment is
// in awaiting_configuration Pending or Active, and false otherwise (None,
// not-enrolled, non-Windows platforms).
func TestGetOrbitConfigWindowsSetupExperience(t *testing.T) {
	setupSvc := func(t *testing.T) (*mock.Store, fleet.Service, context.Context, *fleet.Host) {
		ds := new(mock.Store)
		license := &fleet.LicenseInfo{Tier: fleet.TierPremium}
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: license, SkipCreateTestUsers: true})

		host := &fleet.Host{
			ID:            1,
			OsqueryHostID: ptr.String("test"),
			UUID:          "host-uuid-1",
			Platform:      "windows",
		}

		appCfg := &fleet.AppConfig{
			MDM: fleet.MDM{
				EnabledAndConfigured:        true,
				WindowsEnabledAndConfigured: true,
			},
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return appCfg, nil
		}
		ds.GetHostOperatingSystemFunc = func(ctx context.Context, hostID uint) (*fleet.OperatingSystem, error) {
			return &fleet.OperatingSystem{Platform: "windows", Version: "10.0.19045"}, nil
		}
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) {
			return nil, nil
		}
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, h *fleet.Host) (bool, error) {
			return true, nil
		}
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool {
			return false
		}
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet, ConnectedToFleet: true}, nil
		}
		ds.GetHostAwaitingConfigurationFunc = func(ctx context.Context, hostUUID string) (bool, error) {
			return false, nil
		}
		// GetOrbitConfig persists the live sync capability on change; default to a no-op so subtests that don't assert it don't nil-panic.
		ds.SetMDMWindowsEnrollmentFleetdSyncCapableFunc = func(ctx context.Context, hostUUID string, capable bool) error {
			return nil
		}

		ctx = test.HostContext(ctx, host)
		return ds, svc, ctx, host
	}

	// withWindowsMDMSyncCapability returns a context whose X-Fleet-Capabilities advertise CapabilityWindowsMDMSync, as a Windows fleetd that
	// supports on-demand sync would send on its orbit config request.
	withWindowsMDMSyncCapability := func(ctx context.Context) context.Context {
		req := httptest.NewRequest("POST", "/api/fleet/orbit/config", nil)
		cm := fleet.CapabilityMap{fleet.CapabilityWindowsMDMSync: struct{}{}}
		req.Header.Set(fleet.CapabilitiesHeader, cm.String())
		return capabilities.NewContext(ctx, req)
	}

	t.Run("Windows host awaiting=Pending sets RunSetupExperience", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationPending}, nil
		}

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		assert.True(t, cfg.Notifications.RunSetupExperience)
		assert.False(t, cfg.Notifications.WindowsMDMSyncRequest)
		assert.True(t, ds.GetMDMWindowsHostConfigStateFuncInvoked)
	})

	t.Run("Windows host awaiting=Active sets RunSetupExperience", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationActive}, nil
		}

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		assert.True(t, cfg.Notifications.RunSetupExperience)
	})

	t.Run("Windows host awaiting=None, no pending commands sets neither notification", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationNone}, nil
		}

		cfg, err := svc.GetOrbitConfig(withWindowsMDMSyncCapability(ctx))
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.RunSetupExperience)
		assert.False(t, cfg.Notifications.WindowsMDMSyncRequest)
	})

	t.Run("Windows host with pending commands and capability sets WindowsMDMSyncRequest", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationNone, HasPendingCommands: true}, nil
		}

		cfg, err := svc.GetOrbitConfig(withWindowsMDMSyncCapability(ctx))
		require.NoError(t, err)
		assert.True(t, cfg.Notifications.WindowsMDMSyncRequest)
		assert.False(t, cfg.Notifications.RunSetupExperience)
		assert.True(t, ds.GetMDMWindowsHostConfigStateFuncInvoked)
	})

	t.Run("persists fleetd sync capability only on change", func(t *testing.T) {
		// Capability advertised but the persisted flag is still false -> write it true (so the OMA-DM session can relax the poll).
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationNone, FleetdSyncCapable: false}, nil
		}
		var wrote *bool
		ds.SetMDMWindowsEnrollmentFleetdSyncCapableFunc = func(ctx context.Context, hostUUID string, capable bool) error {
			wrote = &capable
			return nil
		}
		_, err := svc.GetOrbitConfig(withWindowsMDMSyncCapability(ctx))
		require.NoError(t, err)
		require.NotNil(t, wrote, "should persist the capability when it differs from the stored flag")
		assert.True(t, *wrote)

		// Capability advertised and the persisted flag already true -> no write.
		ds, svc, ctx, _ = setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationNone, FleetdSyncCapable: true}, nil
		}
		_, err = svc.GetOrbitConfig(withWindowsMDMSyncCapability(ctx))
		require.NoError(t, err)
		assert.False(t, ds.SetMDMWindowsEnrollmentFleetdSyncCapableFuncInvoked, "no write when the stored flag already matches")

		// No capability but the persisted flag is true (e.g. fleetd downgrade) -> write it false.
		ds, svc, ctx, _ = setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationNone, FleetdSyncCapable: true}, nil
		}
		wrote = nil
		ds.SetMDMWindowsEnrollmentFleetdSyncCapableFunc = func(ctx context.Context, hostUUID string, capable bool) error {
			wrote = &capable
			return nil
		}
		_, err = svc.GetOrbitConfig(ctx) // no capability header
		require.NoError(t, err)
		require.NotNil(t, wrote, "should clear the capability when fleetd stops advertising it")
		assert.False(t, *wrote)
	})

	t.Run("Windows host with pending commands but no capability does not set WindowsMDMSyncRequest", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationNone, HasPendingCommands: true}, nil
		}

		// ctx has no X-Fleet-Capabilities, as an older fleetd that cannot sync on demand would send.
		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.WindowsMDMSyncRequest)
	})

	t.Run("Windows host in ESP with pending commands prefers RunSetupExperience over sync", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{AwaitingConfiguration: fleet.WindowsMDMAwaitingConfigurationPending, HasPendingCommands: true}, nil
		}

		cfg, err := svc.GetOrbitConfig(withWindowsMDMSyncCapability(ctx))
		require.NoError(t, err)
		assert.True(t, cfg.Notifications.RunSetupExperience)
		assert.False(t, cfg.Notifications.WindowsMDMSyncRequest)
	})

	t.Run("Windows host not enrolled (NotFound) sets neither notification", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return nil, &orbitTestNotFoundErr{}
		}

		cfg, err := svc.GetOrbitConfig(withWindowsMDMSyncCapability(ctx))
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.RunSetupExperience)
		assert.False(t, cfg.Notifications.WindowsMDMSyncRequest)
	})

	t.Run("Windows host with non-NotFound lookup error returns the error", func(t *testing.T) {
		ds, svc, ctx, _ := setupSvc(t)
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return nil, errors.New("transient db error")
		}

		_, err := svc.GetOrbitConfig(ctx)
		require.Error(t, err)
	})

	t.Run("non-Windows host does not query Windows host config state", func(t *testing.T) {
		ds, svc, ctx, host := setupSvc(t)
		host.Platform = "darwin"

		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.RunSetupExperience)
		assert.False(t, cfg.Notifications.WindowsMDMSyncRequest)
		assert.False(t, ds.GetMDMWindowsHostConfigStateFuncInvoked,
			"non-Windows hosts must not invoke the Windows lookup")
	})
}

// orbitTestNotFoundErr is a minimal IsNotFound error type for orbit config tests.
type orbitTestNotFoundErr struct{}

func (e *orbitTestNotFoundErr) Error() string    { return "not found" }
func (e *orbitTestNotFoundErr) IsNotFound() bool { return true }

func rawJSON(s string) *json.RawMessage {
	r := json.RawMessage(s)
	return &r
}

func TestMaybeStampOrbitDebugFromAgentOptions(t *testing.T) {
	getInternal := func(svc fleet.Service) *Service {
		return ((svc.(validationMiddleware)).Service).(*Service)
	}

	t.Run("no agent options -> no stamp", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
		host := &fleet.Host{ID: 1}
		appCfg := &fleet.AppConfig{}

		err := getInternal(svc).maybeStampOrbitDebugFromAgentOptions(ctx, host, appCfg)
		require.NoError(t, err)
		require.False(t, ds.ExtendHostOrbitDebugUntilFuncInvoked)
	})

	t.Run("zero duration -> no stamp", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
		host := &fleet.Host{ID: 1}
		appCfg := &fleet.AppConfig{
			AgentOptions: rawJSON(`{"orbit": {"debug_logging_on_enroll_duration": 0}}`),
		}

		err := getInternal(svc).maybeStampOrbitDebugFromAgentOptions(ctx, host, appCfg)
		require.NoError(t, err)
		require.False(t, ds.ExtendHostOrbitDebugUntilFuncInvoked)
	})

	t.Run("global option set, no team -> stamps from app config", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
		host := &fleet.Host{ID: 42}
		appCfg := &fleet.AppConfig{
			AgentOptions: rawJSON(`{"orbit": {"debug_logging_on_enroll_duration": 3600}}`),
		}

		var gotID uint
		var gotUntil time.Time
		ds.ExtendHostOrbitDebugUntilFunc = func(ctx context.Context, hostID uint, until time.Time) error {
			gotID = hostID
			gotUntil = until
			return nil
		}

		before := time.Now()
		err := getInternal(svc).maybeStampOrbitDebugFromAgentOptions(ctx, host, appCfg)
		require.NoError(t, err)
		require.True(t, ds.ExtendHostOrbitDebugUntilFuncInvoked)
		require.Equal(t, host.ID, gotID)
		require.WithinDuration(t, before.Add(time.Hour), gotUntil, time.Minute)
	})

	t.Run("team option set -> stamps from team agent options, ignores global", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
		teamID := uint(7)
		host := &fleet.Host{ID: 99, TeamID: &teamID}
		appCfg := &fleet.AppConfig{
			// Team membership: team options win, global is ignored.
			AgentOptions: rawJSON(`{"orbit": {"debug_logging_on_enroll_duration": 86400}}`),
		}

		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			require.Equal(t, teamID, id)
			return rawJSON(`{"orbit": {"debug_logging_on_enroll_duration": 1800}}`), nil
		}
		var gotUntil time.Time
		ds.ExtendHostOrbitDebugUntilFunc = func(ctx context.Context, hostID uint, until time.Time) error {
			gotUntil = until
			return nil
		}

		before := time.Now()
		err := getInternal(svc).maybeStampOrbitDebugFromAgentOptions(ctx, host, appCfg)
		require.NoError(t, err)
		require.True(t, ds.TeamAgentOptionsFuncInvoked)
		require.True(t, ds.ExtendHostOrbitDebugUntilFuncInvoked)
		require.WithinDuration(t, before.Add(30*time.Minute), gotUntil, time.Minute)
	})

	t.Run("team has no agent options row -> no stamp, no fallback to global", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
		teamID := uint(7)
		host := &fleet.Host{ID: 99, TeamID: &teamID}
		appCfg := &fleet.AppConfig{
			AgentOptions: rawJSON(`{"orbit": {"debug_logging_on_enroll_duration": 3600}}`),
		}
		ds.TeamAgentOptionsFunc = func(ctx context.Context, id uint) (*json.RawMessage, error) {
			return nil, nil
		}

		err := getInternal(svc).maybeStampOrbitDebugFromAgentOptions(ctx, host, appCfg)
		require.NoError(t, err)
		require.True(t, ds.TeamAgentOptionsFuncInvoked)
		require.False(t, ds.ExtendHostOrbitDebugUntilFuncInvoked)
	})

	t.Run("over-cap value defensively clamped at 24h", func(t *testing.T) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{SkipCreateTestUsers: true})
		host := &fleet.Host{ID: 1}
		// Bypass the validator by stuffing a too-large value directly.
		appCfg := &fleet.AppConfig{
			AgentOptions: rawJSON(`{"orbit": {"debug_logging_on_enroll_duration": 360000}}`),
		}
		var gotUntil time.Time
		ds.ExtendHostOrbitDebugUntilFunc = func(ctx context.Context, hostID uint, until time.Time) error {
			gotUntil = until
			return nil
		}

		before := time.Now()
		err := getInternal(svc).maybeStampOrbitDebugFromAgentOptions(ctx, host, appCfg)
		require.NoError(t, err)
		require.WithinDuration(t, before.Add(fleet.MaxOrbitDebugLoggingOnEnrollDuration), gotUntil, time.Minute)
	})
}

func TestResolveOrbitDebugLogging(t *testing.T) {
	ctx := t.Context()
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	cases := []struct {
		name      string
		host      *fleet.Host
		inFlags   json.RawMessage
		wantDebug *bool
		wantFlags map[string]any
	}{
		{
			name:      "no host -> nil debug, flags unchanged",
			host:      nil,
			inFlags:   nil,
			wantDebug: nil,
		},
		{
			name:      "no override -> nil debug, flags unchanged",
			host:      &fleet.Host{},
			inFlags:   json.RawMessage(`{"distributed_interval":10}`),
			wantDebug: nil,
		},
		{
			name:      "unexpired override -> debug on, flags merged",
			host:      &fleet.Host{OrbitDebugUntil: &future},
			inFlags:   nil,
			wantDebug: new(true),
			wantFlags: map[string]any{
				"verbose": true,
			},
		},
		{
			name:      "unexpired override with admin flags -> merged",
			host:      &fleet.Host{OrbitDebugUntil: &future},
			inFlags:   json.RawMessage(`{"distributed_interval":10}`),
			wantDebug: new(true),
			wantFlags: map[string]any{
				"distributed_interval": float64(10),
				"verbose":              true,
			},
		},
		{
			name:      "admin verbose:false wins over debug-on",
			host:      &fleet.Host{OrbitDebugUntil: &future},
			inFlags:   json.RawMessage(`{"verbose":false}`),
			wantDebug: new(true),
			wantFlags: map[string]any{
				"verbose": false,
			},
		},
		{
			name:      "expired override is ignored",
			host:      &fleet.Host{OrbitDebugUntil: &past},
			inFlags:   nil,
			wantDebug: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFlags, gotDebug, err := resolveOrbitDebugLogging(ctx, tc.host, tc.inFlags)
			require.NoError(t, err)

			if tc.wantDebug == nil {
				require.Nil(t, gotDebug)
				require.Equal(t, tc.inFlags, gotFlags)
				return
			}

			require.NotNil(t, gotDebug)
			require.Equal(t, *tc.wantDebug, *gotDebug)
			var got map[string]any
			require.NoError(t, json.Unmarshal(gotFlags, &got))
			require.Equal(t, tc.wantFlags, got)
		})
	}
}

// TestGetOrbitConfigWindowsManagedLocalAccount covers the CreateWindowsManagedLocalAccount notification gating: it is
// set for any Windows MDM host (not just during the setup experience) when the team or No-team setting is enabled,
// fleetd advertises the capability, and it stops once the host has escrowed a password for its current enrollment.
func TestGetOrbitConfigWindowsManagedLocalAccount(t *testing.T) {
	// withMLACapability returns a context whose X-Fleet-Capabilities advertise the managed local
	// account capability, as a capable Windows fleetd would send.
	withMLACapability := func(ctx context.Context) context.Context {
		req := httptest.NewRequest("POST", "/api/fleet/orbit/config", nil)
		cm := fleet.CapabilityMap{fleet.CapabilityWindowsManagedLocalAccount: struct{}{}}
		req.Header.Set(fleet.CapabilitiesHeader, cm.String())
		return capabilities.NewContext(ctx, req)
	}

	setupSvc := func(t *testing.T, tier string, settingEnabled bool, awaiting fleet.WindowsMDMAwaitingConfiguration,
		alreadyEscrowed bool,
	) (*mock.Store, fleet.Service, context.Context) {
		ds := new(mock.Store)
		svc, ctx := newTestService(t, ds, nil, nil, &TestServerOpts{License: &fleet.LicenseInfo{Tier: tier}, SkipCreateTestUsers: true})

		host := &fleet.Host{ID: 1, OsqueryHostID: new("test"), UUID: "host-uuid-1", Platform: "windows"}
		appCfg := &fleet.AppConfig{MDM: fleet.MDM{EnabledAndConfigured: true, WindowsEnabledAndConfigured: true}}
		appCfg.MDM.WindowsSettings.EnableManagedLocalAccount = optjson.SetBool(settingEnabled)

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return appCfg, nil }
		ds.ListReadyToExecuteScriptsForHostFunc = func(ctx context.Context, hostID uint, onlyShowInternal bool) ([]*fleet.HostScriptResult, error) {
			return nil, nil
		}
		ds.ListReadyToExecuteSoftwareInstallsFunc = func(ctx context.Context, hostID uint) ([]string, error) { return nil, nil }
		ds.IsHostConnectedToFleetMDMFunc = func(ctx context.Context, h *fleet.Host) (bool, error) { return true, nil }
		ds.IsHostPendingEscrowFunc = func(ctx context.Context, hostID uint) bool { return false }
		ds.GetHostMDMFunc = func(ctx context.Context, hostID uint) (*fleet.HostMDM, error) {
			return &fleet.HostMDM{Enrolled: true, Name: fleet.WellKnownMDMFleet, ConnectedToFleet: true}, nil
		}
		ds.SetMDMWindowsEnrollmentFleetdSyncCapableFunc = func(ctx context.Context, hostUUID string, capable bool) error { return nil }
		ds.GetMDMWindowsHostConfigStateFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsHostConfigState, error) {
			return &fleet.MDMWindowsHostConfigState{
				AwaitingConfiguration:       awaiting,
				ManagedLocalAccountEscrowed: alreadyEscrowed,
			}, nil
		}

		ctx = test.HostContext(ctx, host)
		return ds, svc, ctx
	}

	// Enabling the setting provisions the whole fleet: a host long past its ESP is asked to create the account just like
	// one that just enrolled. This guards against the notification being re-scoped to the ESP.
	t.Run("set regardless of setup experience state", func(t *testing.T) {
		for _, awaiting := range []fleet.WindowsMDMAwaitingConfiguration{
			fleet.WindowsMDMAwaitingConfigurationPending,
			fleet.WindowsMDMAwaitingConfigurationActive,
			fleet.WindowsMDMAwaitingConfigurationNone,
		} {
			_, svc, ctx := setupSvc(t, fleet.TierPremium, true, awaiting, false)
			cfg, err := svc.GetOrbitConfig(withMLACapability(ctx))
			require.NoError(t, err)
			assert.True(t, cfg.Notifications.CreateWindowsManagedLocalAccount, "awaiting_configuration=%v", awaiting)
		}
	})

	// Idempotence: a host that already escrowed for this enrollment is left alone, so the account is not recreated and
	// the created activity is not logged on every poll.
	t.Run("already escrowed for this enrollment does not set it", func(t *testing.T) {
		_, svc, ctx := setupSvc(t, fleet.TierPremium, true, fleet.WindowsMDMAwaitingConfigurationNone, true)
		cfg, err := svc.GetOrbitConfig(withMLACapability(ctx))
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.CreateWindowsManagedLocalAccount)
	})

	t.Run("setting disabled does not set it", func(t *testing.T) {
		_, svc, ctx := setupSvc(t, fleet.TierPremium, false, fleet.WindowsMDMAwaitingConfigurationPending, false)
		cfg, err := svc.GetOrbitConfig(withMLACapability(ctx))
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.CreateWindowsManagedLocalAccount)
	})

	t.Run("missing capability does not set it", func(t *testing.T) {
		_, svc, ctx := setupSvc(t, fleet.TierPremium, true, fleet.WindowsMDMAwaitingConfigurationPending, false)
		// no capability header on the context
		cfg, err := svc.GetOrbitConfig(ctx)
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.CreateWindowsManagedLocalAccount)
	})

	t.Run("free license does not set it", func(t *testing.T) {
		_, svc, ctx := setupSvc(t, fleet.TierFree, true, fleet.WindowsMDMAwaitingConfigurationPending, false)
		cfg, err := svc.GetOrbitConfig(withMLACapability(ctx))
		require.NoError(t, err)
		assert.False(t, cfg.Notifications.CreateWindowsManagedLocalAccount)
	})
}

// TestEscrowWindowsManagedLocalAccountPassword covers the orbit escrow endpoint: eligibility via Windows MDM enrollment,
// input validation, the created activity, and that an escrow is stored even when the setting was toggled off after the
// notification (never orphan the on-device account).
func TestEscrowWindowsManagedLocalAccountPassword(t *testing.T) {
	setup := func(t *testing.T, enrolled bool, settingEnabled bool) (*mock.Store, fleet.Service, context.Context, *TestServerOpts) {
		ds := new(mock.Store)
		opts := &TestServerOpts{License: &fleet.LicenseInfo{Tier: fleet.TierPremium}, SkipCreateTestUsers: true}
		svc, ctx := newTestService(t, ds, nil, nil, opts)
		host := &fleet.Host{ID: 1, UUID: "host-uuid-1", OsqueryHostID: new("test")}
		ctx = test.HostContext(ctx, host)

		ds.MDMWindowsGetEnrolledDeviceWithHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMWindowsEnrolledDevice, error) {
			if !enrolled {
				return nil, newNotFoundError()
			}
			return &fleet.MDMWindowsEnrolledDevice{HostUUID: hostUUID}, nil
		}
		appCfg := &fleet.AppConfig{MDM: fleet.MDM{WindowsEnabledAndConfigured: true}}
		appCfg.MDM.WindowsSettings.EnableManagedLocalAccount = optjson.SetBool(settingEnabled)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return appCfg, nil }
		ds.SaveHostManagedLocalAccountFromEscrowFunc = func(ctx context.Context, hostUUID, plaintextPassword string) error { return nil }
		ds.ReportManagedLocalAccountEscrowErrorFunc = func(ctx context.Context, hostUUID, clientError string) error { return nil }
		ds.SetMDMWindowsManagedLocalAccountEscrowedFunc = func(ctx context.Context, hostUUID string, escrowed bool) (bool, error) {
			return escrowed, nil
		}
		return ds, svc, ctx, opts
	}

	t.Run("host without Windows MDM enrollment is rejected", func(t *testing.T) {
		ds, svc, ctx, _ := setup(t, false, true)
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "pw", "")
		require.Error(t, err)
		var badReq *fleet.BadRequestError
		require.ErrorAs(t, err, &badReq)
		require.False(t, ds.SaveHostManagedLocalAccountFromEscrowFuncInvoked)
	})

	t.Run("invalid password is rejected", func(t *testing.T) {
		for name, password := range map[string]string{
			"empty":    "",
			"too long": strings.Repeat("a", managedLocalAccountMaxPasswordLength+1),
		} {
			t.Run(name, func(t *testing.T) {
				ds, svc, ctx, _ := setup(t, true, true)
				err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, password, "")
				require.Error(t, err)
				require.False(t, ds.SaveHostManagedLocalAccountFromEscrowFuncInvoked)
			})
		}
	})

	t.Run("client error is recorded and no password is stored", func(t *testing.T) {
		ds, svc, ctx, opts := setup(t, true, true)
		activityLogged := false
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			activityLogged = true
			return nil
		}
		var reportedError string
		ds.ReportManagedLocalAccountEscrowErrorFunc = func(ctx context.Context, hostUUID, clientError string) error {
			reportedError = clientError
			return nil
		}
		var escrowedFlag bool
		ds.SetMDMWindowsManagedLocalAccountEscrowedFunc = func(ctx context.Context, hostUUID string, escrowed bool) (bool, error) {
			escrowedFlag = escrowed
			return true, nil
		}
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "", "netapi32 add failed")
		require.NoError(t, err)
		require.True(t, ds.ReportManagedLocalAccountEscrowErrorFuncInvoked)
		require.Equal(t, "netapi32 add failed", reportedError)
		require.False(t, ds.SaveHostManagedLocalAccountFromEscrowFuncInvoked)
		require.False(t, activityLogged)
		// The flag is cleared so the host keeps being asked and a transient failure self-heals.
		require.True(t, ds.SetMDMWindowsManagedLocalAccountEscrowedFuncInvoked)
		require.False(t, escrowedFlag)
	})

	t.Run("client error is truncated by rune to fit the column", func(t *testing.T) {
		ds, svc, ctx, _ := setup(t, true, true)
		var reportedError string
		ds.ReportManagedLocalAccountEscrowErrorFunc = func(ctx context.Context, hostUUID, clientError string) error {
			reportedError = clientError
			return nil
		}
		// Multi-byte runes so a byte-wise truncation would produce invalid UTF-8.
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "", strings.Repeat("é", 400))
		require.NoError(t, err)
		require.Equal(t, 255, utf8.RuneCountInString(reportedError))
		require.True(t, utf8.ValidString(reportedError))
	})

	t.Run("successful escrow stores the password and logs the created activity once", func(t *testing.T) {
		ds, svc, ctx, opts := setup(t, true, true)
		var savedPassword string
		ds.SaveHostManagedLocalAccountFromEscrowFunc = func(ctx context.Context, hostUUID, plaintextPassword string) error {
			savedPassword = plaintextPassword
			return nil
		}
		activityCount := 0
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, a activity_api.ActivityDetails) error {
			require.Equal(t, fleet.ActivityTypeCreatedManagedLocalAccount{}.ActivityName(), a.ActivityName())
			activityCount++
			return nil
		}
		var escrowedFlag bool
		ds.SetMDMWindowsManagedLocalAccountEscrowedFunc = func(ctx context.Context, hostUUID string, escrowed bool) (bool, error) {
			escrowedFlag = escrowed
			return true, nil
		}
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "device-generated-pw", "")
		require.NoError(t, err)
		require.True(t, ds.SaveHostManagedLocalAccountFromEscrowFuncInvoked)
		require.Equal(t, "device-generated-pw", savedPassword)
		require.Equal(t, 1, activityCount)
		// Marking the enrollment provisioned is what stops the host being asked again.
		require.True(t, escrowedFlag)
	})

	// A device that re-sends an escrow it already made stores the password again but must not claim a
	// second account, mirroring how BitLocker only logs when the key was actually archived.
	t.Run("re-sent escrow stores the password but does not log the activity again", func(t *testing.T) {
		ds, svc, ctx, opts := setup(t, true, true)
		activityCount := 0
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error {
			activityCount++
			return nil
		}
		// The enrollment is already marked provisioned, so the flag does not change.
		ds.SetMDMWindowsManagedLocalAccountEscrowedFunc = func(ctx context.Context, hostUUID string, escrowed bool) (bool, error) {
			return false, nil
		}
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "device-generated-pw", "")
		require.NoError(t, err)
		require.True(t, ds.SaveHostManagedLocalAccountFromEscrowFuncInvoked)
		require.Zero(t, activityCount)
	})

	t.Run("stores the password even when the setting was disabled after the notification", func(t *testing.T) {
		ds, svc, ctx, opts := setup(t, true, false)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error { return nil }
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "device-generated-pw", "")
		require.NoError(t, err)
		require.True(t, ds.SaveHostManagedLocalAccountFromEscrowFuncInvoked)
	})

	// A failed save must surface as an error rather than a silent success, and must not mark the enrollment provisioned.
	t.Run("failed save is reported and leaves the host to be asked again", func(t *testing.T) {
		ds, svc, ctx, _ := setup(t, true, true)
		ds.SaveHostManagedLocalAccountFromEscrowFunc = func(ctx context.Context, hostUUID, plaintextPassword string) error {
			return errors.New("transient db failure")
		}
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "device-generated-pw", "")
		require.Error(t, err)
		require.False(t, ds.SetMDMWindowsManagedLocalAccountEscrowedFuncInvoked)
	})

	// The setting check only decides whether to warn, so a failure to read it must never cost the
	// password: the account already exists on the device and this is the only chance to record it. (The fallback is re-creating the account.)
	t.Run("stores the password even when the setting cannot be read", func(t *testing.T) {
		ds, svc, ctx, opts := setup(t, true, true)
		opts.ActivityMock.NewActivityFunc = func(_ context.Context, _ *activity_api.User, _ activity_api.ActivityDetails) error { return nil }
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) { return nil, errors.New("transient db failure") }
		err := svc.EscrowWindowsManagedLocalAccountPassword(ctx, "device-generated-pw", "")
		require.NoError(t, err)
		require.True(t, ds.SaveHostManagedLocalAccountFromEscrowFuncInvoked)
	})
}

func TestEnrollOrbitEndUserAuthBypass(t *testing.T) {
	// When end user authentication is required and the enrolling agent does not
	// advertise the end_user_auth capability (for example an older agent that
	// does not set the X-Fleet-Capabilities header), the
	// AllowOrbitEndUserAuthBypass config flag decides whether enrollment is
	// blocked or allowed.
	newSvc := func(t *testing.T, allowBypass bool) (*mock.DataStore, fleet.Service, context.Context) {
		// mock.Store hard-codes EnrollOrbit to return (nil, nil), which would make
		// the bypass-allowed success path panic. Use the underlying mock.DataStore so
		// EnrollOrbitFunc is honored.
		ds := new(mock.DataStore)
		cfg := config.TestConfig()
		cfg.MDM.AllowOrbitEndUserAuthBypass = allowBypass
		svc, ctx := newTestServiceWithConfig(t, ds, cfg, nil, nil)

		// Global enroll secret (no team) with end user auth required at the app-config level.
		ds.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
			return &fleet.EnrollSecret{Secret: secret}, nil
		}
		ds.GetHostIdentityCertByNameFunc = func(ctx context.Context, name string) (*hostidentity_types.HostIdentityCertificate, error) {
			return nil, nil
		}
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.EnabledAndConfigured = true
			ac.MDM.MacOSSetup.EnableEndUserAuthentication = true
			return ac, nil
		}
		// No IdP account linked and not previously enrolled: a genuine first-time enrollment.
		ds.GetMDMIdPAccountByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMIdPAccount, error) {
			return nil, nil
		}
		ds.HostPreviouslyOrbitEnrolledFunc = func(ctx context.Context, hostInfo fleet.OrbitHostInfo, isMDMEnabled bool) (bool, error) {
			return false, nil
		}
		ds.EnrollOrbitFunc = func(ctx context.Context, opts ...fleet.DatastoreEnrollOrbitOption) (*fleet.Host, error) {
			return &fleet.Host{ID: 1, UUID: "host-uuid-1", Platform: "ubuntu"}, nil
		}
		ds.MaybeAssociateHostWithScimUserFunc = func(ctx context.Context, hostID uint) error {
			return nil
		}
		return ds, svc, ctx
	}

	hostInfo := fleet.OrbitHostInfo{
		HardwareUUID:   "host-uuid-1",
		HardwareSerial: "serial-1",
		Hostname:       "host-1",
		Platform:       "ubuntu",
		PlatformLike:   "debian",
	}

	// noEUACtx builds a request context advertising only unrelated capabilities,
	// simulating an agent that does not support end user auth.
	noEUACtx := func(ctx context.Context) context.Context {
		req := httptest.NewRequest("POST", "/api/fleet/orbit/enroll", nil)
		req.Header.Set(fleet.CapabilitiesHeader, "foo,bar")
		return capabilities.NewContext(ctx, req)
	}

	t.Run("flag disabled blocks enrollment", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, false)
		_, err := svc.EnrollOrbit(noEUACtx(ctx), hostInfo, "secret", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "END_USER_AUTH_REQUIRED")
		require.False(t, ds.EnrollOrbitFuncInvoked, "no host must be enrolled when EUA is required and the flag is off")
	})

	t.Run("flag enabled allows enrollment", func(t *testing.T) {
		ds, svc, ctx := newSvc(t, true)
		nodeKey, err := svc.EnrollOrbit(noEUACtx(ctx), hostInfo, "secret", "")
		require.NoError(t, err)
		require.NotEmpty(t, nodeKey)
		require.True(t, ds.EnrollOrbitFuncInvoked)
	})

	t.Run("flag enabled still gates agents that support EUA", func(t *testing.T) {
		// The escape hatch only applies to agents that do not support end user
		// auth. A modern agent that advertises the capability must still go
		// through the SSO flow even when the flag is on.
		ds, svc, ctx := newSvc(t, true)
		euaCtx := func(ctx context.Context) context.Context {
			req := httptest.NewRequest("POST", "/api/fleet/orbit/enroll", nil)
			req.Header.Set(fleet.CapabilitiesHeader, string(fleet.CapabilityEndUserAuth))
			return capabilities.NewContext(ctx, req)
		}
		_, err := svc.EnrollOrbit(euaCtx(ctx), hostInfo, "secret", "")
		require.Error(t, err)
		require.Contains(t, err.Error(), "END_USER_AUTH_REQUIRED")
		require.False(t, ds.EnrollOrbitFuncInvoked)
	})

	t.Run("windows EUA token takes precedence over the flag", func(t *testing.T) {
		// A Windows host presenting an EUA token must go through the token path even when
		// the flag is on and the client omits the capability — the token case is ordered
		// first. wstepCertManager is unset in this harness, so the token path falls back to
		// END_USER_AUTH_REQUIRED; the point is that the flag's bypass does not fire (no host
		// is enrolled), proving the token case wins.
		ds, svc, ctx := newSvc(t, true)
		winHost := hostInfo
		winHost.Platform = "windows"
		winHost.PlatformLike = ""
		_, err := svc.EnrollOrbit(noEUACtx(ctx), winHost, "secret", "some-eua-token")
		require.Error(t, err)
		require.Contains(t, err.Error(), "END_USER_AUTH_REQUIRED")
		require.False(t, ds.EnrollOrbitFuncInvoked, "the flag bypass must not fire when an EUA token is present")
	})
}
