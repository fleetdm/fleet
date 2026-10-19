package service

// VPP (App Store app) tests for the MDM suite.
//
// Belongs here: associating App Store apps with fleets (single and batch), installing them on hosts,
// their icons, filtering by MDM status, and refreshing app versions.
//
// Does not belong here: VPP apps installed by policy automations (integration_mdm_vpp_policy_automations_test.go),
// VPP installs in setup experience (integration_mdm_setup_experience_test.go), and the VPP install
// flow covered by integration_vpp_install_test.go.

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/apple_apps"
	"google.golang.org/api/androidmanagement/v1"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type appStoreApp interface {
	GetPlatform() string
	GetAppStoreID() string
}

func (s *integrationMDMTestSuite) TestBatchAssociateAppStoreApps() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	batchURL := "/api/latest/fleet/software/app_store_apps/batch"

	// non-existent team
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusNotFound, "team_name", "foo")

	// create a team
	tmGood, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + " good",
		Description: "desc",
		Secrets:     []*fleet.EnrollSecret{{Secret: "goodsecret"}},
	})
	require.NoError(t, err)

	// create a team
	tmEmpty, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name() + " empty",
		Description: "desc",
	})
	require.NoError(t, err)

	var batchAssociateResponse batchAssociateAppStoreAppsResponse

	// No vpp token set, but request is empty so it succeeds (clears VPP apps for the team).
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
	require.Empty(t, batchAssociateResponse.Apps)

	// No vpp token set, try association
	// FIXME
	// s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID}}}, http.StatusUnprocessableEntity, "team_name", tmGood.Name)

	// Valid token
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var vppRes uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &vppRes)

	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	// Remove all vpp associations from team with no members
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
	require.Empty(t, batchAssociateResponse.Apps)

	// host with valid serial number
	hValid, err := s.ds.NewHost(context.Background(), &fleet.Host{
		DetailUpdatedAt: time.Now(),
		LabelUpdatedAt:  time.Now(),
		PolicyUpdatedAt: time.Now(),
		HardwareSerial:  "123",
		SeenTime:        time.Now().Add(-1 * time.Minute),
		OsqueryHostID:   new(t.Name() + uuid.New().String()),
		NodeKey:         new(t.Name() + uuid.New().String()),
		Hostname:        fmt.Sprintf("%sfoo.local", t.Name()),
		Platform:        "darwin",
	})
	require.NoError(t, err)
	err = s.ds.AddHostsToTeam(context.Background(), fleet.NewAddHostsToTeamParams(&tmGood.ID, []uint{hValid.ID}))
	require.NoError(t, err)

	ctx := context.Background()

	assoc, err := s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Empty(t, assoc)

	// Remove all vpp associations from team with no members
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
	require.Empty(t, batchAssociateResponse.Apps)

	// Incorrect type check
	incorrectTypes := struct {
		Apps []struct {
			AppStoreID  int  `json:"app_store_id"`
			SelfService bool `json:"self_service"`
		} `json:"app_store_apps"`
	}{
		Apps: []struct {
			AppStoreID  int  `json:"app_store_id"`
			SelfService bool `json:"self_service"`
		}{
			{
				AppStoreID: 1,
			},
		},
	}
	badTypeReq := s.Do("POST", batchURL, incorrectTypes, http.StatusBadRequest, "team_name", tmGood.Name)
	badTypeBody := extractServerErrorText(badTypeReq.Body)
	assert.Contains(t, badTypeBody, "must be a string")

	// Associating an app we don't own
	s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: "fake-app"}}}, http.StatusUnprocessableEntity, "team_name", tmGood.Name)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Empty(t, assoc)

	// Associating an app we own
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID},
	}}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
	require.Len(t, batchAssociateResponse.Apps, 1)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 1)
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform})

	// Associating one good and one bad app
	s.Do(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
			{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID},
			{AppStoreID: "fake-app"},
		}}, http.StatusUnprocessableEntity, "team_name", tmGood.Name,
	)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 1)

	// Add a label
	clr := fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		Name:  "label1" + t.Name(),
		Query: "SELECT 1;",
	}, http.StatusOK, &clr)

	label1 := clr.Label

	clr = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		Name:  "label2" + t.Name(),
		Query: "SELECT 2;",
	}, http.StatusOK, &clr)

	label2 := clr.Label

	// Associating two apps we own
	beforeAssociation := time.Now()
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID},
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, SelfService: true, Categories: []string{"Browsers"}, LabelsIncludeAll: []string{label1.Name, label2.Name}},
			},
		}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name,
	)
	afterAssociation := time.Now()
	require.Len(t, batchAssociateResponse.Apps, 4)
	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 4)
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform})
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IOSPlatform})
	assert.Contains(t, assoc, fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IPadOSPlatform})
	meta, err := s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, s.appleVPPConfigSrvConfig.Assets[1].AdamID, fleet.MacOSPlatform, &tmGood.ID)
	require.NoError(t, err)

	var listSwTitles listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSwTitles, "team_id", fmt.Sprint(tmGood.ID))
	for _, st := range listSwTitles.SoftwareTitles {
		if st.AppStoreApp.AppStoreID == s.appleVPPConfigSrvConfig.Assets[1].AdamID {
			var getSWTitle getSoftwareTitleResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", st.ID), nil, http.StatusOK, &getSWTitle, "team_id", fmt.Sprint(tmGood.ID))
			s.ElementsMatch([]string{"🌎 Browsers"}, getSWTitle.SoftwareTitle.AppStoreApp.Categories)
			var labelNames []string
			for _, l := range getSWTitle.SoftwareTitle.AppStoreApp.LabelsIncludeAll {
				labelNames = append(labelNames, l.LabelName)
			}
			s.ElementsMatch([]string{label1.Name, label2.Name}, labelNames)
		}
	}

	actual := assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform}]
	// Only macOS version should be self-service
	assert.Equal(t, fleet.VPPAppTeam{
		AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform,
		SelfService:        true,
		InstallDuringSetup: new(false),
		VersionName:        fleet.DefaultAppStoreAppVersionName,
		AppTeamID:          meta.AppTeamID,
		AddedAt:            actual.AddedAt,
	}, actual)
	require.GreaterOrEqual(t, actual.AddedAt, beforeAssociation)
	require.LessOrEqual(t, actual.AddedAt, afterAssociation)
	require.NotZero(t, actual.AddedAt)

	// Reverse self-service associations
	// Associating two apps we own
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, SelfService: true},
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, SelfService: false},
			},
		}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name,
	)
	require.Len(t, batchAssociateResponse.Apps, 4)
	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 4)
	meta, err = s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, s.appleVPPConfigSrvConfig.Assets[0].AdamID, fleet.MacOSPlatform, &tmGood.ID)
	require.NoError(t, err)
	actual = assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform}]
	assert.Equal(t, fleet.VPPAppTeam{
		AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform,
		SelfService:        true,
		InstallDuringSetup: new(false),
		VersionName:        fleet.DefaultAppStoreAppVersionName,
		AppTeamID:          meta.AppTeamID,
		AddedAt:            actual.AddedAt,
	}, actual)
	meta, err = s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, s.appleVPPConfigSrvConfig.Assets[1].AdamID, fleet.IOSPlatform, &tmGood.ID)
	require.NoError(t, err)
	actual = assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IOSPlatform}]
	assert.Equal(t, fleet.VPPAppTeam{
		AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IOSPlatform,
		InstallDuringSetup: new(false),
		VersionName:        fleet.DefaultAppStoreAppVersionName,
		AppTeamID:          meta.AppTeamID,
		AddedAt:            actual.AddedAt,
	}, actual)
	meta, err = s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, s.appleVPPConfigSrvConfig.Assets[1].AdamID, fleet.IPadOSPlatform, &tmGood.ID)
	require.NoError(t, err)
	actual = assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IPadOSPlatform}]
	assert.Equal(t, fleet.VPPAppTeam{
		AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.IPadOSPlatform,
		InstallDuringSetup: new(false),
		VersionName:        fleet.DefaultAppStoreAppVersionName,
		AppTeamID:          meta.AppTeamID,
		AddedAt:            actual.AddedAt,
	}, actual)
	meta, err = s.ds.GetVPPAppMetadataByAdamIDPlatformTeamID(ctx, s.appleVPPConfigSrvConfig.Assets[1].AdamID, fleet.MacOSPlatform, &tmGood.ID)
	require.NoError(t, err)
	actual = assoc[fleet.VPPAppID{AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform}]
	assert.Equal(t, fleet.VPPAppTeam{
		AdamID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, Platform: fleet.MacOSPlatform,
		InstallDuringSetup: new(false),
		VersionName:        fleet.DefaultAppStoreAppVersionName,
		AppTeamID:          meta.AppTeamID,
		AddedAt:            actual.AddedAt,
	}, actual)

	// Associate an app with a team with no team members
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID}}}, http.StatusOK, &batchAssociateResponse, "team_name", tmEmpty.Name)

	// Remove all vpp associations
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
	require.Empty(t, batchAssociateResponse.Apps)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Empty(t, assoc)

	// Test InstallDuringSetup
	// Helper functions for verifying what's enabled for setup experience
	getReturnedSetupExperienceAdamIDs := func(titles []fleet.SoftwareTitleListResult) []string {
		var adamIDs []string
		for _, title := range titles {
			if (title.AppStoreApp != nil && title.AppStoreApp.InstallDuringSetup != nil && *title.AppStoreApp.InstallDuringSetup == true) ||
				(title.SoftwarePackage != nil && title.SoftwarePackage.InstallDuringSetup != nil && *title.SoftwarePackage.InstallDuringSetup == true) {
				adamIDs = append(adamIDs, title.AppStoreApp.AppStoreID)
			}
		}
		return adamIDs
	}

	checkSetupExperienceVPP := func(t *testing.T, platform string, teamID uint, expectedAdamIDs []string) {
		var respGetSetupExperience getSetupExperienceSoftwareResponse
		s.DoJSON(
			"GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{},
			http.StatusOK,
			&respGetSetupExperience,
			"platform", platform,
			"team_id", fmt.Sprint(teamID),
		)
		assert.ElementsMatch(t, getReturnedSetupExperienceAdamIDs(respGetSetupExperience.SoftwareTitles), expectedAdamIDs)
	}

	// Associate with the SetupExperience flag set to true
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				// macOS only
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, InstallDuringSetup: new(true)},
				// macOS, iOS, iPadOS
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, SelfService: true, InstallDuringSetup: new(true)},
				// iPadOS only
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[2].AdamID, SelfService: true, InstallDuringSetup: new(true)},
			},
		}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name,
	)

	checkSetupExperienceVPP(t, "macos", tmGood.ID, []string{s.appleVPPConfigSrvConfig.Assets[0].AdamID, s.appleVPPConfigSrvConfig.Assets[1].AdamID})
	checkSetupExperienceVPP(t, string(fleet.IPadOSPlatform), tmGood.ID, []string{s.appleVPPConfigSrvConfig.Assets[1].AdamID, s.appleVPPConfigSrvConfig.Assets[2].AdamID})
	checkSetupExperienceVPP(t, string(fleet.IOSPlatform), tmGood.ID, []string{s.appleVPPConfigSrvConfig.Assets[1].AdamID})

	// Associate with the SetupExperience flag set to nil = no change
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				// macOS only
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, InstallDuringSetup: nil},
				// macOS, iOS, iPadOS
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, SelfService: true, InstallDuringSetup: nil},
				// iPadOS only
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[2].AdamID, SelfService: true, InstallDuringSetup: nil},
			},
		}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name,
	)

	checkSetupExperienceVPP(t, "macos", tmGood.ID, []string{s.appleVPPConfigSrvConfig.Assets[0].AdamID, s.appleVPPConfigSrvConfig.Assets[1].AdamID})
	checkSetupExperienceVPP(t, string(fleet.IPadOSPlatform), tmGood.ID, []string{s.appleVPPConfigSrvConfig.Assets[1].AdamID, s.appleVPPConfigSrvConfig.Assets[2].AdamID})
	checkSetupExperienceVPP(t, string(fleet.IOSPlatform), tmGood.ID, []string{s.appleVPPConfigSrvConfig.Assets[1].AdamID})

	// Associate with the SetupExperience flag set to false
	s.DoJSON(
		"POST",
		batchURL,
		batchAssociateAppStoreAppsRequest{
			Apps: []fleet.VPPBatchPayload{
				// macOS only
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, InstallDuringSetup: new(false)},
				// macOS, iOS, iPadOS
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[1].AdamID, SelfService: true, InstallDuringSetup: new(false)},
				// iPadOS only
				{AppStoreID: s.appleVPPConfigSrvConfig.Assets[2].AdamID, SelfService: true, InstallDuringSetup: new(false)},
			},
		}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name,
	)

	checkSetupExperienceVPP(t, "macos", tmGood.ID, []string{})
	checkSetupExperienceVPP(t, string(fleet.IPadOSPlatform), tmGood.ID, []string{})
	checkSetupExperienceVPP(t, string(fleet.IOSPlatform), tmGood.ID, []string{})

	// ===================================================
	//                   Android apps
	// ===================================================

	// Android MDM not set up yet, should fail
	resp := s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID}, {AppStoreID: "com.foo.bar", Platform: fleet.AndroidPlatform},
	}}, http.StatusBadRequest, "team_name", tmGood.Name)

	s.Contains(extractServerErrorText(resp.Body), "Android MDM is not enabled")

	s.enableAndroidMDM(s.T())

	// Android app not found, should fail
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		return nil, &notFoundError{}
	}
	resp = s.Do("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID}, {AppStoreID: "com.app.not.found", Platform: fleet.AndroidPlatform},
	}}, http.StatusUnprocessableEntity, "team_name", tmGood.Name)

	s.Contains(extractServerErrorText(resp.Body), "Validation Failed: Couldn't add software. The application ID \"com.app.not.found\" isn't available in Play Store. Please find ID on the Play Store and try again.")

	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		return &androidmanagement.Application{}, nil
	}

	// Enroll an Android host
	secrets, err := s.ds.GetEnrollSecrets(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]
	require.NotEmpty(t, pubsubToken.Value)

	deviceID1 := createAndroidDeviceID("test-android")

	enterpriseSpecificID1 := strings.ToUpper(uuid.New().String())
	var req android_service.PubSubPushRequest
	for _, d := range []struct {
		id  string
		esi string
	}{{deviceID1, enterpriseSpecificID1}} {
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
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts, "query", enterpriseSpecificID1)

	assert.Len(t, hosts.Hosts, 1)

	// Run setup experience job
	s.runWorker()
	time.Sleep(time.Second)

	chromeAppID := "com.google.chrome"
	driveAppID := "com.google.drive"
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID},
		{AppStoreID: chromeAppID, Platform: fleet.AndroidPlatform},
		{AppStoreID: driveAppID, Platform: fleet.AndroidPlatform},
	}}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
	require.Len(t, batchAssociateResponse.Apps, 3)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 3)

	for _, a := range []fleet.VPPAppID{
		{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform},
		{AdamID: chromeAppID, Platform: fleet.AndroidPlatform},
		{AdamID: driveAppID, Platform: fleet.AndroidPlatform},
	} {
		assert.Contains(t, assoc, a)
	}

	s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked = false

	checkJobs := func(expectedAppIDs []string) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			w := &worker.SoftwareWorker{}
			var j fleet.Job

			err := sqlx.GetContext(ctx, q, &j, "SELECT * FROM jobs WHERE name = ? ORDER BY updated_at DESC LIMIT 1", w.Name())
			s.Require().NoError(err)

			var args map[string]any
			s.Require().NoError(json.Unmarshal(*j.Args, &args))
			s.Equal("bulk_set_android_apps_available_for_host", args["task"])
			s.Equal(fleet.JobStateSuccess, j.State)
			s.ElementsMatch(expectedAppIDs, args["application_ids"])

			return nil
		})
	}

	// Run worker to set apps available to device
	time.Sleep(time.Second)
	s.runWorker()
	s.True(s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)
	s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked = false
	checkJobs([]string{chromeAppID, driveAppID})

	// Remove an app
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, DisplayName: "VPPAppUpdatedName"},
		{AppStoreID: driveAppID, Platform: fleet.AndroidPlatform, DisplayName: "DriveUpdatedName"},
	}}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
	require.Len(t, batchAssociateResponse.Apps, 2)

	assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
	require.NoError(t, err)
	require.Len(t, assoc, 2)

	for _, a := range []fleet.VPPAppID{
		{AdamID: s.appleVPPConfigSrvConfig.Assets[0].AdamID, Platform: fleet.MacOSPlatform},
		{AdamID: driveAppID, Platform: fleet.AndroidPlatform},
	} {
		assert.Contains(t, assoc, a)
	}

	time.Sleep(time.Second)
	s.runWorker()
	s.True(s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)
	checkJobs([]string{driveAppID})

	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSwTitles, "team_id", fmt.Sprint(tmGood.ID))

	s.Len(listSwTitles.SoftwareTitles, 2)
	for _, sw := range listSwTitles.SoftwareTitles {
		switch sw.AppStoreApp.AppStoreID {
		case driveAppID:
			s.Equal("DriveUpdatedName", sw.DisplayName)
		case s.appleVPPConfigSrvConfig.Assets[0].AdamID:
			s.Equal("VPPAppUpdatedName", sw.DisplayName)
		}
	}

	adamIDWithAllPlatforms := s.appleVPPConfigSrvConfig.Assets[1].AdamID

	// change display names
	setDisplayNames := func(expectedAppCount int, expectedApps ...fleet.VPPBatchPayload) {
		displayNameToAppStoreID := make(map[string]string)

		// Make a unique key that uses the id + the platform
		key := func(v appStoreApp) string {
			return fmt.Sprintf(`%s_%s`, v.GetAppStoreID(), v.GetPlatform())
		}

		for _, e := range expectedApps {
			if e.Platform == "" {
				for _, p := range []fleet.InstallableDevicePlatform{fleet.IOSPlatform, fleet.IPadOSPlatform, fleet.MacOSPlatform} {
					e.Platform = p
					displayNameToAppStoreID[key(e)] = e.DisplayName
				}
			}
			displayNameToAppStoreID[key(e)] = e.DisplayName
		}

		s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: expectedApps}, http.StatusOK, &batchAssociateResponse, "team_name", tmGood.Name)
		require.Len(t, batchAssociateResponse.Apps, expectedAppCount)

		assoc, err = s.ds.GetAssignedVPPApps(ctx, &tmGood.ID)
		require.NoError(t, err)
		require.Len(t, assoc, expectedAppCount)

		for _, a := range assoc {
			assert.Contains(t, displayNameToAppStoreID, key(a))
		}

		time.Sleep(time.Second)
		s.runWorker()
		s.True(s.androidAPIClient.EnterprisesPoliciesPatchFuncInvoked)
		checkJobs([]string{driveAppID})

		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSwTitles, "team_id", fmt.Sprint(tmGood.ID))

		s.Len(listSwTitles.SoftwareTitles, expectedAppCount)
		for _, sw := range listSwTitles.SoftwareTitles {
			id, ok := displayNameToAppStoreID[key(sw.AppStoreApp)]
			s.True(ok)
			s.Equal(id, sw.DisplayName)
		}
	}

	setDisplayNames(
		4,
		fleet.VPPBatchPayload{
			AppStoreID:  driveAppID,
			Platform:    fleet.AndroidPlatform,
			DisplayName: "AndroidUpdated2",
		},
		fleet.VPPBatchPayload{
			AppStoreID:  adamIDWithAllPlatforms,
			SelfService: true,
			DisplayName: "AppleUpdated2",
		},
	)

	setDisplayNames(
		3,
		fleet.VPPBatchPayload{
			AppStoreID:  driveAppID,
			Platform:    fleet.AndroidPlatform,
			DisplayName: "AndroidUpdated2",
		},
		fleet.VPPBatchPayload{
			AppStoreID:  adamIDWithAllPlatforms,
			Platform:    fleet.MacOSPlatform,
			SelfService: true,
			DisplayName: "AppleUpdated2",
		},
		fleet.VPPBatchPayload{
			AppStoreID:  adamIDWithAllPlatforms,
			Platform:    fleet.IOSPlatform,
			SelfService: true,
			DisplayName: "IOSAPP",
		},
	)

	setDisplayNames(
		4,
		fleet.VPPBatchPayload{
			AppStoreID:  driveAppID,
			Platform:    fleet.AndroidPlatform,
			DisplayName: "",
		},
		fleet.VPPBatchPayload{
			AppStoreID:  adamIDWithAllPlatforms,
			SelfService: true,
			DisplayName: "",
		},
	)
}

