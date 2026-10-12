package service

// Upcoming activity tests for the MDM suite.
//
// Belongs here: cancelling upcoming activities (including lock and wipe) and what happens to upcoming
// activities when MDM is turned off.
//
// Does not belong here: wipes that cancel upcoming activities (integration_mdm_commands_test.go) and
// the past activities feed.

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestUpcomingActivitiesTurnMDMOff() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// Set up VPP token
	orgName := "Fleet Device Management Inc."
	token := "validtoken"
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

	// Create a couple of hosts
	mdmHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	key := setOrbitEnrollment(t, mdmHost, s.ds)
	mdmHost.OrbitNodeKey = &key
	mdmHost2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	key2 := setOrbitEnrollment(t, mdmHost2, s.ds)
	mdmHost2.OrbitNodeKey = &key2

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial, mdmHost2.HardwareSerial)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	// add VPP app
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appResp, "team_id", "0")
	require.NoError(t, appResp.Err)
	require.True(t, len(appResp.AppStoreApps) > 0)
	addedApp := appResp.AppStoreApps[0]

	var addedMacOSApp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     nil,
		Platform:   addedApp.Platform,
		AppStoreID: addedApp.AdamID,
	}, http.StatusOK, &addedMacOSApp)

	// Get software title ID of the added VPP app
	listSWTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{},
		http.StatusOK, &listSWTitlesResp, "query", addedApp.Name, "team_id", "0")
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	vppAppTitleID := listSWTitlesResp.SoftwareTitles[0].ID

	// make the VPP app install pending on both hosts
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, vppAppTitleID), &installSoftwareRequest{}, http.StatusAccepted)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost2.ID, vppAppTitleID), &installSoftwareRequest{}, http.StatusAccepted)

	// also enqueue a script execution request on host 1
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: mdmHost.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)

	// host 1 has 2 upcoming activities, the vpp app and the script
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 2)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), hostActivitiesResp.Activities[0].Type)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), hostActivitiesResp.Activities[1].Type)
	scriptExecID := hostActivitiesResp.Activities[1].UUID

	// host 2 has the VPP app
	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), hostActivitiesResp.Activities[0].Type)
	host2VppAppExecID := hostActivitiesResp.Activities[0].UUID

	// turn off MDM for host 1
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", mdmHost.ID), nil, http.StatusNoContent)

	// confirm that this host's MDM is off
	res := s.Do("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", mdmHost.ID), nil, http.StatusOK)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	var m map[string]any
	err = json.Unmarshal(b, &m)
	require.NoError(t, err)
	require.Equal(t, "Off", m["enrollment_status"])

	// host 1 has only the script remaining
	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), hostActivitiesResp.Activities[0].Type)
	require.Equal(t, scriptExecID, hostActivitiesResp.Activities[0].UUID)

	// host 2 still has the VPP app
	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	require.Equal(t, fleet.ActivityInstalledAppStoreApp{}.ActivityName(), hostActivitiesResp.Activities[0].Type)
	require.Equal(t, host2VppAppExecID, hostActivitiesResp.Activities[0].UUID)

	// save a script result for host 1, as the script should've been activated
	// when MDM was turned off
	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *mdmHost.OrbitNodeKey, scriptExecID)),
		http.StatusOK, &orbitPostScriptResp)

	// host 1 has no remaining upcoming activity
	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 0)
}

