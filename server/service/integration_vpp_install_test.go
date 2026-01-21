package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
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

	// Add vpp apps to team 1
	macOSApp := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "App 1",
		BundleIdentifier: "a-1",
		IconURL:          "https://example.com/images/1/512x512.png",
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
		IconURL:          "https://example.com/images/2/512x512.png",
		LatestVersion:    "2.0.0",
	}

	iPadOSApp := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "3",
				Platform: fleet.IPadOSPlatform,
			},
		},
		Name:             "App 3",
		BundleIdentifier: "c-3",
		IconURL:          "https://example.com/images/3/512x512.png",
		LatestVersion:    "3.0.0",
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

	// Add iPadOS app to team
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: iPadOSApp.AdamID, SelfService: false, Platform: iPadOSApp.Platform}, http.StatusOK, &addAppResp)

	s.lastActivityMatches(fleet.ActivityAddedAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"team_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "platform": "%s", "self_service": false}`, team.Name,
			iPadOSApp.Name, getSoftwareTitleIDFromApp(iPadOSApp), iPadOSApp.AdamID, team.ID, iPadOSApp.Platform), 0)

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
		IconURL:          "https://example.com/images/2/512x512.png",
		LatestVersion:    "2.0.1", // macOS has different version than iOS
	}
	expectedApps := []*fleet.VPPApp{macOSApp, errApp, iOSApp, iPadOSApp}
	expectedAppsByBundleID := map[string]*fleet.VPPApp{
		macOSApp.BundleIdentifier:  macOSApp,
		errApp.BundleIdentifier:    errApp,
		iOSApp.BundleIdentifier:    iOSApp,
		iPadOSApp.BundleIdentifier: iPadOSApp,
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
		require.Equal(t, expected.IconURL, *got.IconUrl)
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
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "from_auto_update": false, "policy_id": null, "policy_name": null, "host_platform": "%s"}`,
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
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "from_auto_update": false, "policy_id": null, "policy_name": null, "host_platform": "%s"}`,
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
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "from_auto_update": false, "policy_id": null, "policy_name": null, "host_platform": "%s"}`,
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
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "from_auto_update": false, "policy_id": null, "policy_name": null, "host_platform": "%s"}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			addedApp.Name,
			addedApp.AdamID,
			installCmdUUID,
			fleet.SoftwareInstallFailed,
			mdmHost.Platform,
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
	mdmDevice = enrollMacOSHostInMDMManually(t, mdmHost, s.ds, s.server.URL)
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

	var ipadosTitleID uint
	for _, sw := range listSw.SoftwareTitles {
		if sw.Name == iPadOSApp.Name && sw.Source == "ipados_apps" {
			ipadosTitleID = sw.ID
			break
		}
	}
	require.NotZero(t, ipadosTitleID)

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
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": false, "from_auto_update": false, "policy_id": null, "policy_name": null, "host_platform": "%s"}`,
			iosHost.ID,
			iosHost.DisplayName(),
			iOSApp.Name,
			iOSApp.AdamID,
			installCmdUUID,
			fleet.SoftwareInstalled,
			iosHost.Platform,
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

	// ========================================================
	// Test iOS VPP app self service installation
	// ========================================================

	type SSVPPTestData struct {
		host       *fleet.Host
		app        *fleet.VPPApp
		device     *mdmtest.TestAppleMDMClient
		platform   string
		titleID    uint
		certSerial uint64
	}
	// Edit iOS app to enable self service
	require.NotZero(t, iosTitleID)
	require.NotZero(t, ipadosTitleID)

	ssVppData := []SSVPPTestData{
		{platform: "ios", titleID: iosTitleID, app: iOSApp, certSerial: uint64(1111)},
		{platform: "ipados", titleID: ipadosTitleID, app: iPadOSApp, certSerial: uint64(2222)},
	}

	for i, data := range ssVppData {
		// Enroll device, add serial number to fake Apple server, and transfer to team
		data.host, data.device = s.createAppleMobileHostThenEnrollMDM(data.platform)
		s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, data.device.SerialNumber)
		s.Do("POST", "/api/latest/fleet/hosts/transfer",
			&addHostsToTeamRequest{HostIDs: []uint{data.host.ID}, TeamID: &team.ID}, http.StatusOK)

		// Refresh host to get UUID
		data.host, err = s.ds.Host(context.Background(), data.host.ID)
		require.NoError(t, err)

		// Use certificate authentication
		headers := map[string]string{
			"X-Client-Cert-Serial": fmt.Sprintf("%d", data.certSerial),
		}
		s.addHostIdentityCertificate(data.host.UUID, data.certSerial)

		// self-install without cert header (UUID auth fallback for iOS/iPadOS)
		// With fallback auth, UUID auth succeeds for iOS/iPadOS devices, so we get 400 (bad title) instead of 401
		res := s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", data.host.UUID, 999), nil, http.StatusBadRequest)
		errMsg := extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Software title is not available for install.")

		// self-install a non-existing title (with cert header - same result)
		res = s.DoRawWithHeaders("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", data.host.UUID, 999), nil, http.StatusBadRequest, headers)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Software title is not available for install.")

		// self-install an existing title not available for self-install
		res = s.DoRawWithHeaders("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", data.host.UUID, data.titleID), nil, http.StatusBadRequest, headers)
		errMsg = extractServerErrorText(res.Body)
		require.Contains(t, errMsg, "Software title is not available through self-service")

		// Enable self-service for vpp app
		updateAppResp := updateAppStoreAppResponse{}
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", data.titleID),
			&updateAppStoreAppRequest{TitleID: data.titleID, TeamID: &team.ID, SelfService: ptr.Bool(true)}, http.StatusOK, &updateAppResp)

		// Install self-service app correctly
		s.DoRawWithHeaders("POST", fmt.Sprintf("/api/latest/fleet/device/%s/software/install/%d", data.host.UUID, data.titleID), nil, http.StatusAccepted, headers)

		// Verify pending status
		countResp = countHostsResponse{}
		s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
			fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(data.titleID))
		require.Equal(t, 1, countResp.Count)

		// Simulate successful installation on device
		opts := vppInstallOpts{
			appInstallTimeout:  false,
			appInstallVerified: true,
			failOnInstall:      false,
			bundleID:           data.app.BundleIdentifier,
		}
		installCmdUUID = processVPPInstallOnClient(data.device, opts)

		// Verify successful installation
		listResp = listHostsResponse{}
		s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id",
			fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(data.titleID))
		assert.Len(t, listResp.Hosts, 2-i)
		assert.Equal(t, data.host.ID, listResp.Hosts[len(listResp.Hosts)-1].ID)

		// Verify activity log entry
		s.lastActivityMatches(
			fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
			fmt.Sprintf(
				`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": true, "policy_id": null, "policy_name": null, "host_platform": "%s", "from_auto_update": false}`,
				data.host.ID,
				data.host.DisplayName(),
				data.app.Name,
				data.app.AdamID,
				installCmdUUID,
				fleet.SoftwareInstalled,
				data.host.Platform,
			),
			0,
		)
	}
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
		IconURL:          "https://example.com/images/1/512x512.png",
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
		IconURL:          "https://example.com/images/2/512x512.png",
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
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), fmt.Sprintf(`{"enrollment_id": null, "host_display_name":%q, "host_serial":%q, "installed_from_dep":false, "platform": "darwin"}`, mdmHost.DisplayName(), mdmHost.HardwareSerial), 0)

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
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), fmt.Sprintf(`{"enrollment_id": null, "host_display_name":%q, "host_serial":%q, "installed_from_dep":false, "platform": "darwin"}`, mdmHost2.DisplayName(), mdmHost2.HardwareSerial), 0)

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

	s.registerResetVPPProxyData(t)

	s.appleVPPProxySrvData = map[string]string{
		"1": `{"id": "1", "attributes": {"name": "DummyApp", "platformAttributes": {"osx": {"bundleId": "com.example.dummy", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["mac"]}}`,
		"2": `{"id": "2", "attributes": {"name": "NoVersion", "platformAttributes": {"osx": {"bundleId": "com.example.noversion", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "2.0.0"}}}, "deviceFamilies": ["mac"]}}`,
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

	// the Dummy installer has bundle id com.example.dummy, it should fail with a
	// conflict with VPP app 1
	pkgDummy := &fleet.UploadSoftwareInstallerPayload{
		Filename: "dummy_installer.pkg",
		Title:    "DummyApp",
		TeamID:   &team.ID,
	}
	s.uploadSoftwareInstaller(t, pkgDummy, http.StatusConflict, "DummyApp already has an installer available for the Team 1 team.")

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
	require.Contains(t, txt, "NoVersion already has an installer available for the Team 1 team.")

	// --- test with batch-set (gitops) ---

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
	require.Equal(t, fleet.BatchSetSoftwareInstallersStatusFailed, batchResp.Status)
	require.Contains(t, batchResp.Message, "DummyApp already has an installer available for the Team 1 team.")

	// batch-set the VPP apps, including one in conflict
	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: "1"},
		{AppStoreID: "2"},
	}}, http.StatusConflict, "team_name", team.Name)
	txt = extractServerErrorText(res.Body)
	require.Contains(t, txt, "NoVersion already has an installer available for the Team 1 team.")

	// listing software available to install only lists the dummy app and noversion installer
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 2)
	require.Equal(t, "DummyApp", listSw.SoftwareTitles[0].Name)
	require.NotNil(t, listSw.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "1", listSw.SoftwareTitles[0].AppStoreApp.AppStoreID)
	require.Equal(t, "NoVersion", listSw.SoftwareTitles[1].Name)
	require.NotNil(t, listSw.SoftwareTitles[1].SoftwarePackage)
	require.Equal(t, "no_version.pkg", listSw.SoftwareTitles[1].SoftwarePackage.Name)

	// a different team can batch-add the two installers without conflict
	newTeamResp = teamResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 2")}}, http.StatusOK, &newTeamResp)
	team2 := newTeamResp.Team

	batchResponse = batchSetSoftwareInstallersResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/batch", batchSetSoftwareInstallersRequest{
		Software: []*fleet.SoftwareInstallerPayload{
			{URL: srv.URL + "/no_version.pkg", SHA256: "4ba383be20c1020e416958ab10e3b472a4d5532a8cd94ed720d495a9c81958fe"},
			{URL: srv.URL + "/dummy_installer.pkg", SHA256: "7f679541ccfdb56094ca76117fd7cf75071c9d8f43bfd2a6c0871077734ca7c8"},
		},
	}, http.StatusAccepted, &batchResponse, "team_name", team2.Name)
	batchResp = waitBatchSetSoftwareInstallers(t, &s.withServer, team2.Name, batchResponse.RequestUUID)
	require.Equal(t, fleet.BatchSetSoftwareInstallersStatusCompleted, batchResp.Status)
	require.Empty(t, batchResp.Message)
	require.Len(t, batchResp.Packages, 2)

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team2.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 2)
	// both are software packages
	require.Equal(t, "DummyApp", listSw.SoftwareTitles[0].Name)
	require.NotNil(t, listSw.SoftwareTitles[0].SoftwarePackage)
	require.Equal(t, "dummy_installer.pkg", listSw.SoftwareTitles[0].SoftwarePackage.Name)
	require.Equal(t, "NoVersion", listSw.SoftwareTitles[1].Name)
	require.NotNil(t, listSw.SoftwareTitles[1].SoftwarePackage)
	require.Equal(t, "no_version.pkg", listSw.SoftwareTitles[1].SoftwarePackage.Name)

	// a different team can batch-add the two VPP apps without conflict
	newTeamResp = teamResponse{}
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 3")}}, http.StatusOK, &newTeamResp)
	team3 := newTeamResp.Team

	var tokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &tokenResp)
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", tokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID, team3.ID}}, http.StatusOK, &resPatchVPP)

	var batchAppResp batchAssociateAppStoreAppsResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps/batch", batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: "1"},
		{AppStoreID: "2"},
	}}, http.StatusOK, &batchAppResp, "team_name", team3.Name)
	require.Len(t, batchAppResp.Apps, 2)

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team3.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 2)
	// both are VPP apps
	require.Equal(t, "DummyApp", listSw.SoftwareTitles[0].Name)
	require.NotNil(t, listSw.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "1", listSw.SoftwareTitles[0].AppStoreApp.AppStoreID)
	require.Equal(t, "NoVersion", listSw.SoftwareTitles[1].Name)
	require.NotNil(t, listSw.SoftwareTitles[1].AppStoreApp)
	require.Equal(t, "2", listSw.SoftwareTitles[1].AppStoreApp.AppStoreID)
}

