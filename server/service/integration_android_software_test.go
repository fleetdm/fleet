package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/android/service/androidmgmt"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
)

func (s *integrationMDMTestSuite) TestAndroidAppsSelfService() {
	ctx := context.Background()
	t := s.T()

	s.setVPPTokenForTeam(0)
	appConf, err := s.ds.AppConfig(context.Background())
	require.NoError(s.T(), err)
	appConf.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(context.Background(), appConf)
	require.NoError(s.T(), err)

	// Adding android app before android MDM is turned on should fail
	var addAppResp addAppStoreAppResponse
	s.DoJSON(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: "com.should.fail", Platform: fleet.AndroidPlatform},
		http.StatusBadRequest,
		&addAppResp,
	)

	s.enableAndroidMDM(t)

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
	require.Contains(t, extractServerErrorText(r.Body), "Application ID must be a valid Android application ID")

	// Fleet agent app: should fail (cannot be added manually)
	for _, fleetAgentPkg := range []string{
		"com.fleetdm.agent",
		"com.fleetdm.agent.pingali",
		"com.fleetdm.agent.private.testuser",
	} {
		r = s.Do(
			"POST",
			"/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{AppStoreID: fleetAgentPkg, Platform: fleet.AndroidPlatform},
			http.StatusUnprocessableEntity,
		)
		require.Contains(t, extractServerErrorText(r.Body), "The Fleet agent cannot be added manually",
			"expected Fleet Agent package %s to be blocked", fleetAgentPkg)
	}

	// Missing platform: should fail
	r = s.Do(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: "com.valid.app.id"},
		http.StatusUnprocessableEntity,
	)
	s.Assert().Contains(extractServerErrorText(r.Body), "Couldn't add software. com.valid.app.id isn't available in Apple Business Manager or Play Store. Please purchase a license in Apple Business Manager or find the app in Play Store and try again.")

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
	s.Assert().Contains(extractServerErrorText(r.Body), "Couldn't add software. The application ID isn't available in Play Store. Please find ID on the Play Store and try again.")

	amapiConfig := struct {
		AppIDsToNames                     map[string]string
		EnterprisesPoliciesPatchValidator func(policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts)
	}{
		AppIDsToNames:                     map[string]string{},
		EnterprisesPoliciesPatchValidator: func(policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) {},
	}

	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		title := amapiConfig.AppIDsToNames[packageName]

		return &androidmanagement.Application{IconUrl: "https://example.com/1.jpg", Title: title}, nil
	}

	s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		amapiConfig.EnterprisesPoliciesPatchValidator(policyName, policy, opts)

		return &androidmanagement.Policy{}, nil
	}

	// Valid application ID format, but wrong platform specified: should fail
	r = s.Do(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: "com.valid", Platform: fleet.MacOSPlatform},
		http.StatusUnprocessableEntity,
	)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't add software. com.valid isn't available in Apple Business Manager or Play Store. Please purchase a license in Apple Business Manager or find the app in Play Store and try again.")

	// Add Android app
	s.DoJSON(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: androidApp.AdamID, Platform: fleet.AndroidPlatform},
		http.StatusOK,
		&addAppResp,
	)

	// self_service is coerced to be true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var selfService bool
		err := sqlx.GetContext(ctx, q, &selfService, "SELECT self_service FROM vpp_apps_teams WHERE adam_id = ?", androidApp.AdamID)
		s.Require().NoError(err)
		s.Assert().True(selfService)
		return nil
	})

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

	// Should see it in host software library
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	assert.Len(t, getHostSw.Software, 1)
	s.Assert().NotNil(getHostSw.Software[0].AppStoreApp)
	s.Assert().Equal(androidApp.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)

	// Should see it in software titles
	err = s.ds.SyncHostsSoftware(context.Background(), time.Now())
	require.NoError(t, err)
	err = s.ds.SyncHostsSoftwareTitles(context.Background(), time.Now())
	require.NoError(t, err)

	var listSWTitles listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(0))

	s.Assert().Len(listSWTitles.SoftwareTitles, 1)
	s.Assert().Equal(androidApp.AdamID, listSWTitles.SoftwareTitles[0].AppStoreApp.AppStoreID)
	s.Assert().Empty(listSWTitles.SoftwareTitles[0].AppStoreApp.Version)

	// Google AMAPI hasn't been hit yet
	s.Assert().False(s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)

	// Run worker, should run the job that assigns the app to the host's MDM policy
	s.runWorkerUntilDone()

	// Should have hit the android API endpoint
	s.Assert().True(s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked = false

	s.DoJSON(
		"PATCH",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", getHostSw.Software[0].ID),
		&updateAppStoreAppRequest{SelfService: ptr.Bool(false)},
		http.StatusOK,
		&addAppResp,
	)

	// Even though we sent self_service: false, self_service remains true
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var selfService bool
		err := sqlx.GetContext(ctx, q, &selfService, "SELECT self_service FROM vpp_apps_teams WHERE adam_id = ?", getHostSw.Software[0].AppStoreApp.AppStoreID)
		s.Require().NoError(err)
		s.Assert().True(selfService)
		return nil
	})

	// Add some apps to a different team. They shouldn't be sent to our existing host
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team
	// Add Android app
	androidAppNewTeam := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "com.my.cool.app",
				Platform: fleet.AndroidPlatform,
			},
		},
		Name:             "My cool app",
		BundleIdentifier: "com.my.cool.app",
		IconURL:          "https://example.com/images/3",
	}
	s.DoJSON(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: androidAppNewTeam.AdamID, Platform: fleet.AndroidPlatform, TeamID: &team.ID},
		http.StatusOK,
		&addAppResp,
	)

	// New app should not show up in "No team" library
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(0))
	s.Assert().Len(listSWTitles.SoftwareTitles, 1)
	s.Assert().Equal(androidApp.AdamID, listSWTitles.SoftwareTitles[0].AppStoreApp.AppStoreID) // just the app we had before
	s.Assert().Empty(listSWTitles.SoftwareTitles[0].AppStoreApp.Version)

	// New app SHOULD show up in our new team library
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(team.ID))
	s.Assert().Len(listSWTitles.SoftwareTitles, 1)
	s.Assert().Equal(androidAppNewTeam.AdamID, listSWTitles.SoftwareTitles[0].AppStoreApp.AppStoreID)
	s.Assert().Empty(listSWTitles.SoftwareTitles[0].AppStoreApp.Version)

	androidAppNewTeam2 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "com.my.cool.app.two",
				Platform: fleet.AndroidPlatform,
			},
		},
		Name:             "My cool app 2",
		BundleIdentifier: "com.my.cool.app.two",
		IconURL:          "https://example.com/images/4",
	}

	amapiConfig.AppIDsToNames[androidAppNewTeam2.AdamID] = androidAppNewTeam2.Name

	s.DoJSON(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{AppStoreID: androidAppNewTeam2.AdamID, Platform: fleet.AndroidPlatform, TeamID: &team.ID},
		http.StatusOK,
		&addAppResp,
	)

	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(team.ID))
	s.Assert().Len(listSWTitles.SoftwareTitles, 2)
	s.Assert().True(slices.ContainsFunc(listSWTitles.SoftwareTitles, func(t fleet.SoftwareTitleListResult) bool {
		return t.AppStoreApp.AppStoreID == androidAppNewTeam.AdamID || t.AppStoreApp.AppStoreID == androidAppNewTeam2.AdamID
	}))

	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %s, "fleet_id": %s, "platform": "%s", "self_service": true}`,
			team.Name, team.Name, androidAppNewTeam2.Name, addAppResp.TitleID, androidAppNewTeam2.AdamID, fmt.Sprint(team.ID), fmt.Sprint(team.ID), androidAppNewTeam2.Platform), 0)

	s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked = false
	s.runWorkerUntilDone()
	// We shouldn't have hit the AMAPI, since there are no hosts in the team
	s.Assert().False(s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	s.Assert().False(s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)

	amapiConfig.EnterprisesPoliciesPatchValidator = func(policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) {
		var appIDs []string
		for _, a := range policy.Applications {
			appIDs = append(appIDs, a.PackageName)
		}

		s.Assert().ElementsMatch(appIDs, []string{androidAppNewTeam.AdamID, androidAppNewTeam2.AdamID, "com.fleetdm.agent"})
		s.Assert().Contains(policyName, host1.UUID)
	}

	// Transfer a host to the team
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &team.ID,
		HostIDs: []uint{host1.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	s.runWorkerUntilDone()
	s.Assert().True(s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)

	// Transfer host back to "No team"
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  nil,
		HostIDs: []uint{host1.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	// =========================================
	//       Android app configurations
	// =========================================

	// Title with no configuration should omit it from response
	var getAppResp map[string]any
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", addAppResp.TitleID), &getSoftwareTitleRequest{
		ID:     addAppResp.TitleID,
		TeamID: nil,
	}, http.StatusOK, &getAppResp)
	require.Nil(t, getAppResp["configuration"])

	// Android app with configuration
	appConfiguration := json.RawMessage(`{"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"}`)
	androidAppWithConfig := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "com.fooooooo",
				Platform: fleet.AndroidPlatform,
			},
			Configuration: appConfiguration,
		},
		Name:             "foo",
		BundleIdentifier: "com.fooooooo",
		IconURL:          "https://example.com/images/2",
	}

	amapiConfig.AppIDsToNames[androidAppWithConfig.AdamID] = androidAppWithConfig.Name

	// Add Android app
	var appWithConfigResp addAppStoreAppResponse
	s.DoJSON(
		"POST",
		"/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{
			AppStoreID:    androidAppWithConfig.AdamID,
			Platform:      androidAppWithConfig.VPPAppID.Platform,
			Configuration: androidAppWithConfig.Configuration,
		},
		http.StatusOK,
		&appWithConfigResp,
	)

	// Verify that activity includes configuration
	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %s, "fleet_id": %s, "platform": "%s", "self_service": true,"configuration": %s}`,
			fleet.TeamNameNoTeam, fleet.TeamNameNoTeam, androidAppWithConfig.Name, appWithConfigResp.TitleID, androidAppWithConfig.AdamID, "null", "null", androidAppWithConfig.Platform, androidAppWithConfig.Configuration), 0)

	// Should see it in host software library
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	assert.Len(t, getHostSw.Software, 2)
	s.Assert().NotNil(getHostSw.Software[1].AppStoreApp)
	s.Assert().Equal(androidAppWithConfig.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)

	// Edit app without changing configuration
	s.DoJSON(
		"PATCH",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", appWithConfigResp.TitleID),
		&updateAppStoreAppRequest{},
		http.StatusOK,
		&addAppResp,
	)
	s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", 0)

	var titleWithConfigResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", appWithConfigResp.TitleID), &getSoftwareTitleRequest{
		ID:     appWithConfigResp.TitleID,
		TeamID: nil,
	}, http.StatusOK, &titleWithConfigResp)

	require.Contains(t, string(titleWithConfigResp.SoftwareTitle.AppStoreApp.Configuration), "workProfileWidgets")

	// Edit app and change configuration
	newConfig := json.RawMessage(`{"managedConfiguration": {"key": "value"}}`)
	s.DoJSON(
		"PATCH",
		fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", appWithConfigResp.TitleID),
		&updateAppStoreAppRequest{
			Configuration: newConfig,
		},
		http.StatusOK,
		&addAppResp,
	)

	// Verify that configuration changed and last activity is correct
	s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_icon_url":"https://example.com/1.jpg", "software_title_id": %d, "app_store_id": "%s", "team_id": %s, "fleet_id": %s, "software_display_name":"", "platform": "%s", "self_service": true,"configuration": %s}`,
			"", "", androidAppWithConfig.Name, appWithConfigResp.TitleID, androidAppWithConfig.AdamID, "null", "null", androidAppWithConfig.Platform, newConfig), 0)
}