func (s *integrationMDMTestSuite) TestCancelUpcomingActivity() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// Set up VPP token
	orgName := "Fleet Device Management Inc."
	token := "validtoken"
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

	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	key := setOrbitEnrollment(t, mdmHost, s.ds)
	mdmHost.OrbitNodeKey = &key

	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)

	// Associate no team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &resPatchVPP)

	// add VPP app
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", nil, http.StatusOK, &appResp, "team_id", "0")
	require.NoError(t, appResp.Err)
	require.True(t, len(appResp.AppStoreApps) > 0)
	addedApp := appResp.AppStoreApps[0]

	var addedMacOSApp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     nil,
		Platform:   addedApp.Platform,
		AppStoreID: addedApp.AdamID,
	}, http.StatusOK, &addedMacOSApp)

	// Get software title ID of the added VPP app
	listSWTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{},
		http.StatusOK, &listSWTitlesResp, "query", addedApp.Name, "team_id", "0")
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	vppAppTitleID := listSWTitlesResp.SoftwareTitles[0].ID

	// create a software installer
	payload := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp",
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	swTitleID := getSoftwareTitleID(t, s.ds, payload.Title, "apps")

	// create a saved script
	script, err := s.ds.NewScript(t.Context(), &fleet.Script{Name: "test-script.sh", ScriptContents: "echo"})
	require.NoError(t, err)

	// enqueue a VPP app install and software install
	// sleep in-between to ensure deterministic ordering
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, vppAppTitleID), &installSoftwareRequest{}, http.StatusAccepted)
	time.Sleep(time.Millisecond)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, swTitleID), &installSoftwareRequest{}, http.StatusAccepted)
	time.Sleep(time.Millisecond)

	// check that all are upcoming, get the exec IDs
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 2)

	// record a success for VPP app
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstallApplication":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
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

	// record a failure for software install
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q, "install_uuid": %q, "pre_install_condition_output": "ok", "install_script_exit_code": 1, "install_script_output": "fail"
		}`, *mdmHost.OrbitNodeKey, hostActivitiesResp.Activities[1].UUID)), http.StatusNoContent)

	// Exhaust automatic retries for the failed software install.
	// Server-side retries queue up to MaxSoftwareInstallAttempts attempts.
	for attempt := 2; attempt <= fleet.MaxSoftwareInstallAttempts; attempt++ {
		hostActivitiesResp = listHostUpcomingActivitiesResponse{}
		s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID), nil, http.StatusOK, &hostActivitiesResp)
		require.Len(t, hostActivitiesResp.Activities, 1, "should have pending retry (attempt %d)", attempt)
		s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
				"orbit_node_key": %q, "install_uuid": %q, "pre_install_condition_output": "ok", "install_script_exit_code": 1, "install_script_output": "fail"
			}`, *mdmHost.OrbitNodeKey, hostActivitiesResp.Activities[0].UUID)), http.StatusNoContent)
	}

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 0)

	// enqueue a script run and an uninstall
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: mdmHost.ID, ScriptID: &script.ID}, http.StatusAccepted, &runResp)
	time.Sleep(time.Millisecond)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", mdmHost.ID, swTitleID), &uninstallSoftwareRequest{}, http.StatusAccepted)

	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 2)

	// the script shows up for the host's script runs
	var scriptResp fleet.GetHostScriptDetailsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", mdmHost.ID), nil, http.StatusOK, &scriptResp)
	require.Len(t, scriptResp.Scripts, 1)
	require.NotNil(t, scriptResp.Scripts[0].LastExecution)
	require.Equal(t, "pending", scriptResp.Scripts[0].LastExecution.Status)

	// cancel the script, confirm canceled activity
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", mdmHost.ID, hostActivitiesResp.Activities[0].UUID), nil, http.StatusNoContent)
	lastCanceledActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeCanceledRunScript{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "script_name": "test-script.sh"}`, mdmHost.ID, mdmHost.DisplayName()), 0)

	// record a script result post-cancelation
	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *mdmHost.OrbitNodeKey, hostActivitiesResp.Activities[0].UUID)),
		http.StatusOK, &orbitPostScriptResp)

	// must not generate a past activity
	s.lastActivityMatches(fleet.ActivityTypeCanceledRunScript{}.ActivityName(), "", lastCanceledActID)

	// cancel the uninstall, confirm canceled activity
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", mdmHost.ID, hostActivitiesResp.Activities[1].UUID), nil, http.StatusNoContent)
	lastCanceledActID = s.lastActivityOfTypeMatches(fleet.ActivityTypeCanceledUninstallSoftware{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "software_title": "DummyApp", "software_title_id": %d}`, mdmHost.ID, mdmHost.DisplayName(), swTitleID), 0)

	// record an uninstall result post-cancelation
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *mdmHost.OrbitNodeKey,
			hostActivitiesResp.Activities[1].UUID)),
		http.StatusOK, &orbitPostScriptResp)

	// must not generate a past activity
	s.lastActivityMatches(fleet.ActivityTypeCanceledUninstallSoftware{}.ActivityName(), "", lastCanceledActID)

	// enqueue a VPP app install and software install again
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, vppAppTitleID), &installSoftwareRequest{}, http.StatusAccepted)
	time.Sleep(time.Millisecond)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", mdmHost.ID, swTitleID), &installSoftwareRequest{}, http.StatusAccepted)

	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 2)

	// cancel the VPP app install, confirm canceled activity
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", mdmHost.ID, hostActivitiesResp.Activities[0].UUID), nil, http.StatusNoContent)
	lastCanceledActID = s.lastActivityOfTypeMatches(fleet.ActivityTypeCanceledInstallAppStoreApp{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "software_title": "App 1", "software_title_id": %d, "from_setup_experience": false}`, mdmHost.ID, mdmHost.DisplayName(), vppAppTitleID), 0)

	// to be able to simulate the host sending a MDM result post-cancelation,
	// we need to re-activate the MDM command.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(t.Context(), `UPDATE nano_enrollment_queue SET active = 1 WHERE id = ? AND command_uuid = ?`, mdmHost.UUID, hostActivitiesResp.Activities[0].UUID)
		return err
	})

	// record a result for the VPP app post-cancelation
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	assert.NotNil(t, cmd)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstallApplication":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, []fleet.Software{{Name: addedApp.Name, BundleIdentifier: addedApp.BundleIdentifier, Version: addedApp.LatestVersion, Installed: true}})
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected command", cmd.Command.RequestType)
		}
	}
	// must not generate a past activity
	s.lastActivityMatches(fleet.ActivityTypeCanceledInstallAppStoreApp{}.ActivityName(), "", lastCanceledActID)

	// cancel the software install, confirm canceled activity
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", mdmHost.ID, hostActivitiesResp.Activities[1].UUID), nil, http.StatusNoContent)
	lastCanceledActID = s.lastActivityOfTypeMatches(fleet.ActivityTypeCanceledInstallSoftware{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "software_title": "DummyApp", "software_title_id": %d, "from_setup_experience": false}`, mdmHost.ID, mdmHost.DisplayName(), swTitleID), 0)

	// record a software install result post-cancelation
	s.Do("POST", "/api/fleet/orbit/software_install/result", json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q, "install_uuid": %q, "pre_install_condition_output": "ok", "install_script_exit_code": 1, "install_script_output": "fail"
		}`, *mdmHost.OrbitNodeKey, hostActivitiesResp.Activities[1].UUID)), http.StatusNoContent)

	// must not generate a past activity
	s.lastActivityMatches(fleet.ActivityTypeCanceledInstallSoftware{}.ActivityName(), "", lastCanceledActID)

	// the VPP install count only sees the successful VPP app install
	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", vppAppTitleID), nil, http.StatusOK, &titleResponse, "team_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.AppStoreApp)
	require.Equal(t, uint(1), titleResponse.SoftwareTitle.AppStoreApp.Status.Installed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.AppStoreApp.Status.Failed)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.AppStoreApp.Status.Pending)

	// the SW install count only sees the failed install
	titleResponse = getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/v1/fleet/software/titles/%d", swTitleID), nil, http.StatusOK, &titleResponse, "team_id", "0")
	require.NotNil(t, titleResponse.SoftwareTitle.SoftwarePackage)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.Installed)
	require.Equal(t, uint(1), titleResponse.SoftwareTitle.SoftwarePackage.Status.FailedInstall)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.FailedUninstall)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingInstall)
	require.Equal(t, uint(0), titleResponse.SoftwareTitle.SoftwarePackage.Status.PendingUninstall)

	// status of software is the one before cancellation
	getHostSoftwareResp := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", mdmHost.ID), nil, http.StatusOK, &getHostSoftwareResp, "available_for_install", "1")
	require.Len(t, getHostSoftwareResp.Software, 2)
	for _, sw := range getHostSoftwareResp.Software {
		switch sw.ID {
		case vppAppTitleID:
			require.NotNil(t, sw.Status)
			require.Equal(t, fleet.SoftwareInstalled, *sw.Status)
		case swTitleID:
			require.NotNil(t, sw.Status)
			require.Equal(t, fleet.SoftwareInstallFailed, *sw.Status)
		default:
			require.Failf(t, "unexpected software title", "%d %s", sw.ID, sw.Name)
		}
	}

	// script has no last execution (only execution was canceled)
	scriptResp = fleet.GetHostScriptDetailsResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/scripts", mdmHost.ID), nil, http.StatusOK, &scriptResp)
	require.Len(t, scriptResp.Scripts, 1)
	require.Nil(t, scriptResp.Scripts[0].LastExecution)
}

func (s *integrationMDMTestSuite) TestCancelLockWipeUpcomingActivity() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// create a couple linux hosts, which are locked/wiped via scripts
	h1 := createOrbitEnrolledHost(t, "ubuntu", "h1", s.ds)
	h2 := createOrbitEnrolledHost(t, "ubuntu", "h2", s.ds)

	// enqueue a lock and a wipe respectively as immediate upcoming activities
	var lockResp fleet.LockHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", h1.ID), nil, http.StatusOK, &lockResp)
	require.Equal(t, fleet.DeviceStatusUnlocked, lockResp.DeviceStatus)
	require.Equal(t, fleet.PendingActionLock, lockResp.PendingAction)
	s.lastActivityMatches(fleet.ActivityTypeLockedHost{}.ActivityName(), "", 0)

	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", h2.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	s.lastActivityMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), "", 0)

	// check that all are upcoming, get the exec IDs
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", h1.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	lockExecID := hostActivitiesResp.Activities[0].UUID

	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", h2.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	wipeExecID := hostActivitiesResp.Activities[0].UUID

	// trying to cancel those activities fail, as the action could've started
	res := s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", h1.ID, lockExecID), nil, http.StatusBadRequest)
	msg := extractServerErrorText(res.Body)
	require.Contains(t, msg, "Lock and wipe can't be canceled if they're about to run")
	res = s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", h2.ID, wipeExecID), nil, http.StatusBadRequest)
	msg = extractServerErrorText(res.Body)
	require.Contains(t, msg, "Lock and wipe can't be canceled if they're about to run")

	// create a couple more linux hosts, which are locked/wiped via scripts
	h3 := createOrbitEnrolledHost(t, "ubuntu", "h3", s.ds)
	h4 := createOrbitEnrolledHost(t, "ubuntu", "h4", s.ds)

	// this time enqueue a script before the lock/wipe for each host, so that lock/wipe are cancelable
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: h3.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{HostID: h4.ID, ScriptContents: "echo"}, http.StatusAccepted, &runResp)
	time.Sleep(time.Millisecond)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", h3.ID), nil, http.StatusOK, &lockResp)
	require.Equal(t, fleet.DeviceStatusUnlocked, lockResp.DeviceStatus)
	require.Equal(t, fleet.PendingActionLock, lockResp.PendingAction)
	lockActID := s.lastActivityMatches(fleet.ActivityTypeLockedHost{}.ActivityName(), "", 0)

	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", h4.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	wipeActID := s.lastActivityMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), "", 0)

	// check that all are upcoming, get the exec IDs
	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", h3.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 2)
	lockExecID = hostActivitiesResp.Activities[1].UUID

	hostActivitiesResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", h4.ID), nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 2)
	wipeExecID = hostActivitiesResp.Activities[1].UUID

	// cancel the lock and wipe succeeds
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", h3.ID, lockExecID), nil, http.StatusNoContent)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", h4.ID, wipeExecID), nil, http.StatusNoContent)

	// host lock/wipe status should've been reset
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h3.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.EqualValues(t, fleet.DeviceStatusUnlocked, *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.EqualValues(t, fleet.PendingActionNone, *getHostResp.Host.MDM.PendingAction)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", h4.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.EqualValues(t, fleet.DeviceStatusUnlocked, *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.EqualValues(t, fleet.PendingActionNone, *getHostResp.Host.MDM.PendingAction)

	// original lock/wipe activities are preserved in the audit log, with the cancelation as the latest activity
	var listAct listActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", h3.ID), nil, http.StatusOK, &listAct)
	require.True(t, len(listAct.Activities) > 1)
	require.Equal(t, fleet.ActivityTypeCanceledRunScript{}.ActivityName(), listAct.Activities[0].Type)
	require.Equal(t, lockActID, listAct.Activities[1].ID)
	require.Equal(t, fleet.ActivityTypeLockedHost{}.ActivityName(), listAct.Activities[1].Type)

	listAct = listActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities", h4.ID), nil, http.StatusOK, &listAct)
	require.True(t, len(listAct.Activities) > 1)
	require.Equal(t, fleet.ActivityTypeCanceledRunScript{}.ActivityName(), listAct.Activities[0].Type)
	require.Equal(t, wipeActID, listAct.Activities[1].ID)
	require.Equal(t, fleet.ActivityTypeWipedHost{}.ActivityName(), listAct.Activities[1].Type)
}
