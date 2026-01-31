package osquery_utils

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
)

func TestIsDiskEncryptionEnabledForHost(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()

	t.Run("team has disk encryption enabled", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: ptr.Uint(1)}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			require.Equal(t, uint(1), teamID)
			return &fleet.TeamMDM{
				EnableDiskEncryption: true,
			}, nil
		}

		result := IsDiskEncryptionEnabledForHost(ctx, logger, ds, host)
		require.True(t, result)
		require.True(t, ds.TeamMDMConfigFuncInvoked)
	})

	t.Run("team has disk encryption disabled", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: ptr.Uint(1)}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{
				EnableDiskEncryption: false,
			}, nil
		}

		result := IsDiskEncryptionEnabledForHost(ctx, logger, ds, host)
		require.False(t, result)
		require.True(t, ds.TeamMDMConfigFuncInvoked)
	})

	t.Run("team has disk encryption disabled even when global is enabled", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: ptr.Uint(1)}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return &fleet.TeamMDM{
				EnableDiskEncryption: false,
			}, nil
		}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			require.Fail(t, "AppConfig should not be called when host has a team")
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
			}, nil
		}

		result := IsDiskEncryptionEnabledForHost(ctx, logger, ds, host)
		require.False(t, result, "Team setting should take precedence over global setting")
		require.True(t, ds.TeamMDMConfigFuncInvoked)
		require.False(t, ds.AppConfigFuncInvoked, "Global config should not be checked when host is on a team")
	})

	t.Run("global disk encryption enabled (no team)", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: nil}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(true),
				},
			}, nil
		}

		result := IsDiskEncryptionEnabledForHost(ctx, logger, ds, host)
		require.True(t, result)
		require.True(t, ds.AppConfigFuncInvoked)
	})

	t.Run("global disk encryption disabled (no team)", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: nil}

		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			return &fleet.AppConfig{
				MDM: fleet.MDM{
					EnableDiskEncryption: optjson.SetBool(false),
				},
			}, nil
		}

		result := IsDiskEncryptionEnabledForHost(ctx, logger, ds, host)
		require.False(t, result)
		require.True(t, ds.AppConfigFuncInvoked)
	})

	t.Run("error getting team config returns false", func(t *testing.T) {
		ds := new(mock.Store)
		host := &fleet.Host{ID: 1, TeamID: ptr.Uint(1)}

		ds.TeamMDMConfigFunc = func(ctx context.Context, teamID uint) (*fleet.TeamMDM, error) {
			return nil, &fleet.Error{Message: "db error"}
		}

		result := IsDiskEncryptionEnabledForHost(ctx, logger, ds, host)
		require.False(t, result)
	})
}