func (s *integrationMDMTestSuite) TestAndroidSetupExperienceSoftware() {
	t := s.T()
	s.enableAndroidMDM(t)

	app1 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "com.test1",
				Platform: fleet.AndroidPlatform,
			},
		},
		Name:             "Test1",
		BundleIdentifier: "com.test1",
		IconURL:          "https://example.com/1",
	}
	app2 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "com.test2",
				Platform: fleet.AndroidPlatform,
			},
		},
		Name:             "Test2",
		BundleIdentifier: "com.test2",
		IconURL:          "https://example.com/2",
	}

	androidApps := []*fleet.VPPApp{app1, app2}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		for _, app := range androidApps {
			if app.AdamID == packageName {
				return &androidmanagement.Application{IconUrl: app.IconURL, Title: app.Name}, nil
			}
		}
		return nil, &notFoundError{}
	}

	// add Android app 1
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app1.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app1TitleID := addAppResp.TitleID

	// add Android app 2
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app2.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app2TitleID := addAppResp.TitleID

	require.NotEqual(t, app1TitleID, app2TitleID)

	// add app 1 to Android setup experience
	var putResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID},
	}, http.StatusOK, &putResp)

	// verify that the expected activity got created
	s.lastActivityOfTypeMatches(fleet.ActivityEditedSetupExperienceSoftware{}.ActivityName(),
		`{"platform": "android", "team_id": 0, "team_name": "", "fleet_id": 0, "fleet_name": ""}`, 0)

	// list the available setup experience software and verify that only app 1 is installed at setup
	var getResp getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", nil, http.StatusOK, &getResp,
		"team_id", "0", "platform", string(fleet.AndroidPlatform), "order_key", "name")
	require.Len(t, getResp.SoftwareTitles, 2)
	require.Equal(t, app1TitleID, getResp.SoftwareTitles[0].ID)
	require.Equal(t, app1.Name, getResp.SoftwareTitles[0].Name)
	require.Equal(t, app1.AdamID, getResp.SoftwareTitles[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.True(t, *getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.Equal(t, app2TitleID, getResp.SoftwareTitles[1].ID)
	require.Equal(t, app2.Name, getResp.SoftwareTitles[1].Name)
	require.Equal(t, app2.AdamID, getResp.SoftwareTitles[1].AppStoreApp.AppStoreID)
	require.NotNil(t, getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)
	require.False(t, *getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)

	// set app1 and app2 to be installed at setup
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID, app2TitleID},
	}, http.StatusOK, &putResp)

	getResp = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", nil, http.StatusOK, &getResp,
		"team_id", "0", "platform", string(fleet.AndroidPlatform), "order_key", "name")
	require.Len(t, getResp.SoftwareTitles, 2)
	require.Equal(t, app1TitleID, getResp.SoftwareTitles[0].ID)
	require.NotNil(t, getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.True(t, *getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.Equal(t, app2TitleID, getResp.SoftwareTitles[1].ID)
	require.NotNil(t, getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)
	require.True(t, *getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)

	// unset all apps to be installed at setup
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{},
	}, http.StatusOK, &putResp)

	getResp = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", nil, http.StatusOK, &getResp,
		"team_id", "0", "platform", string(fleet.AndroidPlatform), "order_key", "name")
	require.Len(t, getResp.SoftwareTitles, 2)
	require.Equal(t, app1TitleID, getResp.SoftwareTitles[0].ID)
	require.NotNil(t, getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.False(t, *getResp.SoftwareTitles[0].AppStoreApp.InstallDuringSetup)
	require.Equal(t, app2TitleID, getResp.SoftwareTitles[1].ID)
	require.NotNil(t, getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)
	require.False(t, *getResp.SoftwareTitles[1].AppStoreApp.InstallDuringSetup)
}

