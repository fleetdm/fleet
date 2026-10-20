package service

// iOS and iPadOS setup experience tests for the MDM suite.
//
// Belongs here: setup experience for ADE-enrolled iPhones and iPads (VPP and in-house apps, release)
// and for BYOD iOS devices.
//
// Does not belong here: macOS setup experience (integration_mdm_setup_experience_test.go,
// integration_mdm_setup_experience_vpp_test.go) and iOS/iPadOS enrollment outside setup experience.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestSetupExperienceIOSAndIPadOS() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()
	abmOrgName := "fleet_ade_ios_ipados_team_test"

	s.enableABM(abmOrgName)

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
			"mdm": {
			       "apple_business_manager": [{
			         "organization_name": %q,
			         "macos_team": %q,
			         "ios_team": %q,
			         "ipados_team": %q
			       }]
			}
		}`, abmOrgName, team.Name, team.Name, team.Name)), http.StatusOK, &acResp)

	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	var getVPPTokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &getVPPTokenResp)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// app 1 macOS only
	macOSApp1 := &fleet.VPPApp{
		AdamID:           "1",
		Platform:         fleet.MacOSPlatform,
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1",
		LatestVersion:    "1.0.0",
	}
	// App 2 supports macOS, iOS, iPadOS
	macOSApp2 := &fleet.VPPApp{
		AdamID:           "2",
		Platform:         fleet.MacOSPlatform,
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	iOSApp2 := &fleet.VPPApp{
		AdamID:           "2",
		Platform:         fleet.IOSPlatform,
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	iPadOSApp2 := &fleet.VPPApp{
		AdamID:           "2",
		Platform:         fleet.IPadOSPlatform,
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}

	// App 3 is iPadOS only
	iPadOSApp3 := &fleet.VPPApp{
		AdamID:           "3",
		Platform:         fleet.IPadOSPlatform,
		Name:             "App 3",
		BundleIdentifier: "c-3",
		IconURL:          "https://example.com/images/3",
		LatestVersion:    "3.0.0",
	}

	expectedApps := []*fleet.VPPApp{macOSApp1, macOSApp2, iOSApp2, iPadOSApp2, iPadOSApp3}

	var addAppResp addAppStoreAppResponse
	// Add apps
	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, app.AdamID, app.Platform)
		})
		require.NoError(t, err)

		return titleID
	}

	titleIDsByApp := make(map[*fleet.VPPApp]uint)
	for _, app := range expectedApps {
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app.AdamID, Platform: app.Platform},
			http.StatusOK, &addAppResp)
		titleIDsByApp[app] = getSoftwareTitleIDFromApp(app)
	}

	putSetupExperienceSoftwareForPlatform := func(t *testing.T, platform string, teamID uint, titleIDs []uint) {
		var swInstallResp putSetupExperienceSoftwareResponse
		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
			Platform: platform,
			TeamID:   teamID,
			TitleIDs: titleIDs,
		}, http.StatusOK, &swInstallResp)
	}

	// Set the 2 apps for macOS
	putSetupExperienceSoftwareForPlatform(t, "macos", team.ID, []uint{titleIDsByApp[macOSApp1], titleIDsByApp[macOSApp2]})

	// Add an app for iOS
	putSetupExperienceSoftwareForPlatform(t, "ios", team.ID, []uint{titleIDsByApp[iOSApp2]})

	// Add apps for iPadOS
	putSetupExperienceSoftwareForPlatform(t, "ipados", team.ID, []uint{titleIDsByApp[iPadOSApp2], titleIDsByApp[iPadOSApp3]})

	// Add a profile
	teamProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{teamProfile}}, http.StatusNoContent, "team_id", fmt.Sprint(team.ID))

	devices := []godep.Device{
		{
			Model:        "iPad Pro 12.9\" (Wi-Fi Only - 3rd Gen)",
			OS:           "iPadOS",
			DeviceFamily: "iPad",
			OpType:       "added",
			SerialNumber: "ipad-123456",
		},
		{
			Model:        "iPhone 16 Pro",
			OS:           "iOS",
			DeviceFamily: "iPhone",
			OpType:       "added",
			SerialNumber: "iphone-123456",
		},
	}

	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, devices[0].SerialNumber, devices[1].SerialNumber)

	vppAppIDsByDeviceFamily := map[string][]*fleet.VPPApp{
		"iPhone": {iOSApp2},
		"iPad":   {iPadOSApp2, iPadOSApp3},
	}

	for _, enableReleaseManually := range []bool{false, true} {
		for _, enrollmentProfileFromDEPUsingPost := range []bool{false, true} {
			for _, mdmMigrationDeadline := range []bool{false, true} {
				for _, device := range devices {
					t.Run(fmt.Sprintf("%sSetupExperience;enableReleaseManually=%t;EnrollmentProfileFromDEPUsingPost=%t;WithMDMMigrationDeadline=%t", device.DeviceFamily, enableReleaseManually, enrollmentProfileFromDEPUsingPost, mdmMigrationDeadline), func(t *testing.T) {
						if mdmMigrationDeadline {
							deadline := time.Now().Add(24 * time.Hour)
							device.MDMMigrationDeadline = &deadline
						} else {
							device.MDMMigrationDeadline = nil
						}
						s.runDEPEnrollReleaseMobileDeviceWithVPPTest(t, device, DEPEnrollMobileTestOpts{
							ABMOrg:                            abmOrgName,
							EnableReleaseManually:             enableReleaseManually,
							TeamID:                            &team.ID,
							CustomProfileIdent:                "N1",
							EnrollmentProfileFromDEPUsingPost: enrollmentProfileFromDEPUsingPost,
							VppAppsToInstall:                  vppAppIDsByDeviceFamily[device.DeviceFamily],
						})
					})
				}
			}
		}
	}
}

// TestSetupExperienceIOSInHouseApp runs the end-to-end iOS setup experience
// flow with a mixed payload: an in-house app (.ipa) and a VPP app on the same
// enrolling iPhone. Both must install while the device is held in Setup
// Assistant, drive the release, and be recorded with setup-experience
// attribution.
func (s *integrationMDMTestSuite) TestSetupExperienceIOSInHouseApp() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()
	abmOrgName := "fleet_ade_ios_in_house_test"

	s.enableABM(abmOrgName)

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team in-house se"})
	require.NoError(t, err)

	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
			"mdm": {
			       "apple_business_manager": [{
			         "organization_name": %q,
			         "macos_team": %q,
			         "ios_team": %q,
			         "ipados_team": %q
			       }]
			}
		}`, abmOrgName, team.Name, team.Name, team.Name)), http.StatusOK, &acResp)

	// VPP token for the App Store half of the payload
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)
	var getVPPTokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &getVPPTokenResp)
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	iOSApp := &fleet.VPPApp{
		AdamID: "2", Platform: fleet.IOSPlatform,
		Name:             "App 2",
		BundleIdentifier: "b-2",
		LatestVersion:    "2.0.0",
	}
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: iOSApp.AdamID, Platform: iOSApp.Platform},
		http.StatusOK, &addAppResp)
	var vppTitleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &vppTitleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, iOSApp.AdamID, iOSApp.Platform)
	})

	// upload the .ipa (creates the iOS and iPadOS titles) and select the iOS title
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "ipa_test.ipa", TeamID: &team.ID}, http.StatusOK, "")
	var ipaTitleID uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &ipaTitleID, `SELECT title_id FROM in_house_apps WHERE global_or_team_id = ? AND platform = 'ios'`, team.ID)
	})

	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "ios",
		TeamID:   team.ID,
		TitleIDs: []uint{vppTitleID, ipaTitleID},
	}, http.StatusOK, &swInstallResp)

	// custom profile, expected by the enroll/release helper
	teamProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{teamProfile}}, http.StatusNoContent, "team_id", fmt.Sprint(team.ID))

	device := godep.Device{
		Model:        "iPhone 16 Pro",
		OS:           "iOS",
		DeviceFamily: "iPhone",
		OpType:       "added",
		SerialNumber: "iphone-inhouse-1",
	}
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, device.SerialNumber)

	// wrapped in t.Run so the helper's cleanups run before the suite teardown
	t.Run("iPhoneInHouseSetupExperience", func(t *testing.T) {
		s.runDEPEnrollReleaseMobileDeviceWithVPPTest(t, device, DEPEnrollMobileTestOpts{
			ABMOrg:             abmOrgName,
			TeamID:             &team.ID,
			CustomProfileIdent: "N1",
			VppAppsToInstall:   []*fleet.VPPApp{iOSApp},
			InHouseAppsToInstall: []fleet.Software{
				{BundleIdentifier: "com.ipa-test.ipa-test", Name: "ipa_test", Version: "1.0"},
			},
		})

		// both setup experience items reached success, and the in-house row carries
		// its app pointer and MDM command UUID
		listHostsRes := listHostsResponse{}
		s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
		require.Len(t, listHostsRes.Hosts, 1)
		hostUUID := listHostsRes.Hosts[0].UUID
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, hostUUID, team.ID)
		require.NoError(t, err)
		require.Len(t, results, 2)
		var inHouseSeen, vppSeen bool
		for _, res := range results {
			require.Equal(t, fleet.SetupExperienceStatusSuccess, res.Status)
			switch {
			case res.IsForInHouseApp():
				inHouseSeen = true
				require.NotNil(t, res.NanoCommandUUID)
			case res.IsForVPPApp():
				vppSeen = true
			}
		}
		require.True(t, inHouseSeen)
		require.True(t, vppSeen)

		// the verified in-house install is recorded with setup-experience attribution
		var activitiesResp listActivitiesResponse
		s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &activitiesResp, "per_page", "50", "order_key", "id", "order_direction", "desc")
		var inHouseActivitySeen bool
		for _, act := range activitiesResp.Activities {
			if act.Type != (fleet.ActivityTypeInstalledSoftware{}).ActivityName() || act.Details == nil {
				continue
			}
			var details fleet.ActivityTypeInstalledSoftware
			require.NoError(t, json.Unmarshal(*act.Details, &details))
			if details.SoftwareTitle != "ipa_test" {
				continue
			}
			inHouseActivitySeen = true
			require.True(t, details.FromSetupExperience)
			require.Equal(t, string(fleet.SoftwareInstalled), details.Status)
		}
		require.True(t, inHouseActivitySeen, "expected an installed_software activity for the in-house app")
	})
}

