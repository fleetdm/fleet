package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/fleetdm/fleet/v4/server/authz"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mock"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListWindowsEnforcementProfiles(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expected := []*fleet.WindowsEnforcementProfile{
		{ProfileUUID: "e-uuid-1", Name: "policy1"},
		{ProfileUUID: "e-uuid-2", Name: "policy2"},
	}

	ds.ListWindowsEnforcementProfilesFunc = func(ctx context.Context, teamID *uint) ([]*fleet.WindowsEnforcementProfile, error) {
		return expected, nil
	}

	profiles, err := svc.ListWindowsEnforcementProfiles(test.UserContext(ctx, test.UserAdmin), nil)
	require.NoError(t, err)
	require.Len(t, profiles, 2)
	assert.Equal(t, "e-uuid-1", profiles[0].ProfileUUID)

	_, err = svc.ListWindowsEnforcementProfiles(test.UserContext(ctx, test.UserNoRoles), nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)

	_, err = svc.ListWindowsEnforcementProfiles(ctx, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestNewWindowsEnforcementProfile(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	teamID := uint(1)
	rawPolicy := []byte(`{"registry":[]}`)
	callCount := 0

	ds.ListWindowsEnforcementProfilesFunc = func(ctx context.Context, tid *uint) ([]*fleet.WindowsEnforcementProfile, error) {
		callCount++
		if callCount == 1 {
			return nil, nil // first call: no existing profiles
		}
		return []*fleet.WindowsEnforcementProfile{
			{ProfileUUID: "e-uuid-1", TeamID: tid, Name: "test-policy", RawPolicy: rawPolicy},
		}, nil
	}

	ds.BatchSetWindowsEnforcementProfilesFunc = func(ctx context.Context, tid *uint, profiles []*fleet.WindowsEnforcementProfile) error {
		require.NotNil(t, tid)
		assert.Equal(t, teamID, *tid)
		require.Len(t, profiles, 1)
		assert.Equal(t, "test-policy", profiles[0].Name)
		return nil
	}

	profile, err := svc.NewWindowsEnforcementProfile(test.UserContext(ctx, test.UserAdmin), teamID, "test-policy", rawPolicy)
	require.NoError(t, err)
	require.NotNil(t, profile)
	assert.Equal(t, "test-policy", profile.Name)
	assert.True(t, ds.BatchSetWindowsEnforcementProfilesFuncInvoked)

	// unauthorized
	callCount = 0
	_, err = svc.NewWindowsEnforcementProfile(test.UserContext(ctx, test.UserNoRoles), teamID, "test-policy", rawPolicy)
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestNewWindowsEnforcementProfileReplace(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	teamID := uint(0)
	rawPolicy := []byte(`{"registry":[{"path":"HKLM\\Test","name":"v","type":"dword","value":1}]}`)
	updatedPolicy := []byte(`{"registry":[{"path":"HKLM\\Test","name":"v","type":"dword","value":2}]}`)

	ds.ListWindowsEnforcementProfilesFunc = func(ctx context.Context, tid *uint) ([]*fleet.WindowsEnforcementProfile, error) {
		return []*fleet.WindowsEnforcementProfile{
			{ProfileUUID: "e-uuid-1", Name: "existing", RawPolicy: rawPolicy},
		}, nil
	}

	var batchProfiles []*fleet.WindowsEnforcementProfile
	ds.BatchSetWindowsEnforcementProfilesFunc = func(ctx context.Context, tid *uint, profiles []*fleet.WindowsEnforcementProfile) error {
		batchProfiles = profiles
		return nil
	}

	profile, err := svc.NewWindowsEnforcementProfile(test.UserContext(ctx, test.UserAdmin), teamID, "existing", updatedPolicy)
	require.NoError(t, err)
	require.NotNil(t, profile)

	// Should replace existing profile, not add a new one
	require.Len(t, batchProfiles, 1)
	assert.Equal(t, updatedPolicy, batchProfiles[0].RawPolicy)
}

func TestGetWindowsEnforcementProfile(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	expected := &fleet.WindowsEnforcementProfile{
		ProfileUUID: "e-uuid-1",
		Name:        "test-policy",
	}

	ds.GetWindowsEnforcementProfileFunc = func(ctx context.Context, uuid string) (*fleet.WindowsEnforcementProfile, error) {
		if uuid == "e-uuid-1" {
			return expected, nil
		}
		return nil, &notFoundError{}
	}

	profile, err := svc.GetWindowsEnforcementProfile(test.UserContext(ctx, test.UserAdmin), "e-uuid-1")
	require.NoError(t, err)
	assert.Equal(t, "e-uuid-1", profile.ProfileUUID)
	assert.Equal(t, "test-policy", profile.Name)

	_, err = svc.GetWindowsEnforcementProfile(test.UserContext(ctx, test.UserNoRoles), "e-uuid-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestDeleteWindowsEnforcementProfile(t *testing.T) {
	ds := new(mock.Store)
	svc, ctx := newTestService(t, ds, nil, nil)

	profile := &fleet.WindowsEnforcementProfile{
		ProfileUUID: "e-uuid-1",
		Name:        "test-policy",
	}

	ds.GetWindowsEnforcementProfileFunc = func(ctx context.Context, uuid string) (*fleet.WindowsEnforcementProfile, error) {
		return profile, nil
	}

	deleted := false
	ds.DeleteWindowsEnforcementProfileFunc = func(ctx context.Context, uuid string) error {
		deleted = true
		assert.Equal(t, "e-uuid-1", uuid)
		return nil
	}

	err := svc.DeleteWindowsEnforcementProfile(test.UserContext(ctx, test.UserAdmin), "e-uuid-1")
	require.NoError(t, err)
	assert.True(t, deleted)

	// unauthorized
	err = svc.DeleteWindowsEnforcementProfile(test.UserContext(ctx, test.UserNoRoles), "e-uuid-1")
	require.Error(t, err)
	require.Contains(t, err.Error(), authz.ForbiddenErrorMessage)
}

func TestReconcileWindowsEnforcement(t *testing.T) {
	ds := new(mock.Store)

	t.Run("nothing to do", func(t *testing.T) {
		ds.ListWindowsEnforcementToInstallFunc = func(ctx context.Context) ([]*fleet.HostWindowsEnforcement, error) {
			return nil, nil
		}
		ds.ListWindowsEnforcementToRemoveFunc = func(ctx context.Context) ([]*fleet.HostWindowsEnforcement, error) {
			return nil, nil
		}

		logger := slog.New(slog.DiscardHandler)
		err := ReconcileWindowsEnforcement(context.Background(), ds, logger)
		require.NoError(t, err)
	})

	t.Run("install and remove", func(t *testing.T) {
		ds.ListWindowsEnforcementToInstallFunc = func(ctx context.Context) ([]*fleet.HostWindowsEnforcement, error) {
			return []*fleet.HostWindowsEnforcement{
				{HostUUID: "host-1", ProfileUUID: "e-uuid-1", Name: "policy1"},
				{HostUUID: "host-2", ProfileUUID: "e-uuid-1", Name: "policy1"},
			}, nil
		}
		ds.ListWindowsEnforcementToRemoveFunc = func(ctx context.Context) ([]*fleet.HostWindowsEnforcement, error) {
			return []*fleet.HostWindowsEnforcement{
				{HostUUID: "host-3", ProfileUUID: "e-uuid-2"},
			}, nil
		}

		var upsertedPayload []*fleet.HostWindowsEnforcement
		ds.BulkUpsertHostWindowsEnforcementFunc = func(ctx context.Context, payload []*fleet.HostWindowsEnforcement) error {
			upsertedPayload = payload
			return nil
		}

		logger := slog.New(slog.DiscardHandler)
		err := ReconcileWindowsEnforcement(context.Background(), ds, logger)
		require.NoError(t, err)
		require.Len(t, upsertedPayload, 3)

		// First two should be install operations
		assert.Equal(t, fleet.MDMOperationTypeInstall, upsertedPayload[0].OperationType)
		assert.Equal(t, fleet.MDMOperationTypeInstall, upsertedPayload[1].OperationType)
		// Third should be remove
		assert.Equal(t, fleet.MDMOperationTypeRemove, upsertedPayload[2].OperationType)

		// All should have pending status
		for _, p := range upsertedPayload {
			require.NotNil(t, p.Status)
			assert.Equal(t, fleet.MDMDeliveryPending, *p.Status)
		}
	})
}
