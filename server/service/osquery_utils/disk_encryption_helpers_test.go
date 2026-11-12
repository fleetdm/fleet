package osquery_utils

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func TestIsDiskEncryptionEscrowEnabledForHost(t *testing.T) {
	ctx := t.Context()
	logger := slog.New(slog.DiscardHandler)

	// a mixed state: macOS and Windows on, Linux off
	mixedTeamMDM := &fleet.TeamMDM{
		MacOSSettings: fleet.MacOSSettings{
			EnableDiskEncryption:          optjson.SetBool(true),
			EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
		},
		WindowsSettings: fleet.WindowsSettings{
			EnableDiskEncryption: optjson.SetBool(true),
		},
		LinuxSettings: fleet.LinuxSettings{
			EnableEscrowDiskEncryptionKey: optjson.SetBool(false),
		},
	}
	mixedGlobalMDM := fleet.MDM{
		MacOSSettings: fleet.MacOSSettings{
			EnableDiskEncryption:          optjson.SetBool(true),
			EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
		},
		WindowsSettings: fleet.WindowsSettings{
			EnableDiskEncryption: optjson.SetBool(true),
		},
		LinuxSettings: fleet.LinuxSettings{
			EnableEscrowDiskEncryptionKey: optjson.SetBool(false),
		},
	}

	for _, tc := range []struct {
		name     string
		platform string
		want     bool
	}{
		{"darwin follows the macOS escrow setting", "darwin", true},
		{"windows follows the windows setting", "windows", true},
		{"linux follows the linux escrow setting", "ubuntu", false},
		{"unknown platform is never enabled", "chrome", false},
	} {
		t.Run("team "+tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			host := &fleet.Host{ID: 1, TeamID: new(uint(1)), Platform: tc.platform}

			ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
				require.Equal(t, uint(1), teamID)
				return mixedTeamMDM, nil
			}

			require.Equal(t, tc.want, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
			require.True(t, ds.TeamMDMConfigFuncInvoked)
			require.False(t, ds.AppConfigFuncInvoked, "Global config should not be checked when host is on a team")
		})

		t.Run("global "+tc.name, func(t *testing.T) {
			ds := new(mock.Store)
			host := &fleet.Host{ID: 1, TeamID: nil, Platform: tc.platform}

			ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{MDM: mixedGlobalMDM}, nil
			}

			require.Equal(t, tc.want, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
			require.True(t, ds.AppConfigFuncInvoked)
		})
	}

	t.Run("macOS: escrow alone is enough", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: new(uint(1)), Platform: "darwin"}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{
				MacOSSettings: fleet.MacOSSettings{
					EnableEscrowDiskEncryptionKey: optjson.SetBool(true),
				},
			}, nil
		}

		require.True(t, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
	})

	t.Run("macOS: enforcement alone escrows nothing", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: new(uint(1)), Platform: "darwin"}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{
				MacOSSettings: fleet.MacOSSettings{
					EnableDiskEncryption: optjson.SetBool(true),
				},
			}, nil
		}

		// the profile carries no escrow payload, so no key reaches Fleet and
		// ingesting one would store a key nobody asked to be escrowed
		require.False(t, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
	})

	t.Run("team has disk encryption disabled", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: new(uint(1)), Platform: "darwin"}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{}, nil
		}

		require.False(t, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
		require.True(t, ds.TeamMDMConfigFuncInvoked)
	})

	t.Run("nil team config returns false", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: new(uint(1)), Platform: "darwin"}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return nil, nil
		}

		require.False(t, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
	})

	t.Run("error getting team config returns false", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: new(uint(1)), Platform: "darwin"}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return nil, &fleet.Error{Message: "db error"}
		}

		require.False(t, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
	})

	t.Run("error getting app config returns false", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: nil, Platform: "darwin"}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return nil, &fleet.Error{Message: "db error"}
		}

		require.False(t, IsDiskEncryptionEscrowEnabledForHost(ctx, logger, ds, host))
	})
}