type DEPEnrollMobileTestOpts struct {
	ABMOrg                            string
	EnableReleaseManually             bool
	TeamID                            *uint
	CustomProfileIdent                string
	EnrollmentProfileFromDEPUsingPost bool
	VppAppsToInstall                  []*fleet.VPPApp
	// InHouseAppsToInstall lists in-house apps (.ipa) expected to install during
	// setup, as the device would report them in InstalledApplicationList.
	InHouseAppsToInstall []fleet.Software
}

func (s *integrationMDMTestSuite) runDEPEnrollReleaseMobileDeviceWithVPPTest(t *testing.T, device godep.Device, opts DEPEnrollMobileTestOpts) {
	ctx := context.Background()

	// set the enable release device manually option
	payload := map[string]any{
		"enable_release_device_manually": opts.EnableReleaseManually,
		"manual_agent_install":           false,
	}
	if opts.TeamID != nil {
		payload["team_id"] = *opts.TeamID
	}

	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)
	t.Cleanup(func() {
		// Get back to the default state.
		payload["enable_release_device_manually"] = false
		s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)
	})

	// query all hosts - none yet
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Empty(t, listHostsRes.Hosts)

	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	s.mockDEPResponse(opts.ABMOrg, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			assert.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			assert.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}})
			assert.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}, Cursor: "foo"})
			assert.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			var prof profileAssignmentReq
			assert.NoError(t, json.Unmarshal(b, &prof))

			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			assert.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// trigger a profile sync
	s.runDEPSchedule()

	listHostsRes = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, listHostsRes.Hosts[0].HardwareSerial, device.SerialNumber)
	enrolledHost := listHostsRes.Hosts[0].Host

	t.Cleanup(func() {
		// delete the enrolled host
		err := s.ds.DeleteHost(ctx, enrolledHost.ID)
		require.NoError(t, err)
		// clear out any left behind jobs
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `DELETE FROM jobs`)
			return err
		})
	})

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	clientOpts := make([]mdmtest.TestMDMAppleClientOption, 0)
	if opts.EnrollmentProfileFromDEPUsingPost {
		clientOpts = append(clientOpts, mdmtest.WithEnrollmentProfileFromDEPUsingPost())
	}
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken, clientOpts...)
	switch device.DeviceFamily {
	case "iPhone":
		mdmDevice.Model = "iPhone14,6"
	case "iPad":
		mdmDevice.Model = "iPad8,7"
	default:
		// Only expecting mobile devices for this test
		t.Fatalf("unexpected device family: %s", device.DeviceFamily)
	}
	mdmDevice.SerialNumber = device.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// The host should be awaiting configuration
	awaitingConfiguration, err := s.ds.GetHostAwaitingConfiguration(ctx, mdmDevice.UUID)
	require.NoError(t, err)
	require.True(t, awaitingConfiguration)

	// run the cron to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	// run the worker to process the DEP enroll request
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)

	totalAppsToInstall := len(opts.VppAppsToInstall) + len(opts.InHouseAppsToInstall)
	// For reporting back via InstalledApplicationList
	reportedInstalledApps := make([]fleet.Software, 0, totalAppsToInstall)
	// For verifying number of installs
	installedApps := make(map[string]int, totalAppsToInstall)
	var inHouseInstallCount int

	var installProfileCount, installAppCount, refetchVerifyCount, otherCount int
	var profileCustomSeen, profileFleetCASeen, unexpectedProfileSeen bool

	// Can be useful for debugging
	logCommands := true
	for cmd != nil {
		if cmd.Command.RequestType == "DeclarativeManagement" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		if strings.HasPrefix(cmd.CommandUUID, fleet.RefetchAppsCommandUUID()) {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}

		switch cmd.Command.RequestType {
		case "InstallProfile":
			if logCommands {
				fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, string(fullCmd.Command.InstallProfile.Payload))
			}
			installProfileCount++
			if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), //nolint:gocritic // ignore ifElseChain
				fmt.Sprintf("<string>%s</string>", opts.CustomProfileIdent)) {
				profileCustomSeen = true
			} else if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetdConfigPayloadIdentifier)) {
				unexpectedProfileSeen = true
			} else if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetCARootConfigPayloadIdentifier)) {
				profileFleetCASeen = true
			} else if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string", mobileconfig.FleetFileVaultPayloadIdentifier)) &&
				strings.Contains(string(fullCmd.Command.InstallProfile.Payload), "ForceEnableInSetupAssistant") {
				unexpectedProfileSeen = true
			}
		case "InstallApplication":
			if fullCmd.Command.InstallApplication.ITunesStoreID != nil {
				if logCommands {
					fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, fmt.Sprint(*fullCmd.Command.InstallApplication.ITunesStoreID))
				}
				for _, app := range opts.VppAppsToInstall {
					if app.AdamID == fmt.Sprint(*fullCmd.Command.InstallApplication.ITunesStoreID) {
						reportedInstalledApps = append(reportedInstalledApps, fleet.Software{BundleIdentifier: app.BundleIdentifier, Name: app.Name, Version: app.LatestVersion, Installed: true})
						installedApps[app.AdamID]++
					}
				}
			} else {
				// in-house app (.ipa) installs carry a manifest URL instead of an App Store ID
				require.NotNil(t, fullCmd.Command.InstallApplication.ManifestURL)
				if logCommands {
					fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, *fullCmd.Command.InstallApplication.ManifestURL)
				}
				require.Contains(t, *fullCmd.Command.InstallApplication.ManifestURL, "/in_house_app/manifest/")
				require.Less(t, inHouseInstallCount, len(opts.InHouseAppsToInstall), "unexpected extra in-house app install command")
				sw := opts.InHouseAppsToInstall[inHouseInstallCount]
				sw.Installed = true
				reportedInstalledApps = append(reportedInstalledApps, sw)
				installedApps["inhouse:"+sw.BundleIdentifier]++
				inHouseInstallCount++
			}
			installAppCount++

		case "InstallEnterpriseApplication":
			if logCommands {
				if fullCmd.Command.InstallEnterpriseApplication.ManifestURL != nil {
					fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, *fullCmd.Command.InstallEnterpriseApplication.ManifestURL)
				} else {
					fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
				}
			}
		case "InstalledApplicationList":
			if logCommands {
				fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
			}
			// If we are polling to verify the install, we should get an
			// InstalledApplicationList command instead of an InstallApplication command.
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			// Withhold the last install from the completed report so release
			// provably waits for verification. Only the last entry can be
			// withheld, whatever its kind: installs activate one at a time
			// through the host's activity queue, so withholding an earlier one
			// stalls the remaining install commands, and earlier apps are
			// already verified by their own post-ack round by now.
			if len(reportedInstalledApps) == totalAppsToInstall {
				reportedInstalledApps[len(reportedInstalledApps)-1].Installed = false
			}
			cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(
				mdmDevice.UUID,
				cmd.CommandUUID,
				reportedInstalledApps,
			)
			// flip the status back for later
			reportedInstalledApps[len(reportedInstalledApps)-1].Installed = true
			require.NoError(t, err)
			// TODO: We don't actually normally get a command back from the acknowledgement of the InstalledAppList
			// but we'll get additional install commands if we follow it up with an idle. Is this a bug? I think it
			// may be because of how we handle activating the next upcoming activity?
			if cmd == nil {
				cmd, err = mdmDevice.Idle()
				require.NoError(t, err)
			}
			continue
		default:
			if logCommands {
				fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
			}
			otherCount++
		}

		cmds = append(cmds, &fullCmd)

		if cmd.Command.RequestType == "InstallApplication" {
			pending, err := s.ds.GetQueuedJobs(ctx, 5, time.Now().UTC().Add(time.Minute))
			require.NoError(t, err)
			for _, job := range pending {
				if job.Name == "apple_software" {
					mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
						_, err := q.ExecContext(ctx, `UPDATE jobs SET not_before = ? WHERE id = ?`, time.Now().Add(-1*time.Minute).UTC(), job.ID)
						return err
					})
				}
			}
			// Run the worker to process the VPP verification job before acking so that the Verify command is waiting for us
			s.runWorker()
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// expected commands: install CA, install profile (only the custom one),
	// not expected: account configuration, since enrollment_reference not set
	require.Len(t, cmds, 2+totalAppsToInstall)

	require.Equal(t, 2, installProfileCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetCASeen)
	require.False(t, unexpectedProfileSeen)

	require.Equal(t, totalAppsToInstall, installAppCount)
	require.Len(t, installedApps, totalAppsToInstall)

	// Each expected app should be installed exactly once
	for _, app := range opts.VppAppsToInstall {
		require.Equal(t, 1, installedApps[app.AdamID])
	}
	for _, sw := range opts.InHouseAppsToInstall {
		require.Equal(t, 1, installedApps["inhouse:"+sw.BundleIdentifier])
	}

	require.Equal(t, 0, otherCount)

	pendingReleaseJobs := []*fleet.Job{}
	if opts.EnableReleaseManually {
		// get the worker's pending job from the future, there should not be any
		// because it needs to be released manually
		pending, err := s.ds.GetQueuedJobs(ctx, 5, time.Now().UTC().Add(time.Minute))
		require.NoError(t, err)
		for _, job := range pending {
			if job.Name == "apple_mdm" && strings.Contains(string(*job.Args), string(worker.AppleMDMPostDEPReleaseDeviceTask)) {
				pendingReleaseJobs = append(pendingReleaseJobs, job)
			} else if job.Name == "apple_software" {
				// Just delete the job for now to keep things clean
				mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, job.ID)
					return err
				})
			}
		}
		require.Empty(t, pendingReleaseJobs)
		return
	}

	// Automatic release - device release job should be enqueued
	pending, err := s.ds.GetQueuedJobs(ctx, 5, time.Now().UTC().Add(time.Minute))
	pendingVerifyJobs := []*fleet.Job{}
	require.NoError(t, err)
	for _, job := range pending {
		if job.Name == "apple_mdm" && strings.Contains(string(*job.Args), string(worker.AppleMDMPostDEPReleaseDeviceTask)) {
			pendingReleaseJobs = append(pendingReleaseJobs, job)
		}
		if job.Name == "apple_software" {
			pendingVerifyJobs = append(pendingVerifyJobs, job)
		}
	}
	require.Len(t, pendingReleaseJobs, 1)
	require.Equal(t, "apple_mdm", pendingReleaseJobs[0].Name)
	require.Contains(t, string(*pendingReleaseJobs[0].Args), worker.AppleMDMPostDEPReleaseDeviceTask)

	require.Len(t, pendingVerifyJobs, 1)

	// make the pending jobs ready to run immediately and run the job
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE jobs SET not_before = ? WHERE id IN (?, ?)`, time.Now().Add(-1*time.Minute).UTC(), pendingReleaseJobs[0].ID, pendingVerifyJobs[0].ID)
		return err
	})

	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	// make the device process the commands, it should receive the VPP Verify.
	// It should not receive a DeviceConfigured command!
	cmds = cmds[:0]
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		if strings.HasPrefix(cmd.CommandUUID, fleet.RefetchAppsCommandUUID()) {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		if cmd.Command.RequestType == "InstalledApplicationList" {
			if logCommands {
				fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
			}
			cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(
				mdmDevice.UUID,
				cmd.CommandUUID,
				reportedInstalledApps,
			)
			require.NoError(t, err)
			// See above comment about cmd==nil, just want to make sure we don't get any additional
			// commands on the acknowledgement
			if cmd == nil {
				cmd, err = mdmDevice.Idle()
				require.NoError(t, err)
			}
			continue
		}
		require.FailNowf(t, "unexpected command", "got command %s of type %s", cmd.CommandUUID, cmd.Command.RequestType)
	}

	pending, err = s.ds.GetQueuedJobs(ctx, 5, time.Now().UTC().Add(time.Minute))
	pendingReleaseJobs = pendingReleaseJobs[:0]
	pendingVerifyJobs = pendingVerifyJobs[:0]
	require.NoError(t, err)
	for _, job := range pending {
		if job.Name == "apple_mdm" && strings.Contains(string(*job.Args), string(worker.AppleMDMPostDEPReleaseDeviceTask)) {
			pendingReleaseJobs = append(pendingReleaseJobs, job)
		}
		if job.Name == "apple_software" {
			pendingVerifyJobs = append(pendingVerifyJobs, job)
		}
	}
	require.Len(t, pendingReleaseJobs, 1)
	require.Equal(t, "apple_mdm", pendingReleaseJobs[0].Name)
	require.Contains(t, string(*pendingReleaseJobs[0].Args), worker.AppleMDMPostDEPReleaseDeviceTask)

	require.Empty(t, pendingVerifyJobs)

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE jobs SET not_before = ? WHERE id = ?`, time.Now().Add(-1*time.Minute).UTC(), pendingReleaseJobs[0].ID)
		return err
	})

	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	// make the device process the commands, it should receive the
	// DeviceConfigured one.
	cmds = cmds[:0]
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.Len(t, cmds, 1)
	var deviceConfiguredCount int
	for _, cmd := range cmds {
		if strings.HasPrefix(cmd.CommandUUID, fleet.RefetchAppsCommandUUIDPrefix) || strings.HasPrefix(cmd.CommandUUID, fleet.VerifySoftwareInstallVPPPrefix) {
			refetchVerifyCount++
			continue
		}
		switch cmd.Command.RequestType {
		case "DeviceConfigured":
			deviceConfiguredCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 1, deviceConfiguredCount)
	require.Equal(t, 0, otherCount)
}

