package service

// VPP setup experience tests for the MDM suite.
//
// Belongs here: App Store (VPP) apps in macOS setup experience — adding and removing them, install
// errors (including Apple server errors), required VPP software, and large numbers of VPP apps.
//
// Does not belong here: VPP apps on iOS/iPadOS setup experience
// (integration_mdm_setup_experience_ios_ipados_test.go), VPP apps outside setup experience
// (integration_vpp_install_test.go), and setup experience packages and scripts
// (integration_mdm_setup_experience_test.go).

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/dev_mode"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestSetupExperienceVPPInstallError() {
	t := s.T()
	ctx := context.Background()

	s.setSkipWorkerJobs(t)

	teamDevice, enrolledHost, team := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

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

	// Add an app with 0 licenses available
	s.appleVPPConfigSrvConfig.Assets = append(s.appleVPPConfigSrvConfig.Assets, vpp.Asset{
		AdamID:         "5",
		PricingParam:   "STDQ",
		AvailableCount: 0,
	})

	t.Cleanup(func() {
		s.appleVPPConfigSrvConfig.Assets = defaultVPPAssetList
	})

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Add the app with 0 licenses available
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: "5", SelfService: true}, http.StatusOK)

	// Add the VPP app to setup experience
	vppTitleID := getSoftwareTitleID(t, s.ds, "App 5", "apps")
	installerTitleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: team.ID, TitleIDs: []uint{vppTitleID, installerTitleID}}, http.StatusOK, &swInstallResp)

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)
	// run the worker to process the DEP enroll request
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "DeclarativeManagement" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// expected commands: install fleetd (install enterprise), install profiles
	// (custom one, fleetd configuration, fleet CA root)
	require.Len(t, cmds, 4)
	var installProfileCount, installEnterpriseCount, otherCount int
	var profileCustomSeen, profileFleetdSeen, profileFleetCASeen, profileFileVaultSeen bool
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "InstallProfile":
			installProfileCount++
			switch {
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), "<string>I1</string>"):
				profileCustomSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetdConfigPayloadIdentifier)):
				profileFleetdSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetCARootConfigPayloadIdentifier)):
				profileFleetCASeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string", mobileconfig.FleetFileVaultPayloadIdentifier)) &&
				strings.Contains(string(cmd.Command.InstallProfile.Payload), "ForceEnableInSetupAssistant"):
				profileFileVaultSeen = true
			}

		case "InstallEnterpriseApplication":
			installEnterpriseCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 3, installProfileCount)
	require.Equal(t, 1, installEnterpriseCount)
	require.Equal(t, 0, otherCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetdSeen)
	require.True(t, profileFleetCASeen)
	require.False(t, profileFileVaultSeen)

	// simulate fleetd being installed and the host being orbit-enrolled now
	enrolledHost.OsqueryHostID = ptr.String(mdmDevice.UUID)
	enrolledHost.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// there shouldn't be a worker Release Device pending job (we don't release that way anymore)
	pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 0)

	// call the /status endpoint, the software and script should be pending
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	var profNames []string
	var profStatuses []fleet.MDMDeliveryStatus
	for _, prof := range statusResp.Results.ConfigurationProfiles {
		profNames = append(profNames, prof.Name)
		profStatuses = append(profStatuses, prof.Status)
	}
	require.ElementsMatch(t, []string{"N1", "Fleetd configuration", "Fleet root certificate authority (CA)"}, profNames)
	require.ElementsMatch(t, []fleet.MDMDeliveryStatus{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying}, profStatuses)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 2)
	require.Equal(t, "App 5", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, "DummyApp", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)

	// Get status: the VPP app install should have run and failed.
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.Len(t, statusResp.Results.Software, 2)
	// App 5 has no licenses available, so we should get a status failed here...
	require.Equal(t, "App 5", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[0].Status)
	// ...but setup experience should still continue with the next app in the list
	require.Equal(t, "DummyApp", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)
	// Script goes last
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)

	// The status for DummyApp should be "running" now, since it's started installing
	// but we haven't sent back an installation status from orbit yet
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Len(t, statusResp.Results.Software, 2)
	require.Equal(t, "DummyApp", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[1].Status)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// exec ID for "DummyApp" out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, team.ID)
	require.NoError(t, err)
	require.Len(t, results, 3)
	var installUUID string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil &&
			r.SoftwareInstallerID != nil &&
			r.Name == statusResp.Results.Software[1].Name {
			installUUID = *r.HostSoftwareInstallsExecutionID
			break
		}
	}

	require.NotEmpty(t, installUUID)

	// Need to get the software title to get the package name
	var getSoftwareTitleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *statusResp.Results.Software[1].SoftwareTitleID), nil, http.StatusOK, &getSoftwareTitleResp, "team_id", fmt.Sprintf("%d", *enrolledHost.TeamID))
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle)
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage)

	// record a result for software installation
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`, *enrolledHost.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// Script is already running, poll again to confirm status
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "App 5", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Script.Status)
	require.Equal(t, "DummyApp", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[1].Status)

	// Get script exec ID
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, team.ID)
	require.NoError(t, err)
	require.Len(t, results, 3)
	var execID string
	for _, r := range results {
		if r.ScriptExecutionID != nil {
			execID = *r.ScriptExecutionID
		}
	}

	// record a result for script execution
	var scriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *enrolledHost.OrbitNodeKey, execID)),
		http.StatusOK, &scriptResp)

	// Get status again, now the script should be complete. This should also trigger the automatic
	// release of the device, as all setup experience steps are now complete.
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "App 5", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, "DummyApp", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[1].Status)

	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Script.Status)

	// check that the host received the device configured command automatically
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	cmds = cmds[:0]
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

func (s *integrationMDMTestSuite) TestSetupExperienceWithLotsOfVPPApps() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	s.registerResetVPPProxyData(t)

	// Set up some additional VPP apps on the mock the Fleet proxy to Apple servers
	s.appleVPPProxySrvData["6"] = `{"id": "6", "attributes": {"name": "App 6", "platformAttributes": {"osx": {"bundleId": "f-6", "artwork": {"url": "https://example.com/images/6/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "6.0.0"}}}, "deviceFamilies": ["mac"]}}`
	s.appleVPPProxySrvData["7"] = `{"id": "7", "attributes": {"name": "App 7", "platformAttributes": {"osx": {"bundleId": "g-7", "artwork": {"url": "https://example.com/images/7/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "7.0.0"}}}, "deviceFamilies": ["mac"]}}`
	s.appleVPPProxySrvData["8"] = `{"id": "8", "attributes": {"name": "App 8", "platformAttributes": {"osx": {"bundleId": "h-8", "artwork": {"url": "https://example.com/images/8/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "8.0.0"}}}, "deviceFamilies": ["mac"]}}`
	s.appleVPPProxySrvData["9"] = `{"id": "9", "attributes": {"name": "App 9", "platformAttributes": {"osx": {"bundleId": "i-9", "artwork": {"url": "https://example.com/images/9/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "9.0.0"}}}, "deviceFamilies": ["mac"]}}`
	s.appleVPPProxySrvData["10"] = `{"id": "10", "attributes": {"name": "App 10", "platformAttributes": {"osx": {"bundleId": "j-10", "artwork": {"url": "https://example.com/images/10/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "10.0.0"}}}, "deviceFamilies": ["mac"]}}`

	s.appleVPPConfigSrvConfig.Assets = append(s.appleVPPConfigSrvConfig.Assets, []vpp.Asset{
		{
			AdamID:         "6",
			PricingParam:   "STDQ",
			AvailableCount: 1,
		},
		{
			AdamID:         "7",
			PricingParam:   "STDQ",
			AvailableCount: 1,
		},
		{
			AdamID:         "8",
			PricingParam:   "STDQ",
			AvailableCount: 1,
		},
		{
			AdamID:         "9",
			PricingParam:   "STDQ",
			AvailableCount: 1,
		},
		{
			AdamID:         "10",
			PricingParam:   "STDQ",
			AvailableCount: 1,
		},
	}...)

	t.Cleanup(func() {
		s.appleVPPConfigSrvConfig.Assets = defaultVPPAssetList
	})

	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, app.AdamID, app.Platform)
		})

		return titleID
	}

	teamDevice, enrolledHost, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()
	require.NotNil(t, enrolledHost.TeamID)

	s.setVPPTokenForTeam(*enrolledHost.TeamID)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, teamDevice.SerialNumber)

	// Add some VPP apps
	macOSApp1 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1",
		LatestVersion:    "1.0.0",
	}

	macOSApp2 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "6",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 6",
		BundleIdentifier: "f-6",
		IconURL:          "https://example.com/images/6",
		LatestVersion:    "6.0.0",
	}
	macOSApp3 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "7",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 7",
		BundleIdentifier: "g-7",
		IconURL:          "https://example.com/images/7",
		LatestVersion:    "7.0.0",
	}
	macOSApp4 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "8",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 8",
		BundleIdentifier: "h-8",
		IconURL:          "https://example.com/images/8",
		LatestVersion:    "8.0.0",
	}
	macOSApp5 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "9",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 9",
		BundleIdentifier: "i-9",
		IconURL:          "https://example.com/images/9",
		LatestVersion:    "9.0.0",
	}
	macOSApp6 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "10",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 10",
		BundleIdentifier: "j-10",
		IconURL:          "https://example.com/images/10",
		LatestVersion:    "10.0.0",
	}

	expectedApps := []*fleet.VPPApp{macOSApp1, macOSApp2, macOSApp3, macOSApp4, macOSApp5, macOSApp6}

	slices.SortFunc(expectedApps, func(a, b *fleet.VPPApp) int {
		return strings.Compare(a.Name, b.Name)
	})

	expectedAppsByName := map[string]*fleet.VPPApp{
		macOSApp1.Name: macOSApp1,
		macOSApp2.Name: macOSApp2,
		macOSApp3.Name: macOSApp3,
		macOSApp4.Name: macOSApp4,
		macOSApp5.Name: macOSApp5,
		macOSApp6.Name: macOSApp6,
	}

	var addAppResp addAppStoreAppResponse
	// Add remaining as non-self-service
	var titleIDs []uint
	for _, app := range expectedApps {
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: enrolledHost.TeamID, AppStoreID: app.AdamID, Platform: app.Platform},
			http.StatusOK, &addAppResp)
		titleIDs = append(titleIDs, getSoftwareTitleIDFromApp(app))
	}

	// Add VPP apps to setup experience
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: *enrolledHost.TeamID, TitleIDs: titleIDs}, http.StatusOK, &swInstallResp)

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// run the worker to ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)
	// run the worker to process the DEP enroll request
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "DeclarativeManagement" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		// Can be useful for debugging
		// switch cmd.Command.RequestType {
		// case "InstallProfile":
		// 	fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, string(fullCmd.Command.InstallProfile.Payload))
		// case "InstallEnterpriseApplication":
		// 	if fullCmd.Command.InstallEnterpriseApplication.ManifestURL != nil {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, *fullCmd.Command.InstallEnterpriseApplication.ManifestURL)
		// 	} else {
		// 		fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType)
		// 	}
		// default:
		// 	fmt.Println(">>>> device received command: ", cmd.Command.RequestType)
		// }

		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// expected commands: install fleetd (install enterprise), install profiles
	// (custom one, fleetd configuration, fleet CA root)
	require.Len(t, cmds, 4)
	var installProfileCount, installEnterpriseCount, otherCount int
	var profileCustomSeen, profileFleetdSeen, profileFleetCASeen, profileFileVaultSeen bool
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "InstallProfile":
			installProfileCount++
			switch {
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), "<string>I1</string>"):
				profileCustomSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetdConfigPayloadIdentifier)):
				profileFleetdSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetCARootConfigPayloadIdentifier)):
				profileFleetCASeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string", mobileconfig.FleetFileVaultPayloadIdentifier)) &&
				strings.Contains(string(cmd.Command.InstallProfile.Payload), "ForceEnableInSetupAssistant"):
				profileFileVaultSeen = true
			}

		case "InstallEnterpriseApplication":
			installEnterpriseCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 3, installProfileCount)
	require.Equal(t, 1, installEnterpriseCount)
	require.Equal(t, 0, otherCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetdSeen)
	require.True(t, profileFleetCASeen)
	require.False(t, profileFileVaultSeen)

	// simulate fleetd being installed and the host being orbit-enrolled now
	enrolledHost.OsqueryHostID = ptr.String(mdmDevice.UUID)
	enrolledHost.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// there shouldn't be a worker Release Device pending job (we don't release that way anymore)
	pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 0)

	// call the /status endpoint, the software and script should be pending
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	var profNames []string
	var profStatuses []fleet.MDMDeliveryStatus
	for _, prof := range statusResp.Results.ConfigurationProfiles {
		profNames = append(profNames, prof.Name)
		profStatuses = append(profStatuses, prof.Status)
	}
	require.ElementsMatch(t, []string{"N1", "Fleetd configuration", "Fleet root certificate authority (CA)"}, profNames)
	require.ElementsMatch(t, []fleet.MDMDeliveryStatus{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying}, profStatuses)

	// the software and script are still pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 6)
	for _, software := range statusResp.Results.Software {
		_, ok := expectedAppsByName[software.Name]
		require.True(t, ok)
		require.Equal(t, fleet.SetupExperienceStatusPending, software.Status)
		require.NotNil(t, software.SoftwareTitleID)
		require.NotZero(t, *software.SoftwareTitleID)
	}

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		mysqltest.DumpTable(t, q, "host_vpp_software_installs", "command_uuid", "adam_id")
		return nil
	})

	checkCommandsInFlight := func(expectedCount int) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var count int
			err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_mdm_commands WHERE command_type = ?", fleet.VerifySoftwareInstallVPPPrefix)
			require.NoError(t, err)
			require.Equal(t, expectedCount, count)
			return nil
		})
	}

	type vppInstallOpts struct {
		failOnInstall      bool
		appInstallTimeout  bool
		softwareResultList []fleet.Software
	}

	processVPPInstallOnClient := func(mdmClient *mdmtest.TestAppleMDMClient, opts *vppInstallOpts) string {
		var installCmdUUID string

		// Process the InstallApplication command
		s.runWorker()
		cmd, err := mdmClient.Idle()
		require.NoError(t, err)

		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType {
			case "InstallApplication":
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				installCmdUUID = cmd.CommandUUID
				if opts.failOnInstall {
					t.Logf("Failed command UUID: %s", installCmdUUID)
					cmd, err = mdmClient.Err(cmd.CommandUUID, []mdm.ErrorChain{{ErrorCode: 1234}})
					require.NoError(t, err)
					continue
				}

				cmd, err = mdmClient.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			case "InstalledApplicationList":
				// If we are polling to verify the install, we should get an
				// InstalledApplicationList command instead of an InstallApplication command.
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				_, err = mdmClient.AcknowledgeInstalledApplicationList(
					mdmClient.UUID,
					cmd.CommandUUID,
					opts.softwareResultList,
				)
				require.NoError(t, err)
				return ""
			default:
				require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
			}
		}

		if opts.failOnInstall {
			return installCmdUUID
		}

		if opts.appInstallTimeout {
			mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(context.Background(), "UPDATE nano_command_results SET updated_at = ? WHERE command_uuid = ?", time.Now().Add(-11*time.Minute), installCmdUUID)
				return err
			})
		}

		// Process the verification command (InstalledApplicationList)
		s.runWorker()
		// Check that there is now a verify command in flight
		checkCommandsInFlight(1)
		cmd, err = mdmClient.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType {
			case "InstalledApplicationList":
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				cmd, err = mdmClient.AcknowledgeInstalledApplicationList(
					mdmClient.UUID,
					cmd.CommandUUID,
					opts.softwareResultList,
				)
				require.NoError(t, err)
			default:
				require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
			}
		}

		return installCmdUUID
	}

	// Simulate successful installation on the host
	opts := &vppInstallOpts{}
	opts.appInstallTimeout = false
	opts.failOnInstall = false
	// App 1 is installed now
	opts.softwareResultList = []fleet.Software{
		{
			Name:             macOSApp1.Name,
			BundleIdentifier: macOSApp1.BundleIdentifier,
			Version:          macOSApp1.LatestVersion,
			Installed:        true,
		},
	}
	processVPPInstallOnClient(mdmDevice, opts)

	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)

	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 6)
	for _, software := range statusResp.Results.Software {
		_, ok := expectedAppsByName[software.Name]
		require.True(t, ok)
		if software.Name == macOSApp1.Name {
			require.Equalf(t, fleet.SetupExperienceStatusSuccess, software.Status, "software %s should have succeeded", software.Name)
		} else {
			require.Equalf(t, fleet.SetupExperienceStatusPending, software.Status, "software %s should be pending", software.Name)
		}
		require.NotNil(t, software.SoftwareTitleID)
		require.NotZero(t, *software.SoftwareTitleID)
	}

	// Only 2 apps have an installation attempt at this point
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var count int
		err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_vpp_software_installs")
		require.NoError(t, err)
		require.Equal(t, 2, count)
		return nil
	})

	installedApps := map[string]struct{}{
		macOSApp1.Name: {},
	}

	for _, app := range expectedApps[1:] {
		opts.softwareResultList = append(opts.softwareResultList, fleet.Software{
			Name:             app.Name,
			BundleIdentifier: app.BundleIdentifier,
			Version:          app.LatestVersion,
			Installed:        true,
		})

		installedApps[app.Name] = struct{}{}

		processVPPInstallOnClient(mdmDevice, opts)

		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)

		require.NotNil(t, statusResp.Results.Script)
		require.Equal(t, "script.sh", statusResp.Results.Script.Name)
		require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
		require.Len(t, statusResp.Results.Software, 6)
		for _, software := range statusResp.Results.Software {
			_, ok := expectedAppsByName[software.Name]
			require.True(t, ok)
			_, shouldBeInstalled := installedApps[software.Name]
			if shouldBeInstalled {
				require.Equalf(t, fleet.SetupExperienceStatusSuccess, software.Status, "software %s should have succeeded", software.Name)
			} else {
				require.Equalf(t, fleet.SetupExperienceStatusPending, software.Status, "software %s should be pending", software.Name)
			}
			require.NotNil(t, software.SoftwareTitleID)
			require.NotZero(t, *software.SoftwareTitleID)
		}

	}
}

