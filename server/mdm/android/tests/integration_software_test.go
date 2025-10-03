package tests

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
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
	s.Token = "bozo"
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
	res := s.Do("GET", s.ProxyCallbackURL, nil, http.StatusOK, "enterpriseToken", enterpriseToken)
	s.FleetSvc.AssertNumberOfCalls(s.T(), "NewActivity", 1)
	body, err := io.ReadAll(res.Body)
	require.NoError(s.T(), err)
	assert.Equal(s.T(), "text/html; charset=UTF-8", res.Header.Get("Content-Type"))
	assert.Contains(s.T(), string(body), "If this page does not close automatically, please close it manually.")
	assert.Contains(s.T(), string(body), "window.close()")

	// Update the LIST mock to return the enterprise after "creation"
	s.AndroidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{
			{Name: "enterprises/" + EnterpriseID},
		}, nil
	}

	// Now enterprise exists and we can retrieve it.
	resp := android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(s.T(), EnterpriseID, resp.EnterpriseID)

	// Need to set this because app config is mocked in this setup
	s.AppConfig.MDM.AndroidEnabledAndConfigured = true

	err = s.DS.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: "enrollsecret"}})
	require.NoError(t, err)

	secrets, err := s.DS.Datastore.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, secrets, 1)

	deviceID := CreateAndroidDeviceID("test-android")

	enterpriseSpecificID := strings.ToUpper(uuid.New().String())
	enrollmentMessage := EnrollmentMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                deviceID,
			EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, secrets[0].Secret),
		},
		enterpriseSpecificID,
	)

	assets, err := s.DS.Datastore.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]
	require.NotEmpty(t, pubsubToken.Value)

	req := service.PubSubPushRequest{
		PubSubMessage: *enrollmentMessage,
	}

	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	device := CreateAndroidDevice(t, enterpriseSpecificID, "test-android", CreateAndroidDeviceID("test-policy"), ptr.Int(1), nil)
	req = service.PubSubPushRequest{
		PubSubMessage: CreateStatusReportMessageFromDevice(t, device),
	}

	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		var software []*fleet.Software
		err := sqlx.SelectContext(ctx, q, &software, "SELECT id, name, application_id, source FROM software")
		require.NoError(t, err)
		assert.Len(t, software, len(device.ApplicationReports))

		for i, s := range software {
			assert.Equal(t, device.ApplicationReports[i].PackageName, s.ApplicationID)
		}
		return nil
	})

	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		mysql.DumpTable(t, q, "android_enterprises")
		mysql.DumpTable(t, q, "mdm_config_assets")
		mysql.DumpTable(t, q, "enroll_secrets")
		mysql.DumpTable(t, q, "android_devices")
		mysql.DumpTable(t, q, "software_titles")
		return nil
	})
}

func CreateEnrollmentMessage(t *testing.T, deviceInfo androidmanagement.Device) *android.PubSubMessage {
	return EnrollmentMessageWithEnterpriseSpecificID(t, deviceInfo, strings.ToUpper(uuid.New().String()))
}

func EnrollmentMessageWithEnterpriseSpecificID(t *testing.T, deviceInfo androidmanagement.Device, enterpriseSpecificID string) *android.PubSubMessage {
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

func CreateAndroidDeviceID(name string) string {
	return "enterprises/mock-enterprise-id/devices/" + name
}

func CreateAndroidDevice(t *testing.T, deviceId, name, policyName string, policyVersion *int, nonComplianceDetails []*androidmanagement.NonComplianceDetail) androidmanagement.Device {
	return androidmanagement.Device{
		Name:                 CreateAndroidDeviceID(name),
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
		ApplicationReports: []*androidmanagement.ApplicationReport{{
			DisplayName: "Google Chrome",
			PackageName: "com.google.chrome",
			VersionName: "1.0.0",
			State:       "INSTALLED",
		}},
	}
}

func CreateStatusReportMessageFromDevice(t *testing.T, device androidmanagement.Device) android.PubSubMessage {
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

func CreateStatusReportMessage(t *testing.T, deviceId, name, policyName string, policyVersion *int, nonComplianceDetails []*androidmanagement.NonComplianceDetail) android.PubSubMessage {
	return CreateStatusReportMessageFromDevice(t, CreateAndroidDevice(t, deviceId, name, policyName, policyVersion, nonComplianceDetails))
}
