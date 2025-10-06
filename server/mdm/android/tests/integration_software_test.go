package tests

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/go-json-experiment/json"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/api/androidmanagement/v1"
)

func TestServiceSoftware(t *testing.T) {
	testingSuite := new(softwareTestSuite)
	suite.Run(t, testingSuite)
}

type softwareTestSuite struct {
	WithServer
}

func (s *softwareTestSuite) SetupSuite() {
	s.WithServer.SetupSuite(s.T(), "androidSoftwareTestSuite")
	s.Token = "testtoken"
}

func (s *softwareTestSuite) TestAndroidSoftwareIngestion() {
	ctx := context.Background()
	t := s.T()

	// Create enterprise
	var signupResp android.EnterpriseSignupResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupResp)
	assert.Equal(s.T(), EnterpriseSignupURL, signupResp.Url)
	s.T().Logf("callbackURL: %s", s.ProxyCallbackURL)

	s.FleetSvc.On("NewActivity", mock.Anything, mock.Anything, mock.AnythingOfType("fleet.ActivityTypeEnabledAndroidMDM")).Return(nil)
	const enterpriseToken = "enterpriseToken"
	s.Do("GET", s.ProxyCallbackURL, nil, http.StatusOK, "enterpriseToken", enterpriseToken)

	// Update the LIST mock to return the enterprise after "creation"
	s.AndroidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{
			{Name: "enterprises/" + EnterpriseID},
		}, nil
	}

	resp := android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(t, EnterpriseID, resp.EnterpriseID)

	// Need to set this because app config is mocked in this setup
	s.AppConfig.MDM.AndroidEnabledAndConfigured = true

	err := s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "enrollsecret"}})
	require.NoError(t, err)

	secrets, err := s.DS.Datastore.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, secrets, 1)

	assets, err := s.DS.Datastore.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]
	require.NotEmpty(t, pubsubToken.Value)

	deviceID1 := createAndroidDeviceID("test-android")
	deviceID2 := createAndroidDeviceID("test-android-2")

	enterpriseSpecificID1 := strings.ToUpper(uuid.New().String())
	enterpriseSpecificID2 := strings.ToUpper(uuid.New().String())
	var req service.PubSubPushRequest
	for _, d := range []struct {
		id  string
		esi string
	}{{deviceID1, enterpriseSpecificID1}, {deviceID2, enterpriseSpecificID2}} {
		enrollmentMessage := enrollmentMessageWithEnterpriseSpecificID(
			t,
			androidmanagement.Device{
				Name:                d.id,
				EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, secrets[0].Secret),
			},
			d.esi,
		)

		req = service.PubSubPushRequest{
			PubSubMessage: *enrollmentMessage,
		}

		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))
	}

	// Send device data including software for device 1
	software1 := []*androidmanagement.ApplicationReport{{
		DisplayName: "Google Chrome",
		PackageName: "com.google.chrome",
		VersionName: "1.0.0",
		State:       "INSTALLED",
	}}
	deviceData1 := createAndroidDeviceWithSoftware(enterpriseSpecificID1, "test-android", createAndroidDeviceID("test-policy"), ptr.Int(1), nil, software1)
	req = service.PubSubPushRequest{
		PubSubMessage: createStatusReportMessageFromDevice(t, deviceData1),
	}

	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	// Send device data including software for device 2
	software2 := []*androidmanagement.ApplicationReport{{
		DisplayName: "Google Chrome",
		PackageName: "com.google.chrome",
		VersionName: "2.0.0",
		State:       "INSTALLED",
	}}
	deviceData2 := createAndroidDeviceWithSoftware(enterpriseSpecificID2, "test-android-2", createAndroidDeviceID("test-policy"), ptr.Int(1), nil, software2)
	req = service.PubSubPushRequest{
		PubSubMessage: createStatusReportMessageFromDevice(t, deviceData2),
	}

	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		// Check software table for correct values, we should have as many rows as there were ApplicationReports sent
		var software []*fleet.Software
		err := sqlx.SelectContext(ctx, q, &software, "SELECT id, name, application_id, source, title_id FROM software")
		require.NoError(t, err)
		assert.Len(t, software, len(deviceData1.ApplicationReports)+len(deviceData2.ApplicationReports))

		// Check software_titles, we should have fewer rows here, because some ApplicationRows map to the same title.
		var titles []fleet.SoftwareTitle
		err = sqlx.SelectContext(ctx, q, &titles, "SELECT id, name, application_id, source FROM software_titles")
		require.NoError(t, err)

		require.Len(t, titles, 1)

		for _, s := range software {
			// Validate that both softwares map to the same title
			assert.Equal(t, *s.TitleID, titles[0].ID)

			// Check other fields are as expected
			assert.Equal(t, "com.google.chrome", s.ApplicationID)
			assert.Equal(t, "Google Chrome", s.Name)
			assert.Equal(t, "android_apps", s.Source)
		}
		return nil
	})

	// Add some new software for the first device
	software1 = []*androidmanagement.ApplicationReport{
		{
			DisplayName: "Google Chrome",
			PackageName: "com.google.chrome",
			VersionName: "1.0.0",
			State:       "INSTALLED",
		},
		{
			DisplayName: "YouTube",
			PackageName: "com.google.youtube",
			VersionName: "1.0.0",
			State:       "INSTALLED",
		},
		{
			DisplayName: "Google Drive",
			PackageName: "com.google.drive",
			VersionName: "1.0.0",
			State:       "INSTALLED",
		},
	}
	deviceData1 = createAndroidDeviceWithSoftware(enterpriseSpecificID1, "test-android", createAndroidDeviceID("test-policy"), ptr.Int(1), nil, software1)
	req = service.PubSubPushRequest{
		PubSubMessage: createStatusReportMessageFromDevice(t, deviceData1),
	}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	var hostID1 uint
	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		// Get host id
		err := sqlx.GetContext(ctx, q, &hostID1, "SELECT id FROM hosts WHERE uuid = ?", enterpriseSpecificID1)
		require.NoError(t, err)

		var software []*fleet.Software
		err = sqlx.SelectContext(ctx, q, &software, "SELECT id, name, application_id, source, title_id FROM software JOIN host_software ON host_software.software_id = software.id WHERE host_software.host_id = ?", hostID1)
		require.NoError(t, err)
		assert.Len(t, software, len(deviceData1.ApplicationReports)) // chrome, youtube, google drive

		expectedSW := map[string]fleet.Software{
			"com.google.chrome":  {ApplicationID: "com.google.chrome", Name: "Google Chrome", Source: "android_apps"},
			"com.google.youtube": {ApplicationID: "com.google.youtube", Name: "YouTube", Source: "android_apps"},
			"com.google.drive":   {ApplicationID: "com.google.drive", Name: "Google Drive", Source: "android_apps"},
		}

		for _, got := range software {
			// Check other fields are as expected
			expected, ok := expectedSW[got.ApplicationID]
			require.Truef(t, ok, "got unexpected software with application id %s", got.ApplicationID)

			assert.Equal(t, expected.ApplicationID, got.ApplicationID)
			assert.Equal(t, expected.Name, got.Name)
			assert.Equal(t, expected.Source, got.Source)

		}

		return nil
	})

	// Remove some software from the first device
	software1 = []*androidmanagement.ApplicationReport{
		{
			DisplayName: "YouTube",
			PackageName: "com.google.youtube",
			VersionName: "1.0.0",
			State:       "INSTALLED",
		},
	}
	deviceData1 = createAndroidDeviceWithSoftware(enterpriseSpecificID1, "test-android", createAndroidDeviceID("test-policy"), ptr.Int(1), nil, software1)
	req = service.PubSubPushRequest{
		PubSubMessage: createStatusReportMessageFromDevice(t, deviceData1),
	}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		var software []*fleet.Software
		err = sqlx.SelectContext(ctx, q, &software, "SELECT id, name, application_id, source, title_id FROM software JOIN host_software ON host_software.software_id = software.id WHERE host_software.host_id = ?", hostID1)
		require.NoError(t, err)
		assert.Len(t, software, len(deviceData1.ApplicationReports)) // just youtube now

		expectedSW := map[string]fleet.Software{
			"com.google.youtube": {ApplicationID: "com.google.youtube", Name: "YouTube", Source: "android_apps"},
		}

		for _, got := range software {
			// Check other fields are as expected
			expected, ok := expectedSW[got.ApplicationID]
			require.Truef(t, ok, "got unexpected software with application id %s", got.ApplicationID)

			assert.Equal(t, expected.ApplicationID, got.ApplicationID)
			assert.Equal(t, expected.Name, got.Name)
			assert.Equal(t, expected.Source, got.Source)

		}

		return nil
	})

	// Send the same software again, nothing changes
	deviceData1 = createAndroidDeviceWithSoftware(enterpriseSpecificID1, "test-android", createAndroidDeviceID("test-policy"), ptr.Int(1), nil, software1)
	req = service.PubSubPushRequest{
		PubSubMessage: createStatusReportMessageFromDevice(t, deviceData1),
	}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		var software []*fleet.Software
		err = sqlx.SelectContext(ctx, q, &software, "SELECT id, name, application_id, source, title_id FROM software JOIN host_software ON host_software.software_id = software.id WHERE host_software.host_id = ?", hostID1)
		require.NoError(t, err)
		assert.Len(t, software, len(deviceData1.ApplicationReports)) // just youtube now

		expectedSW := map[string]fleet.Software{
			"com.google.youtube": {ApplicationID: "com.google.youtube", Name: "YouTube", Source: "android_apps"},
		}

		for _, got := range software {
			// Check other fields are as expected
			expected, ok := expectedSW[got.ApplicationID]
			require.Truef(t, ok, "got unexpected software with application id %s", got.ApplicationID)

			assert.Equal(t, expected.ApplicationID, got.ApplicationID)
			assert.Equal(t, expected.Name, got.Name)
			assert.Equal(t, expected.Source, got.Source)

		}

		return nil
	})

}

