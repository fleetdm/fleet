package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// TODO: there may be a better package to put this test in, as it required some
// copying of helper functions but I tried moving them around (e.g. in
// android/tests) and it caused import cycles, same when I tried putting this
// whole test in the general service package, so I left it here for now with
// copies of some helpers.

// TestReconcileProfiles uses a real mysql datastore to test Android profile
// reconciliation scenarios, in a similar pattern to the datastore tests.
func TestReconcileProfiles(t *testing.T) {
	ds := mysql.CreateMySQLDS(t)
	mysql.TruncateTables(t, ds)

	setupEnterprise := func(t *testing.T, ds fleet.Datastore) {
		u := *test.UserAdmin
		err := u.SetPassword(test.GoodPassword, 10, 10)
		require.NoError(t, err)
		admin, err := ds.NewUser(t.Context(), &u)
		require.NoError(t, err)
		enterpriseID, err := ds.CreateEnterprise(t.Context(), admin.ID)
		require.NoError(t, err)

		// signupToken is used to authenticate the signup callback URL -- to ensure
		// that the callback came from our Android enterprise signup flow
		signupToken, err := server.GenerateRandomURLSafeText(32)
		require.NoError(t, err)

		signupDetails := android.SignupDetails{
			Name: "test",
		}

		err = ds.UpdateEnterprise(t.Context(), &android.EnterpriseDetails{
			Enterprise: android.Enterprise{
				ID:           enterpriseID,
				EnterpriseID: "test",
			},
			SignupName:  signupDetails.Name,
			SignupToken: signupToken,
		})
		require.NoError(t, err)
	}

	cases := []struct {
		name string
		fn   func(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler)
	}{
		{"NoHost", testNoHost},
		{"HostsWithoutProfile", testHostsWithoutProfile},
		{"HostsWithProfile", testHostsWithProfile},
		{"HostsWithConflictProfile", testHostsWithConflictProfile},
		{"HostsWithMultiOverrideProfile", testHostsWithMultiOverrideProfile},
		{"HostsWithAPIFailures", testHostsWithAPIFailures},
		{"HostsWithAddRemoveUpdateProfiles", testHostsWithAddRemoveUpdateProfiles},
		{"HostsWithLabelProfiles", testHostsWithLabelProfiles},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			setupEnterprise(t, ds)

			client := &mock.Client{}
			client.InitCommonMocks()
			enterprise, err := ds.GetEnterprise(t.Context())
			require.NoError(t, err)

			reconciler := &profileReconciler{
				DS:         ds,
				Enterprise: enterprise,
				Client:     client,
			}

			c.fn(t, ds, client, reconciler)
		})
	}
}

