package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
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

	// Insert/deletion flow for macOS app
	// Add an app store app to team 1
	addedApp := &fleet.VPPApp{
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
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, SelfService: true, AutomaticInstall: true}, http.StatusOK, &addAppResp)

	s.lastActivityOfTypeMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": true}`, team.Name,
			addedApp.Name, getSoftwareTitleIDFromApp(addedApp), addedApp.AdamID, team.ID, addedApp.Platform), 0)

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
	// ensure a valid alternate device token for self-service status access checking later
	updateDeviceTokenForHost(t, s.ds, mdmHost.ID, "foobar")

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID, orbitHost.ID, selfServiceHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Add all apps to the team
	errApp := &fleet.VPPApp{
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
	expectedApps := []*fleet.VPPApp{addedApp, errApp}

	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
		"available_for_install", "true")
	// Add remaining as non-self-service
	for _, app := range expectedApps {
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

	checkCommandsInFlight := func(expectedCount int) {
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			var count int
			err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_mdm_commands WHERE command_type = ?", fleet.VerifySoftwareInstallVPPPrefix)
			require.NoError(t, err)
			require.Equal(t, expectedCount, count)
			return nil
		})
	}

	processVPPInstallOnClient := func(failOnInstall, appInstallVerified, appInstallTimeout bool) string {
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
			case "InstalledApplicationList":
				// If we are polling to verify the install, we should get an
				// InstalledApplicationList command instead of an InstallApplication command.
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				_, err = mdmDevice.AcknowledgeInstalledApplicationList(
					mdmDevice.UUID,
					cmd.CommandUUID,
					[]fleet.Software{
						{
							Name:             "RandomApp",
							BundleIdentifier: "com.example.randomapp",
							Version:          "9.9.9",
							Installed:        false,
						},
						{
							Name:             addedApp.Name,
							BundleIdentifier: addedApp.BundleIdentifier,
							Version:          addedApp.LatestVersion,
							Installed:        appInstallVerified,
						},
					},
				)
				require.NoError(t, err)
				return ""
			default:
				require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
			}
		}

		if failOnInstall {
			return installCmdUUID
		}

		if appInstallTimeout {
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
				_, err := q.ExecContext(context.Background(), "UPDATE nano_command_results SET updated_at = ? WHERE command_uuid = ?", time.Now().Add(-11*time.Minute), installCmdUUID)
				return err
			})
		}

		// Process the verification command (InstalledApplicationList)
		s.runWorker()
		// Check that there is now a verify command in flight
		checkCommandsInFlight(1)
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType {
			case "InstalledApplicationList":
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(
					mdmDevice.UUID,
					cmd.CommandUUID,
					[]fleet.Software{
						{
							Name:             "RandomApp",
							BundleIdentifier: "com.example.randomapp",
							Version:          "9.9.9",
							Installed:        false,
						},
						{
							Name:             addedApp.Name,
							BundleIdentifier: addedApp.BundleIdentifier,
							Version:          addedApp.LatestVersion,
							Installed:        appInstallVerified,
						},
					},
				)
				require.NoError(t, err)
			default:
				require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
			}
		}

		return installCmdUUID
	}

	checkVPPApp := func(got *fleet.HostSoftwareWithInstaller, expected *fleet.VPPApp, expectedCmdUUID string, expectedStatus fleet.SoftwareInstallerStatus) {
		require.Equal(t, expected.Name, got.Name)
		require.NotNil(t, got.AppStoreApp)
		require.Equal(t, expected.AdamID, got.AppStoreApp.AppStoreID)
		require.Equal(t, ptr.String(expected.IconURL), got.AppStoreApp.IconURL)
		require.Empty(t, got.AppStoreApp.Name) // Name is only present for installer packages
		require.Equal(t, expected.LatestVersion, got.AppStoreApp.Version)
		require.NotNil(t, got.Status)
		require.Equal(t, expectedStatus, *got.Status)
		require.Equal(t, expectedCmdUUID, got.AppStoreApp.LastInstall.CommandUUID)
		require.NotNil(t, got.AppStoreApp.LastInstall.InstalledAt)
	}

	// ================================
	// Install command failed
	// ================================

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
	failedCmdUUID := processVPPInstallOnClient(true, false, false)

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

	// ================================================
	// Successful install and immediate verification
	// ================================================

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// Simulate successful installation on the host
	installCmdUUID := processVPPInstallOnClient(false, true, false)

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

	checkVPPApp(got1, addedApp, installCmdUUID, fleet.SoftwareInstalled)
	checkVPPApp(got2, errApp, failedCmdUUID, fleet.SoftwareInstallFailed)

	// ================================================
	// Successful install and delayed verification
	// ================================================

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// Install is ACK, but not verified yet
	installCmdUUID = processVPPInstallOnClient(false, false, false)

	// We should have 0 installed, because the verification is not done yet
	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 0)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// We should instead have 1 pending
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	checkVPPApp(gotSW[0], addedApp, installCmdUUID, fleet.SoftwareInstallPending)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 1)

	// Install is ACK, but not verified yet
	// Don't update the command UUID because we didn't trigger a new install command
	// (the command UUID is the same as the one we got when we triggered the install)
	processVPPInstallOnClient(false, false, false)

	// We should have 0 installed, because the verification is not done yet
	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 0)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// We should instead have 1 pending
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	checkVPPApp(gotSW[0], addedApp, installCmdUUID, fleet.SoftwareInstallPending)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Len(t, listResp.Hosts, 1)

	// Install is ACK, and now it's verified
	// Don't update the command UUID because we didn't trigger a new install command
	// (the command UUID is the same as the one we got when we triggered the install)
	processVPPInstallOnClient(false, true, false)

	checkCommandsInFlight(0)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	checkVPPApp(gotSW[0], addedApp, installCmdUUID, fleet.SoftwareInstalled)

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

	// ========================================================
	// Install command succeeds, but verification fails
	// ========================================================

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	installCmdUUID = processVPPInstallOnClient(false, false, true)

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Empty(t, listResp.Hosts)

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
	checkVPPApp(got1, addedApp, installCmdUUID, fleet.SoftwareInstallFailed)
	checkVPPApp(got2, errApp, failedCmdUUID, fleet.SoftwareInstallFailed)

	// ========================================================
	// Mark installs as failed when MDM turned off on host
	// ========================================================

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", mdmHost.ID), nil, http.StatusNoContent)

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// We should have cleared out upcoming_activies when disabling MDM
		var count uint
		err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?", mdmHost.ID)
		require.NoError(t, err)
		require.Zero(t, count)

		installCmdUUID = ""
		// Get the UUID for the latest install
		err = sqlx.GetContext(
			context.Background(),
			q,
			&installCmdUUID,
			"SELECT command_uuid FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ? ORDER BY verification_failed_at DESC",
			mdmHost.ID,
			addedApp.AdamID,
		)
		require.NotEmpty(t, installCmdUUID)
		require.NoError(t, err)

		return nil
	})

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1, got2 = gotSW[0], gotSW[1]
	checkVPPApp(got1, addedApp, installCmdUUID, fleet.SoftwareInstallFailed)
	checkVPPApp(got2, errApp, failedCmdUUID, fleet.SoftwareInstallFailed)

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// ========================================================
	// Mark installs as failed when MDM turned off globally
	// ========================================================

	// Re-enroll host in MDM
	mdmDevice = enrollMacOSHostInMDM(t, mdmHost, s.ds, s.server.URL)
	s.runWorker()
	checkInstallFleetdCommandSent(mdmDevice, true)

	// Trigger install to the host
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	s.Do("DELETE", "/api/latest/fleet/mdm/apple/apns_certificate", nil, http.StatusOK)

	t.Cleanup(s.appleCoreCertsSetup)

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// We should have cleared out upcoming_activies when disabling MDM
		var count uint
		err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM upcoming_activities WHERE host_id = ?", mdmHost.ID)
		require.NoError(t, err)
		require.Zero(t, count)

		installCmdUUID = ""
		// Get the UUID for the latest install
		err = sqlx.GetContext(
			context.Background(),
			q,
			&installCmdUUID,
			"SELECT command_uuid FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ? ORDER BY verification_failed_at DESC",
			mdmHost.ID,
			addedApp.AdamID,
		)
		require.NoError(t, err)
		require.NotEmpty(t, installCmdUUID)

		return nil
	})

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw)
	gotSW = getHostSw.Software
	require.Len(t, gotSW, 2) // App 1 and App 2
	got1, got2 = gotSW[0], gotSW[1]
	checkVPPApp(got1, addedApp, installCmdUUID, fleet.SoftwareInstallFailed)
	checkVPPApp(got2, errApp, failedCmdUUID, fleet.SoftwareInstallFailed)

	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)
}