func (s *integrationMDMTestSuite) enableAndroidMDM(t *testing.T) string {
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

	enterpriseID := "LC02k5wxw7"
	enterpriseSignupURL := "https://enterprise.google.com/signup/android/email?origin=android&thirdPartyToken=B4D779F1C4DD9A440"
	s.androidAPIClient.InitCommonMocks()

	s.androidAPIClient.EnterprisesCreateFunc = func(_ context.Context, _ androidmgmt.EnterprisesCreateRequest) (androidmgmt.EnterprisesCreateResponse, error) {
		return androidmgmt.EnterprisesCreateResponse{
			EnterpriseName: "enterprises/" + enterpriseID,
			TopicName:      "projects/android/topics/ae98ed130-5ce2-4ddb-a90a-191ec76976d5",
		}, nil
	}

	s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(_ context.Context, policyName string, _ *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		assert.Contains(t, policyName, enterpriseID)
		return &androidmanagement.Policy{}, nil
	}

	s.androidAPIClient.EnterpriseDeleteFunc = func(_ context.Context, enterpriseName string) error {
		assert.Equal(t, "enterprises/"+enterpriseID, enterpriseName)
		return nil
	}

	s.androidAPIClient.SignupURLsCreateFunc = func(_ context.Context, _, callbackURL string) (*android.SignupDetails, error) {
		s.proxyCallbackURL = callbackURL
		return &android.SignupDetails{
			Url:  enterpriseSignupURL,
			Name: "signupUrls/Cb08124d0999c464f",
		}, nil
	}

	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		return &androidmanagement.Policy{}, nil
	}

	s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFunc = func(ctx context.Context, policyName string, packageNames []string) (*androidmanagement.Policy, error) {
		return &androidmanagement.Policy{}, nil
	}

	s.androidAPIClient.EnterprisesDevicesPatchFunc = func(ctx context.Context, deviceName string, device *androidmanagement.Device) (*androidmanagement.Device, error) {
		return &androidmanagement.Device{}, nil
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
			{Name: "enterprises/" + enterpriseID},
		}, nil
	}

	resp := android.GetEnterpriseResponse{}
	s.DoJSON("GET", "/api/v1/fleet/android_enterprise", nil, http.StatusOK, &resp)
	assert.Equal(t, enterpriseID, resp.EnterpriseID)

	return enterpriseID
}