func testNoHost(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	client.InitCommonMocks()
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// no host, so no calls to the Android API
	err := reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// run again, still nothing
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithoutProfile(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// create a couple hosts
	createAndroidHost(t, ds, 1)
	createAndroidHost(t, ds, 2)

	// nothing to process, no profiles missing nor extraneous
	err := reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// run again, still nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithProfile(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		policy.Version = 1
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// create a couple hosts and a non-android one to ensure no unwanted side-effects
	h1 := createAndroidHost(t, ds, 1)
	h2 := createAndroidHost(t, ds, 2)
	createNonAndroidHost(t, ds, 3)

	// add an android profile
	p1 := androidProfileForTest("p1")
	p1, err := ds.NewMDMAndroidConfigProfile(ctx, *p1)
	require.NoError(t, err)

	// profile gets delivered to both hosts
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// run again, nothing to process as everything is pending
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithConflictProfile(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		policy.Version = 1
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// create a couple hosts and a non-android one to ensure no unwanted side-effects
	h1 := createAndroidHost(t, ds, 1)
	h2 := createAndroidHost(t, ds, 2)
	createNonAndroidHost(t, ds, 3)

	// add an android profile
	p1 := androidProfileWithPayloadForTest("p1", `{"key1": "a"}`)
	p1, err := ds.NewMDMAndroidConfigProfile(ctx, *p1)
	require.NoError(t, err)
	// add another one that overrides the first one
	p2 := androidProfileWithPayloadForTest("p2", `{"key1": "b", "key2": "c"}`)
	p2, err = ds.NewMDMAndroidConfigProfile(ctx, *p2)
	require.NoError(t, err)

	// profiles get delivered to both hosts, but p1 is failed
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"key1" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h1.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"key1" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h2.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// run again, nothing to process as everything is pending/failed
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithMultiOverrideProfile(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		policy.Version = 1
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// create h2 in a team, to validate that it isn't impacted by the non-team profiles
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "t1"})
	require.NoError(t, err)

	// create a couple hosts and a non-android one to ensure no unwanted side-effects
	h1 := createAndroidHost(t, ds, 1)
	createAndroidHostInTeam(t, ds, 2, &tm.ID)
	createNonAndroidHost(t, ds, 3)

	// add android profiles with multiple keys and multiple profiles overridden,
	// and insert in different order than the names to verify that name ordering
	// is applied.
	p3 := androidProfileWithPayloadForTest("p3", `{"key1": "c", "key2": "c", "key3": "c", "key4": "c"}`)
	p3, err = ds.NewMDMAndroidConfigProfile(ctx, *p3)
	require.NoError(t, err)
	p1 := androidProfileWithPayloadForTest("p1", `{"key1": "a", "key2": "a"}`)
	p1, err = ds.NewMDMAndroidConfigProfile(ctx, *p1)
	require.NoError(t, err)
	p2 := androidProfileWithPayloadForTest("p2", `{"key1": "b", "key2": "b", "key3": "b"}`)
	p2, err = ds.NewMDMAndroidConfigProfile(ctx, *p2)
	require.NoError(t, err)

	// profiles get delivered to h1 only
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"key1", and "key2" aren't applied. They are overridden by other configuration profile.`},
		{HostUUID: h1.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"key1", "key2", and "key3" aren't applied. They are overridden by other configuration profile.`},
		{HostUUID: h1.UUID, ProfileUUID: p3.ProfileUUID, ProfileName: p3.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// run again, nothing to process as everything is pending/failed
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithAPIFailures(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		return nil, errors.New("nope")
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// create a couple hosts and a non-android one to ensure no unwanted side-effects
	h1 := createAndroidHost(t, ds, 1)
	h2 := createAndroidHost(t, ds, 2)
	createNonAndroidHost(t, ds, 3)

	// add an android profile
	p1 := androidProfileForTest("p1")
	p1, err := ds.NewMDMAndroidConfigProfile(ctx, *p1)
	require.NoError(t, err)

	for i := range 3 {
		err = reconciler.ReconcileProfiles(ctx)
		require.NoError(t, err)
		require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
		client.EnterprisesPoliciesPatchFuncInvoked = false
		require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

		// status remains null, failure count is incremented
		assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
			{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: nil, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: nil, RequestFailCount: i + 1, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: nil},
			{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: nil, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: nil, RequestFailCount: i + 1, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: nil},
		})
	}

	// next run marks as failed
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: nil, RequestFailCount: 0, PolicyRequestUUID: nil, DeviceRequestUUID: nil, Detail: `Couldn't apply profile. Google returned error. Please re-add profile to try again.`},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: nil, RequestFailCount: 0, PolicyRequestUUID: nil, DeviceRequestUUID: nil, Detail: `Couldn't apply profile. Google returned error. Please re-add profile to try again.`},
	})

	// next run has nothing to do
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// add a new profile that "resets" the set of profiles to send, so will retry
	p2 := androidProfileWithPayloadForTest("p2", `{"key1": "b"}`)
	p2, err = ds.NewMDMAndroidConfigProfile(ctx, *p2)
	require.NoError(t, err)

	// and this time make it succeed
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		policy.Version = 1
		return policy, nil
	}

	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h1.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})
}

