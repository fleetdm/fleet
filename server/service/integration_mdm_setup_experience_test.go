package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestSetupExperienceScript() {
	t := s.T()

	tm, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:        t.Name(),
		Description: "desc",
	})
	require.NoError(t, err)

	// create new team script
	var newScriptResp setSetupExperienceScriptResponse
	body, headers := generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res := s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)

	// test script secret validation
	body, headers = generateNewScriptMultipartRequest(t,
		"script.sh", []byte(`echo "$FLEET_SECRET_INVALID"`), s.token, map[string][]string{})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusUnprocessableEntity, headers)

	// get team script metadata
	var getScriptResp getSetupExperienceScriptResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK, &getScriptResp)
	require.Equal(t, "script42.sh", getScriptResp.Name)
	require.NotNil(t, getScriptResp.TeamID)
	require.Equal(t, tm.ID, *getScriptResp.TeamID)
	require.NotZero(t, getScriptResp.ID)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)

	// get team script contents
	res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d&alt=media", tm.ID), nil, http.StatusOK)
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "hello"`, string(b))
	require.Equal(t, int64(len(`echo "hello"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script42.sh"), res.Header.Get("Content-Disposition"))

	// try to create script with same name, should fail because already exists with this name for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusConflict, headers)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "already exists") // TODO: confirm expected error message with product/frontend

	// try to create with a different name for this team, should fail because another script already exists
	// for this team
	body, headers = generateNewScriptMultipartRequest(t,
		"different.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusConflict, headers)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "already exists") // TODO: confirm expected error message with product/frontend

	// create no-team script
	body, headers = generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)
	// // TODO: confirm if we will allow team_id=0 requests
	// noTeamID := uint(0) // TODO: confirm if we will allow team_id=0 requests
	// body, headers = generateNewScriptMultipartRequest(t,
	// 	"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", noTeamID)}})

	// get no-team script metadata
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK, &getScriptResp)
	require.Equal(t, "script42.sh", getScriptResp.Name)
	require.Nil(t, getScriptResp.TeamID)
	require.NotZero(t, getScriptResp.ID)
	require.NotZero(t, getScriptResp.CreatedAt)
	require.NotZero(t, getScriptResp.UpdatedAt)
	// // TODO: confirm if we will allow team_id=0 requests
	// s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", noTeamID), nil, http.StatusOK, &getScriptResp)

	// get no-team script contents
	res = s.Do("GET", "/api/latest/fleet/setup_experience/script?alt=media", nil, http.StatusOK)
	b, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.Equal(t, `echo "hello"`, string(b))
	require.Equal(t, int64(len(`echo "hello"`)), res.ContentLength)
	require.Equal(t, fmt.Sprintf("attachment;filename=\"%s %s\"", time.Now().Format(time.DateOnly), "script42.sh"), res.Header.Get("Content-Disposition"))
	// // TODO: confirm if we will allow team_id=0 requests
	// res = s.Do("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d&alt=media", noTeamID), nil, http.StatusOK)

	// delete the no-team script
	s.Do("DELETE", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK)

	// try get the no-team script
	s.Do("GET", "/api/latest/fleet/setup_experience/script", nil, http.StatusNotFound)

	// try deleting the no-team script again
	s.Do("DELETE", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK) // TODO: confirm if we want to return not found

	// // TODO: confirm if we will allow team_id=0 requests
	// s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script/?team_id=%d", noTeamID), nil, http.StatusOK)

	// delete the team script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK)

	// try get the team script
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusNotFound)

	// try deleting the team script again
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK) // TODO: confirm if we want to return not found
}