func (s *integrationMDMTestSuite) TestInHouseAppInstall() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	// Enroll iPhone
	iosHost, iosDevice := s.createAppleMobileHostThenEnrollMDM("ios")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, iosDevice.SerialNumber)

	// Create a label
	clr := createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{
			Name:    "foo",
			HostIDs: []uint{iosHost.ID},
		},
	}, http.StatusOK, &clr)

	// Upload in-house app for iOS, with the label as "exclude any"
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "ipa_test.ipa", LabelsExcludeAny: []string{"foo"}}, http.StatusOK, "")

	// Get title ID
	var titleID uint
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &titleID, "SELECT title_id FROM in_house_apps WHERE filename = 'ipa_test.ipa'")
	})

	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &resp, "team_id", "0")

	assert.Len(t, resp.SoftwareTitles, 2)
	assert.Equal(t, "ipa_test", resp.SoftwareTitles[0].Name)
	titleID = resp.SoftwareTitles[0].ID

	// Attempt installation on non-scoped app, should fail
	var installResp installSoftwareResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install",
		iosHost.ID, titleID), nil, http.StatusBadRequest, &installResp)

	// Update label to be include any, install should succeed
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{TitleID: titleID, Filename: "ipa_test.ipa", LabelsIncludeAny: []string{"foo"}}, http.StatusOK, "")

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install",
		iosHost.ID, titleID), nil, http.StatusAccepted, &installResp)

	var installCmdUUID string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installCmdUUID, "SELECT command_uuid FROM host_in_house_software_installs WHERE host_id = ?", iosHost.ID)
	})
	require.NotEmpty(t, installCmdUUID)

	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", iosHost.ID), nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 1)
	require.Equal(t, fleet.ActivityTypeInstalledSoftware{}.ActivityName(), listResp.Activities[0].Type)

	// Process the InstallApplication command
	s.runWorker()
	cmd, err := iosDevice.Idle()
	require.NoError(t, err)

	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		if cmd.Command.RequestType == "InstallApplication" {
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			assert.Equal(t, installCmdUUID, cmd.CommandUUID)

			// Points at the expected manifest URL
			expectedManifestURL := fmt.Sprintf("%s/api/latest/fleet/software/titles/%d/in_house_app/manifest?team_id=%d", s.server.URL, titleID, 0)
			assert.Contains(t, string(cmd.Raw), expectedManifestURL)

			cmd, err = iosDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	// Install verification command should be sent

	// Simulate a verification command not finding the app (maybe it takes a little while to install)
	s.runWorker()
	cmd, err = iosDevice.Idle()
	var cmd1 string
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType {
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmd1 = cmd.CommandUUID
			require.Contains(t, cmd.CommandUUID, fleet.VerifySoftwareInstallVPPPrefix)
			cmd, err = iosDevice.AcknowledgeInstalledApplicationList(
				iosDevice.UUID,
				cmd.CommandUUID,
				[]fleet.Software{},
			)
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
		}
	}

	s.runWorker()

	cmd, err = iosDevice.Idle()
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	var verificationCmdUUID string
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType {
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			assert.NotEqual(t, cmd1, cmd.CommandUUID)
			verificationCmdUUID = cmd.CommandUUID
			require.Contains(t, cmd.CommandUUID, fleet.VerifySoftwareInstallVPPPrefix)
			cmd, err = iosDevice.AcknowledgeInstalledApplicationList(
				iosDevice.UUID,
				cmd.CommandUUID,
				[]fleet.Software{
					{
						Name:             "test",
						BundleIdentifier: "com.ipa-test.ipa-test",
						Version:          "1.0",
						Installed:        true,
					},
				},
			)
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
		}
	}

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var install struct {
			CommandUUID         string     `db:"command_uuid"`
			VerificationCmdUUID string     `db:"verification_command_uuid"`
			VerificationAt      *time.Time `db:"verification_at"`
		}
		err = sqlx.GetContext(
			context.Background(),
			q,
			&install,
			"SELECT command_uuid, verification_command_uuid, verification_at FROM host_in_house_software_installs WHERE host_id = ?",
			iosHost.ID,
		)
		require.NoError(t, err)
		assert.Equal(t, installCmdUUID, install.CommandUUID)
		assert.Equal(t, verificationCmdUUID, install.VerificationCmdUUID)
		assert.NotNil(t, install.VerificationAt)

		return nil
	})

	// Get title and software package details
	var st getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", titleID),
		nil, http.StatusOK, &st)

	require.Equal(t, "ipa_test", st.SoftwareTitle.Name)
	require.Equal(t, "ipa_test.ipa", st.SoftwareTitle.SoftwarePackage.Name)
	require.Equal(t, "ios", st.SoftwareTitle.SoftwarePackage.Platform)
	require.WithinDuration(t, time.Now(), st.SoftwareTitle.SoftwarePackage.UploadedAt, time.Hour)
}

