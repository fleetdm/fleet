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

// TestReconcileHostNameEnforcementOnEnrollment verifies that both Apple
// enrollment lifecycle branches — turn-on (TokenUpdate) and reset (Authenticate,
// covering re-enrollment) — reconcile the host's host-name template enforcement,
// so a host enrolling into a team with a template gets a queued row.
func TestReconcileHostNameEnforcementOnEnrollment(t *testing.T) {
	ctx := license.NewContext(context.Background(), &fleet.LicenseInfo{Tier: fleet.TierPremium})
	const hostID = uint(99)

	t.Run("turn-on reconciles with the enrolled host id", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, uuid string) (*fleet.NanoEnrollment, error) {
			return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
		}
		ds.GetHostMDMCheckinInfoFunc = func(ctx context.Context, uuid string) (*fleet.HostMDMCheckinInfo, error) {
			return &fleet.HostMDMCheckinInfo{HostID: hostID, Platform: "darwin"}, nil
		}
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) { return job, nil }
		var gotIDs []uint
		ds.ReconcileHostDeviceNamesForHostsFunc = func(ctx context.Context, hostIDs []uint) error {
			gotIDs = hostIDs
			return nil
		}

		lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
		require.NoError(t, lc.Do(ctx, HostOptions{Action: HostActionTurnOn, Platform: "darwin", UUID: "host-uuid"}))
		require.True(t, ds.ReconcileHostDeviceNamesForHostsFuncInvoked)
		require.Equal(t, []uint{hostID}, gotIDs)
	})

	t.Run("turn-on reconciles on the DEP branch before its early return", func(t *testing.T) {
		ds := new(mock.Store)
		ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, uuid string) (*fleet.NanoEnrollment, error) {
			return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 1}, nil
		}
		// DEPAssignedToFleet takes the DEP branch, which queues a job and returns
		// early — the reconcile must run before that branch.
		ds.GetHostMDMCheckinInfoFunc = func(ctx context.Context, uuid string) (*fleet.HostMDMCheckinInfo, error) {
			return &fleet.HostMDMCheckinInfo{HostID: hostID, Platform: "darwin", DEPAssignedToFleet: true}, nil
		}
		ds.NewJobFunc = func(ctx context.Context, job *fleet.Job) (*fleet.Job, error) { return job, nil }
		var gotIDs []uint
		ds.ReconcileHostDeviceNamesForHostsFunc = func(ctx context.Context, hostIDs []uint) error {
			gotIDs = hostIDs
			return nil
		}

		lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
		require.NoError(t, lc.Do(ctx, HostOptions{Action: HostActionTurnOn, Platform: "darwin", UUID: "host-uuid"}))
		require.True(t, ds.ReconcileHostDeviceNamesForHostsFuncInvoked)
		require.Equal(t, []uint{hostID}, gotIDs)
		require.True(t, ds.NewJobFuncInvoked, "DEP branch should have been taken")
	})

	t.Run("turn-on skips reconcile when the enrollment is not ready", func(t *testing.T) {
		ds := new(mock.Store)
		// TokenUpdateTally != 1 makes turnOnApple short-circuit before reconciling.
		ds.GetNanoMDMEnrollmentFunc = func(ctx context.Context, uuid string) (*fleet.NanoEnrollment, error) {
			return &fleet.NanoEnrollment{Enabled: true, Type: "Device", TokenUpdateTally: 2}, nil
		}
		ds.ReconcileHostDeviceNamesForHostsFunc = func(ctx context.Context, hostIDs []uint) error { return nil }

		lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
		require.NoError(t, lc.Do(ctx, HostOptions{Action: HostActionTurnOn, Platform: "darwin", UUID: "host-uuid"}))
		require.False(t, ds.ReconcileHostDeviceNamesForHostsFuncInvoked)
	})

	t.Run("reset reconciles with the upserted host id", func(t *testing.T) {
		ds := new(mock.Store)
		ds.MDMAppleUpsertHostFunc = func(ctx context.Context, mdmHost *fleet.Host, fromPersonalEnrollment bool) error {
			mdmHost.ID = hostID
			return nil
		}
		ds.MDMResetEnrollmentFunc = func(ctx context.Context, uuid string, scepRenewalInProgress bool) error { return nil }
		var gotIDs []uint
		ds.ReconcileHostDeviceNamesForHostsFunc = func(ctx context.Context, hostIDs []uint) error {
			gotIDs = hostIDs
			return nil
		}

		lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
		require.NoError(t, lc.Do(ctx, HostOptions{
			Action: HostActionReset, Platform: "darwin",
			UUID: "host-uuid", HardwareSerial: "serial", HardwareModel: "MacBookPro",
		}))
		require.True(t, ds.ReconcileHostDeviceNamesForHostsFuncInvoked)
		require.Equal(t, []uint{hostID}, gotIDs)
	})

	t.Run("reset skips reconcile during SCEP renewal", func(t *testing.T) {
		ds := new(mock.Store)
		ds.MDMResetEnrollmentFunc = func(ctx context.Context, uuid string, scepRenewalInProgress bool) error { return nil }
		ds.ReconcileHostDeviceNamesForHostsFunc = func(ctx context.Context, hostIDs []uint) error { return nil }

		lc := New(ds, slog.New(slog.DiscardHandler), nopNewActivity)
		require.NoError(t, lc.Do(ctx, HostOptions{
			Action: HostActionReset, Platform: "darwin",
			UUID: "host-uuid", HardwareSerial: "serial", HardwareModel: "MacBookPro",
			SCEPRenewalInProgress: true,
		}))
		require.False(t, ds.ReconcileHostDeviceNamesForHostsFuncInvoked)
	})
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