func (s *integrationMDMTestSuite) TestBatchAndroidApps() {
	t := s.T()
	ctx := context.Background()

	appConf, err := s.ds.AppConfig(ctx)
	require.NoError(s.T(), err)
	appConf.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appConf)
	require.NoError(s.T(), err)

	s.enableAndroidMDM(t)

	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		return &androidmanagement.Application{IconUrl: "https://example.com/1.jpg", Title: "Test App"}, nil
	}

	teamName := "Android Team For All Tests"
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{
		Name: teamName,
	}, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	teamID := &createTeamResp.Team.ID

	t.Run("android app configurations", func(t *testing.T) {
		// Android app with configuration
		exampleConfiguration := json.RawMessage(`{"workProfileWidgets":"WORK_PROFILE_WIDGETS_ALLOWED"}`)
		androidAppFoo := &fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{
				AppTeamID: ptr.ValOrZero(teamID),
				VPPAppID: fleet.VPPAppID{
					AdamID:   "com.foo",
					Platform: fleet.AndroidPlatform,
				},
				Configuration: exampleConfiguration,
			},
		}

		// Add Android app
		var appWithConfigResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{
				TeamID:        teamID,
				AppStoreID:    androidAppFoo.AdamID,
				Platform:      androidAppFoo.VPPAppID.Platform,
				Configuration: androidAppFoo.Configuration,
			},
			http.StatusOK, &appWithConfigResp,
		)

		// Verify that activity includes configuration
		s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(`{"team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": true,"configuration": %s}`,
				teamName, teamName, "Test App", appWithConfigResp.TitleID, androidAppFoo.AdamID, ptr.ValOrZero(teamID), ptr.ValOrZero(teamID), androidAppFoo.Platform, androidAppFoo.Configuration), 0)

		var listSWTitles listSoftwareTitlesResponse
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(*teamID))
		s.Assert().Len(listSWTitles.SoftwareTitles, 1)
		s.Assert().Equal(androidAppFoo.AdamID, listSWTitles.SoftwareTitles[0].AppStoreApp.AppStoreID)

		// Batch app store apps call won't create an activity
		var batchResp batchAssociateAppStoreAppsResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
					{AppStoreID: "app_2", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
					{AppStoreID: "app_3", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
					{AppStoreID: "app_4", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: exampleConfiguration},
				},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)

		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(*teamID))
		s.Assert().Len(listSWTitles.SoftwareTitles, 4)
		s.Assert().Equal("app_1", listSWTitles.SoftwareTitles[0].AppStoreApp.AppStoreID)
		titleApp1 := listSWTitles.SoftwareTitles[0].ID
		titleApp2 := listSWTitles.SoftwareTitles[1].ID

		// Batch app store apps call won't create an activity

		// Add apps to team 0
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
					{AppStoreID: "app_2", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
					{AppStoreID: "app_3", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
					{AppStoreID: "app_4", SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
				},
			}, http.StatusOK, &batchResp,
		)

		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(0))
		s.Assert().Len(listSWTitles.SoftwareTitles, 4)

		// Update configurations
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", Platform: fleet.AndroidPlatform, Configuration: nil},
					{AppStoreID: "app_2", Platform: fleet.AndroidPlatform, Configuration: exampleConfiguration},
					{AppStoreID: "app_3", Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
					{AppStoreID: "app_4", Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
				},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)

		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(*teamID))
		s.Assert().Len(listSWTitles.SoftwareTitles, 4)

		var titleResp getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleApp1), &getSoftwareTitleRequest{
			ID:     titleApp1,
			TeamID: teamID,
		}, http.StatusOK, &titleResp)
		require.Equal(t, "app_1", *titleResp.SoftwareTitle.ApplicationID)
		require.Equal(t, json.RawMessage(`{}`), titleResp.SoftwareTitle.AppStoreApp.Configuration)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleApp2), &getSoftwareTitleRequest{
			ID:     titleApp2,
			TeamID: teamID,
		}, http.StatusOK, &titleResp)
		require.Equal(t, "app_2", *titleResp.SoftwareTitle.ApplicationID)
		require.Contains(t, string(titleResp.SoftwareTitle.AppStoreApp.Configuration), `"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"`)

		// Remove 2 other apps, 2 configurations should be deleted and 2 should be emptied/remain
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", Platform: fleet.AndroidPlatform, Configuration: nil},
					{AppStoreID: "app_2", Platform: fleet.AndroidPlatform, Configuration: exampleConfiguration},
				},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)

		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitles, "team_id", fmt.Sprint(*teamID))
		s.Assert().Len(listSWTitles.SoftwareTitles, 2)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleApp1), &getSoftwareTitleRequest{
			ID:     titleApp1,
			TeamID: teamID,
		}, http.StatusOK, &titleResp)
		require.Equal(t, "app_1", *titleResp.SoftwareTitle.ApplicationID)
		require.Equal(t, json.RawMessage(`{}`), titleResp.SoftwareTitle.AppStoreApp.Configuration)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleApp2), &getSoftwareTitleRequest{
			ID:     titleApp2,
			TeamID: teamID,
		}, http.StatusOK, &titleResp)
		require.Equal(t, "app_2", *titleResp.SoftwareTitle.ApplicationID)
		require.Contains(t, string(titleResp.SoftwareTitle.AppStoreApp.Configuration), `"workProfileWidgets": "WORK_PROFILE_WIDGETS_ALLOWED"`)

		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps:   []fleet.VPPBatchPayload{},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)
	})
	t.Run("android app setup experience", func(t *testing.T) {
		// Get initial count of edited setup experience activities
		var initialCount int
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			err := sqlx.GetContext(ctx, q, &initialCount, `SELECT COUNT(id) FROM activities WHERE activity_type = 'edited_setup_experience_software'`)
			require.NoError(t, err)
			return nil
		})

		var batchResp batchAssociateAppStoreAppsResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps:   []fleet.VPPBatchPayload{},
			},
			http.StatusOK, &batchResp,
		)
		// Should create an edited setup experience activity, as new software was added
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(false)},
					{AppStoreID: "app_2", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(true)},
				},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)

		// Should not create an edited setup experience activity, nothing changed
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(false)},
					{AppStoreID: "app_2", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(true)},
				},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)

		// Should create an edited setup experience activity, existing software changed
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(true)},
					{AppStoreID: "app_2", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(true)},
				},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)

		// Should create an edited setup experience activity, as new software was added
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
			batchAssociateAppStoreAppsRequest{
				DryRun: false,
				Apps: []fleet.VPPBatchPayload{
					{AppStoreID: "app_1", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(true)},
					{AppStoreID: "app_2", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(true)},
					{AppStoreID: "app_3", Platform: fleet.AndroidPlatform, InstallDuringSetup: ptr.Bool(true)},
				},
			},
			http.StatusOK, &batchResp, "team_name", teamName,
		)
		var count int
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			err := sqlx.GetContext(ctx, q, &count, `SELECT COUNT(id) FROM activities WHERE activity_type = 'edited_setup_experience_software'`)
			require.NoError(t, err)
			return nil
		})
		require.Equal(t, initialCount+3, count)
	})
}