func testHostsWithAddRemoveUpdateProfiles(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		// use the received maximumTimeToLock value as the version to simplify testing
		policy.Version = policy.MaximumTimeToLock
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// create a couple hosts and a non-android one to ensure no unwanted side-effects
	h1 := createAndroidHost(t, ds, 1)
	h2 := createAndroidHost(t, ds, 2)
	createNonAndroidHost(t, ds, 3)

	// add a first android profile
	p1 := androidProfileWithPayloadForTest("p1", `{"maximumTimeToLock": "1"}`)
	p1, err := ds.NewMDMAndroidConfigProfile(ctx, *p1)
	require.NoError(t, err)

	// profiles get delivered
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// run again, nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// mark as verified (it will clear the request uuids, but that's not critical
	// for this test)
	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1)},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1)},
	})
	require.NoError(t, err)

	// run again, nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// Update p1, will now have version 3
	p1.RawJSON = []byte(`{"maximumTimeToLock": "3"}`)
	_, err = ds.BatchSetMDMProfiles(ctx, nil, nil, nil, nil, []*fleet.MDMAndroidConfigProfile{
		{ProfileUUID: p1.ProfileUUID, Name: p1.Name, RawJSON: p1.RawJSON},
	}, nil)
	require.NoError(t, err)

	// that batch-set operation, at the service level, also nulls the status and included version
	// of the affected hosts.
	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: nil, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: nil},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: nil, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: nil},
	})
	require.NoError(t, err)

	// profile gets re-delivered
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(3), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(3), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// add a second android profile
	p2 := androidProfileWithPayloadForTest("p2", `{"maximumTimeToLock": "4"}`)
	p2, err = ds.NewMDMAndroidConfigProfile(ctx, *p2)
	require.NoError(t, err)

	// profiles get re-delivered
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(4), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h1.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(4), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(4), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h2.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(4), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// run again, nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// mark as verified the ones not failed (it will clear the request uuids, but
	// that's not critical for this test)
	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(4)},
		{HostUUID: h2.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(4)},
	})
	require.NoError(t, err)

	// run again, nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// delete profile p1
	err = ds.DeleteMDMAndroidConfigProfile(ctx, p1.ProfileUUID)
	require.NoError(t, err)

	// update the patch policy mock to return a higher version
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		policy.Version = 5
		return policy, nil
	}

	// profiles get re-delivered
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	// p1 marked as pending removal
	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove, IncludedInPolicyVersion: ptr.Int(5), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h1.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(5), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove, IncludedInPolicyVersion: ptr.Int(5), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(5), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// run again, nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// mark as verified (it will clear the request uuids, but that's not critical
	// for this test)
	err = ds.BulkUpsertMDMAndroidHostProfiles(ctx, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(5)},
		{HostUUID: h2.UUID, ProfileUUID: p2.ProfileUUID, ProfileName: p2.Name, Status: &fleet.MDMDeliveryVerified, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(5)},
	})
	require.NoError(t, err)

	// manually delete the p1 removal entries, simulating the verification
	mysql.ExecAdhocSQL(t, ds.(*mysql.Datastore), func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `DELETE FROM host_mdm_android_profiles WHERE profile_uuid = ?`, p1.ProfileUUID)
		return err
	})

	// run again, nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithLabelProfiles(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	// create some labels
	linclAny, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclany-1", Query: "select 1"})
	require.NoError(t, err)
	linclAll, err := ds.NewLabel(ctx, &fleet.Label{Name: "inclall-1", Query: "select 1"})
	require.NoError(t, err)
	lexclAny, err := ds.NewLabel(ctx, &fleet.Label{Name: "exclude-1", Query: "select 1"})
	require.NoError(t, err)

	// make this test in a team
	tm, err := ds.NewTeam(ctx, &fleet.Team{Name: "t1"})
	require.NoError(t, err)

	// create a couple hosts and a non-android one to ensure no unwanted side-effects
	h1 := createAndroidHostInTeam(t, ds, 1, &tm.ID)
	h2 := createAndroidHostInTeam(t, ds, 2, &tm.ID)
	createNonAndroidHost(t, ds, 3)

	// create profiles based on each label, and one with no label
	pNoLabel := androidProfileWithPayloadForTest("pNoLabel", `{"maximumTimeToLock": "1"}`)
	pNoLabel.TeamID = &tm.ID
	pNoLabel, err = ds.NewMDMAndroidConfigProfile(ctx, *pNoLabel)
	require.NoError(t, err)
	pInclAny := androidProfileWithPayloadForTest("pInclAny", `{"maximumTimeToLock": "2"}`, linclAny)
	pInclAny.TeamID = &tm.ID
	pInclAny, err = ds.NewMDMAndroidConfigProfile(ctx, *pInclAny)
	require.NoError(t, err)
	pInclAll := androidProfileWithPayloadForTest("pInclAll", `{"maximumTimeToLock": "3"}`, linclAll)
	pInclAll.TeamID = &tm.ID
	pInclAll, err = ds.NewMDMAndroidConfigProfile(ctx, *pInclAll)
	require.NoError(t, err)
	pExclAny := androidProfileWithPayloadForTest("pExclAny", `{"maximumTimeToLock": "4"}`, lexclAny)
	pExclAny.TeamID = &tm.ID
	pExclAny, err = ds.NewMDMAndroidConfigProfile(ctx, *pExclAny)
	require.NoError(t, err)

	// mock and control the version number, and validate the expected MaximumTimeToLock value
	expectedMaxTimeToLock := int64(1) // will always be this due to pNoLabel always winning the setting
	version := int64(1)
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		policy.Version = version
		require.Equal(t, expectedMaxTimeToLock, policy.MaximumTimeToLock)
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	// currently only the no-label profile is applied
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// update the last label membership time, so that exclude any can start applying
	h1.LabelUpdatedAt, h1.PolicyUpdatedAt = time.Now().UTC(), time.Now().UTC()
	h2.LabelUpdatedAt, h2.PolicyUpdatedAt = time.Now().UTC(), time.Now().UTC()
	err = ds.UpdateHost(ctx, h1.Host)
	require.NoError(t, err)
	err = ds.UpdateHost(ctx, h2.Host)
	require.NoError(t, err)

	// no-label and exclude any are applied
	version++
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h1.UUID, ProfileUUID: pExclAny.ProfileUUID, ProfileName: pExclAny.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h2.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: pExclAny.ProfileUUID, ProfileName: pExclAny.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
	})

	// make h1 member of inclany and h2 of inclall
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, linclAny.ID, []uint{h1.Host.ID}, fleet.TeamFilter{})
	require.NoError(t, err)
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, linclAll.ID, []uint{h2.Host.ID}, fleet.TeamFilter{})
	require.NoError(t, err)

	// no-label, exclude any and the respective include profiles are applied
	version++
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h1.UUID, ProfileUUID: pInclAny.ProfileUUID, ProfileName: pInclAny.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h1.UUID, ProfileUUID: pExclAny.ProfileUUID, ProfileName: pExclAny.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h2.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: pInclAll.ProfileUUID, ProfileName: pInclAll.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h2.UUID, ProfileUUID: pExclAny.ProfileUUID, ProfileName: pExclAny.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
	})

	// make h1 member of exclAny so it stops receiving this profile
	_, _, err = ds.UpdateLabelMembershipByHostIDs(ctx, lexclAny.ID, []uint{h1.Host.ID}, fleet.TeamFilter{})
	require.NoError(t, err)

	// this only affects h1, h2 version is unchanged
	h2Version := version
	version++
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h1.UUID, ProfileUUID: pInclAny.ProfileUUID, ProfileName: pInclAny.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h1.UUID, ProfileUUID: pExclAny.ProfileUUID, ProfileName: pExclAny.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeRemove, IncludedInPolicyVersion: ptr.Int(int(version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},

		{HostUUID: h2.UUID, ProfileUUID: pNoLabel.ProfileUUID, ProfileName: pNoLabel.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(h2Version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: pInclAll.ProfileUUID, ProfileName: pInclAll.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(h2Version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
		{HostUUID: h2.UUID, ProfileUUID: pExclAny.ProfileUUID, ProfileName: pExclAny.Name, Status: &fleet.MDMDeliveryFailed, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(int(h2Version)), PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String(""), Detail: `"maximumTimeToLock" isn't applied. It's overridden by other configuration profile.`},
	})

	// run again, nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func assertHostProfiles(t *testing.T, ds fleet.Datastore, hostProfiles []*fleet.MDMAndroidProfilePayload) {
	ctx := t.Context()
	mds := ds.(*mysql.Datastore)
	var got []*fleet.MDMAndroidProfilePayload
	mysql.ExecAdhocSQL(t, mds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &got, `SELECT host_uuid, status, operation_type, detail, profile_uuid, profile_name,
			policy_request_uuid, device_request_uuid, request_fail_count, included_in_policy_version
		FROM host_mdm_android_profiles`)
	})

	// for policy and device request UUIDs, we just check NULL vs non-NULL
	for _, hp := range got {
		if hp.PolicyRequestUUID != nil {
			hp.PolicyRequestUUID = ptr.String("")
		}
		if hp.DeviceRequestUUID != nil {
			hp.DeviceRequestUUID = ptr.String("")
		}
	}
	for _, hp := range hostProfiles {
		if hp.PolicyRequestUUID != nil {
			hp.PolicyRequestUUID = ptr.String("")
		}
		if hp.DeviceRequestUUID != nil {
			hp.DeviceRequestUUID = ptr.String("")
		}
	}
	require.ElementsMatch(t, hostProfiles, got)
}