func enrollmentMessageWithEnterpriseSpecificID(t *testing.T, deviceInfo androidmanagement.Device, enterpriseSpecificID string) *android.PubSubMessage {
	deviceInfo.HardwareInfo = &androidmanagement.HardwareInfo{
		EnterpriseSpecificId: enterpriseSpecificID,
		Brand:                "TestBrand",
		Model:                "TestModel",
		SerialNumber:         "test-serial",
		Hardware:             "test-hardware",
	}
	deviceInfo.SoftwareInfo = &androidmanagement.SoftwareInfo{
		AndroidBuildNumber: "test-build",
		AndroidVersion:     "1",
	}
	deviceInfo.MemoryInfo = &androidmanagement.MemoryInfo{
		TotalRam:             int64(8 * 1024 * 1024 * 1024),  // 8GB RAM in bytes
		TotalInternalStorage: int64(64 * 1024 * 1024 * 1024), // 64GB system partition
	}

	deviceInfo.MemoryEvents = []*androidmanagement.MemoryEvent{
		{
			EventType:  "EXTERNAL_STORAGE_DETECTED",
			ByteCount:  int64(64 * 1024 * 1024 * 1024), // 64GB external/built-in storage total capacity
			CreateTime: "2024-01-15T09:00:00Z",
		},
		{
			EventType:  "INTERNAL_STORAGE_MEASURED",
			ByteCount:  int64(10 * 1024 * 1024 * 1024), // 10GB free in system partition
			CreateTime: "2024-01-15T10:00:00Z",
		},
		{
			EventType:  "EXTERNAL_STORAGE_MEASURED",
			ByteCount:  int64(25 * 1024 * 1024 * 1024), // 25GB free in external/built-in storage
			CreateTime: "2024-01-15T10:00:00Z",
		},
	}

	data, err := json.Marshal(deviceInfo)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return &android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubEnrollment),
		},
		Data: encodedData,
	}
}