func (s *integrationMDMTestSuite) TestSetupExperienceVPPCRUD() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// Just for testing isolation
	otherTeam, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 2"})
	require.NoError(t, err)

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
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID, otherTeam.ID}}, http.StatusOK, &resPatchVPP)

	// app 1 macOS only
	macOSApp1 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1",
		LatestVersion:    "1.0.0",
	}
	// App 2 supports macOS, iOS, iPadOS
	macOSApp2 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	iOSApp2 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.IOSPlatform,
			},
		},
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}
	iPadOSApp2 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.IPadOSPlatform,
			},
		},
		Name:             "App 2",
		BundleIdentifier: "b-2",
		IconURL:          "https://example.com/images/2",
		LatestVersion:    "2.0.0",
	}

	// App 3 is iPadOS only
	iPadOSApp3 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "3",
				Platform: fleet.IPadOSPlatform,
			},
		},
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

		// Add apps to the other team as well so that they are available but should not show up in setup experience
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &otherTeam.ID, AppStoreID: app.AdamID, Platform: app.Platform},
			http.StatusOK, &addAppResp)
	}

	// Helper function for inspecting returned list of software marked for setup experience
	getReturnedSetupExperienceTitleIDs := func(titles []fleet.SoftwareTitleListResult) []uint {
		var titleIDs []uint
		for _, title := range titles {
			if (title.AppStoreApp != nil && title.AppStoreApp.InstallDuringSetup != nil && *title.AppStoreApp.InstallDuringSetup == true) ||
				(title.SoftwarePackage != nil && title.SoftwarePackage.InstallDuringSetup != nil && *title.SoftwarePackage.InstallDuringSetup == true) {
				titleIDs = append(titleIDs, title.ID)
			}
		}
		return titleIDs
	}

	checkSetupExperienceSoftware := func(t *testing.T, platform string, teamID uint, expectedTitleIDs []uint) {
		var respGetSetupExperience getSetupExperienceSoftwareResponse
		s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{},
			http.StatusOK,
			&respGetSetupExperience,
			"platform", platform,
			"team_id", fmt.Sprint(teamID),
		)
		assert.ElementsMatch(t, getReturnedSetupExperienceTitleIDs(respGetSetupExperience.SoftwareTitles), expectedTitleIDs)
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

	// Should return both of the items we set for macOS
	checkSetupExperienceSoftware(t, "macos", team.ID, []uint{titleIDsByApp[macOSApp1], titleIDsByApp[macOSApp2]})

	// Should return nothing for iOS/iPadOS
	checkSetupExperienceSoftware(t, "ios", team.ID, []uint{})
	checkSetupExperienceSoftware(t, "ipados", team.ID, []uint{})

	// Should return nothing for macOS on other team
	checkSetupExperienceSoftware(t, "macos", otherTeam.ID, []uint{})

	// Add an app for iOS
	putSetupExperienceSoftwareForPlatform(t, "ios", team.ID, []uint{titleIDsByApp[iOSApp2]})

	// Fetch iOS apps for the team and now it should be listed
	checkSetupExperienceSoftware(t, "ios", team.ID, []uint{titleIDsByApp[iOSApp2]})

	// Should still return nothing for iPadOS
	checkSetupExperienceSoftware(t, "ipados", team.ID, []uint{})

	// Should still return both of the items we set for macOS
	checkSetupExperienceSoftware(t, "macos", team.ID, []uint{titleIDsByApp[macOSApp1], titleIDsByApp[macOSApp2]})

	// Add apps for iPadOS
	putSetupExperienceSoftwareForPlatform(t, "ipados", team.ID, []uint{titleIDsByApp[iPadOSApp2], titleIDsByApp[iPadOSApp3]})

	// Fetch iPadOS apps for the team and now they should be listed
	checkSetupExperienceSoftware(t, "ipados", team.ID, []uint{titleIDsByApp[iPadOSApp2], titleIDsByApp[iPadOSApp3]})

	// Should return nothing for iOS/iPadOS/macOS on the other team
	checkSetupExperienceSoftware(t, "ios", otherTeam.ID, []uint{})
	checkSetupExperienceSoftware(t, "ipados", otherTeam.ID, []uint{})
	checkSetupExperienceSoftware(t, "macos", otherTeam.ID, []uint{})

	// try to add an ipadOS app to macOS and iOS, both should fail
	res := s.DoRaw("PUT", "/api/v1/fleet/setup_experience/software", []byte(fmt.Sprintf(`{"platform": "macos", "team_id": %d, "software_title_ids": [%d]}`, team.ID, titleIDsByApp[iPadOSApp2])), http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.NoError(t, res.Body.Close())
	assert.Contains(t, errMsg, "invalid platform for requested AppStoreApp")

	res = s.DoRaw("PUT", "/api/v1/fleet/setup_experience/software", []byte(fmt.Sprintf(`{"platform": "ios", "team_id": %d, "software_title_ids": [%d]}`, team.ID, titleIDsByApp[iPadOSApp2])), http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.NoError(t, res.Body.Close())
	assert.Contains(t, errMsg, "invalid platform for requested AppStoreApp")

	// Lists should be unchanged after the failed attempts
	checkSetupExperienceSoftware(t, "macos", team.ID, []uint{titleIDsByApp[macOSApp1], titleIDsByApp[macOSApp2]})
	checkSetupExperienceSoftware(t, "ios", team.ID, []uint{titleIDsByApp[iOSApp2]})

	// Clear iPadOS and verify macOS and iPadOS are unaffected
	putSetupExperienceSoftwareForPlatform(t, "ipados", team.ID, []uint{})

	// iPadOS should be empty now
	checkSetupExperienceSoftware(t, "ipados", team.ID, []uint{})

	// macOS/iPadOS lists should be unchanged
	checkSetupExperienceSoftware(t, "macos", team.ID, []uint{titleIDsByApp[macOSApp1], titleIDsByApp[macOSApp2]})
	checkSetupExperienceSoftware(t, "ios", team.ID, []uint{titleIDsByApp[iOSApp2]})
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithRequiredSoftwareVPP() {
	t := s.T()
	ctx := context.Background()

	s.setSkipWorkerJobs(t)

	teamDevice, enrolledHost, team := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	payload := map[string]any{
		"require_all_software_macos": true,
		"team_id":                    enrolledHost.TeamID,
	}
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)

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

	// Add an app with 1 license available
	s.appleVPPConfigSrvConfig.Assets = append(s.appleVPPConfigSrvConfig.Assets, vpp.Asset{
		AdamID:         "5",
		PricingParam:   "STDQ",
		AvailableCount: 1,
	})

	// Add an app with 0 licenses available
	s.appleVPPConfigSrvConfig.Assets = append(s.appleVPPConfigSrvConfig.Assets, vpp.Asset{
		AdamID:         "4",
		PricingParam:   "STDQ",
		AvailableCount: 0,
	})

	t.Cleanup(func() {
		s.appleVPPConfigSrvConfig.Assets = defaultVPPAssetList
	})

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Add the app with 0 licenses available
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: "5", SelfService: true}, http.StatusOK)
	// Add the app with 1 licenses available
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: "4", SelfService: true}, http.StatusOK)

	// Set custom display names on the VPP apps so they sort alphabetically
	// after DummyApp, with the no-license app sorting first among the VPP
	// apps. This ensures the installer (DummyApp) installs first, then the
	// VPP app with no license fails immediately, triggering the
	// "require all software" cancel-on-failure flow after a successful
	// installer install.
	// Alphabetical order by display name: DummyApp < VPP AAA No License < VPP ZZZ Has License
	//
	// This also exercises the fix for #41741: setup experience ordering
	// should use the custom display name when set.
	vppTitleID := getSoftwareTitleID(t, s.ds, "App 5", "apps")
	vppTitleID2 := getSoftwareTitleID(t, s.ds, "App 4", "apps")

	var updateAppResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", vppTitleID),
		&updateAppStoreAppRequest{TeamID: &team.ID, DisplayName: ptr.String("VPP ZZZ Has License")},
		http.StatusOK, &updateAppResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", vppTitleID2),
		&updateAppStoreAppRequest{TeamID: &team.ID, DisplayName: ptr.String("VPP AAA No License")},
		http.StatusOK, &updateAppResp)

	// Add the VPP apps to setup experience
	installerTitleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: team.ID, TitleIDs: []uint{vppTitleID, vppTitleID2, installerTitleID}}, http.StatusOK, &swInstallResp)

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)
	// run the worker to process the DEP enroll request
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "DeclarativeManagement" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}

		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// expected commands: install fleetd (install enterprise), install profiles
	// (custom one, fleetd configuration, fleet CA root)
	require.Len(t, cmds, 4)
	var installProfileCount, installEnterpriseCount, otherCount int
	var profileCustomSeen, profileFleetdSeen, profileFleetCASeen, profileFileVaultSeen bool
	for _, cmd := range cmds {
		switch cmd.Command.RequestType {
		case "InstallProfile":
			installProfileCount++
			switch {
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), "<string>I1</string>"):
				profileCustomSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetdConfigPayloadIdentifier)):
				profileFleetdSeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string>", mobileconfig.FleetCARootConfigPayloadIdentifier)):
				profileFleetCASeen = true
			case strings.Contains(string(cmd.Command.InstallProfile.Payload), fmt.Sprintf("<string>%s</string", mobileconfig.FleetFileVaultPayloadIdentifier)) &&
				strings.Contains(string(cmd.Command.InstallProfile.Payload), "ForceEnableInSetupAssistant"):
				profileFileVaultSeen = true
			}

		case "InstallEnterpriseApplication":
			installEnterpriseCount++
		default:
			otherCount++
		}
	}
	require.Equal(t, 3, installProfileCount)
	require.Equal(t, 1, installEnterpriseCount)
	require.Equal(t, 0, otherCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetdSeen)
	require.True(t, profileFleetCASeen)
	require.False(t, profileFileVaultSeen)

	// simulate fleetd being installed and the host being orbit-enrolled now
	enrolledHost.OsqueryHostID = ptr.String(mdmDevice.UUID)
	enrolledHost.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// there shouldn't be a worker Release Device pending job (we don't release that way anymore)
	pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 0)

	// call the /status endpoint, the software and script should be pending
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "reset_failed_setup_steps": true}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	var profNames []string
	var profStatuses []fleet.MDMDeliveryStatus
	for _, prof := range statusResp.Results.ConfigurationProfiles {
		profNames = append(profNames, prof.Name)
		profStatuses = append(profStatuses, prof.Status)
	}
	require.ElementsMatch(t, []string{"N1", "Fleetd configuration", "Fleet root certificate authority (CA)"}, profNames)
	require.ElementsMatch(t, []fleet.MDMDeliveryStatus{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerifying}, profStatuses)

	// the software and script are still pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 3)
	// Ordered by display name: DummyApp (no display name), App 4 (display name
	// "VPP AAA No License"), App 5 (display name "VPP ZZZ Has License").
	// Note: Name stores the original st.name, ordering uses display name.
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, "App 4", statusResp.Results.Software[1].Name)
	require.Equal(t, "VPP AAA No License", statusResp.Results.Software[1].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)
	require.Equal(t, "App 5", statusResp.Results.Software[2].Name)
	require.Equal(t, "VPP ZZZ Has License", statusResp.Results.Software[2].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[2].Status)

	// call /status endpoint again, DummyApp (first by display name) should be running
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Len(t, statusResp.Results.Software, 3)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name, "DummyApp should be first by display name")
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, team.ID)
	require.NoError(t, err)
	require.Len(t, results, 4)
	var installUUID string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil &&
			r.SoftwareInstallerID != nil &&
			r.Name == statusResp.Results.Software[0].Name {
			installUUID = *r.HostSoftwareInstallsExecutionID
			break
		}
	}

	require.NotEmpty(t, installUUID)

	// Need to get the software title to get the package name
	var getSoftwareTitleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *statusResp.Results.Software[0].SoftwareTitleID), nil, http.StatusOK, &getSoftwareTitleResp, "team_id", fmt.Sprintf("%d", *enrolledHost.TeamID))
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle)
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage)

	// record a result for software installation
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`, *enrolledHost.OrbitNodeKey, installUUID)), http.StatusNoContent)

	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 3)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	// App 4 (display name "VPP AAA No License") has no licenses available, so
	// it should fail immediately. Because we have "require_all_software_macos"
	// set, the remaining software (App 5) and the script should be marked as
	// failed too.
	require.Equal(t, "App 4", statusResp.Results.Software[1].Name)
	require.Equal(t, "VPP AAA No License", statusResp.Results.Software[1].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[1].Status)
	require.Equal(t, "App 5", statusResp.Results.Software[2].Name)
	require.Equal(t, "VPP ZZZ Has License", statusResp.Results.Software[2].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[2].Status)

	// Reset the setup experience items.
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "reset_failed_setup_steps": true}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	// the software and script are still pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 3)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, "App 4", statusResp.Results.Software[1].Name)
	require.Equal(t, "VPP AAA No License", statusResp.Results.Software[1].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)
	require.Equal(t, "App 5", statusResp.Results.Software[2].Name)
	require.Equal(t, "VPP ZZZ Has License", statusResp.Results.Software[2].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[2].Status)
}

func (s *integrationMDMTestSuite) TestSetupExperienceVPPInstallAppleServerError() {
	t := s.T()
	ctx := context.Background()

	s.setSkipWorkerJobs(t)

	teamDevice, enrolledHost, team := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()
	require.NotNil(t, enrolledHost.TeamID)

	s.setVPPTokenForTeam(team.ID)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, teamDevice.SerialNumber)

	// Add an App Store app standing in for 1Password for Safari, with licenses available so the
	// install is enqueued and only fails once Apple rejects the MDM command
	s.Do("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: "1", Platform: fleet.MacOSPlatform}, http.StatusOK)
	vppTitleID := getSoftwareTitleID(t, s.ds, "App 1", "apps")

	// Install only the App Store app during setup experience, leaving "Cancel setup if software
	// install fails" disabled so the script is the only step after it
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software",
		putSetupExperienceSoftwareRequest{TeamID: team.ID, TitleIDs: []uint{vppTitleID}}, http.StatusOK, &swInstallResp)

	// ADE enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	// Acknowledge the profiles and fleetd install sent during Setup Assistant
	var enrollCommands []string
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		enrollCommands = append(enrollCommands, cmd.Command.RequestType)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.Contains(t, enrollCommands, "InstallEnterpriseApplication")

	// simulate fleetd being installed and the host being orbit-enrolled now
	enrolledHost.OsqueryHostID = new(mdmDevice.UUID)
	enrolledHost.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// Poll for the steps the way orbit does once the setup experience screen is up, which
	// enqueues the App Store app install
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "App 1", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)

	// Reject every InstallApplication with the error the affected hosts reported
	appStoreRejection := []mdm.ErrorChain{{
		ErrorDomain:          "ASDServerErrorDomain",
		ErrorCode:            9610,
		LocalizedDescription: "Unhandled exception",
		USEnglishDescription: "Unhandled exception",
	}}

	// Answer commands and poll the status endpoint the way orbit does until the step resolves
	var installCommandUUIDs []string
	var deviceConfiguredCount int
	for range 10 {
		s.runWorker()

		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			switch cmd.Command.RequestType {
			case "InstallApplication":
				installCommandUUIDs = append(installCommandUUIDs, cmd.CommandUUID)
				cmd, err = mdmDevice.Err(cmd.CommandUUID, appStoreRejection)
			case "InstalledApplicationList":
				cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, nil)
			case "DeviceConfigured":
				deviceConfiguredCount++
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			default:
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			}
			require.NoError(t, err)
		}

		statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
		require.Len(t, statusResp.Results.Software, 1)
		if statusResp.Results.Software[0].Status.IsTerminalStatus() {
			break
		}
	}

	// Check the retries stop at the attempt limit and the host is not released part way through
	require.Len(t, installCommandUUIDs, fleet.MaxSoftwareInstallAttempts+1)
	require.Zero(t, deviceConfiguredCount)

	// Check the app ends failed, so the setup experience can move on to the script
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[0].Status)

	// Poll again to start the script, now that the failed app is no longer holding the queue
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Script.Status)

	// Report the script result the way orbit does
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, team.ID)
	require.NoError(t, err)
	var scriptExecutionID string
	for _, r := range results {
		if r.ScriptExecutionID != nil {
			scriptExecutionID = *r.ScriptExecutionID
		}
	}
	require.NotEmpty(t, scriptExecutionID)

	var scriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *enrolledHost.OrbitNodeKey, scriptExecutionID)),
		http.StatusOK, &scriptResp)

	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Script.Status)

	// Check the host is released instead of sitting on the setup experience screen
	var releaseCommands []string
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		releaseCommands = append(releaseCommands, cmd.Command.RequestType)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.Equal(t, []string{"DeviceConfigured"}, releaseCommands)
}