func (s *integrationMDMTestSuite) TestVPPApps() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// Invalid token
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL+"?invalidToken", t)
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte("foobar"), http.StatusUnprocessableEntity, "Invalid token. Please provide a valid content token from Apple Business.", nil)
	// Attempt to renew an invalid (nonexistent) token, should fail
	s.uploadDataViaFormWithVerb("/api/latest/fleet/vpp_tokens/999/renew", "PATCH", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte("foobar"))), http.StatusUnprocessableEntity, "Invalid token. Please provide a valid content token from Apple Business.", nil)

	// Simulate a server error from the Apple API
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL+"?serverError", t)
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte("foobar"), http.StatusInternalServerError, "Apple VPP endpoint returned error: Internal server error (error number: 9603)", nil)

	// Valid token
	orgName := "Fleet Device Management Inc."
	location := "Fleet Location One"
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	s.lastActivityMatches(fleet.ActivityEnabledVPP{}.ActivityName(), "", 0)

	// Renew valid token
	expTime = expTime.Add(24 * time.Hour).UTC().Round(time.Second)
	expDate = expTime.Format(fleet.VPPTimeFormat)
	tokenJSON = fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	s.uploadDataViaFormWithVerb(fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/renew", validToken.Token.ID), "PATCH", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)
	// Valid token bytes, but non-existent token ID. Should get not found
	s.uploadDataViaFormWithVerb("/api/latest/fleet/vpp_tokens/999/renew", "PATCH", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusNotFound, "VPPToken was not found in the datastore", nil)

	// Get the token
	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)

	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, app.AdamID, app.Platform)
		})

		return titleID
	}

	t.Run("vpp apps with labels", func(t *testing.T) {
		// Create a team
		var newTeamResp teamResponse
		s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Team Labels" + t.Name())}, http.StatusOK, &newTeamResp)
		team := newTeamResp.Team

		// Add an MDM macOS host to the team
		host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
		orbitKey := setOrbitEnrollment(t, host, s.ds)
		host.OrbitNodeKey = &orbitKey
		s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{host.ID}, TeamID: &team.ID}, http.StatusOK)

		// Associate team to the VPP token.
		var resPatchVPP patchVPPTokensTeamsResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

		var createLabelResp fleet.CreateLabelResponse
		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "label1" + t.Name()}, http.StatusOK, &createLabelResp)
		l1 := createLabelResp.Label
		require.NotNil(t, l1)

		s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "label2" + t.Name()}, http.StatusOK, &createLabelResp)
		l2 := createLabelResp.Label
		require.NotNil(t, l2)

		includeAnyApp := fleet.VPPApp{
			AdamID:           "1",
			Platform:         fleet.MacOSPlatform,
			Name:             "App 1",
			BundleIdentifier: "a-1",
			IconURL:          "https://example.com/images/1/512x512.png",
			LatestVersion:    "1.0.0",
		}

		// Attempt to add an app with both types of labels. Should fail
		var addAppResp addAppStoreAppResponse
		addAppReq := &addAppStoreAppRequest{
			TeamID:           &team.ID,
			AppStoreID:       includeAnyApp.AdamID,
			SelfService:      true,
			LabelsIncludeAny: []string{l1.Name},
			LabelsExcludeAny: []string{l2.Name},
		}
		res := s.Do("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), `Only one of "labels_include_all", "labels_include_any" or "labels_exclude_any" can be included`)

		// Now add it for real
		addAppReq.LabelsExcludeAny = []string{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusOK, &addAppResp)
		titleID := getSoftwareTitleIDFromApp(&includeAnyApp)
		require.Equal(t, titleID, addAppResp.TitleID, "addAppResp should contain the correct software title ID")
		activityData := `{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": true, "labels_include_any": [{"id": %d, "name": %q}]}`
		s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(activityData, team.Name, team.Name,
				includeAnyApp.Name, titleID, includeAnyApp.AdamID, team.ID, team.ID, includeAnyApp.Platform, l1.ID, l1.Name), 0)

		var getSWTitle getSoftwareTitleResponse
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &getSWTitle, "team_id", fmt.Sprint(team.ID))
		require.NotNil(t, getSWTitle.SoftwareTitle.AppStoreApp)
		require.Equal(t, getSWTitle.SoftwareTitle.AppStoreApp.AdamID, includeAnyApp.AdamID)
		require.Empty(t, getSWTitle.SoftwareTitle.AppStoreApp.LabelsExcludeAny)
		require.Empty(t, getSWTitle.SoftwareTitle.AppStoreApp.LabelsIncludeAll)
		require.Equal(t, []fleet.SoftwareScopeLabel{{LabelName: l1.Name, LabelID: l1.ID}}, getSWTitle.SoftwareTitle.AppStoreApp.LabelsIncludeAny)

		// Add an app with exclude_any labels
		excludeAnyApp := fleet.VPPApp{
			AdamID:           "2",
			Platform:         fleet.MacOSPlatform,
			Name:             "App 2",
			BundleIdentifier: "b-2",
			IconURL:          "https://example.com/images/1/512x512.png",
			LatestVersion:    "1.0.0",
		}

		addAppReq = &addAppStoreAppRequest{
			TeamID:           &team.ID,
			AppStoreID:       excludeAnyApp.AdamID,
			SelfService:      true,
			LabelsExcludeAny: []string{l2.Name},
		}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", addAppReq, http.StatusOK, &addAppResp)
		titleID = getSoftwareTitleIDFromApp(&excludeAnyApp)
		activityData = `{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": true, "labels_exclude_any": [{"id": %d, "name": %q}]}`
		s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(activityData, team.Name, team.Name,
				excludeAnyApp.Name, titleID, excludeAnyApp.AdamID, team.ID, team.ID, excludeAnyApp.Platform, l2.ID, l2.Name), 0)

		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &getSWTitle, "team_id", fmt.Sprint(team.ID))
		require.NotNil(t, getSWTitle.SoftwareTitle.AppStoreApp)
		require.Equal(t, getSWTitle.SoftwareTitle.AppStoreApp.AdamID, excludeAnyApp.AdamID)
		require.Empty(t, getSWTitle.SoftwareTitle.AppStoreApp.LabelsIncludeAny)
		require.Empty(t, getSWTitle.SoftwareTitle.AppStoreApp.LabelsIncludeAll)
		require.Equal(t, []fleet.SoftwareScopeLabel{{LabelName: l2.Name, LabelID: l2.ID}}, getSWTitle.SoftwareTitle.AppStoreApp.LabelsExcludeAny)
		require.True(t, getSWTitle.SoftwareTitle.AppStoreApp.SelfService)

		// Add a non-VPP software
		payloadRubyTm1 := &fleet.UploadSoftwareInstallerPayload{
			InstallScript: "install",
			Filename:      "ruby.deb",
			SelfService:   false,
			TeamID:        &team.ID,
		}
		s.uploadSoftwareInstaller(t, payloadRubyTm1, http.StatusOK, "")

		resp := listSoftwareTitlesResponse{}
		s.DoJSON(
			"GET", "/api/latest/fleet/software/titles",
			listSoftwareTitlesRequest{},
			http.StatusOK, &resp,
			"query", "ruby",
			"team_id", fmt.Sprintf("%d", team.ID),
		)

		require.Len(t, resp.SoftwareTitles, 1)
		nonVPPTitleID := resp.SoftwareTitles[0].ID

		updateAppReq := &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: new(false)}

		// Attempt to update the non-VPP software using the VPP path. Should fail.
		s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", nonVPPTitleID), updateAppReq, http.StatusNotFound)

		// Attempt to update a non-existent app. Should fail.
		s.Do("PATCH", "/api/latest/fleet/software/titles/9999/app_store_app", updateAppReq, http.StatusNotFound)

		// Attempt to update with both types of labels. Should fail.
		updateAppReq.LabelsIncludeAny = []string{l1.Name}
		updateAppReq.LabelsExcludeAny = []string{l1.Name}
		res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID), updateAppReq, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), `Only one of "labels_include_all", "labels_include_any" or "labels_exclude_any" can be included.`)

		// Attempt to update with a non-existent label. Should fail.
		updateAppReq.LabelsExcludeAny = []string{}
		updateAppReq.LabelsIncludeAny = []string{"404_notfound"}
		res = s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID), updateAppReq, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), `Couldn't update. Label "404_notfound" doesn't exist. Please remove the label from the software.`)

		// Update App2. Unset self service and update the labels
		updateAppReq.LabelsIncludeAny = []string{l2.Name}
		var updateAppResp updateAppStoreAppResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID), updateAppReq, http.StatusOK, &updateAppResp)

		require.NotNil(t, updateAppResp.AppStoreApp)
		require.Equal(t, updateAppResp.AppStoreApp.AppStoreID, excludeAnyApp.AdamID)
		require.Equal(t, []fleet.SoftwareScopeLabel{{LabelName: l2.Name, LabelID: l2.ID}}, updateAppResp.AppStoreApp.LabelsIncludeAny)
		require.Empty(t, updateAppResp.AppStoreApp.LabelsExcludeAny)
		require.Empty(t, updateAppResp.AppStoreApp.LabelsIncludeAll)
		require.False(t, updateAppResp.AppStoreApp.SelfService)
		require.Equal(t, fleet.MacOSPlatform, updateAppResp.AppStoreApp.Platform)

		activityData = `{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "app_store_id": "%s", "software_icon_url": "https://example.com/images/2/512x512.png", "team_id": %d, "fleet_id": %d, "software_title_id": %d, "platform": "%s", "self_service": false, "labels_include_any": [{"id": %d, "name": %q}], "software_display_name": ""}`
		s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(activityData, team.Name, team.Name,
				excludeAnyApp.Name, excludeAnyApp.AdamID, team.ID, team.ID, titleID, excludeAnyApp.Platform, l2.ID, l2.Name), 0)

		// double check that our updates worked
		getSWTitle = getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID), nil, http.StatusOK, &getSWTitle, "team_id", fmt.Sprint(team.ID))
		require.NotNil(t, getSWTitle.SoftwareTitle.AppStoreApp)
		require.Equal(t, getSWTitle.SoftwareTitle.AppStoreApp.AdamID, excludeAnyApp.AdamID)
		require.Equal(t, []fleet.SoftwareScopeLabel{{LabelName: l2.Name, LabelID: l2.ID}}, getSWTitle.SoftwareTitle.AppStoreApp.LabelsIncludeAny)
		require.Empty(t, getSWTitle.SoftwareTitle.AppStoreApp.LabelsExcludeAny)
		require.Empty(t, getSWTitle.SoftwareTitle.AppStoreApp.LabelsIncludeAll)
		require.False(t, getSWTitle.SoftwareTitle.AppStoreApp.SelfService)

		// Attempt an install on the host. This should fail because the host doesn't have the label
		// l2.
		res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), &installSoftwareRequest{}, http.StatusBadRequest)
		require.Contains(t, extractServerErrorText(res.Body), "Couldn't install. This host isn't a member of the labels defined for this software title.")

		// Add l2 to the host. Attempt install again, should succeed
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", l2.ID), &fleet.ModifyLabelPayload{Hosts: []string{host.HardwareSerial}}, http.StatusOK, &createLabelResp)

		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmDevice.SerialNumber)

		s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, titleID), &installSoftwareRequest{}, http.StatusAccepted)

		// Attempt to delete a non-existent app. Should fail.
		s.Do("DELETE", "/api/latest/fleet/software/titles/9999/available_for_install", nil, http.StatusNotFound, "team_id", fmt.Sprintf("%d", team.ID))

		// delete the VPP app
		s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", titleID), nil, http.StatusNoContent, "team_id", fmt.Sprintf("%d", team.ID))
		activityData = `{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "app_store_id": "%s", "software_icon_url": "https://example.com/images/2/512x512.png", "team_id": %d, "fleet_id": %d, "platform": "%s", "labels_include_any": [{"id": %d, "name": %q}]}`
		s.lastActivityMatches(fleet.ActivityDeletedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(activityData, team.Name, team.Name,
				excludeAnyApp.Name, excludeAnyApp.AdamID, team.ID, team.ID, excludeAnyApp.Platform, l2.ID, l2.Name), 0)

		// iOS and iPadOS

		iOSApp := fleet.VPPApp{
			AdamID:           "2",
			Platform:         fleet.IOSPlatform,
			Name:             "App 2",
			BundleIdentifier: "b-2",
			IconURL:          "https://example.com/images/2/512x512.png",
			LatestVersion:    "2.0.0",
		}

		iPadOSApp := fleet.VPPApp{
			AdamID:           "2",
			Platform:         fleet.IPadOSPlatform,
			Name:             "App 2",
			BundleIdentifier: "b-2",
			IconURL:          "https://example.com/images/2/512x512.png",
			LatestVersion:    "2.0.0",
		}

		testCases := []struct {
			desc      string
			app       fleet.VPPApp
			addAppReq *addAppStoreAppRequest
		}{
			{
				desc: "ios device",
				app:  iOSApp,
				addAppReq: &addAppStoreAppRequest{
					TeamID:           &team.ID,
					AppStoreID:       iOSApp.AdamID,
					LabelsIncludeAny: []string{l2.Name},
					Platform:         iOSApp.Platform,
				},
			},

			{
				desc: "ipados device",
				app:  iPadOSApp,
				addAppReq: &addAppStoreAppRequest{
					TeamID:           &team.ID,
					AppStoreID:       iPadOSApp.AdamID,
					LabelsIncludeAny: []string{l2.Name},
					Platform:         iPadOSApp.Platform,
				},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				mobileHost, mobileMDMDevice := s.createAppleMobileHostThenEnrollMDM(string(tc.app.Platform))
				s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{mobileHost.ID}, TeamID: &team.ID}, http.StatusOK)

				s.Do("POST", "/api/latest/fleet/software/app_store_apps", tc.addAppReq, http.StatusOK)
				titleID = getSoftwareTitleIDFromApp(&tc.app)
				s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mobileMDMDevice.SerialNumber)

				// Should fail because label is include_any and the host doesn't have it
				res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mobileHost.ID, titleID), &installSoftwareRequest{}, http.StatusBadRequest)
				require.Contains(t, extractServerErrorText(res.Body), "Couldn't install. This host isn't a member of the labels defined for this software title.")

				// Create a new label
				s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "label3" + t.Name(), Hosts: []string{mobileMDMDevice.SerialNumber}}, http.StatusOK, &createLabelResp)
				l3 := createLabelResp.Label
				require.NotNil(t, l3)

				// Should fail because label is exclude_any and the host does have it
				updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, LabelsExcludeAny: []string{l3.Name}}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID), updateAppReq, http.StatusOK, &updateAppResp)
				res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mobileHost.ID, titleID), &installSoftwareRequest{}, http.StatusBadRequest)
				require.Contains(t, extractServerErrorText(res.Body), "Couldn't install. This host isn't a member of the labels defined for this software title.")

				// Should succeed because the label is include_any and the host does have it
				updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, LabelsIncludeAny: []string{l3.Name}}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID), updateAppReq, http.StatusOK, &updateAppResp)
				s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mobileHost.ID, titleID), &installSoftwareRequest{}, http.StatusAccepted)

				// Create a new host
				mobileHost, mobileMDMDevice = s.createAppleMobileHostThenEnrollMDM(string(tc.app.Platform))
				s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{mobileHost.ID}, TeamID: &team.ID}, http.StatusOK)
				s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mobileMDMDevice.SerialNumber)

				// Should succeed because the label is exclude_any and the iOS host doesn't have it
				updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, LabelsExcludeAny: []string{l3.Name}}
				s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", titleID), updateAppReq, http.StatusOK, &updateAppResp)
				s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mobileHost.ID, titleID), &installSoftwareRequest{}, http.StatusAccepted)
			})
		}
	})

	// Create a team with unique name to avoid conflicts with other tests
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("VPPApps Test Team " + t.Name())}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// A PATCH endpoint omitting mdm.volume_purchasing_program should not remove the VPP token association.
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"agent_options": {
			"config": {
				"options": {
					"pack_delimiter": "/",
					"logger_tls_period": 10,
					"distributed_plugin": "tls",
					"disable_distributed": false,
					"logger_tls_endpoint": "/api/osquery/log",
					"distributed_interval": 10,
					"distributed_tls_max_attempts": 3
				}
			}
		}
  }`), http.StatusOK, &acResp)

	// Check that the VPP token is still associated to the team.
	resp = getVPPTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)
	require.Len(t, resp.Tokens[0].Teams, 1)
	require.Equal(t, team.ID, resp.Tokens[0].Teams[0].ID)
	require.Equal(t, team.Name, resp.Tokens[0].Teams[0].Name)

	// Reset the token's teams by omitting the token from app config
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": { "volume_purchasing_program": [] }
  }`), http.StatusOK, &acResp)

	resp = getVPPTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)
	require.Empty(t, resp.Tokens[0].Teams)

	// Add the team back using the PATCH /api/latest/fleet/config endpoint now (what GitOps uses).
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": [ "%s" ]} ] }
  }`, location, team.Name)), http.StatusOK, &acResp)

	// Check again that the VPP token is associated to the team.
	resp = getVPPTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)
	require.Len(t, resp.Tokens[0].Teams, 1)
	require.Equal(t, team.ID, resp.Tokens[0].Teams[0].ID)
	require.Equal(t, team.Name, resp.Tokens[0].Teams[0].Name)

	// Get list of VPP apps from "Apple"
	// We're passing team 1 here, but we haven't added any app store apps to that team, so we get
	// back all available apps in our VPP location.
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)

	macOSApp := fleet.VPPApp{
		AdamID:           "1",
		Platform:         fleet.MacOSPlatform,
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1/512x512.png",
		LatestVersion:    "1.0.0",
	}
	iPadOSApp := fleet.VPPApp{
		AdamID:           "2",
		Platform:         fleet.IPadOSPlatform,
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2/512x512.png",
		LatestVersion:    "2.0.0",
	}
	iOSApp := fleet.VPPApp{
		AdamID:           "2",
		Platform:         fleet.IOSPlatform,
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2/512x512.png",
		LatestVersion:    "2.0.0",
	}
	expectedApps := []*fleet.VPPApp{
		&macOSApp,
		&iPadOSApp,
		&iOSApp,
		{
			AdamID:           "2",
			Platform:         fleet.MacOSPlatform,
			Name:             "App 2",
			BundleIdentifier: "b-2",
			IconURL:          "https://example.com/images/2/512x512.png",
			LatestVersion:    "2.0.1",
		},
		{
			AdamID:           "3",
			Platform:         fleet.IPadOSPlatform,
			Name:             "App 3",
			BundleIdentifier: "c-3",
			IconURL:          "https://example.com/images/3/512x512.png",
			LatestVersion:    "3.0.0",
		},
	}
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	// Insert/deletion flow for macOS app
	// Add an app store app to team 1
	addedApp := expectedApps[0]
	var addAppResp addAppStoreAppResponse
	// Add an app store app to non-existent team
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: new(uint(9999)), AppStoreID: addedApp.AdamID}, http.StatusNotFound, &addAppResp)

	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, SelfService: true, AutomaticInstall: true}, http.StatusOK, &addAppResp)

	s.lastActivityOfTypeMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": true}`, team.Name, team.Name,
			addedApp.Name, getSoftwareTitleIDFromApp(addedApp), addedApp.AdamID, team.ID, team.ID, addedApp.Platform), 0)

	// Now we should be filtering out the app we added to team 1
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, expectedApps[1:], appResp.AppStoreApps)

	// list the software titles for that team, to get the title id of the VPP app
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	require.True(t, *listSw.SoftwareTitles[0].AppStoreApp.SelfService)
	macOSTitleID := listSw.SoftwareTitles[0].ID
	require.Len(t, listSw.SoftwareTitles[0].AppStoreApp.AutomaticInstallPolicies, 1)

	// check that we created an activity for the policy creation
	wantAct := fleet.ActivityTypeCreatedPolicy{
		ID:   listSw.SoftwareTitles[0].AppStoreApp.AutomaticInstallPolicies[0].ID,
		Name: listSw.SoftwareTitles[0].AppStoreApp.AutomaticInstallPolicies[0].Name,
	}
	s.lastActivityMatches(wantAct.ActivityName(), string(jsonMustMarshal(t, wantAct)), 0)

	// listing with the self-service filter also returns it
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "self_service", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	require.Equal(t, macOSTitleID, listSw.SoftwareTitles[0].ID)

	// delete the automatic install policy (so we can delete the app next)
	var deletePolicyResp fleet.DeleteTeamPoliciesResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/delete", team.ID), &fleet.DeleteGlobalPoliciesRequest{IDs: []uint{listSw.SoftwareTitles[0].AppStoreApp.AutomaticInstallPolicies[0].ID}}, http.StatusOK, &deletePolicyResp)

	// delete the app store app for team 1
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNoContent,
		"team_id", fmt.Sprint(team.ID))
	s.lastActivityMatches(fleet.ActivityDeletedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "app_store_id": "%s", "software_icon_url": "https://example.com/images/1/512x512.png", "team_id": %d, "fleet_id": %d, "platform": "%s"}`, team.Name, team.Name,
			addedApp.Name, addedApp.AdamID, team.ID, team.ID, addedApp.Platform), 0)

	// deleting it again fails, not found
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNotFound,
		"team_id", fmt.Sprint(team.ID))

	// get the list of available apps, returns all apps now
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	// Insert/deletion flow for iPadOS app
	addedApp = expectedApps[1]
	addAppResp = addAppStoreAppResponse{}
	// Accepts self-service for iPadOS
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, Platform: addedApp.Platform, SelfService: true},
		http.StatusOK, &addAppResp)
	// No auto-install for iPadOS
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, Platform: addedApp.Platform, AutomaticInstall: true},
		http.StatusBadRequest, &addAppResp)
	// Delete the self-service app so it can be added again as non-self-service
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", getSoftwareTitleIDFromApp(addedApp)), nil, http.StatusNoContent,
		"fleet_id", fmt.Sprint(team.ID))
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, Platform: addedApp.Platform},
		http.StatusOK, &addAppResp)
	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": false}`, team.Name, team.Name,
			addedApp.Name, getSoftwareTitleIDFromApp(addedApp), addedApp.AdamID, team.ID, team.ID, addedApp.Platform), 0)

	// Now we should be filtering out the app we added to team 1
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, append(expectedApps[2:], expectedApps[0]), appResp.AppStoreApps)

	// list the software titles for that team, to get the title id of the VPP app
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
		"available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	macOSTitleID = listSw.SoftwareTitles[0].ID
	assert.Equal(t, "ipados_apps", listSw.SoftwareTitles[0].Source)

	// filtering by self-service returns nothing
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
		"self_service", "true")
	require.Empty(t, listSw.SoftwareTitles)

	_, err := s.ds.CreateOrUpdateSoftwareTitleIcon(context.Background(), &fleet.UploadSoftwareTitleIconPayload{
		TitleID:   macOSTitleID,
		Filename:  "icon.png",
		TeamID:    team.ID,
		StorageID: "icon_storage_id",
	})
	require.NoError(t, err)

	// delete the app store app for team 1
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNoContent,
		"team_id",
		fmt.Sprint(team.ID))
	s.lastActivityMatches(fleet.ActivityDeletedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "app_store_id": "%s", "software_icon_url": "/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", "team_id": %d, "fleet_id": %d, "platform": "%s"}`, team.Name, team.Name,
			addedApp.Name, addedApp.AdamID, macOSTitleID, team.ID, team.ID, team.ID, addedApp.Platform), 0)

	var count int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &count, `SELECT COUNT(1) FROM software_title_icons WHERE team_id = ? and software_title_id = ?`, team.ID, macOSTitleID)
	})
	require.Equal(t, 0, count)

	// deleting it again fails, not found
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNotFound,
		"team_id",
		fmt.Sprint(team.ID))

	// get the list of available apps, returns all apps now
	appResp = getAppStoreAppsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	// Installation flow

	// Create a couple of hosts
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)
	selfServiceHost, selfServiceDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, selfServiceHost, s.ds)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	checkInstallFleetdCommandSent(t, selfServiceDevice, true)
	selfServiceToken := "selfservicetoken"
	updateDeviceTokenForHost(t, s.ds, selfServiceHost.ID, selfServiceToken)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, selfServiceDevice.SerialNumber)
	iOSHost, iOSMdmClient := s.createAppleMobileHostThenEnrollMDM("ios")
	iPadOSHost, iPadOSMdmClient := s.createAppleMobileHostThenEnrollMDM("ipados")
	// ensure a valid alternate device token for self-service status access checking later
	updateDeviceTokenForHost(t, s.ds, mdmHost.ID, "foobar")
	s.awaitRunAppleMDMWorkerSchedule()

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial,
		iOSHost.HardwareSerial, iPadOSHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID, orbitHost.ID, iOSHost.ID, iPadOSHost.ID, selfServiceHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add all apps to the team
	addedApp = expectedApps[0]
	errApp := expectedApps[3]
	appSelfService := expectedApps[0]
	// Add app 1 as self-service
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: appSelfService.AdamID, Platform: appSelfService.Platform, SelfService: true},
		http.StatusOK, &addAppResp)
	s.lastActivityMatches(
		fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": true}`, team.Name, team.Name,
			appSelfService.Name, getSoftwareTitleIDFromApp(appSelfService), appSelfService.AdamID, team.ID, team.ID, appSelfService.Platform),
		0,
	)
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
		"available_for_install", "true")
	// Add remaining as non-self-service
	for _, app := range expectedApps[1:] {
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app.AdamID, Platform: app.Platform},
			http.StatusOK, &addAppResp)
		s.lastActivityMatches(
			fleet.ActivityAddedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(`{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": false}`, team.Name, team.Name,
				app.Name, getSoftwareTitleIDFromApp(app), app.AdamID, team.ID, team.ID, app.Platform),
			0,
		)
		listSw = listSoftwareTitlesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
			"available_for_install", "true")
	}

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, len(expectedApps))
	macOSTitleID = 99999
	errTitleID := uint(99999)
	iOSTitleID := uint(99999)
	iPadOSTitleID := uint(99999)
	for _, sw := range listSw.SoftwareTitles {
		assert.NotNil(t, sw.AppStoreApp)
		switch {
		case sw.Name == addedApp.Name && sw.Source == "apps":
			macOSTitleID = sw.ID
		case sw.Name == errApp.Name && sw.Source == "apps":
			errTitleID = sw.ID
		case sw.Name == iOSApp.Name && sw.Source == "ios_apps":
			iOSTitleID = sw.ID
		case sw.Name == iPadOSApp.Name && sw.Source == "ipados_apps":
			iPadOSTitleID = sw.ID
		}
	}

	// attempt to install a VPP app on the non-MDM enrolled host
	r := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", orbitHost.ID, macOSTitleID), &installSoftwareRequest{}, http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), "Error: Couldn't install. To install App Store app, turn on MDM for this host.")

	// Disable all teams token
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", validToken.Token.ID), patchVPPTokensTeamsRequest{}, http.StatusOK, &resPatchVPP)

	// Spoof an expired VPP token and attempt to install VPP app
	tokenJSONBad := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, "2030-06-24T15:50:50+0000", "badtoken", "Evil Fleet")
	s.appleVPPConfigSrvConfig.Location = "Spooky Haunted House"
	var vppRes uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSONBad))), http.StatusAccepted, "", &vppRes)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID, 99999}}, http.StatusUnprocessableEntity, &resPatchVPP)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Disable the token
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", vppRes.Token.ID), patchVPPTokensTeamsRequest{}, http.StatusOK, &resPatchVPP)

	// Enable all teams token
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", validToken.Token.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	// Attempt to install non-existent app
	r = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, 99999), &installSoftwareRequest{},
		http.StatusBadRequest)
	require.Contains(t, extractServerErrorText(r.Body), "Couldn't install software. Software title is not available for install. Please add software package or App Store app to install.")

	// Add the apps again, self-service only for the apps with the macOS app's ID
	for _, app := range expectedApps {
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app.AdamID, Platform: app.Platform, SelfService: app.AdamID == macOSApp.AdamID},
			http.StatusOK, &addAppResp)
	}

	var failedCmdUUID string
	errorOnInstallApplicationCommand := func(errorCode int) {
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		assert.NotNil(t, cmd)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
			case "InstallApplication":
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				failedCmdUUID = cmd.CommandUUID
				t.Logf("Failed command UUID: %s", failedCmdUUID)
				cmd, err = mdmDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{{ErrorCode: errorCode}})
				require.NoError(t, err)
			}
		}
	}

	// Trigger install to the host
	installResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, errTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d", vppRes.Token.ID), &deleteVPPTokenRequest{}, http.StatusNoContent)

	// Check if the host is listed as pending
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "pending", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(errTitleID))
	require.Len(t, listResp.Hosts, 1)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(errTitleID))
	require.Equal(t, 1, countResp.Count)

	// Simulate failed installation on the host, exhaust retries (MaxSoftwareInstallAttempts = 3)
	// First error triggers retry 1, second triggers retry 2, third triggers retry 3,
	// fourth exhausts retries and marks as failed.
	for range fleet.MaxSoftwareInstallAttempts + 1 {
		errorOnInstallApplicationCommand(1234)
	}

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "failed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(errTitleID))
	require.Len(t, listResp.Hosts, 1)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "failed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(errTitleID))
	require.Equal(t, 1, countResp.Count)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"version_name": "Default version", "host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null, "host_platform": "%s", "from_setup_experience": false, "from_auto_update": false, "from_configuration_resend": false}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			errApp.Name,
			errApp.AdamID,
			failedCmdUUID,
			fleet.SoftwareInstallFailed,
			mdmHost.Platform,
		),
		0,
	)

	// We store the old failedCmdUUID to restore after this 9610, since it's used further down.
	storedFailedCmdUUID := failedCmdUUID

	// Trigger a 9610 failure to see retry happening
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.runWorker()

	// Simulate failed installation on the host
	errorOnInstallApplicationCommand(9610)

	// Verify the retry_count was increased to 1 for the failedCmdUUID
	var retryCount int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &retryCount, `SELECT retry_count FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ?`, mdmHost.ID, macOSApp.AdamID)
	})
	require.Equal(t, 1, retryCount)

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "failed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(macOSTitleID))
	require.Empty(t, listResp.Hosts)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "failed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	oldFailedCmdUUID := failedCmdUUID
	errorOnInstallApplicationCommand(9610)
	require.NotEqual(t, oldFailedCmdUUID, failedCmdUUID)

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &retryCount, `SELECT retry_count FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ?`, mdmHost.ID, macOSApp.AdamID)
	})
	require.Equal(t, 2, retryCount)

	oldFailedCmdUUID = failedCmdUUID
	errorOnInstallApplicationCommand(9610)
	require.NotEqual(t, failedCmdUUID, oldFailedCmdUUID)

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &retryCount, `SELECT retry_count FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ?`, mdmHost.ID, macOSApp.AdamID)
	})
	require.Equal(t, 3, retryCount)

	oldFailedCmdUUID = failedCmdUUID
	// Final time where we mark the app as failed
	errorOnInstallApplicationCommand(9610)
	require.NotEqual(t, failedCmdUUID, oldFailedCmdUUID)

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &retryCount, `SELECT retry_count FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ?`, mdmHost.ID, macOSApp.AdamID)
	})
	require.Equal(t, 3, retryCount)

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "failed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 1)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "failed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"version_name": "Default version", "host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null, "host_platform": "%s", "from_setup_experience": false, "from_auto_update": false, "from_configuration_resend": false}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			macOSApp.Name,
			macOSApp.AdamID,
			failedCmdUUID,
			fleet.SoftwareInstallFailed,
			mdmHost.Platform,
		),
		0,
	)

	// Restore failedCmdUUID to the original failed command UUID for further processing.
	failedCmdUUID = storedFailedCmdUUID

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.runWorker()

	// Simulate successful installation on the host
	var installCmdUUID string
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstallApplication":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			installCmdUUID = cmd.CommandUUID
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	s.runWorker()

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, []fleet.Software{{Name: addedApp.Name, BundleIdentifier: addedApp.BundleIdentifier, Version: addedApp.LatestVersion, Installed: true}})
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected command type", cmd.Command.RequestType)
		}
	}

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var types []string
		err := sqlx.SelectContext(context.Background(), q, &types, "SELECT activity_type FROM upcoming_activities WHERE host_id = ?", mdmHost.ID)
		require.NoError(t, err)
		require.Empty(t, types)
		return nil
	})
	require.Len(t, listResp.Hosts, 1)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"version_name": "Default version", "host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null, "host_platform": "%s", "from_setup_experience": false, "from_auto_update": false, "from_configuration_resend": false}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			addedApp.Name,
			addedApp.AdamID,
			installCmdUUID,
			fleet.SoftwareInstalled,
			mdmHost.Platform,
		),
		0,
	)

	// Check list host software

	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "include_available_for_install", "true")
	gotSW := getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1, got2 := gotSW[0], gotSW[1]
	require.Equal(t, "App 1", got1.Name)
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.IconUrl, new(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	require.NotNil(t, got1.Status)
	require.Equal(t, fleet.SoftwareInstalled, *got1.Status)
	require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, installCmdUUID)
	require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)
	require.Equal(t, "App 2", got2.Name)
	require.NotNil(t, got2.Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *got2.Status)
	require.NotNil(t, got2.AppStoreApp)
	require.Equal(t, got2.AppStoreApp.AppStoreID, errApp.AdamID)
	require.Equal(t, got2.IconUrl, new(errApp.IconURL))
	require.Empty(t, got2.AppStoreApp.Name)
	require.Equal(t, got2.AppStoreApp.Version, errApp.LatestVersion)
	require.Equal(t, got2.AppStoreApp.LastInstall.CommandUUID, failedCmdUUID)
	require.NotNil(t, got2.AppStoreApp.LastInstall.InstalledAt)

	// Check with a query
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "query", "App 1", "include_available_for_install", "true")
	require.Len(t, getHostSw.Software, 1) // App 1 only
	got1 = getHostSw.Software[0]
	require.Equal(t, "App 1", got1.Name)
	require.NotNil(t, got1.AppStoreApp)
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.IconUrl, new(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name)
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	require.NotNil(t, got1.Status)
	require.Equal(t, fleet.SoftwareInstalled, *got1.Status)
	require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, installCmdUUID)
	require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)

	// Filter the self-service apps for that host
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "self_service", "true")
	require.Len(t, getHostSw.Software, 1)
	require.Equal(t, appSelfService.Name, got1.Name)

	// Return installed app with software detail query
	distributedReq := submitDistributedQueryResultsRequestShim{
		NodeKey: *mdmHost.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_macos": json.RawMessage(fmt.Sprintf(
				`[{"name": "%s", "version": "%s", "type": "Application (macOS)",
					"bundle_identifier": "%s", "source": "apps", "last_opened_at": "",
					"installed_path": "/Applications/a.app"}]`, addedApp.Name, addedApp.LatestVersion, addedApp.BundleIdentifier,
			)),
		},
		Statuses: map[string]any{
			hostDistributedQueryPrefix + "software_macos": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	// Remove the installed app by not returning it
	distributedReq = submitDistributedQueryResultsRequestShim{
		NodeKey: *mdmHost.NodeKey,
		Results: map[string]json.RawMessage{
			hostDetailQueryPrefix + "software_macos": json.RawMessage(`[]`),
		},
		Statuses: map[string]any{
			hostDistributedQueryPrefix + "software_macos": 0,
		},
		Messages: map[string]string{},
		Stats:    map[string]*fleet.Stats{},
	}
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSON("POST", "/api/osquery/distributed/write", distributedReq, http.StatusOK, &distributedResp)

	// Check list host software
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "include_available_for_install", "true")
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1 = gotSW[0]
	require.Equal(t, "App 1", got1.Name)
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.IconUrl, new(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	assert.Nil(t, got1.Status)
	assert.Nil(t, got1.AppStoreApp.LastInstall)

	// Install on iOS and iPadOS devices
	installs := map[string]struct {
		installHost    *fleet.Host
		titleID        uint
		mdmClient      *mdmtest.TestAppleMDMClient
		app            fleet.VPPApp
		extraAvailable int
		hostCount      int
		deviceToken    string
	}{
		"iOS app install": {
			installHost: iOSHost,
			titleID:     iOSTitleID,
			mdmClient:   iOSMdmClient,
			app:         iOSApp,
			hostCount:   1,
		},
		"iPadOS app install": {
			installHost:    iPadOSHost,
			titleID:        iPadOSTitleID,
			mdmClient:      iPadOSMdmClient,
			app:            iPadOSApp,
			extraAvailable: 1,
			hostCount:      1,
		},
		"macOS app install": {
			installHost: selfServiceHost,
			titleID:     macOSTitleID,
			mdmClient:   selfServiceDevice,
			app:         macOSApp,
			hostCount:   2,
			deviceToken: selfServiceToken,
		},
	}

	for name, install := range installs {
		t.Run(name, func(t *testing.T) {
			installHost := install.installHost
			titleID := install.titleID
			mdmClient := install.mdmClient
			app := install.app

			// Self-service install
			if install.deviceToken != "" {
				var ssInstallResp submitSelfServiceSoftwareInstallResponse
				s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", install.deviceToken, install.titleID),
					&fleetSelfServiceSoftwareInstallRequest{}, http.StatusAccepted, &ssInstallResp)
			} else {
				installResp = installSoftwareResponse{}
				s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", installHost.ID, titleID),
					&installSoftwareRequest{}, http.StatusAccepted, &installResp)
			}
			countResp = countHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
				fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(titleID))
			require.Equal(t, 1, countResp.Count)

			// send an idle request to grab the command uuid
			cmd, err = mdmClient.Idle()
			require.NoError(t, err)
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			installCmdUUID = cmd.CommandUUID

			if install.deviceToken != "" {
				var cmdResResp getMDMCommandResultsResponse
				res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/software/commands/%s/results", install.deviceToken, installCmdUUID), nil, http.StatusOK)
				err = json.NewDecoder(res.Body).Decode(&cmdResResp)
				require.NoError(t, err)
				require.Empty(t, cmdResResp.Results)
			}

			// Get pending activity
			var hostActivitiesResp listHostUpcomingActivitiesResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", installHost.ID),
				nil, http.StatusOK, &hostActivitiesResp)
			activitiesToString := func(activities []*fleet.UpcomingActivity) []string {
				var res []string
				for _, activity := range activities {
					res = append(res, fmt.Sprintf("%+v", activity))
				}
				return res
			}
			require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
			require.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityInstalledAppStoreApp{}.ActivityName())
			require.EqualValues(t, 1, hostActivitiesResp.Count)
			require.JSONEq(
				t,
				fmt.Sprintf(
					`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "software_display_name": null, "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "host_platform": "%s"}`,
					installHost.ID,
					installHost.DisplayName(),
					app.Name,
					app.AdamID,
					installCmdUUID,
					fleet.SoftwareInstallPending,
					install.deviceToken != "",
					installHost.Platform,
				),
				string(*hostActivitiesResp.Activities[0].Details),
			)

			// Simulate successful installation on the host
			cmd, err = mdmClient.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)

			// Process InstalledApplicationList command for install verification
			s.runWorker()

			cmd, err = mdmClient.Idle()
			require.NoError(t, err)
			for cmd != nil {
				var fullCmd micromdm.CommandPayload
				switch cmd.Command.RequestType {
				case "InstalledApplicationList":
					require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
					cmd, err = mdmClient.AcknowledgeInstalledApplicationList(mdmClient.UUID, cmd.CommandUUID, []fleet.Software{{Name: app.Name, BundleIdentifier: app.BundleIdentifier, Version: app.LatestVersion, Installed: true}})
					require.NoError(t, err)
				default:
					require.Fail(t, "unexpected command type", cmd.Command.RequestType)
				}
			}

			if install.deviceToken != "" {
				var cmdResResp getMDMCommandResultsResponse
				res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/software/commands/%s/results", install.deviceToken, installCmdUUID), nil, http.StatusOK)
				err = json.NewDecoder(res.Body).Decode(&cmdResResp)
				require.NoError(t, err)
				require.Len(t, cmdResResp.Results, 1)
				require.Equal(t, "Acknowledged", cmdResResp.Results[0].Status)

				s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/%s/software/commands/foobar/results", install.deviceToken), nil, http.StatusNotFound)
				s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/device/foobar/software/commands/%s/results", installCmdUUID), nil, http.StatusNotFound)
			}

			listResp = listHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
				fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(titleID))
			require.Len(t, listResp.Hosts, install.hostCount)
			countResp = countHostsResponse{}
			s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
				fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(titleID))
			require.Equal(t, install.hostCount, countResp.Count)

			s.lastActivityMatches(
				fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
				fmt.Sprintf(
					`{"version_name": "Default version", "host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "policy_id": null, "policy_name": null, "host_platform": "%s", "from_setup_experience": false, "from_auto_update": false, "from_configuration_resend": false}`,
					installHost.ID,
					installHost.DisplayName(),
					app.Name,
					app.AdamID,
					installCmdUUID,
					fleet.SoftwareInstalled,
					install.deviceToken != "",
					installHost.Platform,
				),
				0,
			)

			// Check list host software
			getHostSw = getHostSoftwareResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", installHost.ID), nil, http.StatusOK, &getHostSw, "include_available_for_install", "true")
			require.Len(t, getHostSw.Software, install.hostCount+install.extraAvailable)
			var foundInstalledApp bool
			for index := range getHostSw.Software {
				got1 = getHostSw.Software[index]
				if got1.Status != nil {
					require.Equal(t, got1.Name, app.Name)
					require.NotNil(t, got1.AppStoreApp)
					require.Equal(t, got1.AppStoreApp.AppStoreID, app.AdamID)
					require.Equal(t, got1.IconUrl, new(app.IconURL))
					require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
					require.Equal(t, got1.AppStoreApp.Version, app.LatestVersion)
					require.Equal(t, fleet.SoftwareInstalled, *got1.Status)
					require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, installCmdUUID)
					require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)
					foundInstalledApp = true
				}
			}
			assert.True(t, foundInstalledApp)
		})
	}

	// Attempt (and fail) to self-service install iPad and iOS titles
	var ssInstallResp submitSelfServiceSoftwareInstallResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", selfServiceToken, iPadOSTitleID), &fleetSelfServiceSoftwareInstallRequest{},
		http.StatusBadRequest, &ssInstallResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", selfServiceToken, iOSTitleID), &fleetSelfServiceSoftwareInstallRequest{},
		http.StatusBadRequest, &ssInstallResp)

	// Delete VPP token and check that it's not appearing anymore

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d", validToken.Token.ID), &deleteVPPTokenResponse{}, http.StatusNoContent)
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
}