func createAndroidDeviceID(name string) string {
	return "enterprises/mock-enterprise-id/devices/" + name
}

func createAndroidDeviceWithSoftware(deviceId, name, policyName string, policyVersion *int, nonComplianceDetails []*androidmanagement.NonComplianceDetail, software []*androidmanagement.ApplicationReport) androidmanagement.Device {
	return androidmanagement.Device{
		Name:                 createAndroidDeviceID(name),
		NonComplianceDetails: nonComplianceDetails,
		HardwareInfo: &androidmanagement.HardwareInfo{
			EnterpriseSpecificId: deviceId,
			Brand:                "TestBrand",
			Model:                "TestModel",
			SerialNumber:         "test-serial",
			Hardware:             "test-hardware",
		},
		SoftwareInfo: &androidmanagement.SoftwareInfo{
			AndroidBuildNumber: "test-build",
			AndroidVersion:     "1",
		},
		MemoryInfo: &androidmanagement.MemoryInfo{
			TotalRam:             int64(8 * 1024 * 1024 * 1024),  // 8GB RAM in bytes
			TotalInternalStorage: int64(64 * 1024 * 1024 * 1024), // 64GB system partition
		},
		AppliedPolicyName:    policyName,
		AppliedPolicyVersion: int64(*policyVersion),
		LastPolicySyncTime:   "2001-01-01T00:00:00Z",
		ApplicationReports:   software,
	}
}

func createStatusReportMessageFromDevice(t *testing.T, device androidmanagement.Device) android.PubSubMessage {
	data, err := json.Marshal(device)
	require.NoError(t, err)

	encodedData := base64.StdEncoding.EncodeToString(data)

	return android.PubSubMessage{
		Attributes: map[string]string{
			"notificationType": string(android.PubSubStatusReport),
		},
		Data: encodedData,
	}
}