func (s *integrationMDMTestSuite) TestAndroidAppsUninstallOnDelete() {
	ctx := context.Background()
	t := s.T()

	s.setSkipWorkerJobs(t)
	s.setVPPTokenForTeam(0)
	appConf, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConf.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appConf)
	require.NoError(t, err)

	// create a team for the test host that will not be affected
	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &fleet.Team{Name: "test"}, http.StatusOK, &createTeamResp)
	teamID := createTeamResp.Team.ID

	s.enableAndroidMDM(t)

	amapiConfig := struct {
		AppIDsToNames                     map[string]string
		EnterprisesPoliciesPatchValidator func(policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts)
	}{
		AppIDsToNames:                     map[string]string{},
		EnterprisesPoliciesPatchValidator: func(policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) {},
	}

	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		title := amapiConfig.AppIDsToNames[packageName]
		return &androidmanagement.Application{IconUrl: "https://example.com/1.jpg", Title: title}, nil
	}

	s.androidAPIClient.EnterprisesPoliciesPatchFunc = func(ctx context.Context, policyName string, policy *androidmanagement.Policy, opts androidmgmt.PoliciesPatchOpts) (*androidmanagement.Policy, error) {
		amapiConfig.EnterprisesPoliciesPatchValidator(policyName, policy, opts)
		return &androidmanagement.Policy{}, nil
	}

	// add some Android apps
	androidApps := make([]*fleet.VPPApp, 5)
	titleIDs := make([]uint, len(androidApps))
	for i := range androidApps {
		androidApps[i] = &fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{AdamID: "com.app" + fmt.Sprint(i), Platform: fleet.AndroidPlatform},
			},
			Name:             "App" + fmt.Sprint(i),
			BundleIdentifier: "com.app" + fmt.Sprint(i),
			IconURL:          "https://example.com/images/" + fmt.Sprint(i),
		}
		amapiConfig.AppIDsToNames[androidApps[i].AdamID] = androidApps[i].Name

		var addAppResp addAppStoreAppResponse

		// last app goes on the team, will not affect the test
		var teamIDPtr *uint
		if i == len(androidApps)-1 {
			teamIDPtr = &teamID
		}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			AppStoreID: androidApps[i].AdamID,
			Platform:   fleet.AndroidPlatform,
			TeamID:     teamIDPtr,
		}, http.StatusOK, &addAppResp)
		titleIDs[i] = addAppResp.TitleID
	}

	// delete app [0], does not affect any host so no removeApplicationsPolicy call
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleIDs[0]), nil, http.StatusNoContent, "team_id", "0")
	s.runWorkerUntilDone()
	require.False(t, s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFuncInvoked)
	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)

	// enroll a few Android devices
	secrets, err := s.ds.GetEnrollSecrets(ctx, nil)
	require.NoError(t, err)
	require.Len(t, secrets, 1)

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]
	require.NotEmpty(t, pubsubToken.Value)

	deviceID1 := createAndroidDeviceID("test-android")
	deviceID2 := createAndroidDeviceID("test-android-2")
	deviceID3 := createAndroidDeviceID("test-android-3")

	enterpriseSpecificID1 := strings.ToUpper(uuid.New().String())
	enterpriseSpecificID2 := strings.ToUpper(uuid.New().String())
	enterpriseSpecificID3 := strings.ToUpper(uuid.New().String())
	var req android_service.PubSubPushRequest
	for _, d := range []struct {
		id  string
		esi string
	}{{deviceID1, enterpriseSpecificID1}, {deviceID2, enterpriseSpecificID2}, {deviceID3, enterpriseSpecificID3}} {
		enrollmentMessage := enrollmentMessageWithEnterpriseSpecificID(t, androidmanagement.Device{
			Name:                d.id,
			EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, secrets[0].Secret),
		}, d.esi)
		req = android_service.PubSubPushRequest{PubSubMessage: *enrollmentMessage}
		s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))
	}

	var hosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts)
	require.Len(t, hosts.Hosts, 3)
	host1 := hosts.Hosts[0]
	host2 := hosts.Hosts[1]
	host3 := hosts.Hosts[2] // isolated host, not affected by test

	// transfer host3 to the team
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &teamID,
		HostIDs: []uint{host3.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	// run the worker, the hosts get the 3 remaining android apps as self-service-available
	s.runWorkerUntilDone()
	require.False(t, s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFuncInvoked)
	require.True(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked = false
	s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked = false // from now on (after device enrollment), this gets called only when we expect it to

	for _, host := range []fleet.HostResponse{host1, host2} {
		var getHostSw getHostSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw,
			"available_for_install", "true", "order_key", "name")
		require.Len(t, getHostSw.Software, 3)
		require.NotNil(t, getHostSw.Software[0].AppStoreApp)
		require.Equal(t, androidApps[1].AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
		require.NotNil(t, getHostSw.Software[1].AppStoreApp)
		require.Equal(t, androidApps[2].AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
		require.NotNil(t, getHostSw.Software[2].AppStoreApp)
		require.Equal(t, androidApps[3].AdamID, getHostSw.Software[2].AppStoreApp.AppStoreID)
	}

	// delete app [1], should trigger remove from both hosts
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleIDs[1]), nil, http.StatusNoContent, "team_id", "0")
	s.runWorkerUntilDone()
	require.True(t, s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFuncInvoked)
	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	require.False(t, s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)
	s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFuncInvoked = false

	for _, host := range []fleet.HostResponse{host1, host2} {
		var getHostSw getHostSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw,
			"available_for_install", "true", "order_key", "name")
		require.Len(t, getHostSw.Software, 2)
		require.NotNil(t, getHostSw.Software[0].AppStoreApp)
		require.Equal(t, androidApps[2].AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
		require.NotNil(t, getHostSw.Software[1].AppStoreApp)
		require.Equal(t, androidApps[3].AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	}

	// batch-set to keep only app [3] (effectively deletes app[2]), as per a gitops run
	var batchResp batchAssociateAppStoreAppsResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{
		Apps: []fleet.VPPBatchPayload{
			{AppStoreID: androidApps[3].AdamID, SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage("{}")},
		},
	}, http.StatusOK, &batchResp)

	s.runWorkerUntilDone() // calls policies patch, which sets (replaces) the list of apps
	require.False(t, s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFuncInvoked)
	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	require.True(t, s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)
	s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked = false

	for _, host := range []fleet.HostResponse{host1, host2} {
		var getHostSw getHostSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw,
			"available_for_install", "true", "order_key", "name")
		require.Len(t, getHostSw.Software, 1)
		require.NotNil(t, getHostSw.Software[0].AppStoreApp)
		require.Equal(t, androidApps[3].AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	}

	// batch-set to remove all apps
	batchResp = batchAssociateAppStoreAppsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{
		Apps: []fleet.VPPBatchPayload{},
	}, http.StatusOK, &batchResp)

	s.runWorkerUntilDone() // calls policies patch, which sets (replaces) the list of apps
	require.False(t, s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFuncInvoked)
	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	require.True(t, s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)
	s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked = false

	for _, host := range []fleet.HostResponse{host1, host2} {
		var getHostSw getHostSoftwareResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw,
			"available_for_install", "true", "order_key", "name")
		require.Len(t, getHostSw.Software, 0)
	}

	// batch-set again without any app is a no-op
	batchResp = batchAssociateAppStoreAppsResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{
		Apps: []fleet.VPPBatchPayload{},
	}, http.StatusOK, &batchResp)

	s.runWorkerUntilDone()
	require.False(t, s.androidAPIClient.EnterprisesPoliciesRemovePolicyApplicationsFuncInvoked)
	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	require.False(t, s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)

	// isolated host was unaffected, still has the same team app
	var getHostSw getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host3.ID), nil, http.StatusOK, &getHostSw,
		"available_for_install", "true", "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, androidApps[4].AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
}

