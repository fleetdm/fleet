package tests

import (
	"context"
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
	s.WithServer.SetupSuite(s.T(), "androidEnterpriseTestSuite")
	s.Token = "bozo"
}

func (s *softwareTestSuite) SetupTest() {
	s.AppConfig.MDM.AndroidEnabledAndConfigured = false
	s.CreateCommonDSMocks()
	// Override EnterprisesListFunc to return empty list initially (no enterprises exist)
	s.AndroidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{}, nil
	}
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

	s.AppConfig.MDM.AndroidEnabledAndConfigured = true

	// rawData, err := os.ReadFile("./testdata/status_report.json")
	// require.NoError(t, err)
	// data := base64.StdEncoding.EncodeToString(rawData)

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

	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		mysql.DumpTable(t, q, "android_enterprises")
		mysql.DumpTable(t, q, "mdm_config_assets")
		mysql.DumpTable(t, q, "enroll_secrets")
		mysql.DumpTable(t, q, "android_devices")
		return nil
	})

	req = service.PubSubPushRequest{
		PubSubMessage: CreateStatusReportMessage(t, enterpriseSpecificID, "test-android", CreateAndroidDeviceID("test-policy"), ptr.Int(1), nil),
	}

	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	// TODO: how can we hit "normal" server/service endpoints? Can't do it now because those types are private to that package...
	mysql.ExecAdhocSQL(t, s.DS.Datastore, func(q sqlx.ExtContext) error {
		var software []*fleet.Software
		err := sqlx.SelectContext(ctx, q, &software, "SELECT id, name, source FROM software")
		require.NoError(t, err)
		fmt.Printf("software: %v\n", software)
		return nil
	})
}