func (s *integrationMDMTestSuite) TestInHouseAppSelfInstall() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	// Enroll iPhone
	iosHost, iosDevice := s.createAppleMobileHostThenEnrollMDM("ios")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, iosDevice.SerialNumber)

	// Upload in-house app for iOS, not available in self-service for now
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "ipa_test.ipa"}, http.StatusOK, "")

	// get its title ID
	var resp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{
		SoftwareTitleListOptions: fleet.SoftwareTitleListOptions{Platform: "ios"},
	}, http.StatusOK, &resp, "team_id", "0")
	require.Len(t, resp.SoftwareTitles, 1)
	require.Equal(t, "ipa_test", resp.SoftwareTitles[0].Name)
	titleID := resp.SoftwareTitles[0].ID

	activityData := fmt.Sprintf(`{"software_title": "ipa_test", "software_package": "ipa_test.ipa", "team_name": null,
		"team_id": null, "self_service": false, "software_title_id": %d}`, titleID)
	s.lastActivityMatches(fleet.ActivityTypeAddedSoftware{}.ActivityName(), activityData, 0)

	// Add certificate authentication for iPhone
	iosHost, err := s.ds.Host(ctx, iosHost.ID)
	require.NoError(t, err)
	certSerial := uint64(123456789)
	headers := map[string]string{
		"X-Client-Cert-Serial": fmt.Sprintf("%d", certSerial),
	}
	s.addHostIdentityCertificate(iosHost.UUID, certSerial)

	// self-install without cert header (UUID auth fallback for iOS)
	// With fallback auth, UUID auth succeeds for iOS devices, so we get 400 (bad title) instead of 401
	res := s.DoRawNoAuth("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", iosHost.UUID, 999), nil, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Software title is not available for install.")

	// self-install a non-existing title (with cert header - same result)
	res = s.DoRawWithHeaders("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", iosHost.UUID, 999), nil, http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Software title is not available for install.")

	// self-install an existing title not available for self-install
	res = s.DoRawWithHeaders("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", iosHost.UUID, titleID), nil, http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Software title is not available through self-service")

	// update the in-house app to make it self-service
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{SelfService: ptr.Bool(true), TitleID: titleID, TeamID: nil},
		http.StatusOK, "")
	activityData = fmt.Sprintf(`{"software_title": "ipa_test", "software_package": "ipa_test.ipa", "software_display_name": "", "software_icon_url": null, "team_name": null,
		"team_id": null, "self_service": true, "software_title_id": %d}`, titleID)
	s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), activityData, 0)

	// self-install request is accepted
	s.DoRawWithHeaders("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", iosHost.UUID, titleID), nil, http.StatusAccepted, headers)

	var installCmdUUID string
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &installCmdUUID, "SELECT command_uuid FROM host_in_house_software_installs WHERE host_id = ?", iosHost.ID)
	})
	require.NotEmpty(t, installCmdUUID)

	// last activity is still "edited software" as the installed activity is created only when
	// the install is verified
	s.lastActivityMatches(fleet.ActivityTypeEditedSoftware{}.ActivityName(), "", 0)

	// Process the InstallApplication command
	s.runWorker()
	cmd, err := iosDevice.Idle()
	require.NoError(t, err)

	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		if cmd.Command.RequestType == "InstallApplication" {
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			assert.Equal(t, installCmdUUID, cmd.CommandUUID)

			// Points at the expected manifest URL
			expectedManifestURL := fmt.Sprintf("%s/api/latest/fleet/software/titles/%d/in_house_app/manifest?team_id=%d", s.server.URL, titleID, 0)
			assert.Contains(t, string(cmd.Raw), expectedManifestURL)

			cmd, err = iosDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	// Install verification command should be sent, acknowledge it
	cmd, err = iosDevice.Idle()
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType {
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			require.Contains(t, cmd.CommandUUID, fleet.VerifySoftwareInstallVPPPrefix)
			cmd, err = iosDevice.AcknowledgeInstalledApplicationList(
				iosDevice.UUID,
				cmd.CommandUUID,
				[]fleet.Software{
					{Name: "test", BundleIdentifier: "com.ipa-test.ipa-test", Version: "1.0", Installed: true},
				},
			)
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
		}
	}

	// installed activity is now created
	activityData = fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "command_uuid": %q, "install_uuid": "",
	"software_title": "ipa_test", "software_package": "", "self_service": true, "status": "installed",
	"policy_id": null, "policy_name": null}`, iosHost.ID, iosHost.DisplayName(), installCmdUUID)
	s.lastActivityMatches(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), activityData, 0)

	// host has no more upcoming activities
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", iosHost.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 0)

	// host has the past activity for the installed app
	var listPastResp listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", iosHost.ID), nil, http.StatusOK, &listPastResp)
	require.Len(t, listPastResp.Activities, 1)

	// update the app to have a label condition
	clr := createLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", createLabelRequest{
		LabelPayload: fleet.LabelPayload{Name: "L1", HostIDs: []uint{}},
	}, http.StatusOK, &clr)

	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID: titleID, TeamID: nil, LabelsIncludeAny: []string{"L1"},
	}, http.StatusOK, "")

	// self-install request is rejected
	res = s.DoRawWithHeaders("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", iosHost.UUID, titleID), nil, http.StatusBadRequest, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "This software is not available for this host.")

	// add the label to the host, so it can be installed
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", iosHost.ID), addLabelsToHostRequest{
		Labels: []string{"L1"},
	}, http.StatusOK, &addLabelsToHostResp)

	// self-install request is now accepted
	s.DoRawWithHeaders("POST", fmt.Sprintf("/api/v1/fleet/device/%s/software/install/%d", iosHost.UUID, titleID), nil, http.StatusAccepted, headers)
}

func (s *integrationMDMTestSuite) TestGetInHouseAppManifestUnsignedURL() {
	// Test that the Fleet URL is used if cloudfrontsigner is nil
	t := s.T()
	s.setSkipWorkerJobs(t)
	teamID := ptr.Uint(0)

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "ipa_test.ipa"}, http.StatusOK, "")

	var titleResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{
		SoftwareTitleListOptions: fleet.SoftwareTitleListOptions{Platform: "ios"},
	}, http.StatusOK, &titleResp, "team_id", "0")
	require.Len(t, titleResp.SoftwareTitles, 1)
	require.Equal(t, "ipa_test", titleResp.SoftwareTitles[0].Name)
	titleID := titleResp.SoftwareTitles[0].ID

	readManifest := func(res *http.Response) []byte {
		buf, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		res.Body.Close()
		return buf
	}
	res := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/in_house_app/manifest?team_id=%d", titleID, *teamID),
		jsonMustMarshal(t, getInHouseAppManifestRequest{TitleID: titleID, TeamID: teamID}), http.StatusOK)

	manifest := readManifest(res)
	require.NotNil(t, manifest)
	require.Contains(t, string(manifest), fmt.Sprintf("/%d/in_house_app?team_id=%d", titleID, *teamID))
}

func (s *integrationMDMTestSuite) addHostIdentityCertificate(hostUUID string, certSerial uint64) {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := context.Background()

	// Generate a real certificate for the device with proper SHA256 hash
	certPEM, certHash, _ := generateTestCertForDeviceAuth(t, certSerial, hostUUID)

	// Insert certificate data using the new nanomdm tables
	mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
		// Insert serial number
		_, err := db.ExecContext(ctx, `INSERT INTO scep_serials (serial) VALUES (?)`, certSerial)
		if err != nil {
			return err
		}

		// Insert certificate
		_, err = db.ExecContext(ctx, `
			INSERT INTO scep_certificates
			(serial, name, not_valid_before, not_valid_after, certificate_pem, revoked)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			certSerial,
			hostUUID,
			time.Now().Add(-24*time.Hour),
			time.Now().Add(365*24*time.Hour),
			certPEM,
			false,
		)
		if err != nil {
			return err
		}

		// Insert certificate association for device authentication
		_, err = db.ExecContext(ctx, `
			INSERT INTO nano_cert_auth_associations (id, sha256)
			VALUES (?, ?)
		`, hostUUID, certHash)
		return err
	})
}

