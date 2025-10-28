package service

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func (s *integrationMDMTestSuite) TestAndroidAppSelfService() {
	ctx := context.Background()
	t := s.T()

	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	t.Cleanup(func() {
		appConf, err := s.ds.AppConfig(context.Background())
		require.NoError(s.T(), err)
		appConf.MDM.AndroidEnabledAndConfigured = true
		err = s.ds.SaveAppConfig(context.Background(), appConf)
		require.NoError(s.T(), err)
	})

	// Adding android app before android MDM is turned on should fail
	var addAppResp addAppStoreAppResponse
	s.DoJSON(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: "com.should.fail", Platform: fleet.AndroidPlatform},
		http.StatusNotFound,
		&addAppResp,
	)

	EnterpriseID := "LC02k5wxw7"
	EnterpriseSignupURL := "https://enterprise.google.com/signup/android/email?origin=android&thirdPartyToken=B4D779F1C4DD9A440"
	s.androidAPIClient.InitCommonMocks()

	s.androidAPIClient.EnterprisesCreateFunc = func(_ context.Context, _ androidmgmt.EnterprisesCreateRequest) (androidmgmt.EnterprisesCreateResponse, error) {
		return androidmgmt.EnterprisesCreateResponse{
			EnterpriseName: "enterprises/" + EnterpriseID,
			TopicName:      "projects/android/topics/ae98ed130-5ce2-4ddb-a90a-191ec76976d5",
		}, nil
	}
	s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(_ context.Context, policyName string, _ *androidmanagement.Policy) (*androidmanagement.Policy, error) {
		assert.Contains(t, policyName, EnterpriseID)
		return &androidmanagement.Policy{}, nil
	}
	s.androidAPIClient.EnterpriseDeleteFunc = func(_ context.Context, enterpriseName string) error {
		assert.Equal(t, "enterprises/"+EnterpriseID, enterpriseName)
		return nil
	}

	s.androidAPIClient.SignupURLsCreateFunc = func(_ context.Context, _, callbackURL string) (*android.SignupDetails, error) {
		s.proxyCallbackURL = callbackURL
		return &android.SignupDetails{
			Url:  EnterpriseSignupURL,
			Name: "signupUrls/Cb08124d0999c464f",
		}, nil
	}

	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicy *androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		return nil, nil
	}

	// Create enterprise
	var signupResp android.EnterpriseSignupResponse
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise/signup_url", nil, http.StatusOK, &signupResp)

	const enterpriseToken = "enterpriseToken"

	// callback URL includes the host, need to extract the path so we can call it with our
	// HTTP request helpers
	u, err := url.Parse(s.proxyCallbackURL)
	require.NoError(t, err)
	s.Do("GET", u.Path, nil, http.StatusOK, "enterpriseToken", enterpriseToken)

	// Update the LIST mock to return the enterprise after "creation"
	s.androidAPIClient.EnterprisesListFunc = func(_ context.Context, _ string) ([]*androidmanagement.Enterprise, error) {
		return []*androidmanagement.Enterprise{
			{Name: "enterprises/" + EnterpriseID},
		}, nil
	}

	resp := android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(t, EnterpriseID, resp.EnterpriseID)

	// Android MDM setup
	androidApp := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "com.whatsapp",
				Platform: fleet.AndroidPlatform,
			},
		},
		Name:             "WhatsApp",
		BundleIdentifier: "com.whatsapp",
		IconURL:          "https://example.com/images/2",
	}

	// Invalid application ID format: should fail
	r := s.Do(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: "thisisnotanappid", Platform: fleet.AndroidPlatform},
		http.StatusUnprocessableEntity,
	)
	require.Contains(t, extractServerErrorText(r.Body), "app_store_id must be a valid Android application ID")

	// Missing platform: should fail
	r = s.Do(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: "com.valid.app.id"},
		http.StatusUnprocessableEntity,
	)
	require.Contains(t, extractServerErrorText(r.Body), "platform is required")

	// Valid application ID format, but app isn't found: should fail
	// Update mock to return a 404
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		return nil, &notFoundError{}
	}

	r = s.Do(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: "com.app.id.not.found", Platform: fleet.AndroidPlatform},
		http.StatusUnprocessableEntity,
	)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't add software. The application ID isn't available in Play Store. Please find ID on the Play Store and try again.")

	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		return &androidmanagement.Application{IconUrl: "https://example.com/1.jpg", Title: "Test App"}, nil
	}

	// Add Android app
	s.DoJSON(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: androidApp.AdamID, Platform: fleet.AndroidPlatform},
		http.StatusOK,
		&addAppResp,
	)

	secrets, err := s.ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, secrets, 1)

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]
	require.NotEmpty(t, pubsubToken.Value)

	deviceID1 := createAndroidDeviceID("test-android")
	deviceID2 := createAndroidDeviceID("test-android-2")

	enterpriseSpecificID1 := strings.ToUpper(uuid.New().String())
	enterpriseSpecificID2 := strings.ToUpper(uuid.New().String())
	var req android_service.PubSubPushRequest
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

		req = android_service.PubSubPushRequest{
			PubSubMessage: *enrollmentMessage,
		}

		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))
	}

	var hosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts)

	assert.Len(t, hosts.Hosts, 2)

	host1 := hosts.Hosts[0]
	assert.Equal(t, host1.Platform, string(fleet.AndroidPlatform))

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		mysql.DumpTable(t, q, "software_titles")
		mysql.DumpTable(t, q, "vpp_apps")
		mysql.DumpTable(t, q, "vpp_apps_teams")
		return nil
	})

	// Should see it in host software library
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	assert.Len(t, getHostSw.Software, 1)

}