func (s *integrationMDMTestSuite) TestNoTeamVPPAppIcons() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// Valid token
	orgName := "Fleet Device Management Inc."
	location := "Fleet Location One"
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	// Get the token
	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)
	require.Len(t, resp.Tokens, 1)
	require.Equal(t, orgName, resp.Tokens[0].OrgName)
	require.Equal(t, location, resp.Tokens[0].Location)
	require.Equal(t, expTime, resp.Tokens[0].RenewDate)

	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{fleet.PolicyNoTeamID}}, http.StatusOK, &resPatchVPP)

	macOSApp := fleet.VPPApp{
		AdamID:           "1",
		Platform:         fleet.MacOSPlatform,
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1/512x512.png",
		LatestVersion:    "1.0.0",
	}

	// Insert/deletion flow for macOS app
	// Add an app store app to team no team
	addedApp := macOSApp
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: new(fleet.PolicyNoTeamID), AppStoreID: addedApp.AdamID, SelfService: true}, http.StatusOK, &addAppResp)

	// list the software titles, to get the title id of the VPP app
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(fleet.PolicyNoTeamID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	require.True(t, *listSw.SoftwareTitles[0].AppStoreApp.SelfService)
	macOSTitleID := listSw.SoftwareTitles[0].ID

	_, err := s.ds.CreateOrUpdateSoftwareTitleIcon(context.Background(), &fleet.UploadSoftwareTitleIconPayload{
		TitleID:   macOSTitleID,
		Filename:  "icon.png",
		TeamID:    fleet.PolicyNoTeamID,
		StorageID: "icon_storage_id",
	})
	require.NoError(t, err)

	var count int
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &count, `SELECT COUNT(1) FROM software_title_icons WHERE team_id = ? and software_title_id = ?`, fleet.PolicyNoTeamID, macOSTitleID)
	})
	require.Equal(t, 1, count)

	// delete the app store app for team no team
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install", macOSTitleID), nil, http.StatusNoContent,
		"team_id",
		fmt.Sprint(fleet.PolicyNoTeamID))
	s.lastActivityMatches(fleet.ActivityDeletedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"version_name": "Default version", "team_name": null, "fleet_name": null, "software_title": "%s", "app_store_id": "%s", "software_icon_url": "/api/latest/fleet/software/titles/%d/icon?fleet_id=%d", "team_id": %d, "fleet_id": %d, "platform": "%s"}`,
			addedApp.Name, addedApp.AdamID, macOSTitleID, fleet.PolicyNoTeamID, fleet.PolicyNoTeamID, fleet.PolicyNoTeamID, addedApp.Platform), 0)

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &count, `SELECT COUNT(1) FROM software_title_icons WHERE team_id = ? and software_title_id = ?`, fleet.PolicyNoTeamID, macOSTitleID)
	})
	require.Equal(t, 0, count)

	// Delete VPP token and check that it's not appearing anymore

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d", validToken.Token.ID), &deleteVPPTokenResponse{}, http.StatusNoContent)
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
}

func (s *integrationMDMTestSuite) TestVPPAppsMDMFiltering() {
	t := s.T()

	ctx := context.Background()

	// Create hosts
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	_, _ = mdmHost, mdmClient

	test.CreateInsertGlobalVPPToken(t, s.ds)

	// Create team and add hosts to team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Team 1")}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{orbitHost.ID, mdmHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add an app so that we don't get a not found error
	_, err := s.ds.InsertVPPAppWithTeam(ctx, &fleet.VPPApp{
		Name:             "App " + t.Name(),
		BundleIdentifier: "bid_" + t.Name(),
		AdamID:           "adam_test_01",
		Platform:         fleet.MacOSPlatform,
	}, &team.ID, nil)
	require.NoError(t, err)

	resp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", orbitHost.ID), getHostSoftwareRequest{}, http.StatusOK, &resp)
	assert.Empty(t, resp.Software)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), getHostSoftwareRequest{}, http.StatusOK, &resp, "include_available_for_install", "true")
	assert.Len(t, resp.Software, 1)
}

func (s *integrationMDMTestSuite) TestRefreshVPPAppVersions() {
	t := s.T()
	ctx := context.Background()

	// Reset the VPP proxy data to what it was before this test
	s.registerResetVPPProxyData(t)

	// Set up 3 apps - macOS, iOS, and iPadOS (using new VPP proxy format)
	s.appleVPPProxySrvData = map[string]string{
		"1": `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"osx": {"bundleId": "a-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["mac"]}}`,
		// Add a space to the version and make sure we are trimming it (we've seen that in Meta Horizon app `"versionDisplay":" 353.0"`)
		"2": `{"id": "2", "attributes": {"name": "App 2", "platformAttributes": {"ios": {"bundleId": "d-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": " 2.0.0 "}}}, "deviceFamilies": ["iphone"]}}`,
		"3": `{"id": "3", "attributes": {"name": "App 3", "platformAttributes": {"ios": {"bundleId": "b-3", "artwork": {"url": "https://example.com/images/3/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "3.0.0"}}}, "deviceFamilies": ["ipad"]}}`,
	}

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Team 1" + t.Name())}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	stubbedConfig := apple_apps.StubbedConfig() // authentication is tested elsewhere

	// Set up VPP token
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	// Get the token
	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// No VPP apps added yet, so this is a no-op
	err := vpp.RefreshVersions(ctx, s.ds, stubbedConfig)
	require.NoError(t, err)

	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	app1 := appResp.AppStoreApps[0]
	app2 := appResp.AppStoreApps[1]
	app3 := appResp.AppStoreApps[2]

	// Add 3 apps: 2 will have version changes, the 3rd will not
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		Platform:   app1.Platform,
		AppStoreID: app1.AdamID,
	}, http.StatusOK, &addAppResp)

	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		Platform:   app2.Platform,
		AppStoreID: app2.AdamID,
	}, http.StatusOK, &addAppResp)

	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		Platform:   app3.Platform,
		AppStoreID: app3.AdamID,
	}, http.StatusOK, &addAppResp)

	// Check versions before refresh
	var listSWTitlesResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitlesResp, "team_id", fmt.Sprintf("%d", team.ID), "query", app1.Name)
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "1.0.0", listSWTitlesResp.SoftwareTitles[0].AppStoreApp.Version)

	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitlesResp, "team_id", fmt.Sprintf("%d", team.ID), "query", app2.Name)
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "2.0.0", listSWTitlesResp.SoftwareTitles[0].AppStoreApp.Version)

	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitlesResp, "team_id", fmt.Sprintf("%d", team.ID), "query", app3.Name)
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "3.0.0", listSWTitlesResp.SoftwareTitles[0].AppStoreApp.Version)

	// "update" the versions
	s.appleVPPProxySrvData["1"] = `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"osx": {"bundleId": "a-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "9.9.9"}}}, "deviceFamilies": ["mac"]}}`
	s.appleVPPProxySrvData["2"] = `{"id": "2", "attributes": {"name": "App 2", "platformAttributes": {"osx": {"bundleId": "b-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "10.10.10"}}, "ios": {"bundleId": "b-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "10.10.10"}}}, "deviceFamilies": ["mac", "iphone", "ipad"]}}`

	err = vpp.RefreshVersions(ctx, s.ds, stubbedConfig)
	require.NoError(t, err)

	// 1 and 2 should be updated
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitlesResp, "team_id", fmt.Sprintf("%d", team.ID), "query", app1.Name)
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "9.9.9", listSWTitlesResp.SoftwareTitles[0].AppStoreApp.Version)

	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitlesResp, "team_id", fmt.Sprintf("%d", team.ID), "query", app2.Name)
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "10.10.10", listSWTitlesResp.SoftwareTitles[0].AppStoreApp.Version)

	// No change
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSWTitlesResp, "team_id", fmt.Sprintf("%d", team.ID), "query", app3.Name)
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "3.0.0", listSWTitlesResp.SoftwareTitles[0].AppStoreApp.Version)

	// Refresh again. There are no version changes this time, so this is a no-op.
	err = vpp.RefreshVersions(ctx, s.ds, stubbedConfig)
	require.NoError(t, err)
}