// TestInHouseAppVPPConflict tests that IPA (in-house apps) and VPP iOS/iPadOS apps
// with the same bundle identifier cannot coexist on the same team.
func (s *integrationMDMTestSuite) TestInHouseAppVPPConflict() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	s.registerResetVPPProxyData(t)

	s.appleVPPProxySrvData = map[string]string{
		"100": `{"id": "100", "attributes": {"name": "IPA Test App", "platformAttributes": {"ios": {"bundleId": "com.ipa-test.ipa-test", "artwork": {"url": "https://example.com/images/100/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["iphone"]}}`,
		"101": `{"id": "101", "attributes": {"name": "IPA Test App iPad", "platformAttributes": {"ios": {"bundleId": "com.ipa-test.ipa-test", "artwork": {"url": "https://example.com/images/101/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["ipad"]}}`,
		"102": `{"id": "102", "attributes": {"name": "Different App", "platformAttributes": {"ios": {"bundleId": "com.example.different", "artwork": {"url": "https://example.com/images/102/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["iphone"]}}`,
	}

	originalAssets := s.appleVPPConfigSrvConfig.Assets
	t.Cleanup(func() { s.appleVPPConfigSrvConfig.Assets = originalAssets })

	s.appleVPPConfigSrvConfig.Assets = append(s.appleVPPConfigSrvConfig.Assets, vpp.Asset{
		AdamID:         "100",
		PricingParam:   "STDQ",
		AvailableCount: 10,
	}, vpp.Asset{
		AdamID:         "101",
		PricingParam:   "STDQ",
		AvailableCount: 10,
	}, vpp.Asset{
		AdamID:         "102",
		PricingParam:   "STDQ",
		AvailableCount: 10,
	})

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("IPA Conflict Team")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.setVPPTokenForTeam(team.ID)

	// Test Case 1: Upload IPA first, then try to add VPP iOS app with same bundle ID
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		Filename: "ipa_test.ipa",
		TeamID:   &team.ID,
	}, http.StatusOK, "")

	res := s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: "100",
		Platform:   "ios",
	}, http.StatusConflict)
	txt := extractServerErrorText(res.Body)
	require.Contains(t, txt, "already has an installer available for the IPA Conflict Team team.")

	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: "101",
		Platform:   "ipados",
	}, http.StatusConflict)
	txt = extractServerErrorText(res.Body)
	require.Contains(t, txt, "already has an installer available for the IPA Conflict Team team.")

	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		AppStoreID: "102",
		Platform:   "ios",
	}, http.StatusOK, &addAppResp)

	// Test Case 2: Add VPP iOS app first, then try to upload IPA with same bundle ID
	var newTeamResp2 teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("IPA Conflict Team 2")}}, http.StatusOK, &newTeamResp2)
	team2 := newTeamResp2.Team

	var tokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &tokenResp)
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", tokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID, team2.ID}}, http.StatusOK, &resPatchVPP)

	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team2.ID,
		AppStoreID: "100",
		Platform:   "ios",
	}, http.StatusOK, &addAppResp)

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		Filename: "ipa_test.ipa",
		TeamID:   &team2.ID,
	}, http.StatusConflict, "already has an installer available for the IPA Conflict Team 2 team.")

	// Test Case 3: Verify "No team" works correctly
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		Filename: "ipa_test.ipa",
		TeamID:   nil,
	}, http.StatusOK, "")

	var newTeamResp3 teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("IPA Conflict Team 3")}}, http.StatusOK, &newTeamResp3)
	team3 := newTeamResp3.Team

	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &tokenResp)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", tokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID, team2.ID, team3.ID, 0}}, http.StatusOK, &resPatchVPP)

	res = s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     nil,
		AppStoreID: "100",
		Platform:   "ios",
	}, http.StatusConflict)
	txt = extractServerErrorText(res.Body)
	require.Contains(t, txt, "already has an installer available for the No team team.")
}

