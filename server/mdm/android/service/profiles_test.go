package service

import (
	"context"
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
		fn   func(t *testing.T, ds fleet.Datastore)
	}{
		{"NoHost", testNoHost},
		{"HostsWithoutProfile", testHostsWithoutProfile},
		{"HostsWithProfile", testHostsWithProfile},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			defer mysql.TruncateTables(t, ds)
			setupEnterprise(t, ds)
			c.fn(t, ds)
		})
	}
}

func testNoHost(t *testing.T, ds fleet.Datastore) {
	ctx := t.Context()

	client := &mock.Client{}
	client.InitCommonMocks()
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	enterprise, err := ds.GetEnterprise(ctx)
	require.NoError(t, err)

	reconciler := &profileReconciler{
		DS:         ds,
		Enterprise: enterprise,
		Client:     client,
	}

	// no host, so no calls to the Android API
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// run again, still nothing
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithoutProfile(t *testing.T, ds fleet.Datastore) {
	ctx := t.Context()

	client := &mock.Client{}
	client.InitCommonMocks()
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	enterprise, err := ds.GetEnterprise(ctx)
	require.NoError(t, err)

	reconciler := &profileReconciler{
		DS:         ds,
		Enterprise: enterprise,
		Client:     client,
	}

	// create a couple hosts
	createAndroidHost(t, ds, 1)
	createAndroidHost(t, ds, 2)

	// nothing to process, no profiles missing nor extraneous
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)

	// run again, still nothing to process
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func testHostsWithProfile(t *testing.T, ds fleet.Datastore) {
	ctx := t.Context()

	client := &mock.Client{}
	client.InitCommonMocks()
	client.EnterprisesPoliciesPatchFunc = func(ctx context.Context, enterpriseID string, policy *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		policy.Version = 1
		return policy, nil
	}
	client.EnterprisesDevicesPatchFunc = func(ctx context.Context, name string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return device, nil
	}

	enterprise, err := ds.GetEnterprise(ctx)
	require.NoError(t, err)

	reconciler := &profileReconciler{
		DS:         ds,
		Enterprise: enterprise,
		Client:     client,
	}

	// create a couple hosts and a non-android one to ensure no unwanted side-effects
	h1 := createAndroidHost(t, ds, 1)
	h2 := createAndroidHost(t, ds, 2)
	createNonAndroidHost(t, ds, 3)

	// add an android profile
	p1 := androidProfileForTest("p1")
	p1, err = ds.NewMDMAndroidConfigProfile(ctx, *p1)
	require.NoError(t, err)

	// profile gets delivered to both hosts
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.True(t, client.EnterprisesPoliciesPatchFuncInvoked)
	client.EnterprisesPoliciesPatchFuncInvoked = false
	require.True(t, client.EnterprisesDevicesPatchFuncInvoked)
	client.EnterprisesDevicesPatchFuncInvoked = false

	assertHostProfiles(t, ds, []*fleet.MDMAndroidBulkUpsertHostProfilePayload{
		{HostUUID: h1.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
		{HostUUID: h2.UUID, ProfileUUID: p1.ProfileUUID, ProfileName: p1.Name, Status: &fleet.MDMDeliveryPending, OperationType: fleet.MDMOperationTypeInstall, IncludedInPolicyVersion: ptr.Int(1), RequestFailCount: 0, PolicyRequestUUID: ptr.String(""), DeviceRequestUUID: ptr.String("")},
	})

	// run again, nothing to process as everything is pending
	err = reconciler.ReconcileProfiles(ctx)
	require.NoError(t, err)
	require.False(t, client.EnterprisesPoliciesPatchFuncInvoked)
	require.False(t, client.EnterprisesDevicesPatchFuncInvoked)
}

func assertHostProfiles(t *testing.T, ds fleet.Datastore, hostProfiles []*fleet.MDMAndroidBulkUpsertHostProfilePayload) {
	ctx := t.Context()
	mds := ds.(*mysql.Datastore)
	var got []*fleet.MDMAndroidBulkUpsertHostProfilePayload
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