func createAndroidHost(t *testing.T, ds fleet.Datastore, suffixID int) *fleet.AndroidHost {
	return createAndroidHostInTeam(t, ds, suffixID, nil)
}

func createAndroidHostInTeam(t *testing.T, ds fleet.Datastore, suffixID int, teamID *uint) *fleet.AndroidHost {
	hostUUID := uuid.NewString()
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       fmt.Sprintf("hostname%d", suffixID),
			ComputerName:   fmt.Sprintf("computer_name%d", suffixID),
			Platform:       "android",
			OSVersion:      "Android 14",
			Build:          fmt.Sprintf("build%d", suffixID),
			Memory:         1024,
			HardwareSerial: uuid.NewString(),
			UUID:           hostUUID,
			TeamID:         teamID,
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""), // Remove dashes to fit in VARCHAR(37)
			EnterpriseSpecificID: ptr.String(hostUUID),
			AppliedPolicyID:      ptr.String("1"),
			LastPolicySyncTime:   ptr.Time(time.Now().Add(-time.Hour)), // 1 hour ago
		},
	}
	host.SetNodeKey(*host.Device.EnterpriseSpecificID)
	_, err := ds.NewAndroidHost(context.Background(), host)
	require.NoError(t, err)

	return host
}

func createNonAndroidHost(t *testing.T, ds fleet.Datastore, suffixID int) *fleet.Host {
	hostUUID := uuid.NewString()
	host := &fleet.Host{
		Hostname:       fmt.Sprintf("hostname%d", suffixID),
		ComputerName:   fmt.Sprintf("computer_name%d", suffixID),
		Platform:       "darwin",
		OSVersion:      "macOS 14",
		Build:          fmt.Sprintf("build%d", suffixID),
		Memory:         1024,
		HardwareSerial: uuid.NewString(),
		UUID:           hostUUID,
	}
	host, err := ds.NewHost(context.Background(), host)
	require.NoError(t, err)

	return host
}

func androidProfileForTest(name string, labels ...*fleet.Label) *fleet.MDMAndroidConfigProfile {
	payload := `{
		"maximumTimeToLock": "1234"
	}`
	return androidProfileWithPayloadForTest(name, payload, labels...)
}

func androidProfileWithPayloadForTest(name, payload string, labels ...*fleet.Label) *fleet.MDMAndroidConfigProfile {
	profile := &fleet.MDMAndroidConfigProfile{
		RawJSON: []byte(payload),
		Name:    name,
	}

	for _, l := range labels {
		switch {
		case strings.HasPrefix(l.Name, "exclude-"):
			profile.LabelsExcludeAny = append(profile.LabelsExcludeAny, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		case strings.HasPrefix(l.Name, "inclany-"):
			profile.LabelsIncludeAny = append(profile.LabelsIncludeAny, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		default:
			profile.LabelsIncludeAll = append(profile.LabelsIncludeAll, fleet.ConfigurationProfileLabel{LabelName: l.Name, LabelID: l.ID})
		}
	}

	return profile
}