func (s *integrationMDMTestSuite) TestAndroidWebApps() {
	ctx := context.Background()
	t := s.T()

	s.setSkipWorkerJobs(t)
	appConf, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConf.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appConf)
	require.NoError(t, err)

	// attempt to create a webapp before android MDM is enabled
	body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
		"title": {"Web App"},
		"url":   {"https://example.com"},
	})
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/software/web_apps", body.Bytes(), http.StatusBadRequest, headers)
	require.Contains(t, extractServerErrorText(res.Body), "Android MDM isn't turned on.")

	enterpriseID := s.enableAndroidMDM(t)

	s.androidAPIClient.EnterprisesWebAppsCreateFunc = func(ctx context.Context, enterpriseName string, app *androidmanagement.WebApp) (*androidmanagement.WebApp, error) {
		id := uuid.NewString()
		return &androidmanagement.WebApp{Name: fmt.Sprintf("enterprises/%s/webApps/%s", enterpriseID, id)}, nil
	}

	cases := []struct {
		desc     string
		title    string
		url      string
		iconFile string // filename in testdata/icons/

		wantStatus int
		wantErrMsg string
	}{
		{
			desc:       "missing title",
			title:      "",
			url:        "http://example.com",
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "title multipart field is required",
		},
		{
			desc:       "missing url",
			title:      "WebApp",
			url:        "",
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "url multipart field is required",
		},
		{
			desc:       "invalid url",
			title:      "WebApp",
			url:        "non-absolute-url",
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "The start URL must be a valid absolute URL.",
		},
		{
			desc:       "valid without icon",
			title:      "WebApp",
			url:        "http://example.com",
			wantStatus: http.StatusOK,
			wantErrMsg: "",
		},
		{
			desc:       "invalid icon not a png",
			title:      "WebApp",
			url:        "http://example.com",
			iconFile:   "not-a-png.txt",
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "The icon must be a PNG file and square, with dimensions of at least 512 x 512px.",
		},
		{
			desc:       "invalid icon not square",
			title:      "WebApp",
			url:        "http://example.com",
			iconFile:   "non-square.png",
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "The icon must be a PNG file and square, with dimensions of at least 512 x 512px.",
		},
		{
			desc:       "invalid icon too small",
			title:      "WebApp",
			url:        "http://example.com",
			iconFile:   "200px-square.png",
			wantStatus: http.StatusBadRequest,
			wantErrMsg: "The icon must be a PNG file and square, with dimensions of at least 512 x 512px.",
		},
		{
			desc:       "valid with icon",
			title:      "WebApp",
			url:        "http://example.com",
			iconFile:   "512px-square.png",
			wantStatus: http.StatusOK,
			wantErrMsg: "",
		},
	}
	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			var filename string
			var iconData []byte
			if c.iconFile != "" {
				filename = c.iconFile
				b, err := os.ReadFile(filepath.Join("testdata", "icons", c.iconFile))
				require.NoError(t, err)
				iconData = b
			}

			body, headers := generateMultipartRequest(t, "icon", filename, iconData, s.token, map[string][]string{
				"title": {c.title},
				"url":   {c.url},
			})
			res := s.DoRawWithHeaders("POST", "/api/latest/fleet/software/web_apps", body.Bytes(), c.wantStatus, headers)
			if c.wantErrMsg != "" {
				require.Contains(t, extractServerErrorText(res.Body), c.wantErrMsg)
			} else {
				var resp createAndroidWebAppResponse
				err := json.NewDecoder(res.Body).Decode(&resp)
				require.NoError(t, err)
				require.NotEmpty(t, resp.AppStoreID)
			}
		})
	}
}

