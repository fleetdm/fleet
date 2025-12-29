package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/mock"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/test"
	kitlog "github.com/go-kit/log"
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
		{"CertificateTemplates", testCertificateTemplates},
		{"BuildAndSendFleetAgentConfigForEnrollment", testBuildAndSendFleetAgentConfigForEnrollment},
		{"CertificateTemplatesIncludesExistingVerified", testCertificateTemplatesIncludesExistingVerified},
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
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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

	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
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

func testCertificateTemplates(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	// Create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "test-team"})
	require.NoError(t, err)

	// insert enroll secret for team
	err = ds.ApplyEnrollSecrets(ctx, &team.ID,
		[]*fleet.EnrollSecret{
			{Secret: "secret", TeamID: &team.ID},
		},
	)
	require.NoError(t, err)

	// Create a test certificate authority
	var caID uint
	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String("Test SCEP CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)
	caID = ca.ID

	// Create certificate templates
	template1 := &fleet.CertificateTemplate{
		Name:                   "cert-template-1",
		TeamID:                 team.ID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Certificate 1",
	}
	_, err = ds.CreateCertificateTemplate(ctx, template1)
	require.NoError(t, err)

	template2 := &fleet.CertificateTemplate{
		Name:                   "cert-template-2",
		TeamID:                 team.ID,
		CertificateAuthorityID: caID,
		SubjectName:            "CN=Test Certificate 2",
	}
	_, err = ds.CreateCertificateTemplate(ctx, template2)
	require.NoError(t, err)

	// Create Android hosts in the team
	host1 := createAndroidHostInTeam(t, ds, 1, &team.ID)
	host2 := createAndroidHostInTeam(t, ds, 2, &team.ID)

	// Add host certificate templates for host 2 with 'delivered' status to exclude it from processing
	// (only 'pending' status templates are picked up by reconcileCertificateTemplates)
	var certificateTemplateIDs []uint
	mysql.ExecAdhocSQL(t, ds.(*mysql.Datastore), func(q sqlx.ExtContext) error {
		query := `
			SELECT id
			FROM certificate_templates
			WHERE team_id = ?
			ORDER BY id
		`
		err := sqlx.SelectContext(ctx, q, &certificateTemplateIDs, query, team.ID)
		require.NoError(t, err)

		for _, certTemplateID := range certificateTemplateIDs {
			_, err = q.ExecContext(ctx,
				"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
				host2.UUID,
				certTemplateID,
				"challenge",
				fleet.CertificateTemplateDelivered,
				fleet.MDMOperationTypeInstall,
				fmt.Sprintf("Cert Template %d", certTemplateID),
			)
			require.NoError(t, err)
		}

		return nil
	})

	// Create pending certificate templates for host1 (this is what triggers processing)
	for _, certTemplateID := range certificateTemplateIDs {
		_, err = ds.CreatePendingCertificateTemplatesForExistingHosts(ctx, certTemplateID, team.ID)
		require.NoError(t, err)
	}

	// Get app config for server URL
	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.ServerSettings.ServerURL = "https://fleet.example.com"
	err = ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// AddFleetAgentToAndroidPolicy calls EnterprisesPoliciesModifyPolicyApplications
	// easier to mock that than the AddFleetAgentToAndroidPolicy
	var capturedPolicyName string
	var capturedPolicies []*androidmanagement.ApplicationPolicy
	client.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, policies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		capturedPolicyName = policyName
		capturedPolicies = policies
		return &androidmanagement.Policy{}, nil
	}

	oldPackageValue := os.Getenv("FLEET_DEV_ANDROID_AGENT_PACKAGE")
	oldSHA256Value := os.Getenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", "com.fleetdm.agent")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", "abc123def456")
	defer func() {
		os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", oldPackageValue)
		os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", oldSHA256Value)
	}()

	err = reconciler.reconcileCertificateTemplates(ctx)
	require.NoError(t, err)

	// verify host1 is targetted by the api call
	require.True(t, client.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	require.Equal(t, fmt.Sprintf("%s/policies/%s", reconciler.Enterprise.Name(), host1.Host.UUID), capturedPolicyName)
	require.Len(t, capturedPolicies, 1)

	// Verify the managed configuration contains certificate template IDs
	var managedConfig android.AgentManagedConfiguration
	err = json.Unmarshal(capturedPolicies[0].ManagedConfiguration, &managedConfig)
	require.NoError(t, err)
	require.Len(t, managedConfig.CertificateTemplateIDs, 2)
	for _, certTemplate := range managedConfig.CertificateTemplateIDs {
		require.Contains(t, certificateTemplateIDs, certTemplate.ID)
	}

	// Verify that host_certificate_template records were created with pending status
	var host1CertTemplates []struct {
		HostUUID              string `db:"host_uuid"`
		CertificateTemplateID uint   `db:"certificate_template_id"`
		FleetChallenge        string `db:"fleet_challenge"`
		Status                string `db:"status"`
	}
	mysql.ExecAdhocSQL(t, ds.(*mysql.Datastore), func(q sqlx.ExtContext) error {
		query := `
			SELECT host_uuid, certificate_template_id, fleet_challenge, status
			FROM host_certificate_templates
			WHERE host_uuid = ?
			ORDER BY certificate_template_id
		`
		return sqlx.SelectContext(ctx, q, &host1CertTemplates, query, host1.Host.UUID)
	})
	require.Len(t, host1CertTemplates, 2)

	for _, hct := range host1CertTemplates {
		require.Equal(t, host1.Host.UUID, hct.HostUUID)
		require.NotEmpty(t, hct.FleetChallenge)
		require.EqualValues(t, fleet.CertificateTemplateDelivered, hct.Status)
	}

	client.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked = false

	// Run reconciliation again - should not create duplicate records or make API calls
	err = reconciler.reconcileCertificateTemplates(ctx)
	require.NoError(t, err)

	// Verify the API was NOT called again
	require.False(t, client.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)

	// no duplicate records were created
	var countHost1 int
	mysql.ExecAdhocSQL(t, ds.(*mysql.Datastore), func(q sqlx.ExtContext) error {
		query := `SELECT COUNT(*) FROM host_certificate_templates WHERE host_uuid = ?`
		return sqlx.GetContext(ctx, q, &countHost1, query, host1.Host.UUID)
	})
	require.Equal(t, 2, countHost1)
}