func (s *integrationMDMTestSuite) TestRefreshVPPAppVersionsForAllPlatforms() {
	t := s.T()

	// Reset the VPP proxy data to what it was before this test
	s.registerResetVPPProxyData(t)

	// Set up app with adamID 1 with iOS, iPadOS, macOS (e.g. WhatsApp).
	// Set up app with adamID 2 with iOS and iPadOS.
	// Set up app with adamID 3 with iOS.
	s.appleVPPProxySrvData = map[string]string{
		"1": `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"osx": {"bundleId": "a-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.2.3"}}, "ios": {"bundleId": "a-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.3.0"}}}, "deviceFamilies": ["mac", "iphone", "ipad"]}}`,
		"2": `{"id": "2", "attributes": {"name": "App 2", "platformAttributes": {"ios": {"bundleId": "d-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "2.0.0"}}}, "deviceFamilies": ["iphone", "ipad"]}}`,
		"3": `{"id": "3", "attributes": {"name": "App 3", "platformAttributes": {"ios": {"bundleId": "b-3", "artwork": {"url": "https://example.com/images/3/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "3.0.0"}}}, "deviceFamilies": ["iphone"]}}`,
	}

	stubbedConfig := apple_apps.StubbedConfig() // authentication is tested elsewhere

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Team 1" + t.Name())}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Set up VPP token
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	// Get the token
	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	require.Len(t, appResp.AppStoreApps, 6)

	var (
		allApps    = appResp.AppStoreApps
		app1MacOS  *fleet.VPPApp
		app1IOS    *fleet.VPPApp
		app1IPadOS *fleet.VPPApp
		app2IOS    *fleet.VPPApp
		app2IPadOS *fleet.VPPApp
		app3IOS    *fleet.VPPApp
	)
	for _, app := range appResp.AppStoreApps {
		switch {
		case app.AdamID == "1" && app.Platform == "darwin":
			app1MacOS = app
		case app.AdamID == "1" && app.Platform == "ios":
			app1IOS = app
		case app.AdamID == "1" && app.Platform == "ipados":
			app1IPadOS = app
		case app.AdamID == "2" && app.Platform == "ios":
			app2IOS = app
		case app.AdamID == "2" && app.Platform == "ipados":
			app2IPadOS = app
		case app.AdamID == "3" && app.Platform == "ios":
			app3IOS = app
		}
	}
	require.NotNil(t, app1MacOS)
	require.NotNil(t, app1IOS)
	require.NotNil(t, app1IPadOS)
	require.NotNil(t, app2IOS)
	require.NotNil(t, app2IPadOS)
	require.NotNil(t, app3IOS)

	// Add all apps (three adam IDs with different platforms) to the team.
	for _, app := range allApps {
		var addAppResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID:     &team.ID,
			Platform:   app.Platform,
			AppStoreID: app.AdamID,
		}, http.StatusOK, &addAppResp)
		switch {
		case app.AdamID == "1" && app.Platform == "darwin":
			app1MacOS.TitleID = addAppResp.TitleID
		case app.AdamID == "1" && app.Platform == "ios":
			app1IOS.TitleID = addAppResp.TitleID
		case app.AdamID == "1" && app.Platform == "ipados":
			app1IPadOS.TitleID = addAppResp.TitleID
		case app.AdamID == "2" && app.Platform == "ios":
			app2IOS.TitleID = addAppResp.TitleID
		case app.AdamID == "2" && app.Platform == "ipados":
			app2IPadOS.TitleID = addAppResp.TitleID
		case app.AdamID == "3" && app.Platform == "ios":
			app3IOS.TitleID = addAppResp.TitleID
		}
	}

	// Check versions before refresh
	for titleID, expectedVersion := range map[uint]string{
		app1MacOS.TitleID:  "1.2.3",
		app1IOS.TitleID:    "1.3.0",
		app1IPadOS.TitleID: "1.3.0",
		app2IOS.TitleID:    "2.0.0",
		app2IPadOS.TitleID: "2.0.0",
		app3IOS.TitleID:    "3.0.0",
	} {
		titleResponse := getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResponse, "team_id", fmt.Sprint(team.ID))
		require.NotNil(t, titleResponse.SoftwareTitle)
		require.Equal(t, expectedVersion, titleResponse.SoftwareTitle.AppStoreApp.LatestVersion)
	}

	// "Update" the versions for Adam ID "1" and "2".
	s.appleVPPProxySrvData["1"] = `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"osx": {"bundleId": "a-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "9.9.9"}}, "ios": {"bundleId": "a-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "9.9.8"}}}, "deviceFamilies": ["mac", "iphone", "ipad"]}}`
	s.appleVPPProxySrvData["2"] = `{"id": "2", "attributes": {"name": "App 2", "platformAttributes": {"ios": {"bundleId": "b-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "10.10.10"}}}, "deviceFamilies": ["iphone", "ipad"]}}`

	err := vpp.RefreshVersions(t.Context(), s.ds, stubbedConfig)
	require.NoError(t, err)

	// Check versions after refresh
	for titleID, expectedVersion := range map[uint]string{
		app1MacOS.TitleID:  "9.9.9",
		app1IOS.TitleID:    "9.9.8",
		app1IPadOS.TitleID: "9.9.8",
		app2IOS.TitleID:    "10.10.10",
		app2IPadOS.TitleID: "10.10.10",
		app3IOS.TitleID:    "3.0.0",
	} {
		titleResponse := getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResponse, "team_id", fmt.Sprint(team.ID))
		require.NotNil(t, titleResponse.SoftwareTitle)
		require.Equal(t, expectedVersion, titleResponse.SoftwareTitle.AppStoreApp.LatestVersion)
	}

	// "Update" the version for Adam ID "3".
	s.appleVPPProxySrvData["3"] = `{"id": "3", "attributes": {"name": "App 3", "platformAttributes": {"ios": {"bundleId": "b-3", "artwork": {"url": "https://example.com/images/3/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "11.11.11"}}}, "deviceFamilies": ["iphone"]}}`

	err = vpp.RefreshVersions(t.Context(), s.ds, stubbedConfig)
	require.NoError(t, err)

	// Check versions after refresh
	for titleID, expectedVersion := range map[uint]string{
		app1MacOS.TitleID:  "9.9.9",
		app1IOS.TitleID:    "9.9.8",
		app1IPadOS.TitleID: "9.9.8",
		app2IOS.TitleID:    "10.10.10",
		app2IPadOS.TitleID: "10.10.10",
		app3IOS.TitleID:    "11.11.11",
	} {
		titleResponse := getSoftwareTitleResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", titleID), nil, http.StatusOK, &titleResponse, "team_id", fmt.Sprint(team.ID))
		require.NotNil(t, titleResponse.SoftwareTitle)
		require.Equal(t, expectedVersion, titleResponse.SoftwareTitle.AppStoreApp.LatestVersion)
	}

	// Refresh again. There are no version changes this time, so this is a no-op.
	err = vpp.RefreshVersions(t.Context(), s.ds, stubbedConfig)
	require.NoError(t, err)
}

