package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) setVPPTokenForTeam(teamID uint) {
	t := s.T()
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

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{teamID}}, http.StatusOK, &resPatchVPP)
}

func (s *integrationMDMTestSuite) TestVPPAppInstallVerification() {
	// ===============================
	// Initial setup
	// ===============================

	t := s.T()
	s.setSkipWorkerJobs(t)

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.setVPPTokenForTeam(team.ID)

	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			ctx := context.Background()
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?`, app.AdamID, app.Platform)
		})

		return titleID
	}

	// Add macOS and iOS apps to team 1
	macOSApp := &fleet.VPPApp{
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

	iOSApp := &fleet.VPPApp{
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
	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: macOSApp.AdamID, SelfService: true}, http.StatusOK, &addAppResp)

	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": true}`, team.Name,
			macOSApp.Name, getSoftwareTitleIDFromApp(macOSApp), macOSApp.AdamID, team.ID, macOSApp.Platform), 0)

	// Add iOS app to team
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: iOSApp.AdamID, Platform: iOSApp.Platform}, http.StatusOK, &addAppResp)

	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": false}`, team.Name,
			iOSApp.Name, getSoftwareTitleIDFromApp(iOSApp), iOSApp.AdamID, team.ID, iOSApp.Platform), 0)

	// Create hosts for testing
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)
	selfServiceHost, selfServiceDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.runWorker()
	setOrbitEnrollment(t, selfServiceHost, s.ds)
	selfServiceToken := "selfservicetoken"
	checkInstallFleetdCommandSent(t, selfServiceDevice, true)
	updateDeviceTokenForHost(t, s.ds, selfServiceHost.ID, selfServiceToken)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, selfServiceDevice.SerialNumber)

	// Create and enroll an iOS device
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
	expectedApps := []*fleet.VPPApp{macOSApp, errApp, iOSApp}
	expectedAppsByBundleID := map[string]*fleet.VPPApp{
		macOSApp.BundleIdentifier: macOSApp,
		errApp.BundleIdentifier:   errApp,
		iOSApp.BundleIdentifier:   iOSApp,
	}
	addedApp := expectedApps[0]

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

	type vppInstallOpts struct {
		failOnInstall      bool
		appInstallVerified bool
		appInstallTimeout  bool
		bundleID           string
	}

	processVPPInstallOnClient := func(mdmClient *mdmtest.TestAppleMDMClient, opts vppInstallOpts) string {
		var installCmdUUID string

		// Process the InstallApplication command
		s.runWorker()
		cmd, err := mdmClient.Idle()
		require.NoError(t, err)

		app, ok := expectedAppsByBundleID[opts.bundleID]
		require.Truef(t, ok, "unexpected bundle ID: %s", opts.bundleID)
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
					[]fleet.Software{
						{
							Name:             "RandomApp",
							BundleIdentifier: "com.example.randomapp",
							Version:          "9.9.9",
							Installed:        false,
						},
						{
							Name:             app.Name,
							BundleIdentifier: app.BundleIdentifier,
							Version:          app.LatestVersion,
							Installed:        opts.appInstallVerified,
						},
					},
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
			mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
					[]fleet.Software{
						{
							Name:             "RandomApp",
							BundleIdentifier: "com.example.randomapp",
							Version:          "9.9.9",
							Installed:        false,
						},
						{
							Name:             app.Name,
							BundleIdentifier: app.BundleIdentifier,
							Version:          app.LatestVersion,
							Installed:        opts.appInstallVerified,
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
		require.Equal(t, ptr.String(expected.IconURL), got.IconUrl)
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

	// Cancel any pending refetch requests
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), mdmHost.ID, false))

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
	opts := vppInstallOpts{
		failOnInstall:      true,
		appInstallVerified: false,
		appInstallTimeout:  false,
		bundleID:           addedApp.BundleIdentifier,
	}
	failedCmdUUID := processVPPInstallOnClient(mdmDevice, opts)

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

	// No refetch requested since the install failed
	var hostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost.ID), nil, http.StatusOK, &hostResp)
	require.False(t, hostResp.Host.RefetchRequested, "RefetchRequested should be false after failed software install")

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
	opts.appInstallTimeout = false
	opts.failOnInstall = false
	opts.appInstallVerified = true
	installCmdUUID := processVPPInstallOnClient(mdmDevice, opts)

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

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost.ID), nil, http.StatusOK, &hostResp)
	require.True(t, hostResp.Host.RefetchRequested, "RefetchRequested should be true after successful software install")

	s.lq.On("QueriesForHost", mdmHost.ID).Return(map[string]string{fmt.Sprintf("%d", mdmHost.ID): "select 1 from osquery;"}, nil)

	req := getDistributedQueriesRequest{NodeKey: *mdmHost.NodeKey}
	var dqResp getDistributedQueriesResponse
	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), mdmHost.ID, false))

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
	opts.appInstallTimeout = false
	opts.appInstallVerified = false
	opts.failOnInstall = false
	installCmdUUID = processVPPInstallOnClient(mdmDevice, opts)

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
	processVPPInstallOnClient(mdmDevice, opts)

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
	opts.appInstallVerified = true
	processVPPInstallOnClient(mdmDevice, opts)

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

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost.ID), nil, http.StatusOK, &hostResp)
	require.True(t, hostResp.Host.RefetchRequested, "RefetchRequested should be true after successful software install")

	s.DoJSON("POST", "/api/osquery/distributed/read", req, http.StatusOK, &dqResp)
	require.Contains(t, dqResp.Queries, "fleet_detail_query_software_macos")
	require.NoError(t, s.ds.UpdateHostRefetchRequested(context.Background(), mdmHost.ID, false))

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

	opts.failOnInstall = false
	opts.appInstallVerified = false
	opts.appInstallTimeout = true
	installCmdUUID = processVPPInstallOnClient(mdmDevice, opts)

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

	// No refetch requested since the install failed
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost.ID), nil, http.StatusOK, &hostResp)
	require.False(t, hostResp.Host.RefetchRequested, "RefetchRequested should be false after failed software install")

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

	// Trigger install to the self-service device (its data shouldn't be changed)
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", selfServiceHost.ID, macOSTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 2, countResp.Count)

	// Trigger verification on other host
	opts.failOnInstall = false
	opts.appInstallVerified = false
	opts.appInstallTimeout = false
	processVPPInstallOnClient(selfServiceDevice, opts)

	s.runWorker()

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", mdmHost.ID), nil, http.StatusNoContent)

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		// We should have cleared out upcoming_activies when disabling MDM
		var count int
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

		count = 99999

		// We also should have cleared out host_mdm_commands to avoid a deadlocked state
		err = sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_mdm_commands WHERE host_id = ? AND command_type = ?", mdmHost.ID, fleet.VerifySoftwareInstallVPPPrefix)
		require.NoError(t, err)
		require.Zero(t, count)

		// The other host should have a verification command pending
		err = sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_mdm_commands WHERE host_id = ? AND command_type = ?", selfServiceHost.ID, fleet.VerifySoftwareInstallVPPPrefix)
		require.NoError(t, err)
		require.Equal(t, 1, count)

		return nil
	})

	// Cancel the install for the other host, we don't need it anymore
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", selfServiceHost.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)

	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", selfServiceHost.ID, listUpcomingAct.Activities[0].UUID), nil, http.StatusNoContent)

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
	checkInstallFleetdCommandSent(t, mdmDevice, true)

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

	// Re-enable MDM
	s.appleCoreCertsSetup()

	// ========================================================
	// Test iOS VPP app installation
	// ========================================================

	// Enroll iOS device, add serial number to fake Apple server, and transfer to team
	iosHost, iosDevice := s.createAppleMobileHostThenEnrollMDM("ios")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, iosDevice.SerialNumber)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{iosHost.ID}, TeamID: &team.ID}, http.StatusOK)

	var iosTitleID uint
	for _, sw := range listSw.SoftwareTitles {
		if sw.Name == iOSApp.Name && sw.Source == "ios_apps" {
			iosTitleID = sw.ID
			break
		}
	}
	require.NotZero(t, iosTitleID)

	// Trigger install to the iOS device
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", iosHost.ID, iosTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	// Verify pending status
	countResp = countHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(iosTitleID))
	require.Equal(t, 1, countResp.Count)

	// Before installation, we should have 0 refetch commands
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var count int
		err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_mdm_commands WHERE host_id = ? AND command_type = ?", iosHost.ID, fleet.RefetchAppsCommandUUIDPrefix)
		require.NoError(t, err)
		require.Zero(t, count)
		return nil
	})

	// Simulate successful installation on iOS device
	opts.appInstallTimeout = false
	opts.appInstallVerified = true
	opts.failOnInstall = false
	opts.bundleID = iOSApp.BundleIdentifier
	installCmdUUID = processVPPInstallOnClient(iosDevice, opts)

	// Verify successful installation
	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(iosTitleID))
	assert.Len(t, listResp.Hosts, 1)
	assert.Equal(t, iosHost.ID, listResp.Hosts[0].ID)

	// Verify activity log entry
	s.lastActivityMatches(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "policy_id": null, "policy_name": null}`,
			iosHost.ID,
			iosHost.DisplayName(),
			iOSApp.Name,
			iOSApp.AdamID,
			installCmdUUID,
			fleet.SoftwareInstalled,
		),
		0,
	)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", iosHost.ID), nil, http.StatusOK, &hostResp)
	require.False(t, hostResp.Host.RefetchRequested, "RefetchRequested should be false after successful software install for iDevice")

	// Now we have a refetch apps command in flight to update the host software inventory
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var count int
		err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_mdm_commands WHERE host_id = ?", iosHost.ID)
		require.NoError(t, err)
		require.Equal(t, count, 1)
		return nil
	})

	s.runWorker()
	cmd, err := iosDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType {
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			require.True(t, strings.HasPrefix(cmd.CommandUUID, fleet.RefetchAppsCommandUUIDPrefix))
			cmd, err = iosDevice.AcknowledgeInstalledApplicationList(
				iosDevice.UUID,
				cmd.CommandUUID,
				[]fleet.Software{
					{
						Name:             "RandomApp",
						BundleIdentifier: "com.example.randomapp",
						Version:          "9.9.9",
						Installed:        false,
					},
					{
						Name:             iOSApp.Name,
						BundleIdentifier: iOSApp.BundleIdentifier,
						Version:          iOSApp.LatestVersion,
						Installed:        true,
					},
				},
			)
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
		}
	}

	// we should also have the installed version, because we update host software inventory on verification
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", iosHost.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	assert.Len(t, getHostSw.Software, 1)
	assert.Equal(t, iosTitleID, getHostSw.Software[0].ID)
	assert.NotNil(t, getHostSw.Software[0].AppStoreApp)
	assert.Len(t, getHostSw.Software[0].InstalledVersions, 1)
	assert.Equal(t, iOSApp.LatestVersion, getHostSw.Software[0].InstalledVersions[0].Version)
}

// for https://github.com/fleetdm/fleet/issues/31083
func (s *integrationMDMTestSuite) TestVPPAppActivitiesOnCancelInstall() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.setVPPTokenForTeam(team.ID)

	// Add app 1 and 2 targeting macOS to the team
	app1 := &fleet.VPPApp{
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

	app2 := &fleet.VPPApp{
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

	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app1.AdamID, SelfService: true}, http.StatusOK, &addAppResp)
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app2.AdamID, SelfService: false}, http.StatusOK, &addAppResp)

	// list the software titles to get the title IDs
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	var app1TitleID, app2TitleID uint
	for _, sw := range listSw.SoftwareTitles {
		require.NotNil(t, sw.AppStoreApp)
		switch sw.Name {
		case app1.Name:
			app1TitleID = sw.ID
		case app2.Name:
			app2TitleID = sw.ID
		}
	}

	// create a control host that will not be used in the test, should be unaffected
	controlHost, controlDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, controlHost, s.ds)
	s.runWorker()
	checkInstallFleetdCommandSent(t, controlDevice, true)
	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, controlHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{controlHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// trigger a VPP app install on the control host, will stay there until the end
	var installResp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", controlHost.ID, app1TitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	// create a host that will receive the VPP install commands AFTER a script execution request
	// (so the VPP installs are not activated when they are cancelled)
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)
	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// enqueue a script run, so the VPP app installs are pending in the unified
	// queue
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: mdmHost.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)

	// trigger install of both apps on the host
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, app1TitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, app2TitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	// confirm the state of this host's upcoming activities
	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 3)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[0].Type)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), listResp.Activities[1].Type)
	require.Contains(t, string(*listResp.Activities[1].Details), fmt.Sprintf(`"app_store_id": %q`, app1.AdamID))
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), listResp.Activities[2].Type)
	require.Contains(t, string(*listResp.Activities[2].Details), fmt.Sprintf(`"app_store_id": %q`, app2.AdamID))

	// listing the host's software shows them as pending install
	var getHostSw getHostSoftwareResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[1].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[1].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)

	// turn off MDM for the host
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", mdmHost.ID), nil, http.StatusNoContent)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), fmt.Sprintf(`{"host_display_name":%q, "host_serial":%q, "installed_from_dep":false}`, mdmHost.DisplayName(), mdmHost.HardwareSerial), 0)

	// upcoming activities now have only the script
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[0].Type)

	// host's past activities do not have the VPP apps cancellation because those app installs
	// were not activated
	var listPastResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", mdmHost.ID), nil, http.StatusOK, &listPastResp)
	require.GreaterOrEqual(t, len(listPastResp.Activities), 0)

	// listing the host's software available for install shows none as MDM is now disabled
	// and no failure was recorded for the attempts (because the apps were not activated)
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	require.Len(t, getHostSw.Software, 0)

	// create another host that will receive the VPP install commands without any
	// other activity in front (so the first VPP install will be activated when
	// they are cancelled)
	mdmHost2, mdmDevice2 := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost2, s.ds)
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice2, true)
	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost2.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost2.ID}, TeamID: &team.ID}, http.StatusOK)

	// trigger install of both apps on the host
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost2.ID, app1TitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost2.ID, app2TitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	// confirm the state of this host's upcoming activities
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 2)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), listResp.Activities[0].Type)
	require.Contains(t, string(*listResp.Activities[0].Details), fmt.Sprintf(`"app_store_id": %q`, app1.AdamID))
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), listResp.Activities[1].Type)
	require.Contains(t, string(*listResp.Activities[1].Details), fmt.Sprintf(`"app_store_id": %q`, app2.AdamID))

	// listing the host's software shows them as pending install
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[1].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[1].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)

	// turn off MDM for the host
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", mdmHost2.ID), nil, http.StatusNoContent)
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), fmt.Sprintf(`{"host_display_name":%q, "host_serial":%q, "installed_from_dep":false}`, mdmHost2.DisplayName(), mdmHost2.HardwareSerial), 0)

	// upcoming activities are now empty
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 0)

	// host's past activities should have the first VPP app cancellation because it was activated
	listPastResp = listActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", mdmHost2.ID), nil, http.StatusOK, &listPastResp)
	require.GreaterOrEqual(t, len(listPastResp.Activities), 1)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), listPastResp.Activities[0].Type)
	require.Contains(t, string(*listPastResp.Activities[0].Details), fmt.Sprintf(`"app_store_id": %q`, app1.AdamID))
	require.Contains(t, string(*listPastResp.Activities[0].Details), `"status": "failed_install"`)
	if len(listPastResp.Activities) > 1 {
		// the second activity should not be the cancellation of the second app
		require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), listPastResp.Activities[1].Type)
	}

	// listing the host's software available for install shows the cancelled app as failed
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)

	// upcoming activities on the control host are unaffected
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", controlHost.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), listResp.Activities[0].Type)
	require.Contains(t, string(*listResp.Activities[0].Details), fmt.Sprintf(`"app_store_id": %q`, app1.AdamID))
}

// for https://github.com/fleetdm/fleet/issues/32082
func (s *integrationMDMTestSuite) TestSoftwareTitleVPPAppSoftwarePackageConflict() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	oldApps := s.appleITunesSrvData
	t.Cleanup(func() { s.appleITunesSrvData = oldApps })
	s.appleITunesSrvData = map[string]string{
		"1": `{"bundleId": "com.example.dummy", "artworkUrl512": "https://example.com/images/1", "version": "1.0.0", "trackName": "DummyApp", "TrackID": 1}`,
		"2": `{"bundleId": "com.example.noversion", "artworkUrl512": "https://example.com/images/2", "version": "2.0.0", "trackName": "NoVersion", "TrackID": 2}`,
	}

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.setVPPTokenForTeam(team.ID)

	// Add VPP app 1 with bundle ID com.example.dummy (conflicts with DummyApp below)
	vppApp1 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
	}

	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: vppApp1.AdamID, SelfService: true}, http.StatusOK, &addAppResp)

	// add the NoVersion installer with bundle id com.example.noversion (conflicts with VPP app 2 below)
	pkgNoVersion := &fleet.UploadSoftwareInstallerPayload{
		Filename: "no_version.pkg",
		Title:    "NoVersion",
		TeamID:   &team.ID,
	}
	s.uploadSoftwareInstaller(t, pkgNoVersion, http.StatusOK, "")

	// list the software titles to get the title IDs
	var listSw listSoftwareTitlesResponse
	// s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	// spew.Dump(listSw.SoftwareTitles)

	// var app1TitleID, dummyTitleID uint
	// for _, sw := range listSw.SoftwareTitles {
	// 	switch sw.Name {
	// 	case vppApp1.Name:
	// 		app1TitleID = sw.ID
	// 	case payloadDummy.Title:
	// 		dummyTitleID = sw.ID
	// 	}
	// }

	// the Dummy installer has bundle id com.example.dummy, it should fail with a
	// conflict with VPP app 1
	pkgDummy := &fleet.UploadSoftwareInstallerPayload{
		Filename: "dummy_installer.pkg",
		Title:    "DummyApp",
		TeamID:   &team.ID,
	}
	// TODO(mna): this is the bug, should fail to upload due to conflict
	// s.uploadSoftwareInstaller(t, pkgDummy, http.StatusConflict, "zzz")
	s.uploadSoftwareInstaller(t, pkgDummy, http.StatusOK, "")

	// list the software titles to get the title IDs
	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	spew.Dump(listSw.SoftwareTitles)

	// Add VPP app 2 with bundle ID com.example.noversion (conflicts with NoVersion)
	vppApp2 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "2",
				Platform: fleet.MacOSPlatform,
			},
		},
	}

	res := s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: vppApp2.AdamID, SelfService: true}, http.StatusConflict)
	txt := extractServerErrorText(res.Body)
	require.Contains(t, txt, "NoVersion already has a package or app available for install on the Team 1 team.")

	// TODO(mna): test with batch-set (gitops) too

	// start the HTTP server to serve package installers
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/no_version.pkg":
			http.ServeFile(w, r, filepath.Join("testdata", "software-installers", "no_version.pkg"))
		case "/dummy_installer.pkg":
			http.ServeFile(w, r, filepath.Join("testdata", "software-installers", "dummy_installer.pkg"))
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}))
	t.Cleanup(srv.Close)

	// try to set DummyApp and NoVersion installers, but DummyApp conflicts with VPP app 1
	var batchResponse batchSetSoftwareInstallersResponse
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{
		Software: []*fleet.SoftwareInstallerPayload{
			{URL: srv.URL + "/no_version.pkg", SHA256: "4ba383be20c1020e416958ab10e3b472a4d5532a8cd94ed720d495a9c81958fe"},
			{URL: srv.URL + "/dummy_installer.pkg", SHA256: "7f679541ccfdb56094ca76117fd7cf75071c9d8f43bfd2a6c0871077734ca7c8"},
		},
	}, http.StatusAccepted, &batchResponse, "team_name", team.Name)
	batchResp := waitBatchSetSoftwareInstallers(t, &s.withServer, team.Name, batchResponse.RequestUUID)
	// TODO(mna): this is the bug, a package that conflicts can be batch-uploaded
	// require.Equal(t, fleet.BatchSetSoftwareInstallersStatusFailed, batchResp.Status)
	// require.Contains(t, batchResp.Message, "zzzz")
	require.Equal(t, fleet.BatchSetSoftwareInstallersStatusCompleted, batchResp.Status)

	fmt.Println(">>>> BEFORE BATCH VPP APPS")
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		mysql.DumpTable(t, q, "vpp_apps_teams")
		return nil
	})

	// batch-set the VPP apps, including one in conflict
	var batchAssociateResponse batchAssociateAppStoreAppsResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: "1"},
		{AppStoreID: "2"},
	}}, http.StatusOK, &batchAssociateResponse, "team_name", team.Name)
	// TODO(mna): this is another bug, it should fail for app store "2"
	require.Len(t, batchAssociateResponse.Apps, 2)

	fmt.Println(">>>> AFTER BATCH VPP APPS")
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		mysql.DumpTable(t, q, "vpp_apps_teams")
		return nil
	})
}