// testBuildAndSendFleetAgentConfigForEnrollment tests the enrollment flow where we send
// the Fleet agent config to hosts even if they don't have certificate templates to install.
// This is needed when new hosts are enrolling to receive the Fleet app.
func testBuildAndSendFleetAgentConfigForEnrollment(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	// Create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "enrollment-test-team"})
	require.NoError(t, err)

	// Insert enroll secret for team
	err = ds.ApplyEnrollSecrets(ctx, &team.ID,
		[]*fleet.EnrollSecret{
			{Secret: "enroll-secret", TeamID: &team.ID},
		},
	)
	require.NoError(t, err)

	// Get app config for server URL
	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.ServerSettings.ServerURL = "https://fleet.example.com"
	err = ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// Create Android host in the team (no certificate templates)
	host := createAndroidHostInTeam(t, ds, 100, &team.ID)

	// Track API calls
	var capturedPolicyName string
	var capturedPolicies []*androidmanagement.ApplicationPolicy
	client.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, policies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		capturedPolicyName = policyName
		capturedPolicies = policies
		return &androidmanagement.Policy{}, nil
	}

	oldPackageValue := os.Getenv("FLEET_DEV_ANDROID_AGENT_PACKAGE")
	oldSHA256Value := os.Getenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", "com.fleetdm.agent")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", "abc123def456")
	defer func() {
		os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", oldPackageValue)
		os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", oldSHA256Value)
	}()

	// Create service and call BuildAndSendFleetAgentConfig with skipHostsWithoutNewCerts=false
	// This simulates the enrollment flow from software_worker.go
	svc := &Service{
		logger:           kitlog.NewNopLogger(),
		fleetDS:          ds,
		ds:               ds.(fleet.AndroidDatastore),
		androidAPIClient: client,
	}

	// Call with skipHostsWithoutNewCerts=false (enrollment scenario)
	err = svc.BuildAndSendFleetAgentConfig(ctx, reconciler.Enterprise.Name(), []string{host.Host.UUID}, false)
	require.NoError(t, err)

	// Verify the API was called for the host even without certificate templates
	require.True(t, client.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	require.Equal(t, fmt.Sprintf("%s/policies/%s", reconciler.Enterprise.Name(), host.Host.UUID), capturedPolicyName)
	require.Len(t, capturedPolicies, 1)

	// Verify the managed configuration was sent without certificate template IDs
	var managedConfig android.AgentManagedConfiguration
	err = json.Unmarshal(capturedPolicies[0].ManagedConfiguration, &managedConfig)
	require.NoError(t, err)
	require.Empty(t, managedConfig.CertificateTemplateIDs)
	require.Equal(t, "https://fleet.example.com", managedConfig.ServerURL)
	require.Equal(t, host.Host.UUID, managedConfig.HostUUID)
	require.Equal(t, "enroll-secret", managedConfig.EnrollSecret)

	// Reset the invocation flag
	client.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked = false

	// Now test with skipHostsWithoutNewCerts=true (reconciliation scenario)
	// The host should be skipped since it has no pending certificate templates
	err = svc.BuildAndSendFleetAgentConfig(ctx, reconciler.Enterprise.Name(), []string{host.Host.UUID}, true)
	require.NoError(t, err)

	// Verify the API was NOT called since we're skipping hosts without new certs
	require.False(t, client.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
}

// testCertificateTemplatesIncludesExistingVerified tests that when a host has existing
// certificates in various statuses AND a new pending certificate, the agent config sent
// to AMAPI includes ALL certificate templates, not just the new pending one.
func testCertificateTemplatesIncludesExistingVerified(t *testing.T, ds fleet.Datastore, client *mock.Client, reconciler *profileReconciler) {
	ctx := t.Context()

	// Create a team
	team, err := ds.NewTeam(ctx, &fleet.Team{Name: "existing-cert-test-team"})
	require.NoError(t, err)

	// Insert enroll secret for team
	err = ds.ApplyEnrollSecrets(ctx, &team.ID,
		[]*fleet.EnrollSecret{
			{Secret: "secret", TeamID: &team.ID},
		},
	)
	require.NoError(t, err)

	// Create a test certificate authority
	ca, err := ds.NewCertificateAuthority(ctx, &fleet.CertificateAuthority{
		Type:      string(fleet.CATypeCustomSCEPProxy),
		Name:      ptr.String("Test SCEP CA"),
		URL:       ptr.String("http://localhost:8080/scep"),
		Challenge: ptr.String("test-challenge"),
	})
	require.NoError(t, err)

	// Create certificate templates for each status we want to test
	templateVerified := &fleet.CertificateTemplate{
		Name:                   "verified-cert-template",
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Verified Certificate",
	}
	verifiedCert, err := ds.CreateCertificateTemplate(ctx, templateVerified)
	require.NoError(t, err)

	templateDelivered := &fleet.CertificateTemplate{
		Name:                   "delivered-cert-template",
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Delivered Certificate",
	}
	deliveredCert, err := ds.CreateCertificateTemplate(ctx, templateDelivered)
	require.NoError(t, err)

	templateDelivering := &fleet.CertificateTemplate{
		Name:                   "delivering-cert-template",
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Delivering Certificate",
	}
	deliveringCert, err := ds.CreateCertificateTemplate(ctx, templateDelivering)
	require.NoError(t, err)

	templateFailed := &fleet.CertificateTemplate{
		Name:                   "failed-cert-template",
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Failed Certificate",
	}
	failedCert, err := ds.CreateCertificateTemplate(ctx, templateFailed)
	require.NoError(t, err)

	templatePending := &fleet.CertificateTemplate{
		Name:                   "pending-cert-template",
		TeamID:                 team.ID,
		CertificateAuthorityID: ca.ID,
		SubjectName:            "CN=Pending Certificate",
	}
	pendingCert, err := ds.CreateCertificateTemplate(ctx, templatePending)
	require.NoError(t, err)

	// Create an Android host in the team
	host := createAndroidHostInTeam(t, ds, 300, &team.ID)

	// Insert certificate templates with various statuses
	mysql.ExecAdhocSQL(t, ds.(*mysql.Datastore), func(q sqlx.ExtContext) error {
		// Verified status
		_, err := q.ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
			host.Host.UUID, verifiedCert.ID, "verified-challenge", fleet.CertificateTemplateVerified, fleet.MDMOperationTypeInstall, verifiedCert.Name,
		)
		if err != nil {
			return err
		}

		// Delivered status
		_, err = q.ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
			host.Host.UUID, deliveredCert.ID, "delivered-challenge", fleet.CertificateTemplateDelivered, fleet.MDMOperationTypeInstall, deliveredCert.Name,
		)
		if err != nil {
			return err
		}

		// Delivering status (from a previous failed run)
		_, err = q.ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
			host.Host.UUID, deliveringCert.ID, "delivering-challenge", fleet.CertificateTemplateDelivering, fleet.MDMOperationTypeInstall, deliveringCert.Name,
		)
		if err != nil {
			return err
		}

		// Failed status
		_, err = q.ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
			host.Host.UUID, failedCert.ID, "failed-challenge", fleet.CertificateTemplateFailed, fleet.MDMOperationTypeInstall, failedCert.Name,
		)
		if err != nil {
			return err
		}

		// Pending status (new certificate to be processed)
		_, err = q.ExecContext(ctx,
			"INSERT INTO host_certificate_templates (host_uuid, certificate_template_id, fleet_challenge, status, operation_type, name) VALUES (?, ?, ?, ?, ?, ?)",
			host.Host.UUID, pendingCert.ID, "", fleet.CertificateTemplatePending, fleet.MDMOperationTypeInstall, pendingCert.Name,
		)
		return err
	})

	// Get app config for server URL
	appConfig, err := ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.ServerSettings.ServerURL = "https://fleet.example.com"
	err = ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// Track API calls
	var capturedPolicies []*androidmanagement.ApplicationPolicy
	client.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, policies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		capturedPolicies = policies
		return &androidmanagement.Policy{}, nil
	}

	oldPackageValue := os.Getenv("FLEET_DEV_ANDROID_AGENT_PACKAGE")
	oldSHA256Value := os.Getenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", "com.fleetdm.agent")
	os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", "abc123def456")
	defer func() {
		os.Setenv("FLEET_DEV_ANDROID_AGENT_PACKAGE", oldPackageValue)
		os.Setenv("FLEET_DEV_ANDROID_AGENT_SIGNING_SHA256", oldSHA256Value)
	}()

	// Run reconciliation
	err = reconciler.reconcileCertificateTemplates(ctx)
	require.NoError(t, err)

	// Verify the API was called
	require.True(t, client.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	require.Len(t, capturedPolicies, 1)

	// Parse the managed configuration
	var managedConfig android.AgentManagedConfiguration
	err = json.Unmarshal(capturedPolicies[0].ManagedConfiguration, &managedConfig)
	require.NoError(t, err)

	// Agent config should include ALL 5 certificate templates regardless of status
	require.Len(t, managedConfig.CertificateTemplateIDs, 5,
		"Agent config should include all certificate templates (verified, delivered, delivering, failed, and pending)")

	// Verify all certificate template IDs are present
	templateIDs := make(map[uint]bool)
	for _, tmpl := range managedConfig.CertificateTemplateIDs {
		templateIDs[tmpl.ID] = true
	}
	require.True(t, templateIDs[verifiedCert.ID], "Verified certificate should be in the config")
	require.True(t, templateIDs[deliveredCert.ID], "Delivered certificate should be in the config")
	require.True(t, templateIDs[deliveringCert.ID], "Delivering certificate should be in the config")
	require.True(t, templateIDs[failedCert.ID], "Failed certificate should be in the config")
	require.True(t, templateIDs[pendingCert.ID], "Pending certificate should be in the config")
}
