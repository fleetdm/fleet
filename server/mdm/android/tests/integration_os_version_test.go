package tests

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/androidmanagement/v1"
)

func TestServiceOSVersion(t *testing.T) {
	testingSuite := new(osVersionTestSuite)
	suite.Run(t, testingSuite)
}

type osVersionTestSuite struct {
	WithServer
}

func (s *osVersionTestSuite) SetupSuite() {
	s.WithServer.SetupSuite(s.T(), "androidOSVersionTestSuite")
	s.Token = "testtoken"
}

// TestAndroidOSVersionSecurityPatchLevel verifies end-to-end (against real MySQL)
// that an Android device's security patch level is folded into the host's OS
// version, that distinct patch levels produce distinct operating_systems rows,
// and that a device reporting no patch level falls back to the bare version.
func (s *osVersionTestSuite) TestAndroidOSVersionSecurityPatchLevel() {
	ctx := context.Background()
	t := s.T()

	// Create enterprise.
	var signupResp android.EnterpriseSignupResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupResp)
	assert.Equal(t, EnterpriseSignupURL, signupResp.Url)

	s.FleetSvc.On("NewActivity", mock.Anything, mock.Anything, mock.AnythingOfType("fleet.ActivityTypeEnabledAndroidMDM")).Return(nil)
	const enterpriseToken = "enterpriseToken"
	s.Do("GET", s.ProxyCallbackURL, nil, http.StatusOK, "enterpriseToken", enterpriseToken)

	s.AndroidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{{Name: "enterprises/" + EnterpriseID}}, nil
	}

	resp := android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(t, EnterpriseID, resp.EnterpriseID)

	s.AppConfig.MDM.AndroidEnabledAndConfigured = true

	require.NoError(t, s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "enrollsecret"}}))
	secrets, err := s.DS.Datastore.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, secrets, 1)

	assets, err := s.DS.Datastore.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]
	require.NotEmpty(t, pubsubToken.Value)

	esi := strings.ToUpper(uuid.New().String())

	// Enroll the device. The enrollment helper reports version "1" and no patch
	// level, so the host starts with a bare version.
	enrollmentMessage := enrollmentMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                createAndroidDeviceID("test-os-version"),
			EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, secrets[0].Secret),
		},
		esi,
	)
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &service.PubSubPushRequest{PubSubMessage: *enrollmentMessage}, http.StatusOK, "token", string(pubsubToken.Value))

	assertOSVersion := func(wantHostOSVersion, wantOSRowVersion string) {
		t.Helper()
		mysqltest.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
			var hostOSVersion string
			require.NoError(t, sqlx.GetContext(ctx, q, &hostOSVersion, "SELECT os_version FROM hosts WHERE uuid = ?", esi))
			assert.Equal(t, wantHostOSVersion, hostOSVersion)

			var osRow struct {
				Name    string `db:"name"`
				Version string `db:"version"`
			}
			require.NoError(t, sqlx.GetContext(ctx, q, &osRow,
				`SELECT os.name, os.version
				 FROM operating_systems os
				 JOIN host_operating_system hos ON hos.os_id = os.id
				 JOIN hosts h ON h.id = hos.host_id
				 WHERE h.uuid = ?`, esi))
			assert.Equal(t, "Android", osRow.Name)
			assert.Equal(t, wantOSRowVersion, osRow.Version)
			return nil
		})
	}

	// Status report with a security patch level folds it into the version.
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub",
		&service.PubSubPushRequest{PubSubMessage: osVersionStatusReportMessage(t, esi, "16", "2026-05-01")},
		http.StatusOK, "token", string(pubsubToken.Value))
	assertOSVersion("Android 16 (2026-05-01)", "16 (2026-05-01)")

	// A newer patch level for the same major version produces a distinct
	// operating_systems row; the host now points at the new one.
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub",
		&service.PubSubPushRequest{PubSubMessage: osVersionStatusReportMessage(t, esi, "16", "2026-06-01")},
		http.StatusOK, "token", string(pubsubToken.Value))
	assertOSVersion("Android 16 (2026-06-01)", "16 (2026-06-01)")

	mysqltest.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		var versions []string
		require.NoError(t, sqlx.SelectContext(ctx, q, &versions,
			`SELECT version FROM operating_systems
			 WHERE name = 'Android' AND version IN ('1', '16 (2026-05-01)', '16 (2026-06-01)')
			 ORDER BY version`))
		// "1" is the bare version from enrollment (now orphaned; the cleanup cron
		// that removes unreferenced rows does not run in this test). The two folded
		// versions confirm each security patch level is a distinct row.
		assert.Equal(t, []string{"1", "16 (2026-05-01)", "16 (2026-06-01)"}, versions,
			"each security patch level should be a distinct operating_systems row")
		return nil
	})

	// A device that does not report a patch level falls back to the bare version.
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub",
		&service.PubSubPushRequest{PubSubMessage: osVersionStatusReportMessage(t, esi, "16", "")},
		http.StatusOK, "token", string(pubsubToken.Value))
	assertOSVersion("Android 16", "16")
}

func osVersionStatusReportMessage(t *testing.T, esi, androidVersion, securityPatchLevel string) android.PubSubMessage {
	return createStatusReportMessageFromDevice(t, androidmanagement.Device{
		Name: createAndroidDeviceID("test-os-version"),
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: esi,
			Brand:                "TestBrand",
			Model:                "TestModel",
			SerialNumber:         "test-serial",
			Hardware:             "test-hardware",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidBuildNumber: "test-build",
			AndroidVersion:     androidVersion,
			SecurityPatchLevel: securityPatchLevel,
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             int64(8 * 1024 * 1024 * 1024),
			TotalInternalStorage: int64(64 * 1024 * 1024 * 1024),
		},
		LastStatusReportTime: "2024-01-01T12:00:00Z",
	})
}