func (s *integrationMDMTestSuite) createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript() (device godep.Device, host *fleet.Host, tm *fleet.Team) {
	t := s.T()
	ctx := context.Background()

	// enroll a device in a team with software to install and a script to execute
	s.enableABM("fleet-setup-experience")
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	teamDevice := godep.Device{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"}

	// add a team profile
	teamProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{teamProfile}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// add a macOS software to install
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp",
		TeamID:        &tm.ID,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payloadDummy.Title, "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: tm.ID, TitleIDs: []uint{titleID}}, http.StatusOK, &swInstallResp)

	// add a script to execute
	body, headers := generateNewScriptMultipartRequest(t,
		"script.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)

	// no bootstrap package, no custom setup assistant (those are already tested
	// in the DEPEnrollReleaseDevice tests).

	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	s.mockDEPResponse("fleet-setup-experience", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}, Cursor: "foo"})
			require.NoError(t, err)
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))

			var resp godep.ProfileResponse
			resp.ProfileUUID = prof.ProfileUUID
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, device := range prof.Devices {
				resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			err = encoder.Encode(resp)
			require.NoError(t, err)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	// trigger a profile sync
	s.runDEPSchedule()

	// the (ghost) host now exists
	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, listHostsRes.Hosts[0].HardwareSerial, teamDevice.SerialNumber)
	enrolledHost := listHostsRes.Hosts[0].Host
	enrolledHost.TeamID = &tm.ID

	// transfer it to the team
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm.ID, HostIDs: []uint{enrolledHost.ID}}, http.StatusOK)

	return teamDevice, enrolledHost, tm
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithSoftwareAndScriptAutoRelease() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	teamDevice, enrolledHost, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the worker to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {

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
	var statusResp getOrbitSetupExperienceStatusResponse
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
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
	require.Len(t, results, 2)
	require.NoError(t, err)
	var installUUID string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			installUUID = *r.HostSoftwareInstallsExecutionID
		}
	}

	require.NotEmpty(t, installUUID)

	// Need to get the software title to get the package name
	var getSoftwareTitleResp getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", *statusResp.Results.Software[0].SoftwareTitleID), nil, http.StatusOK, &getSoftwareTitleResp, "team_id", fmt.Sprintf("%d", *enrolledHost.TeamID))
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle)
	require.NotNil(t, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage)

	debugPrintActivities := func(activities []*fleet.UpcomingActivity) []string {
		var res []string
		for _, activity := range activities {
			res = append(res, fmt.Sprintf("%+v", activity))
		}
		return res
	}

	// Check upcoming activities: we should only have the software upcoming because we don't run the
	// script until after the software is done
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", enrolledHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	expectedActivityDetail := fmt.Sprintf(`
	{
		"status": "pending_install",
		"host_id": %d,
		"policy_id": null,
		"policy_name": null,
		"install_uuid": "%s",
		"self_service": false,
		"software_title": "%s",
		"software_package": "%s",
		"host_display_name": "%s"
	}
	`, enrolledHost.ID, installUUID, getSoftwareTitleResp.SoftwareTitle.Name, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage.Name, enrolledHost.DisplayName())
	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", debugPrintActivities(hostActivitiesResp.Activities))
	require.NotNil(t, hostActivitiesResp.Activities[0].Details)
	require.JSONEq(t, expectedActivityDetail, string(*hostActivitiesResp.Activities[0].Details))

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	// Software is now running, script is still pending
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)

	// record a result for software installation
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`, *enrolledHost.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// status still shows script as pending
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// Software is installed, now we should run the script
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Script.Status)

	// Get script exec ID
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
	require.Len(t, results, 2)
	require.NoError(t, err)
	var execID string
	for _, r := range results {
		if r.ScriptExecutionID != nil {
			execID = *r.ScriptExecutionID
		}
	}

	// Validate past activity for software install
	// For some reason the display name that's included in the `enrolledHost` is _slightly_
	// different than the expected value in the activities. Pulling the host directly gets the
	// correct display name.
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", enrolledHost.ID), nil, http.StatusOK, &getHostResp)

	expectedActivityDetail = fmt.Sprintf(`
{
  "host_id": %d,
  "host_display_name": "%s",
  "software_title": "%s",
  "software_package": "%s",
  "self_service": false,
  "install_uuid": "%s",
  "status": "installed",
  "policy_id": null,
  "policy_name": null
}
	`, enrolledHost.ID, getHostResp.Host.DisplayName, statusResp.Results.Software[0].Name, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage.Name, installUUID)

	s.lastActivityMatchesExtended(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), expectedActivityDetail, 0, ptr.Bool(true))

	// Validate upcoming activity for the script
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", enrolledHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	expectedActivityDetail = fmt.Sprintf(`
{
	"async": true,
	"host_id": %d,
	"policy_id": null,
	"policy_name": null,
	"script_name": "%s",
	"host_display_name": "%s",
	"script_execution_id": "%s"
}
	`, enrolledHost.ID, statusResp.Results.Script.Name, enrolledHost.DisplayName(), execID)
	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", debugPrintActivities(hostActivitiesResp.Activities))
	require.NotNil(t, hostActivitiesResp.Activities[0].Details)
	require.JSONEq(t, expectedActivityDetail, string(*hostActivitiesResp.Activities[0].Details))

	// record a result for script execution
	var scriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *enrolledHost.OrbitNodeKey, execID)),
		http.StatusOK, &scriptResp)

	// Get status again, now the script should be complete. This should also trigger the automatic
	// release of the device, as all setup experience steps are now complete.
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

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

	// Validate activity for script run
	expectedActivityDetail = fmt.Sprintf(`
{
	"async": true,
	"host_id": %d,
	"policy_id": null,
	"policy_name": null,
	"script_name": "%s",
	"host_display_name": "%s",
	"script_execution_id": "%s"
}
	`, enrolledHost.ID, statusResp.Results.Script.Name, getHostResp.Host.DisplayName, execID)

	s.lastActivityMatches(fleet.ActivityTypeRanScript{}.ActivityName(), expectedActivityDetail, 0)
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithSoftwareAndScriptForceRelease() {
	t := s.T()
	ctx := context.Background()

	teamDevice, enrolledHost, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the worker to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {

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
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// there shouldn't be a worker Release Device pending job (we don't release that way anymore)
	pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 0)

	// call the /status endpoint, the software and script should be pending
	var statusResp getOrbitSetupExperienceStatusResponse
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
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// call the /status endpoint again but this time force the release
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "force_release": true}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)

	// the software and script have not completed yet
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)

	// check that the host received the device configured command even if
	// software and script are still pending
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

func (s *integrationMDMTestSuite) TestSetupExperienceVPPInstallError() {
	t := s.T()
	ctx := context.Background()

	teamDevice, enrolledHost, team := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
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

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the worker to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {

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
	var statusResp getOrbitSetupExperienceStatusResponse
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
	require.Len(t, statusResp.Results.Software, 2)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, "App 5", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
	require.Len(t, results, 3)
	require.NoError(t, err)
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

	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 2)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.Equal(t, "App 5", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)

	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 2)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)

	// App 5 has no licenses available, so we should get a status failed here and setup experience
	// should continue
	require.Equal(t, "App 5", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[1].Status)

	// Software installations are done, now we should run the script
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Script.Status)
	require.Equal(t, "App 5", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[1].Status)

	// Get script exec ID
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
	require.Len(t, results, 3)
	require.NoError(t, err)
	var execID string
	for _, r := range results {
		if r.ScriptExecutionID != nil {
			execID = *r.ScriptExecutionID
		}
	}

	// record a result for script execution
	var scriptResp orbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *enrolledHost.OrbitNodeKey, execID)),
		http.StatusOK, &scriptResp)

	// Get status again, now the script should be complete. This should also trigger the automatic
	// release of the device, as all setup experience steps are now complete.
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, "App 5", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[1].Status)

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

func (s *integrationMDMTestSuite) TestSetupExperienceFlowCancelScript() {
	t := s.T()
	ctx := context.Background()

	device, host, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = device.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the worker to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

		cmds = append(cmds, &fullCmd)
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// expected commands: install fleetd (install enterprise), install profiles
	// (custom one, fleetd configuration, fleet CA root)
	require.Len(t, cmds, 4)

	// simulate fleetd being installed and the host being orbit-enrolled now
	host.OsqueryHostID = ptr.String(mdmDevice.UUID)
	host.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// call the /status endpoint, the software and script should be pending
	var statusResp getOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile

	// the software and script are pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually (for now only the software install has its execution id)
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID)
	require.Len(t, results, 2)
	require.NoError(t, err)

	var swExecID string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			swExecID = *r.HostSoftwareInstallsExecutionID
		}
	}
	require.NotEmpty(t, swExecID)

	// Check upcoming activities: we should only have the software upcoming because we don't run the
	// script until after the software is done
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	require.Equal(t, swExecID, hostActivitiesResp.Activities[0].UUID)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// cancel the software install
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", host.ID, swExecID),
		nil, http.StatusNoContent)

	// call the /status endpoint, the software is now failed and script should be pending
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile

	// the software is failed and script is pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually (this time get the script exec ID)
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID)
	require.Len(t, results, 2)
	require.NoError(t, err)

	var scrExecID string
	for _, r := range results {
		if r.ScriptExecutionID != nil {
			scrExecID = *r.ScriptExecutionID
		}
	}
	require.NotEmpty(t, scrExecID)

	// script is now in the upcoming activities
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	require.Equal(t, scrExecID, hostActivitiesResp.Activities[0].UUID)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// cancel the script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", host.ID, scrExecID),
		nil, http.StatusNoContent)

	// call the /status endpoint, both the software and script are now failed
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile

	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

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
	require.Equal(t, "DeviceConfigured", cmds[0].Command.RequestType)
}

func (s *integrationMDMTestSuite) TestSetupExperienceWithLotsOfVPPApps() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	// Set up some additional VPP apps on the mock Apple servers
	s.appleITunesSrvData["6"] = `{"bundleId": "f-6", "artworkUrl512": "https://example.com/images/6", "version": "6.0.0", "trackName": "App 6", "TrackID": 6}`
	s.appleITunesSrvData["7"] = `{"bundleId": "g-7", "artworkUrl512": "https://example.com/images/7", "version": "7.0.0", "trackName": "App 7", "TrackID": 7}`
	s.appleITunesSrvData["8"] = `{"bundleId": "h-8", "artworkUrl512": "https://example.com/images/8", "version": "8.0.0", "trackName": "App 8", "TrackID": 8}`
	s.appleITunesSrvData["9"] = `{"bundleId": "i-9", "artworkUrl512": "https://example.com/images/9", "version": "9.0.0", "trackName": "App 9", "TrackID": 9}`
	s.appleITunesSrvData["10"] = `{"bundleId": "j-10", "artworkUrl512": "https://example.com/images/10", "version": "10.0.0", "trackName": "App 10", "TrackID": 10}`

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
		delete(s.appleITunesSrvData, "6")
		delete(s.appleITunesSrvData, "7")
		delete(s.appleITunesSrvData, "8")
		delete(s.appleITunesSrvData, "9")
		delete(s.appleITunesSrvData, "10")
		s.appleVPPConfigSrvConfig.Assets = defaultVPPAssetList
	})

	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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

	// run the worker to process the DEP enroll request
	s.runWorker()
	// run the worker to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {

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
	var statusResp getOrbitSetupExperienceStatusResponse
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

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		mysql.DumpTable(t, q, "host_vpp_software_installs", "command_uuid", "adam_id")
		return nil
	})

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
			require.Equal(t, fleet.SetupExperienceStatusSuccess, software.Status)
		} else {
			require.Equal(t, fleet.SetupExperienceStatusRunning, software.Status)
		}
		require.NotNil(t, software.SoftwareTitleID)
		require.NotZero(t, *software.SoftwareTitleID)
	}

	// All apps should have an install record at this point
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var count int
		err := sqlx.GetContext(context.Background(), q, &count, "SELECT COUNT(*) FROM host_vpp_software_installs")
		require.NoError(t, err)
		require.Equal(t, 6, count)
		return nil
	})

	installedApps := map[string]struct{}{
		macOSApp1.Name: {},
	}

	for _, app := range expectedApps {
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
				require.Equal(t, fleet.SetupExperienceStatusSuccess, software.Status, software.Name, software.Status)
			} else {
				require.Equal(t, fleet.SetupExperienceStatusRunning, software.Status)
			}
			require.NotNil(t, software.SoftwareTitleID)
			require.NotZero(t, *software.SoftwareTitleID)
		}

	}
}