func (s *integrationMDMTestSuite) TestVPPAppScheduledUpdates() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	ctx := t.Context()

	// Reset the VPP proxy data to what it was before this test
	s.registerResetVPPProxyData(t)

	vppAutoUpdateTest := func(t *testing.T, team *fleet.Team, host *fleet.Host, deviceClient *mdmtest.TestAppleMDMClient) {
		// Set an iOS and iPadOS app on the VPP response.
		s.appleVPPProxySrvData = map[string]string{
			"1": `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"ios": {"bundleId": "app-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["iphone", "ipad"]}}`,
		}

		if team.ID != 0 {
			// Transfer host to team.
			s.Do("POST", "/api/latest/fleet/hosts/transfer",
				&addHostsToTeamRequest{HostIDs: []uint{host.ID}, TeamID: &team.ID}, http.StatusOK)
		}

		// Add iOS VPP application.
		iOSVPPApp := &fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   "1",
					Platform: fleet.IOSPlatform,
				},
			},
		}

		// Add iOS app to the team.
		addAppResp := addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID:     &team.ID,
			AppStoreID: iOSVPPApp.AdamID,
			Platform:   iOSVPPApp.Platform,
		}, http.StatusOK, &addAppResp)

		// Add iPadOS VPP application.
		iPadOSVPPApp := &fleet.VPPApp{
			VPPAppTeam: fleet.VPPAppTeam{
				VPPAppID: fleet.VPPAppID{
					AdamID:   "1",
					Platform: fleet.IPadOSPlatform,
				},
			},
		}

		// Add iPadOS app to the team.
		addAppResp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID:     &team.ID,
			AppStoreID: iPadOSVPPApp.AdamID,
			Platform:   iPadOSVPPApp.Platform,
		}, http.StatusOK, &addAppResp)

		// Get title ID of the VPP app.
		var appTitleID uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &appTitleID, `SELECT title_id FROM vpp_apps WHERE adam_id = '1' AND platform = ?`, host.Platform)
		})
		require.NotZero(t, appTitleID)

		// Trigger install to the host
		installResp := installSoftwareResponse{}
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", host.ID, appTitleID), &installSoftwareRequest{},
			http.StatusAccepted, &installResp)

		// iOS device acknowledges the InstallApplication command.
		s.runWorker()
		cmd, err := deviceClient.Idle()
		require.NoError(t, err)
		require.Equal(t, "InstallApplication", cmd.Command.RequestType)
		// Acknowledge InstallApplication command
		var fullCmd micromdm.CommandPayload
		err = plist.Unmarshal(cmd.Raw, &fullCmd)
		require.NoError(t, err)
		installApplicationCommandUUID := cmd.CommandUUID
		cmd, err = deviceClient.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		require.Nil(t, cmd)

		// Fleet will return an InstalledApplicationList to verify the installation.
		//
		// iOS device processes such command, and simulates the software is
		// installed by returning in the list.
		s.runWorker()
		cmd, err = deviceClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
		fullCmd = micromdm.CommandPayload{}
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		require.Contains(t, cmd.CommandUUID, fleet.VerifySoftwareInstallVPPPrefix)
		cmd, err = deviceClient.AcknowledgeInstalledApplicationList(
			deviceClient.UUID,
			cmd.CommandUUID,
			[]fleet.Software{
				{
					Name:             "App 1",
					BundleIdentifier: "app-1",
					Version:          "1.0.0",
					Installed:        true,
				},
			},
		)
		require.NoError(t, err)

		// Check activity is generated for the installation.
		s.lastActivityMatches(
			fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
			fmt.Sprintf(
				`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "from_auto_update": false, "status": "%s", "self_service": false, "policy_id": null, "policy_name": null, "host_platform": "%s"}`,
				host.ID,
				host.DisplayName(),
				"App 1",
				"1",
				installApplicationCommandUUID,
				fleet.SoftwareInstalled,
				host.Platform,
			),
			0,
		)

		// Issue a refetch on the iOS host, and make sure the commands are queued.
		triggerRefetch := func() {
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", host.ID), nil, http.StatusOK)
			commands, err := s.ds.GetHostMDMCommands(context.Background(), host.ID)
			require.NoError(t, err)
			require.Len(t, commands, 3)
			assert.ElementsMatch(t, []fleet.HostMDMCommand{
				{HostID: host.ID, CommandType: fleet.RefetchAppsCommandUUIDPrefix},
				{HostID: host.ID, CommandType: fleet.RefetchCertsCommandUUIDPrefix},
				{HostID: host.ID, CommandType: fleet.RefetchDeviceCommandUUIDPrefix},
			}, commands)
		}

		handleRefetch := func(software []fleet.Software) {
			s.runWorker()

			// 1. InstalledApplicationList
			cmd, err = deviceClient.Idle()
			require.NoError(t, err)
			require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
			fullCmd = micromdm.CommandPayload{}
			err = plist.Unmarshal(cmd.Raw, &fullCmd)
			require.NoError(t, err)
			cmd, err = deviceClient.AcknowledgeInstalledApplicationList(
				deviceClient.UUID,
				cmd.CommandUUID,
				software,
			)
			require.NoError(t, err)

			// 2. CertificateList
			cmd, err = deviceClient.Idle()
			require.NoError(t, err)
			require.Equal(t, "CertificateList", cmd.Command.RequestType)
			var fullCmd micromdm.CommandPayload
			err := plist.Unmarshal(cmd.Raw, &fullCmd)
			require.NoError(t, err)
			cmd, err = deviceClient.AcknowledgeCertificateList(deviceClient.UUID, cmd.CommandUUID, nil)
			require.NoError(t, err)

			// 3. DeviceInformation
			cmd, err = deviceClient.Idle()
			require.NoError(t, err)
			require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
			fullCmd = micromdm.CommandPayload{}
			err = plist.Unmarshal(cmd.Raw, &fullCmd)
			require.NoError(t, err)
			deviceName := "iPhone 17"
			deviceProductName := "iPhone"
			if host.Platform == "ipados" {
				deviceName = "iPad 17"
				deviceProductName = "iPad"
			}
			cmd, err = deviceClient.AcknowledgeDeviceInformation(deviceClient.UUID, cmd.CommandUUID, deviceName, deviceProductName, "America/Los_Angeles")
			require.NoError(t, err)
		}

		// First refetch will populate the timezone of the device (because DeviceInformation command is always sent last in refetches).
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "1.0.0",
				Installed:        true,
			},
		})

		// Reload information after the refetch.
		host, err = s.ds.Host(ctx, host.ID)
		require.NoError(t, err)

		// Second refetch should perform no auto updates of any kind (nothing configured yet).
		triggerRefetch()
		lastActivityID := s.lastActivityMatches(
			fleet.ActivityInstalledAppStoreApp{}.ActivityName(), "", 0,
		)
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "1.0.0",
				Installed:        true,
			},
		})
		// No new activity is created (no update yet).
		s.lastActivityMatches(fleet.ActivityInstalledAppStoreApp{}.ActivityName(), "", lastActivityID) // no new activity yet

		// Configure auto-updates on the VPP app on a time that is currently not now in America/Los_Angeles.
		nowInLosAngeles, err := getCurrentLocalTimeInHostTimeZone(ctx, "America/Los_Angeles")
		require.NoError(t, err)
		endTime := nowInLosAngeles.Add(-1 * time.Minute)
		startTime := endTime.Add(-1 * time.Hour)
		startTimeHHMM := startTime.Format("15:04")
		endTimeHHMM := endTime.Format("15:04")
		var updateAppStoreAppResponsePayload updateAppStoreAppResponse
		t.Logf("Time in America/Los_Angeles: %s, window = [%s, %s]", nowInLosAngeles, startTimeHHMM, endTimeHHMM)
		s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/software/titles/%d/app_store_app", appTitleID), updateAppStoreAppRequest{
			TeamID:              &team.ID,
			AutoUpdateEnabled:   ptr.Bool(true),
			AutoUpdateStartTime: ptr.String(startTimeHHMM),
			AutoUpdateEndTime:   ptr.String(endTimeHHMM),
		}, http.StatusOK, &updateAppStoreAppResponsePayload)

		// Refetch should perform no auto updates of any kind because the host is not in the configured time window.
		lastActivityID = s.lastActivityMatches(
			fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", 0,
		)
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "1.0.0",
				Installed:        true,
			},
		})
		// No new activity is created (no update yet).
		s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", lastActivityID) // no new activity yet

		// Configure auto-updates on the VPP app on a time that is currently in America/Los_Angeles.
		nowInLosAngeles, err = getCurrentLocalTimeInHostTimeZone(ctx, "America/Los_Angeles")
		require.NoError(t, err)
		startTime = nowInLosAngeles.Add(-30 * time.Minute)
		endTime = endTime.Add(1 * time.Hour)
		startTimeHHMM = startTime.Format("15:04")
		endTimeHHMM = endTime.Format("15:04")
		updateAppStoreAppResponsePayload = updateAppStoreAppResponse{}
		t.Logf("Time in America/Los_Angeles: %s, window = [%s, %s]", nowInLosAngeles, startTimeHHMM, endTimeHHMM)
		s.DoJSON("PATCH", fmt.Sprintf("/api/v1/fleet/software/titles/%d/app_store_app", appTitleID), updateAppStoreAppRequest{
			TeamID:              &team.ID,
			AutoUpdateEnabled:   ptr.Bool(true),
			AutoUpdateStartTime: ptr.String(startTimeHHMM),
			AutoUpdateEndTime:   ptr.String(endTimeHHMM),
		}, http.StatusOK, &updateAppStoreAppResponsePayload)

		// Refetch, but should not auto-update because the app is currently in the latest version.
		lastActivityID = s.lastActivityMatches(
			fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", 0,
		)
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "1.0.0",
				Installed:        true,
			},
		})
		// No new activity is created (no update yet).
		s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", lastActivityID)

		// Update latest version of the app in VPP (simulate the app being updated in Apple App Store).
		s.appleVPPProxySrvData = map[string]string{
			"1": `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"ios": {"bundleId": "app-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "2.0.0"}}}, "deviceFamilies": ["iphone", "ipad"]}}`,
		}

		noopAuthenticator := func(bool) (string, error) { return "", nil } // authentication is tested elsewhere
		err = vpp.RefreshVersions(ctx, s.ds, noopAuthenticator)
		require.NoError(t, err)

		// Spoof the previous installation time to skip the installed-1-hour-ago filtering.
		mysql.ExecAdhocSQL(t, s.ds, func(db sqlx.ExtContext) error {
			_, err := db.ExecContext(ctx, `UPDATE host_vpp_software_installs SET created_at = DATE_SUB(NOW(), INTERVAL 2 HOUR);`)
			return err
		})

		// Refetch, should not trigger auto-update because the app is not listed in the application list.
		// This can happens when the app is in a state of downloaded but "still installing/initializing".
		lastActivityID = s.lastActivityMatches(
			fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", 0,
		)
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{},
		})
		// No new activity is created (no update yet).
		s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", lastActivityID)

		// Refetch, should not trigger auto-update because the app is listed but version is not provided in the application list.
		// This can happens when the app is in a state of downloaded but "still installing/initializing".
		lastActivityID = s.lastActivityMatches(
			fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", 0,
		)
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Installed:        true,
			},
		})
		// No new activity is created (no update yet).
		s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", lastActivityID)

		// Refetch, should not trigger auto-update because the app is listed with an invalid version string.
		// Just testing we handle such scenario.
		lastActivityID = s.lastActivityMatches(
			fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", 0,
		)
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "invalid",
				Installed:        true,
			},
		})
		// No new activity is created (no update yet).
		s.lastActivityMatches(fleet.ActivityEditedAppStoreApp{}.ActivityName(), "", lastActivityID)

		// Refetch, should trigger auto-update because the app is currently not in the latest version.
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "1.0.0",
				Installed:        true,
			},
		})

		// iOS device acknowledges the InstallApplication command associated to the auto-update.
		s.runWorker()
		cmd, err = deviceClient.Idle()
		require.NoError(t, err)
		require.Equal(t, "InstallApplication", cmd.Command.RequestType)
		// Acknowledge InstallApplication command
		fullCmd = micromdm.CommandPayload{}
		err = plist.Unmarshal(cmd.Raw, &fullCmd)
		require.NoError(t, err)
		installApplicationCommandUUID = cmd.CommandUUID
		cmd, err = deviceClient.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
		require.Nil(t, cmd)

		// Fleet will return an InstalledApplicationList to verify the installation.
		// Return the application with the latest version 2.0.0 (simulating the update was successful).
		s.runWorker()
		cmd, err = deviceClient.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd)
		require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
		fullCmd = micromdm.CommandPayload{}
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
		require.Contains(t, cmd.CommandUUID, fleet.VerifySoftwareInstallVPPPrefix)
		cmd, err = deviceClient.AcknowledgeInstalledApplicationList(
			deviceClient.UUID,
			cmd.CommandUUID,
			[]fleet.Software{
				{
					Name:             "App 1",
					BundleIdentifier: "app-1",
					Version:          "2.0.0",
					Installed:        true,
				},
			},
		)
		require.NoError(t, err)

		// Check activity is generated for the installation.
		lastActivityID = s.lastActivityMatches(
			fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
			fmt.Sprintf(
				// See `"from_auto_update": true`.
				`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "from_auto_update": true, "status": "%s", "self_service": false, "policy_id": null, "policy_name": null, "host_platform": "%s"}`,
				host.ID,
				host.DisplayName(),
				"App 1",
				"1",
				installApplicationCommandUUID,
				fleet.SoftwareInstalled,
				host.Platform,
			),
			0,
		)

		// Trigger a refetch to refresh software inventory.
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "2.0.0",
				Installed:        true,
			},
		})

		// Check the host software inventory is updated.
		software, _, err := s.ds.ListSoftware(ctx, fleet.SoftwareListOptions{
			HostID: &host.ID,
		})
		require.NoError(t, err)
		require.Len(t, software, 1)
		require.Equal(t, "2.0.0", software[0].Version)

		// Update latest version of the app in VPP again (simulate the app being updated in Apple App Store).
		s.appleVPPProxySrvData = map[string]string{
			"1": `{"id": "1", "attributes": {"name": "App 1", "platformAttributes": {"ios": {"bundleId": "app-1", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "3.0.0"}}}, "deviceFamilies": ["iphone", "ipad"]}}`,
		}
		err = vpp.RefreshVersions(ctx, s.ds, noopAuthenticator)
		require.NoError(t, err)

		// Refetch, should not trigger auto-update because the app was recently updated (in the last hour).
		// Register the previous activity id for the install.
		triggerRefetch()
		handleRefetch([]fleet.Software{
			{
				Name:             "App 1",
				BundleIdentifier: "app-1",
				Version:          "2.0.0",
				Installed:        true,
			},
		})
		// No new activity.
		s.lastActivityMatches(fleet.ActivityInstalledAppStoreApp{}.ActivityName(), "", lastActivityID)
	}

	// Create a team and a VPP token on it.
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team
	s.setVPPTokenForTeam(team.ID)

	// Enroll iOS device, and add serial number to fake Apple server (for VPP APIs).
	iosHost, iosClientDevice := s.createAppleMobileHostThenEnrollMDM("ios")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, iosClientDevice.SerialNumber)

	t.Run("iphone-on-a-team", func(t *testing.T) {
		vppAutoUpdateTest(t, team, iosHost, iosClientDevice)
	})

	// Enroll iPadOS device, and add serial number to fake Apple server (for VPP APIs).
	ipadosHost, ipadosClientDevice := s.createAppleMobileHostThenEnrollMDM("ipados")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, ipadosClientDevice.SerialNumber)

	t.Run("ipad-on-a-team", func(t *testing.T) {
		vppAutoUpdateTest(t, team, ipadosHost, ipadosClientDevice)
	})

	// Enroll iOS device, and add serial number to fake Apple server (for VPP APIs).
	iosHostNoTeam, iosClientDeviceNoTeam := s.createAppleMobileHostThenEnrollMDM("ios")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, iosClientDeviceNoTeam.SerialNumber)

	// Set VPP token for "No team".
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, "DELETE FROM vpp_tokens;")
		return err
	})
	s.setVPPTokenForTeam(0)

	t.Run("iphone-on-no-team", func(t *testing.T) {
		vppAutoUpdateTest(t, &fleet.Team{ID: 0}, iosHostNoTeam, iosClientDeviceNoTeam)
	})

	// Enroll iOS device, and add serial number to fake Apple server (for VPP APIs).
	ipadosHostNoTeam, ipadosClientDeviceNoTeam := s.createAppleMobileHostThenEnrollMDM("ipados")
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, ipadosClientDeviceNoTeam.SerialNumber)

	t.Run("ipad-on-no-team", func(t *testing.T) {
		vppAutoUpdateTest(t, &fleet.Team{ID: 0}, ipadosHostNoTeam, ipadosClientDeviceNoTeam)
	})
}

