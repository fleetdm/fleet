package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/fleetdbase"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestVPPAppInstallVerification() {
	// ===============================
	// Initial setup
	// ===============================

	t := s.T()
	s.setSkipWorkerJobs(t)

	// Valid token
	orgName := "Fleet Device Management Inc."
	location := "Fleet Location One"
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	s.lastActivityMatches(fleet.ActivityEnabledVPP{}.ActivityName(), "", 0)

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
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, app.AdamID, app.Platform)
		})

		return titleID
	}

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Get list of VPP apps from "Apple"
	// We're passing team 1 here, but we haven't added any app store apps to that team, so we get
	// back all available apps in our VPP location.
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)

	macOSApp := fleet.VPPApp{
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
	iPadOSApp := fleet.VPPApp{
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
	iOSApp := fleet.VPPApp{
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
	expectedApps := []*fleet.VPPApp{
		&macOSApp,
		&iPadOSApp,
		&iOSApp,
		{
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
		},
		{
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
		},
	}
	require.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	// Insert/deletion flow for macOS app
	// Add an app store app to team 1
	addedApp := expectedApps[0]
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, SelfService: true, AutomaticInstall: true}, http.StatusOK, &addAppResp)

	s.lastActivityOfTypeMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": true}`, team.Name,
			addedApp.Name, getSoftwareTitleIDFromApp(addedApp), addedApp.AdamID, team.ID, addedApp.Platform), 0)

	// ===============================
	// Initial setup
	// ===============================

	checkInstallFleetdCommandSent := func(mdmDevice *mdmtest.TestAppleMDMClient, wantCommand bool) {
		foundInstallFleetdCommand := false
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			if manifest := fullCmd.Command.InstallEnterpriseApplication.ManifestURL; manifest != nil {
				foundInstallFleetdCommand = true
				require.Equal(t, "InstallEnterpriseApplication", cmd.Command.RequestType)
				require.Contains(t, *fullCmd.Command.InstallEnterpriseApplication.ManifestURL, fleetdbase.GetPKGManifestURL())
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		require.Equal(t, wantCommand, foundInstallFleetdCommand)
	}

	// Create a couple of hosts
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, true)
	selfServiceHost, selfServiceDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, selfServiceHost, s.ds)
	selfServiceToken := "selfservicetoken"
	updateDeviceTokenForHost(t, s.ds, selfServiceHost.ID, selfServiceToken)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, selfServiceDevice.SerialNumber)
	// iOSHost, _ := s.createAppleMobileHostThenEnrollMDM("ios")
	// iPadOSHost, _ := s.createAppleMobileHostThenEnrollMDM("ipados")
	// ensure a valid alternate device token for self-service status access checking later
	updateDeviceTokenForHost(t, s.ds, mdmHost.ID, "foobar")

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID, orbitHost.ID, selfServiceHost.ID}, TeamID: &team.ID}, http.StatusOK)

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
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": true}`, team.Name,
			appSelfService.Name, getSoftwareTitleIDFromApp(appSelfService), appSelfService.AdamID, team.ID, appSelfService.Platform),
		0,
	)
	var listSw listSoftwareTitlesResponse
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
			fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": false}`, team.Name,
				app.Name, getSoftwareTitleIDFromApp(app), app.AdamID, team.ID, app.Platform),
			0,
		)
	}

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, len(expectedApps))
	var errTitleID, macOSTitleID uint
	for _, sw := range listSw.SoftwareTitles {
		require.NotNil(t, sw.AppStoreApp)
		switch {
		case sw.Name == addedApp.Name && sw.Source == "apps":
			macOSTitleID = sw.ID
		case sw.Name == errApp.Name && sw.Source == "apps":
			errTitleID = sw.ID
		}
	}

	// ================================
	// Install attempts
	// ================================

	processVPPInstallOnClient := func(failOnInstall bool, appInstallVerified bool) string {
		var installCmdUUID string

		// Process the InstallApplication command
		s.runWorker()
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType {
			case "InstallApplication":
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				installCmdUUID = cmd.CommandUUID
				if failOnInstall {
					t.Logf("Failed command UUID: %s", installCmdUUID)
					cmd, err = mdmDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{{ErrorCode: 1234}})
					require.NoError(t, err)
					continue
				}

				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			default:
				require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
			}
		}

		if failOnInstall {
			return installCmdUUID
		}

		// Process the verification command (InstalledApplicationList)
		s.runWorker()
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType {
			case "InstalledApplicationList":
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, []fleet.Software{{Name: addedApp.Name, BundleIdentifier: addedApp.BundleIdentifier, Version: addedApp.LatestVersion, Installed: appInstallVerified}})
				require.NoError(t, err)
			default:
				require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
			}
		}

		return installCmdUUID
	}

	// Trigger install to the host
	installResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, errTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

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

	// Simulate failed installation on the host
	failedCmdUUID := processVPPInstallOnClient(true, false)

	// We should have cleared out upcoming_activies since the install failed
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var count uint
		err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?", mdmHost.ID)
		require.NoError(t, err)
		require.Zero(t, count)
		return nil
	})

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
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			errApp.Name,
			errApp.AdamID,
			failedCmdUUID,
			fleet.SoftwareInstallFailed,
		),
		0,
	)

	// Successful install

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// Simulate successful installation on the host
	installCmdUUID := processVPPInstallOnClient(false, true)

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 1)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			addedApp.Name,
			addedApp.AdamID,
			installCmdUUID,
			fleet.SoftwareInstalled,
		),
		0,
	)

	// Check list host software

	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW := getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1, got2 := gotSW[0], gotSW[1]
	require.Equal(t, got1.Name, "App 1")
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.AppStoreApp.IconURL, ptr.String(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	require.NotNil(t, got1.Status)
	require.Equal(t, *got1.Status, fleet.SoftwareInstalled)
	require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, installCmdUUID)
	require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)
	require.Equal(t, got2.Name, "App 2")
	require.NotNil(t, got2.Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *got2.Status)
	require.NotNil(t, got2.AppStoreApp)
	require.Equal(t, got2.AppStoreApp.AppStoreID, errApp.AdamID)
	require.Equal(t, got2.AppStoreApp.IconURL, ptr.String(errApp.IconURL))
	require.Empty(t, got2.AppStoreApp.Name)
	require.Equal(t, got2.AppStoreApp.Version, errApp.LatestVersion)
	require.Equal(t, got2.AppStoreApp.LastInstall.CommandUUID, failedCmdUUID)
	require.NotNil(t, got2.AppStoreApp.LastInstall.InstalledAt)

	// Unsuccessful install (failed to verify)

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// Simulate timed out verification
	os.Setenv("FLEET_TEST_VPP_VERIFY_TIMEOUT", "1ms")
	t.Cleanup(func() {
		require.NoError(t, os.Unsetenv("FLEET_TEST_VPP_VERIFY_TIMEOUT"))
	})

	installCmdUUID = processVPPInstallOnClient(false, false)

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Empty(t, listResp.Hosts)

	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "failed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 1)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "failed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			addedApp.Name,
			addedApp.AdamID,
			installCmdUUID,
			fleet.SoftwareInstallFailed,
		),
		0,
	)

	// Check list host software

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1, got2 = gotSW[0], gotSW[1]
	require.Equal(t, got1.Name, "App 1")
	require.NotNil(t, got1.AppStoreApp)
	require.Equal(t, got1.AppStoreApp.AppStoreID, addedApp.AdamID)
	require.Equal(t, got1.AppStoreApp.IconURL, ptr.String(addedApp.IconURL))
	require.Empty(t, got1.AppStoreApp.Name) // Name is only present for installer packages
	require.Equal(t, got1.AppStoreApp.Version, addedApp.LatestVersion)
	require.NotNil(t, got1.Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *got1.Status)
	require.Equal(t, got1.AppStoreApp.LastInstall.CommandUUID, installCmdUUID)
	require.NotNil(t, got1.AppStoreApp.LastInstall.InstalledAt)
	require.Equal(t, got2.Name, "App 2")
	require.NotNil(t, got2.Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *got2.Status)
	require.NotNil(t, got2.AppStoreApp)
	require.Equal(t, got2.AppStoreApp.AppStoreID, errApp.AdamID)
	require.Equal(t, got2.AppStoreApp.IconURL, ptr.String(errApp.IconURL))
	require.Empty(t, got2.AppStoreApp.Name)
	require.Equal(t, got2.AppStoreApp.Version, errApp.LatestVersion)
	require.Equal(t, got2.AppStoreApp.LastInstall.CommandUUID, failedCmdUUID)
	require.NotNil(t, got2.AppStoreApp.LastInstall.InstalledAt)
}