func (s *integrationMDMTestSuite) TestVPPInstallsUseAppToken() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)

	origLocation := s.appleVPPConfigSrvConfig.Location
	t.Cleanup(func() { s.appleVPPConfigSrvConfig.Location = origLocation })
	uploadToken := func(location string) *fleet.VPPTokenDB {
		s.appleVPPConfigSrvConfig.Location = location
		tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"Acme"}`,
			time.Now().Add(200*time.Hour).UTC().Format(fleet.VPPTimeFormat), uuid.NewString())
		var resp uploadVPPTokenResponse
		s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
			[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &resp)
		tok, err := s.ds.GetVPPToken(ctx, resp.Token.ID)
		require.NoError(t, err)
		return tok
	}
	usToken, jpToken := uploadToken("Acme US"), uploadToken("Acme Japan")

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new(t.Name())}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, host, s.ds)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, host.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{host.ID}, TeamID: &team.ID}, http.StatusOK)

	// Only the JP token is assigned when app 1 is added, so it binds to it. Once the US token is also assigned, it's the
	// fleet's first-added token, so app 2 binds to it.
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", jpToken.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK)
	addApp := func(adamID string) uint {
		var resp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: adamID, Platform: fleet.MacOSPlatform}, http.StatusOK, &resp)
		return resp.TitleID
	}
	jpTitleID := addApp("1")
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", usToken.ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK)
	usTitleID := addApp("2")

	var listResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", nil, http.StatusOK, &listResp)
	for _, tok := range listResp.Tokens {
		require.Equal(t, []fleet.TeamTuple{{ID: team.ID, Name: team.Name}}, tok.Teams)
	}

	s.appleVPPConfigSrvConfig.AssociateAuthorizations = nil
	s.appleVPPConfigSrvConfig.DisassociateAuthorizations = nil
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, jpTitleID), &installSoftwareRequest{}, http.StatusAccepted)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, usTitleID), &installSoftwareRequest{}, http.StatusAccepted)
	require.Equal(t, []string{"Bearer " + jpToken.Token, "Bearer " + usToken.Token}, s.appleVPPConfigSrvConfig.AssociateAuthorizations)

	var installTokenIDs []uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.SelectContext(ctx, q, &installTokenIDs, `
			SELECT vaua.vpp_token_id FROM upcoming_activities ua
			JOIN vpp_app_upcoming_activities vaua ON vaua.upcoming_activity_id = ua.id
			WHERE ua.host_id = ? ORDER BY ua.id`, host.ID)
	})
	require.Equal(t, []uint{jpToken.ID, usToken.ID}, installTokenIDs)

	// Canceling the pending US install releases the seat from the US token
	var activitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID), nil, http.StatusOK, &activitiesResp)
	require.Len(t, activitiesResp.Activities, 2)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", host.ID, activitiesResp.Activities[1].UUID), nil, http.StatusNoContent)
	require.Equal(t, []string{"Bearer " + usToken.Token}, s.appleVPPConfigSrvConfig.DisassociateAuthorizations)

	// An expired token only blocks installs of its own apps
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE vpp_tokens SET renew_at = ? WHERE id = ?`, time.Now().Add(-time.Hour), jpToken.ID)
		return err
	})
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, jpTitleID), &installSoftwareRequest{}, http.StatusUnprocessableEntity)
	require.Contains(t, extractServerErrorText(res.Body), "VPP token expired")
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, usTitleID), &installSoftwareRequest{}, http.StatusAccepted)
}
