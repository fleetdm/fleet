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
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

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
		ds:         ds,
		enterprise: enterprise,
		client:     client,
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

func createAndroidHost(t *testing.T, ds fleet.Datastore, suffixID int) *fleet.AndroidHost {
	host := &fleet.AndroidHost{
		Host: &fleet.Host{
			Hostname:       fmt.Sprintf("hostname%d", suffixID),
			ComputerName:   fmt.Sprintf("computer_name%d", suffixID),
			Platform:       "android",
			OSVersion:      "Android 14",
			Build:          fmt.Sprintf("build%d", suffixID),
			Memory:         1024,
			HardwareSerial: uuid.NewString(),
		},
		Device: &android.Device{
			DeviceID:             strings.ReplaceAll(uuid.NewString(), "-", ""), // Remove dashes to fit in VARCHAR(37)
			EnterpriseSpecificID: ptr.String(uuid.NewString()),
			AppliedPolicyID:      ptr.String("1"),
			LastPolicySyncTime:   ptr.Time(time.Now().Add(-time.Hour)), // 1 hour ago
		},
	}
	host.SetNodeKey(*host.Device.EnterpriseSpecificID)
	_, err := ds.NewAndroidHost(context.Background(), host)
	require.NoError(t, err)

	return host
}
