package mdmlifecycle

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/contexts/license"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/stretchr/testify/require"
)

func nopNewActivity(ctx context.Context, user *fleet.User, details fleet.ActivityDetails) error {
	return nil
}

func TestDoUnsupportedParams(t *testing.T) {
	ds := new(mock.Store)
	lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)

	err := lc.Do(context.Background(), HostOptions{})
	require.ErrorContains(t, err, "unsupported platform")

	err = lc.Do(context.Background(), HostOptions{Platform: "linux"})
	require.ErrorContains(t, err, "unsupported platform")

	err = lc.Do(context.Background(), HostOptions{Platform: "darwin", Action: "invalid"})
	require.ErrorContains(t, err, "unknown action")

	err = lc.Do(context.Background(), HostOptions{Platform: "windows", Action: "invalid"})
	require.ErrorContains(t, err, "unknown action")
}

func TestDoParamValidation(t *testing.T) {
	ds := new(mock.Store)
	lf := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
	ctx := context.Background()

	cases := []struct {
		platform string
		action   HostAction
		wantErr  bool
	}{

		{"darwin", HostActionTurnOn, true},
		{"darwin", HostActionTurnOff, true},
		{"darwin", HostActionReset, true},
		{"darwin", HostActionDelete, true},
		{"windows", HostActionTurnOn, true},
		{"windows", HostActionTurnOff, true},
		{"windows", HostActionReset, true},
		{"windows", HostActionDelete, false},
	}

	for _, tc := range cases {
		err := lf.Do(ctx, HostOptions{
			Action:   tc.action,
			Platform: tc.platform,
		})
		if tc.wantErr {
			require.ErrorContains(t, err, "required")
		} else {
			require.NoError(t, err)
		}
	}
}

// TestDeleteAppleDuplicateDEPHost verifies that deleting one of a set of
// duplicate DEP hosts (same serial) does not recreate a pending "ghost" host
// when another DEP-assigned host with that serial still exists, while a host
// with no duplicate is still restored as before.
func TestDeleteAppleDuplicateDEPHost(t *testing.T) {
	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})

	const serial = "ABC123XYZ"
	host := &fleet.Host{ID: 1, HardwareSerial: serial, Platform: "darwin"}

	newDS := func(dupExists bool) *mock.Store {
		ds := new(mock.Store)
		ds.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
			ac := &fleet.AppConfig{}
			ac.MDM.AppleBMEnabledAndConfigured = true
			return ac, nil
		}
		abmTokenID := uint(7)
		ds.GetHostDEPAssignmentFunc = func(ctx context.Context, hostID uint) (*fleet.HostDEPAssignment, error) {
			return &fleet.HostDEPAssignment{HostID: hostID, ABMTokenID: &abmTokenID}, nil
		}
		ds.ReconcileDuplicateDEPHostOnDeleteFunc = func(ctx context.Context, s, platform string, deletedHostID uint) (bool, error) {
			require.Equal(t, serial, s)
			require.Equal(t, host.Platform, platform)
			require.Equal(t, host.ID, deletedHostID)
			return dupExists, nil
		}
		ds.RestoreMDMApplePendingDEPHostFunc = func(ctx context.Context, h *fleet.Host) error {
			return nil
		}
		return ds
	}

	t.Run("duplicate exists, ghost host not restored", func(t *testing.T) {
		ds := newDS(true)

		lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
		err := lc.Do(ctx, HostOptions{Action: HostActionDelete, Platform: "darwin", Host: host})
		require.NoError(t, err)

		require.True(t, ds.ReconcileDuplicateDEPHostOnDeleteFuncInvoked)
		require.False(t, ds.RestoreMDMApplePendingDEPHostFuncInvoked)
	})

	t.Run("no duplicate, ghost host restored", func(t *testing.T) {
		ds := newDS(false)
		// "No team" default keeps getDefaultTeamForABMToken from needing TeamExists.
		ds.GetABMTokenByIDFunc = func(ctx context.Context, tokenID uint) (*fleet.ABMToken, error) {
			return &fleet.ABMToken{ID: tokenID}, nil
		}
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) {
			return job, nil
		}

		lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
		err := lc.Do(ctx, HostOptions{Action: HostActionDelete, Platform: "darwin", Host: host})
		require.NoError(t, err)

		require.True(t, ds.ReconcileDuplicateDEPHostOnDeleteFuncInvoked)
		require.True(t, ds.RestoreMDMApplePendingDEPHostFuncInvoked)
	})
}