// Test for this special-case bugfix:
// https://github.com/fleetdm/fleet/issues/37290
func (s *integrationMDMTestSuite) TestVPPAppInstallVerificationXcodeSpecialCase() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	s.setVPPTokenForTeam(team.ID)

	// Reset the VPP proxy data to what it was before this test
	s.registerResetVPPProxyData(t)

	s.appleVPPProxySrvData = map[string]string{
		"1": `{"id": "1", "attributes": {"name": "Xcode", "platformAttributes": {"osx": {"bundleId": "com.apple.dt.Xcode", "artwork": {"url": "https://example.com/images/1/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["mac"]}}`,
		"2": `{"id": "2", "attributes": {"name": "App 2", "platformAttributes": {"osx": {"bundleId": "b-2", "artwork": {"url": "https://example.com/images/2/{w}x{h}.{f}"}, "latestVersionInfo": {"versionDisplay": "1.0.0"}}}, "deviceFamilies": ["mac"]}}`,
	}

	// Add apps targeting macOS to the team
	appXcode := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "1",
				Platform: fleet.MacOSPlatform,
			},
		},
		Name:             "Xcode",
		BundleIdentifier: "com.apple.dt.Xcode",
		IconURL:          "https://example.com/images/1/512x512.png",
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
		IconURL:          "https://example.com/images/2/512x512.png",
		LatestVersion:    "2.0.0",
	}

	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: appXcode.AdamID, SelfService: true}, http.StatusOK, &addAppResp)
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app2.AdamID, SelfService: false}, http.StatusOK, &addAppResp)

	// list the software titles to get the title IDs
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	var appXcodeTitleID, app2TitleID uint
	for _, sw := range listSw.SoftwareTitles {
		require.NotNil(t, sw.AppStoreApp)
		switch sw.Name {
		case appXcode.Name:
			appXcodeTitleID = sw.ID
		case app2.Name:
			app2TitleID = sw.ID
		}
	}

	// create a host that will receive the VPP install commands
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// trigger install of Xcode
	installResp := installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, appXcodeTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	// check that it starts polling with ManagedOnly=true, and switches to ManagedOnly=false once it stops reporting it as Installing
	processInstallAppCmds := func(device *mdmtest.TestAppleMDMClient, appList []fleet.Software, expectManagedOnly bool, expectCommands ...string) {
		s.runWorker()
		cmd, err := device.Idle()
		require.NoError(t, err)

		expectedCommandsSet := make(map[string]bool, len(expectCommands))
		for _, ec := range expectCommands {
			expectedCommandsSet[ec] = true
		}

		requestTypeSeen := make(map[string]bool)
		for cmd != nil {
			requestTypeSeen[cmd.Command.RequestType] = true

			var fullCmd micromdm.CommandPayload
			switch cmd.Command.RequestType {
			case "InstalledApplicationList":
				if !expectedCommandsSet["InstalledApplicationList"] {
					require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
				}

				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				require.Equal(t, expectManagedOnly, fullCmd.Command.InstalledApplicationList.ManagedAppsOnly)
				cmd, err = device.AcknowledgeInstalledApplicationList(
					device.UUID,
					cmd.CommandUUID,
					appList,
				)
				require.NoError(t, err)

			case "InstallApplication":
				if !expectedCommandsSet["InstallApplication"] {
					require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
				}

				cmd, err = device.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)

			default:
				require.Fail(t, "unexpected MDM command on client", cmd.Command.RequestType)
			}
		}

		for ec := range expectedCommandsSet {
			require.True(t, requestTypeSeen[ec], "expected %s command", ec)
		}
	}

	// first iteration, only the command to install the application
	processInstallAppCmds(mdmDevice, nil, true, "InstallApplication")

	// second iteration, return Xcode as installing
	processInstallAppCmds(mdmDevice,
		[]fleet.Software{
			{
				Name:             appXcode.Name,
				BundleIdentifier: appXcode.BundleIdentifier,
				Version:          appXcode.LatestVersion,
				Installed:        false, // installing
			},
		}, true, "InstalledApplicationList")

	// third iteration, stop returning Xcode
	processInstallAppCmds(mdmDevice,
		[]fleet.Software{}, true, "InstalledApplicationList")

	// fourth iteration, apps requested with non-managed too, to verify Xcode
	processInstallAppCmds(mdmDevice,
		[]fleet.Software{
			{
				Name:             appXcode.Name,
				BundleIdentifier: appXcode.BundleIdentifier,
				Version:          appXcode.LatestVersion,
				Installed:        true,
			},
			{
				Name:             "other",
				BundleIdentifier: "some.bundle",
				Version:          "1.2",
				Installed:        true,
			},
		}, false, "InstalledApplicationList")

	// check that it is properly verified as installed
	var listResp listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(appXcodeTitleID))
	require.Len(t, listResp.Hosts, 1)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)

	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "installed", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(appXcodeTitleID))
	require.Equal(t, 1, countResp.Count)

	// trigger install of app2
	installResp = installSoftwareResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, app2TitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	// first iteration, install app2
	processInstallAppCmds(mdmDevice, []fleet.Software{}, true, "InstallApplication")

	// second iteration, return app2 as installing
	processInstallAppCmds(mdmDevice,
		[]fleet.Software{
			{
				Name:             app2.Name,
				BundleIdentifier: app2.BundleIdentifier,
				Version:          app2.LatestVersion,
				Installed:        false, // installing
			},
		}, true, "InstalledApplicationList")

	// third iteration, do not return app2, which should not trigger the Xcode special-case
	processInstallAppCmds(mdmDevice, []fleet.Software{}, true, "InstalledApplicationList")

	// fourth iteration, return app2 as installed, still only managed apps were requested
	processInstallAppCmds(mdmDevice,
		[]fleet.Software{
			{
				Name:             app2.Name,
				BundleIdentifier: app2.BundleIdentifier,
				Version:          app2.LatestVersion,
				Installed:        true,
			},
		}, true, "InstalledApplicationList")

	// check that it is properly verified as installed
	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(app2TitleID))
	require.Len(t, listResp.Hosts, 1)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)

	// trigger install of both apps together on a different host
	mdmHost2, mdmDevice2 := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	mdmHost2.OrbitNodeKey = ptr.String(setOrbitEnrollment(t, mdmHost2, s.ds))
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice2, true)

	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost2.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost2.ID}, TeamID: &team.ID}, http.StatusOK)

	// enqueue a script execution first, so that when it's marked as executed, both
	// vpp app installs activate at the same time (VPP app installs get batch-activated
	// when they are consecutive in the upcoming queue)
	var runResp runScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: mdmHost2.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost2.ID, appXcodeTitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost2.ID, app2TitleID), &installSoftwareRequest{},
		http.StatusAccepted, &installResp)

	// check the host's upcoming activities, should be the script and 2 VPP app installs
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 3)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), hostActivitiesResp.Activities[0].Type)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), hostActivitiesResp.Activities[1].Type)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), hostActivitiesResp.Activities[2].Type)
	scriptExecID := hostActivitiesResp.Activities[0].UUID

	// set a result for the script, activating the 2 VPP installs next
	var orbitPostScriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *mdmHost2.OrbitNodeKey, scriptExecID)),
		http.StatusOK, &orbitPostScriptResp)

	// first iteration, install app2 and Xcode, report both as installing
	processInstallAppCmds(mdmDevice2,
		[]fleet.Software{
			{
				Name:             appXcode.Name,
				BundleIdentifier: appXcode.BundleIdentifier,
				Version:          appXcode.LatestVersion,
				Installed:        false, // installing
			},
			{
				Name:             app2.Name,
				BundleIdentifier: app2.BundleIdentifier,
				Version:          app2.LatestVersion,
				Installed:        false, // installing
			},
		}, true, "InstallApplication", "InstalledApplicationList")

	// second iteration, return app2 and Xcode as installing, no install command sent
	processInstallAppCmds(mdmDevice2,
		[]fleet.Software{
			{
				Name:             appXcode.Name,
				BundleIdentifier: appXcode.BundleIdentifier,
				Version:          appXcode.LatestVersion,
				Installed:        false, // installing
			},
			{
				Name:             app2.Name,
				BundleIdentifier: app2.BundleIdentifier,
				Version:          app2.LatestVersion,
				Installed:        false, // installing
			},
		}, true, "InstalledApplicationList")

	// third iteration, return app2 as installed, Xcode removed
	processInstallAppCmds(mdmDevice2,
		[]fleet.Software{
			{
				Name:             app2.Name,
				BundleIdentifier: app2.BundleIdentifier,
				Version:          app2.LatestVersion,
				Installed:        true,
			},
		}, true, "InstalledApplicationList")

	// fourth iteration, return Xcode and app2 as installed, not-managed-only requested
	processInstallAppCmds(mdmDevice2,
		[]fleet.Software{
			{
				Name:             app2.Name,
				BundleIdentifier: app2.BundleIdentifier,
				Version:          app2.LatestVersion,
				Installed:        true,
			},
			{
				Name:             appXcode.Name,
				BundleIdentifier: appXcode.BundleIdentifier,
				Version:          appXcode.LatestVersion,
				Installed:        true,
			},
		}, false, "InstalledApplicationList")

	// check that both are properly verified as installed
	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(app2TitleID), "order_key", "h.id")
	require.Len(t, listResp.Hosts, 2)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)
	require.Equal(t, listResp.Hosts[1].ID, mdmHost2.ID)

	listResp = listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listResp, "software_status", "installed", "team_id", fmt.Sprint(team.ID),
		"software_title_id", fmt.Sprint(appXcodeTitleID), "order_key", "h.id")
	require.Len(t, listResp.Hosts, 2)
	require.Equal(t, listResp.Hosts[0].ID, mdmHost.ID)
	require.Equal(t, listResp.Hosts[1].ID, mdmHost2.ID)
}