// TestSetupExperienceBYODiOS exercises the post-enrollment integration path
// for an iOS Account-Driven User Enrollment (BYOD): after the device finishes
// TokenUpdate, Fleet should
//   - persist the Managed Apple ID on host_mdm (from the IdP account email),
//     and
//   - have the iphone_ipad_refetcher cron pick the host up on its next tick
//     (via the freshly-enrolled branch of ListIOSAndIPadOSToRefetch) so the
//     three iOS refetch commands are queued without waiting an hour. BYOD
//     restricts InstalledApplicationList to managed apps only — Apple rejects
//     the full-inventory variant on User Enrollments.
//
// The setup-experience VPP install half of the BYOD flow is exercised by
// unit-level tests (see TestInstallVPPAppPostValidation_AssociateAssetsRouting);
// reproducing the full Apple /registerVPPUserSrv + /assets/associate
// clientUserId dance in this integration test would need a more complete VPP
// mock backend.
func (s *integrationMDMTestSuite) TestSetupExperienceBYODiOS() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	// Account-Driven User Enrollment goes through the SAML SSO flow to obtain
	// the bearer token Apple includes on subsequent enrollment requests.
	// setUpMDMSSO swaps server_url to match the (hardcoded) SAML audience; we
	// restore the original URL before performing the actual enrollment so the
	// device gets the regular MDM endpoints.
	originalServerURL := s.server.URL
	s.setUpMDMSSO(t, true)
	abmToken := s.enableABM(t.Name())

	ssoResult := s.LoginAccountDrivenEnrollUser("sso_user", "user123#", string(abmToken.EnrollmentURLToken))
	loc, err := ssoResult.Location()
	require.NoError(t, err)
	require.NotNil(t, loc)
	require.True(t, strings.HasPrefix(loc.String(), "apple-remotemanagement-user-login://authentication-results?access-token="))
	accessToken := strings.Split(loc.String(), "apple-remotemanagement-user-login://authentication-results?access-token=")[1]

	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"server_settings": {"server_url": "`+originalServerURL+`"}
	}`), http.StatusOK, &appConfigResponse{})

	// BYOD-enroll an iPhone with the SSO-issued bearer token.
	iPhoneHwModel := "iPhone14,5"
	mdmDevice := mdmtest.NewTestMDMClientAppleAccountDrivenUserEnrollment(s.server.URL, iPhoneHwModel, accessToken)
	require.NoError(t, mdmDevice.Enroll())
	require.Equal(t, "sso_user@example.com", mdmDevice.EnrollInfo.AssignedManagedAppleID)

	// Process the AppleMDMPostManualEnrollmentTask that the lifecycle queued
	// during TokenUpdate. apple_mdm jobs run on s.appleMDMWorker (not s.worker
	// — that one handles macosJob/vppVerifyJob/softwareWorker).
	require.NoError(t, s.appleMDMWorker.ProcessJobs(ctx))

	// Locate the freshly enrolled host. For Account-Driven User Enrollment the
	// host UUID is the enrollment id, not a real device UDID.
	var hostsResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hostsResp)
	var enrolledHostID uint
	for _, h := range hostsResp.Hosts {
		if h.UUID == mdmDevice.EnrollmentID() {
			enrolledHostID = h.ID
			require.NotNil(t, h.MDM.EnrollmentStatus)
			require.Equal(t, fleet.MDMEnrollmentStatusPersonal, *h.MDM.EnrollmentStatus)
			break
		}
	}
	require.NotZero(t, enrolledHostID, "BYOD-enrolled iPhone not found")

	// Managed Apple ID is persisted on host_mdm so VPP installs can target the
	// user via clientUserId.
	managedAppleID, err := s.ds.GetHostManagedAppleID(ctx, enrolledHostID)
	require.NoError(t, err)
	require.Equal(t, "sso_user@example.com", managedAppleID)

	// Trigger the refetcher cron. With the freshly-enrolled branch in
	// ListIOSAndIPadOSToRefetch, our brand-new host (its details_updated_at is
	// set to the enroll time) gets picked up on this very first tick rather
	// than waiting up to an hour.
	require.NoError(t, apple_mdm.IOSiPadOSRefetch(ctx, s.ds, s.mdmCommander, s.logger, s.fleetSvc.NewActivity))

	tracked, err := s.ds.GetHostMDMCommands(ctx, enrolledHostID)
	require.NoError(t, err)
	requireTrackedRefetchCommands(t, tracked, enrolledHostID,
		fleet.RefetchAppsCommandUUIDPrefix, fleet.RefetchCertsCommandUUIDPrefix, fleet.RefetchDeviceCommandUUIDPrefix)

	// Drain the device's command queue and check the three refetch commands
	// arrive. InstalledApplicationList for BYOD MUST request managed apps only;
	// Apple rejects the full-inventory variant on User Enrollments.
	var sawAppList, sawCertList, sawDeviceInfo, sawManagedOnly bool
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		switch cmd.Command.RequestType {
		case "InstalledApplicationList":
			sawAppList = true
			require.NotNil(t, fullCmd.Command.InstalledApplicationList)
			sawManagedOnly = fullCmd.Command.InstalledApplicationList.ManagedAppsOnly
		case "CertificateList":
			sawCertList = true
		case "DeviceInformation":
			sawDeviceInfo = true
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.True(t, sawAppList, "InstalledApplicationList must be queued for BYOD iPhone")
	require.True(t, sawManagedOnly, "InstalledApplicationList for BYOD must set managedOnly=true")
	require.True(t, sawCertList, "CertificateList must be queued for BYOD iPhone")
	require.True(t, sawDeviceInfo, "DeviceInformation must be queued for BYOD iPhone")
}
