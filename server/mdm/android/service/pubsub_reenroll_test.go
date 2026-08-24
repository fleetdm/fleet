package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

// setupReenrollMocks wires the datastore mocks a re-enrollment goes through, so the
// tests below only have to set up what they actually assert on.
func setupReenrollMocks(t *testing.T, mockDS *AndroidMockDS, existingHost *fleet.AndroidHost) {
	t.Helper()

	mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
		return &fleet.AppConfig{MDM: fleet.MDM{AndroidEnabledAndConfigured: true}}, nil
	}
	mockDS.VerifyEnrollSecretFunc = func(ctx context.Context, secret string) (*fleet.EnrollSecret, error) {
		return &fleet.EnrollSecret{Secret: "global"}, nil
	}
	mockDS.AndroidHostLiteFunc = func(ctx context.Context, esID string) (*fleet.AndroidHost, error) {
		if existingHost == nil {
			return nil, common_mysql.NotFound("android host")
		}
		return existingHost, nil
	}
	mockDS.NewAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, companyOwned bool) (*fleet.AndroidHost, error) {
		return &fleet.AndroidHost{Host: &fleet.Host{}}, nil
	}
	mockDS.UpdateAndroidHostFunc = func(ctx context.Context, host *fleet.AndroidHost, fromEnroll, companyOwned bool) error {
		return nil
	}
	mockDS.ScimUserByHostIDFunc = func(ctx context.Context, hostID uint) (*fleet.ScimUser, error) {
		return nil, common_mysql.NotFound("scim user")
	}
	mockDS.ListHostDeviceMappingFunc = func(ctx context.Context, id uint) ([]*fleet.HostDeviceMapping, error) {
		return nil, nil
	}
	mockDS.GetMDMIdPAccountByHostUUIDFunc = func(ctx context.Context, hostUUID string) (*fleet.MDMIdPAccount, error) {
		return nil, common_mysql.NotFound("mdm idp account")
	}
	mockDS.ClearHostMDMActionsFunc = func(ctx context.Context, hostID uint) error {
		return nil
	}
	mockDS.DeleteAllHostCertificateTemplatesFunc = func(ctx context.Context, hostUUID string) error {
		return nil
	}
}

func enrollmentMessageForSecret(t *testing.T, name string, ownership string) *android.PubSubMessage {
	t.Helper()

	enrollTokenData, err := json.Marshal(enrollmentTokenRequest{EnrollSecret: "global"})
	require.NoError(t, err)
	return createEnrollmentMessage(t, androidmanagement.Device{
		Name:                createAndroidDeviceId(name),
		EnrollmentTokenData: string(enrollTokenData),
		Ownership:           ownership,
	})
}

// TestPubSubEnrollment_ResetsStateOnReEnroll covers the re-enrollment reset: an ENROLLMENT
// notification for a host Fleet already knows about must clear the state the device no
// longer has, and must forward the preserve_host_activities_on_reenrollment setting.
func TestPubSubEnrollment_ResetsStateOnReEnroll(t *testing.T) {
	const existingHostID uint = 42

	for _, preserveActivities := range []bool{true, false} {
		name := "preserves host activities"
		if !preserveActivities {
			name = "does not preserve host activities"
		}
		t.Run(name, func(t *testing.T) {
			svc, mockDS := createAndroidService(t)

			existingHost := &fleet.AndroidHost{
				Host: &fleet.Host{ID: existingHostID, UUID: testBrandTestSerialHashed},
				Device: &android.Device{
					HostID:               existingHostID,
					DeviceID:             "device-reenroll",
					EnterpriseSpecificID: new(testBrandTestSerialHashed),
				},
			}
			setupReenrollMocks(t, mockDS, existingHost)
			mockDS.AppConfigFunc = func(ctx context.Context) (*fleet.AppConfig, error) {
				return &fleet.AppConfig{
					MDM: fleet.MDM{AndroidEnabledAndConfigured: true},
					ActivityExpirySettings: fleet.ActivityExpirySettings{
						PreserveHostActivitiesOnReenrollment: preserveActivities,
					},
				}, nil
			}

			// The reset has to run before the enrollment's own data is written back,
			// otherwise it deletes the vitals that were just reported.
			var resetBeforeUpdate bool
			mockDS.AndroidResetOnReenrollmentFunc = func(ctx context.Context, hostID uint, hostUUID string, preserveHostActivities bool) ([]*fleet.User, []fleet.ActivityDetails, error) {
				require.Equal(t, existingHostID, hostID)
				require.Equal(t, testBrandTestSerialHashed, hostUUID)
				require.Equal(t, preserveActivities, preserveHostActivities,
					"the preserve_host_activities_on_reenrollment setting must be forwarded to the datastore")
				resetBeforeUpdate = !mockDS.UpdateAndroidHostFuncInvoked
				return nil, nil, nil
			}

			msg := enrollmentMessageForSecret(t, "reenroll", DeviceOwnershipCompanyOwned)
			require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", msg))

			require.True(t, mockDS.AndroidResetOnReenrollmentFuncInvoked,
				"re-enrollment must reset the stale state of the previous enrollment")
			require.True(t, mockDS.UpdateAndroidHostFuncInvoked)
			require.True(t, resetBeforeUpdate,
				"the reset must run before UpdateAndroidHost writes this enrollment's vitals")
		})
	}
}

// TestPubSub_NoResetOnStatusReport is the other half of the gate: a plain status report is
// not a re-enrollment, so the host's state must survive it.
func TestPubSub_NoResetOnStatusReport(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	const enterpriseSpecificID = "es-id-status-report"
	existingHost := &fleet.AndroidHost{
		Host:   &fleet.Host{ID: 7, UUID: enterpriseSpecificID},
		Device: &android.Device{HostID: 7, DeviceID: createAndroidDeviceId("status")},
	}
	setupReenrollMocks(t, mockDS, existingHost)
	mockDS.AndroidResetOnReenrollmentFunc = func(ctx context.Context, hostID uint, hostUUID string, preserveHostActivities bool) ([]*fleet.User, []fleet.ActivityDetails, error) {
		t.Error("a status report must not reset host state")
		return nil, nil, nil
	}

	// No applied policy name, so the report only updates the host's details.
	msg := createStatusReportMessage(t, enterpriseSpecificID, "status", "", new(0), nil)
	require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", &msg))

	require.True(t, mockDS.UpdateAndroidHostFuncInvoked, "the status report must still have updated the host")
	require.False(t, mockDS.AndroidResetOnReenrollmentFuncInvoked)
}

// TestPubSubEnrollment_NoResetOnFirstEnrollment covers a device Fleet has never seen: the
// host row is brand new, so there is nothing to reset.
func TestPubSubEnrollment_NoResetOnFirstEnrollment(t *testing.T) {
	svc, mockDS := createAndroidService(t)

	setupReenrollMocks(t, mockDS, nil /* no existing host */)
	mockDS.AndroidResetOnReenrollmentFunc = func(ctx context.Context, hostID uint, hostUUID string, preserveHostActivities bool) ([]*fleet.User, []fleet.ActivityDetails, error) {
		t.Error("a first enrollment must not reset host state")
		return nil, nil, nil
	}

	msg := enrollmentMessageForSecret(t, "first-enroll", DeviceOwnershipPersonallyOwned)
	require.NoError(t, svc.ProcessPubSubPush(t.Context(), "value", msg))

	require.True(t, mockDS.NewAndroidHostFuncInvoked, "a new device must go through NewAndroidHost")
	require.False(t, mockDS.AndroidResetOnReenrollmentFuncInvoked)
}