func (s *integrationMDMTestSuite) TestAndroidWebAppsCannotSetConfiguration() {
	ctx := context.Background()
	t := s.T()

	s.setSkipWorkerJobs(t)
	appConf, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConf.MDM.AndroidEnabledAndConfigured = false
	err = s.ds.SaveAppConfig(ctx, appConf)
	require.NoError(t, err)

	enterpriseID := s.enableAndroidMDM(t)
	var count int
	s.androidAPIClient.EnterprisesWebAppsCreateFunc = func(ctx context.Context, enterpriseName string, app *androidmanagement.WebApp) (*androidmanagement.WebApp, error) {
		count++
		id := "abc" + fmt.Sprint(count)
		return &androidmanagement.WebApp{Name: fmt.Sprintf("enterprises/%s/webApps/com.google.enterprise.webapp.%s", enterpriseID, id)}, nil
	}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		ix := strings.LastIndex(packageName, ".") // title is the final segment
		return &androidmanagement.Application{IconUrl: "https://example.com/1.jpg", Title: packageName[ix+1:]}, nil
	}

	// create a webapp
	body, headers := generateMultipartRequest(t, "", "", nil, s.token, map[string][]string{
		"title": {"Web App"},
		"url":   {"https://example.com"},
	})
	var resp createAndroidWebAppResponse
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/software/web_apps", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&resp)
	require.NoError(t, err)
	webAppID := resp.AppStoreID

	// add it to Fleet with configuration, will fail
	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: webAppID, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage(`{"key":"value"}`),
	}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "Android web apps don't support configurations.")

	// add it without configuration, will work
	var addResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: webAppID, Platform: fleet.AndroidPlatform,
	}, http.StatusOK, &addResp)
	webAppTitleID := addResp.TitleID

	// update it with configuration, will fail
	res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", webAppTitleID),
		&updateAppStoreAppRequest{Configuration: json.RawMessage(`{"key":"value"}`)},
		http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "Android web apps don't support configurations.")

	// update it without configuration, will work
	var updateResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", webAppTitleID),
		&updateAppStoreAppRequest{DisplayName: ptr.String("MyWebApp")},
		http.StatusOK, &updateResp)
	require.Equal(t, webAppID, updateResp.AppStoreApp.AdamID)
	require.Equal(t, "MyWebApp", updateResp.AppStoreApp.DisplayName)
	require.Equal(t, "abc1", updateResp.AppStoreApp.Name)
	require.True(t, updateResp.AppStoreApp.SelfService)
	require.Nil(t, updateResp.AppStoreApp.Configuration)

	// batch-set with configuration, will fail
	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
		batchAssociateAppStoreAppsRequest{
			DryRun: false,
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: webAppID, SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage(`{"key":"value"}`)},
			},
		}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "Android web apps don't support configurations.")

	// batch-set with multiple Android apps, with configuration on the webApp, will fail
	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch",
		batchAssociateAppStoreAppsRequest{
			DryRun: false,
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: webAppID, SelfService: true, Platform: fleet.AndroidPlatform, Configuration: json.RawMessage(`{"key":"value"}`)},
				{AppStoreID: "com.google.chrome", SelfService: true, Platform: fleet.AndroidPlatform},
			},
		}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "Android web apps don't support configurations.")

	// batch-set without configuration, will work
	var batchResp batchAssociateAppStoreAppsResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch",
		batchAssociateAppStoreAppsRequest{
			DryRun: false,
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: webAppID, SelfService: true, Platform: fleet.AndroidPlatform},
				{AppStoreID: "com.google.chrome", SelfService: true, Platform: fleet.AndroidPlatform},
			},
		}, http.StatusOK, &batchResp)
	require.Len(t, batchResp.Apps, 2)
}
