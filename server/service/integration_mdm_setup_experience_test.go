package service

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
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/vpp"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/fleetdm/fleet/v4/server/worker"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
	"google.golang.org/api/googleapi"
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

	// try to update script with same name, should not fail because this is allowed
	body, headers = generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)

	// update with a different name and contents via PUT endpoint, should suceed
	body, headers = generateNewScriptMultipartRequest(t,
		"different.sh", []byte(`echo "hello2"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)

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

	s.lastActivityOfTypeMatches(fleet.ActivityEditedSetupExperienceSoftware{}.ActivityName(),
		fmt.Sprintf(`{"platform": "darwin", "team_id": %d, "team_name": "%s"}`, tm.ID, tm.Name), 0)

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
	require.NoError(t, err)
	require.Len(t, results, 2)
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
		"source": "apps",
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
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Script.Status)
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
	require.NoError(t, err)
	require.Len(t, results, 2)
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
  "source": "apps",
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
	"script_execution_id": "%s",
	"batch_execution_id": null
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
	"script_execution_id": "%s",
	"batch_execution_id": null
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
	require.NoError(t, err)
	require.Len(t, results, 3)
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
	require.NoError(t, err)
	require.Len(t, results, 3)
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

func (s *integrationMDMTestSuite) TestSetupExperienceFlowUpdateScript() {
	t := s.T()
	ctx := context.Background()

	device, host, tm := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

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
	require.NoError(t, err)
	require.Len(t, results, 2)

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

	// call the /status endpoint, the software is now running and script should be pending
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile

	// the software is running and script is pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// update the script but with no actual changes, it does not get cancelled
	body, headers := generateNewScriptMultipartRequest(t,
		"script.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)

	// call the /status endpoint, the software is still running and script should still be pending
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile

	// the software is still running and script is pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// update the script with changes, see it get cancelled
	body, headers = generateNewScriptMultipartRequest(t,
		"script2.sh", []byte(`echo "foobar"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)

	// call the /status endpoint, software is running, script is removed as it got cancelled by the update
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile

	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	require.Nil(t, statusResp.Results.Script)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// them out manually
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	var installUUIDs []string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			installUUIDs = append(installUUIDs, *r.HostSoftwareInstallsExecutionID)
		}
	}
	require.Equal(t, len(installUUIDs), 1)

	// record a result for software installation
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`, *host.OrbitNodeKey, installUUIDs[0])), http.StatusNoContent)

	// Check the setup experience status endpoint to advance the status
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)

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
	require.NoError(t, err)
	require.Len(t, results, 2)

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
	require.NoError(t, err)
	require.Len(t, results, 2)

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

func (s *integrationMDMTestSuite) TestSetupExperienceEndpointsWithPlatform() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// Add a macOS software to the setup experience on team 1.
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp",
		TeamID:        &team1.ID,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payloadDummy.Title, "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "macos",
		TeamID:   team1.ID,
		TitleIDs: []uint{titleID},
	}, http.StatusOK, &swInstallResp)

	// Get "Setup experience" items using platform and the endpoint without platform that we cannot remove (for backwards compatibility).
	var respGetSetupExperience getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{}, http.StatusOK, &respGetSetupExperience, "team_id", fmt.Sprint(team1.ID))
	noPlatformTitles := respGetSetupExperience.SoftwareTitles
	require.Len(t, noPlatformTitles, 1)
	respGetSetupExperience = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{},
		http.StatusOK,
		&respGetSetupExperience,
		"platform", "macos",
		"team_id", fmt.Sprint(team1.ID),
	)
	macOSPlatformTitles := respGetSetupExperience.SoftwareTitles
	require.Equal(t, macOSPlatformTitles, noPlatformTitles)

	// Test invalid platform in GET and PUT.
	res := s.DoRawWithHeaders("PUT", "/api/v1/fleet/setup_experience/software", []byte(`{"platform": "foobar", "team_id": 0}`), http.StatusBadRequest, nil)
	errMsg := extractServerErrorText(res.Body)
	require.NoError(t, res.Body.Close())
	require.Contains(t, errMsg, "platform \"foobar\" unsupported, platform must be one of \"macos\", \"ios\", \"ipados\", \"windows\", \"linux\", \"android\"")
	res = s.DoRawWithHeaders("GET", "/api/v1/fleet/setup_experience/software?platform=foobar&team_id=0", nil, http.StatusBadRequest, nil)
	errMsg = extractServerErrorText(res.Body)
	require.NoError(t, res.Body.Close())
	require.Contains(t, errMsg, "platform \"foobar\" unsupported, platform must be one of \"macos\", \"ios\", \"ipados\", \"windows\", \"linux\", \"android\"")
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
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
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
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	var getVPPTokenResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &getVPPTokenResp)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", getVPPTokenResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

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
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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

type DEPEnrollMobileTestOpts struct {
	ABMOrg                            string
	EnableReleaseManually             bool
	TeamID                            *uint
	CustomProfileIdent                string
	EnrollmentProfileFromDEPUsingPost bool
	VppAppsToInstall                  []*fleet.VPPApp
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
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{device}, Cursor: "foo"})
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
		mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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

	// run the worker to process the DEP enroll request
	s.runWorker()

	// run the cron to assign configuration profiles
	s.awaitTriggerProfileSchedule(t)

	var cmds []*micromdm.CommandPayload
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)

	// For reporting back via InstalledApplicationList
	installedVPPApps := make([]fleet.Software, 0, len(opts.VppAppsToInstall))
	// For verifying number of installs
	installedApps := make(map[string]int, len(opts.VppAppsToInstall))

	var installProfileCount, installAppCount, refetchVerifyCount, otherCount int
	var profileCustomSeen, profileFleetCASeen, unexpectedProfileSeen bool

	// Can be useful for debugging
	logCommands := false
	for cmd != nil {

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
			if logCommands {
				fmt.Println(">>>> device received command: ", cmd.CommandUUID, cmd.Command.RequestType, fmt.Sprint(*fullCmd.Command.InstallApplication.ITunesStoreID))
			}
			for _, app := range opts.VppAppsToInstall {
				if app.AdamID == fmt.Sprint(*fullCmd.Command.InstallApplication.ITunesStoreID) {
					installedVPPApps = append(installedVPPApps, fleet.Software{BundleIdentifier: app.BundleIdentifier, Name: app.Name, Version: app.LatestVersion, Installed: true})
					installedApps[app.AdamID]++
				}
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
			// Hold off on verifying the last install until later so we can ensure it waits for verification
			if len(installedVPPApps) == len(opts.VppAppsToInstall) {
				installedVPPApps[len(installedVPPApps)-1].Installed = false
			}
			cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(
				mdmDevice.UUID,
				cmd.CommandUUID,
				installedVPPApps,
			)
			// flip the status back for later
			installedVPPApps[len(installedVPPApps)-1].Installed = true
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
					mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
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
	require.Len(t, cmds, 2+len(opts.VppAppsToInstall))

	require.Equal(t, 2, installProfileCount)
	require.True(t, profileCustomSeen)
	require.True(t, profileFleetCASeen)
	require.Equal(t, false, unexpectedProfileSeen)

	require.Equal(t, len(opts.VppAppsToInstall), installAppCount)
	require.Equal(t, len(opts.VppAppsToInstall), len(installedApps))

	// Each expected app should be installed exactly once
	for _, app := range opts.VppAppsToInstall {
		require.Equal(t, 1, installedApps[app.AdamID])
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
				mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
					_, err := q.ExecContext(ctx, `DELETE FROM jobs WHERE id = ?`, job.ID)
					return err
				})
			}
		}
		require.Len(t, pendingReleaseJobs, 0)
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
	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE jobs SET not_before = ? WHERE id IN (?, ?)`, time.Now().Add(-1*time.Minute).UTC(), pendingReleaseJobs[0].ID, pendingVerifyJobs[0].ID)
		return err
	})

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
				installedVPPApps,
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

	require.Len(t, pendingVerifyJobs, 0)

	mysql.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx, `UPDATE jobs SET not_before = ? WHERE id = ?`, time.Now().Add(-1*time.Minute).UTC(), pendingReleaseJobs[0].ID)
		return err
	})

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

// TestSetupExperienceEndpointsPlatformIsolation tests that setting the setup experience software items
// for one platform doesn't remove the items for another platform on the same team.
func (s *integrationMDMTestSuite) TestSetupExperienceEndpointsPlatformIsolation() {
	t := s.T()
	ctx := context.Background()

	team1, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "team 1"})
	require.NoError(t, err)

	// Add a macOS software to the setup experience on team 1.
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp",
		TeamID:        &team1.ID,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")
	titleID := getSoftwareTitleID(t, s.ds, payloadDummy.Title, "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "macos",
		TeamID:   team1.ID,
		TitleIDs: []uint{titleID},
	}, http.StatusOK, &swInstallResp)

	// Clear all Linux software on the setup experience.
	swInstallResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "macos",
		TeamID:   team1.ID,
		TitleIDs: []uint{},
	}, http.StatusOK, &swInstallResp)

	// Get setup experience items for macOS should return the one item.
	var respGetSetupExperience getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{},
		http.StatusOK,
		&respGetSetupExperience,
		"platform", "macos",
		"team_id", fmt.Sprint(team1.ID),
	)
	require.Len(t, respGetSetupExperience.SoftwareTitles, 1)
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithRequireSoftware() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	teamDevice, enrolledHost, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	payload := map[string]any{
		"require_all_software_macos": true,
		"team_id":                    enrolledHost.TeamID,
	}
	s.Do("PATCH", "/api/latest/fleet/setup_experience", json.RawMessage(jsonMustMarshal(t, payload)), http.StatusNoContent)

	pk1Title := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	// Add another couple of packages
	// Add 2nd package to the team.
	pk2 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for pkg2",
		Filename:      "no_version.pkg",
		Title:         "pkg2",
		TeamID:        enrolledHost.TeamID,
	}
	s.uploadSoftwareInstaller(t, pk2, http.StatusOK, "")
	pk2Title := getSoftwareTitleID(t, s.ds, "NoVersion", "apps")
	// Add 3rd package to the team.
	pk3 := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install script for pkg3",
		Filename:      "EchoApp.pkg",
		Title:         "pkg3",
		TeamID:        enrolledHost.TeamID,
	}
	s.uploadSoftwareInstaller(t, pk3, http.StatusOK, "")
	pk3Title := getSoftwareTitleID(t, s.ds, "EchoApp", "apps")

	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: *enrolledHost.TeamID, TitleIDs: []uint{pk1Title, pk2Title, pk3Title}}, http.StatusOK, &swInstallResp)

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

	// call the /status endpoint, the software and script should be pending.
	// Note that this kicks off the next step of the setup experience, so while
	// the API response will show the software as "pending", they will now be
	// set as "running" in the database.
	var statusResp getOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "reset_failed_setup_steps": true}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.True(t, statusResp.Results.RequireAllSoftware)
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
	for _, softwareResult := range statusResp.Results.Software {
		require.Equal(t, fleet.SetupExperienceStatusPending, softwareResult.Status)
	}

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// them out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
	require.NoError(t, err)
	require.Len(t, results, 4)
	var installUUIDs []string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			installUUIDs = append(installUUIDs, *r.HostSoftwareInstallsExecutionID)
		}
	}
	require.Equal(t, len(installUUIDs), 3)

	// debugPrintActivities := func(activities []*fleet.UpcomingActivity) []string {
	// 	var res []string
	// 	for _, activity := range activities {
	// 		res = append(res, fmt.Sprintf("%+v", activity))
	// 	}
	// 	return res
	// }

	// Check upcoming activities: we should only have the software upcoming because we don't run the
	// script until after the software is done
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", enrolledHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	// Get the install UUIDs in the activities
	activityInstallUUIDs := make([]string, len(hostActivitiesResp.Activities))
	for i, activity := range hostActivitiesResp.Activities {
		if activity.Details != nil {
			var detail struct {
				InstallUUID string `json:"install_uuid"`
			}
			err := json.Unmarshal(*activity.Details, &detail)
			require.NoError(t, err)
			activityInstallUUIDs[i] = detail.InstallUUID
		}
	}
	// Verify that they match the install IDs in the setup experience.
	for _, installUUID := range installUUIDs {
		require.Contains(t, activityInstallUUIDs, installUUID)
	}

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// Check status again. Software should all be listed as "running" now.
	// Since no results have been recorded, this shouldn't change any database state.
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	// Software is now running, script is still pending
	require.Equal(t, len(statusResp.Results.Software), 3)
	for _, softwareResult := range statusResp.Results.Software {
		require.Equal(t, fleet.SetupExperienceStatusRunning, softwareResult.Status)
	}
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)

	// record a successful result for the first software installation
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 0,
					"install_script_output": "ok"
				}`, *enrolledHost.OrbitNodeKey, installUUIDs[0])), http.StatusNoContent)

	// status still shows script as pending
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 3)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	// Other two software should still be "running"
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[1].Status)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[2].Status)

	// Record a failure for the second software.
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
					"orbit_node_key": %q,
					"install_uuid": %q,
					"install_script_exit_code": 1,
					"install_script_output": "nope"
				}`, *enrolledHost.OrbitNodeKey, installUUIDs[1])), http.StatusNoContent)

	// no MDM command got enqueued due to the /status call (device not released yet)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// Get the setup experience status again.
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	// Script should be marked as failed since required software install failed.
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 3)
	// The successful install should remain successful.
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	// The software we recorded as failed should have the failed state.
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[1].Status)
	// The software we were waiting to install should have the failed state
	// because required software failed.
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[2].Status)

	// There should be no upcoming activities.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", enrolledHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Equal(t, len(hostActivitiesResp.Activities), 0)

	// Reset the setup experience items.
	statusResp = getOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "reset_failed_setup_steps": true}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	// The script should be back to "pending"
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 3)
	// The successful install should remain successful.
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	// Other two software should go back to "pending"
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[2].Status)
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithRequiredSoftwareVPP() {
	t := s.T()
	ctx := context.Background()

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
	t.Setenv("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL)
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

	// Add the VPP app to setup experience
	vppTitleID := getSoftwareTitleID(t, s.ds, "App 5", "apps")
	vppTitleID2 := getSoftwareTitleID(t, s.ds, "App 4", "apps")
	installerTitleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: team.ID, TitleIDs: []uint{vppTitleID, vppTitleID2, installerTitleID}}, http.StatusOK, &swInstallResp)

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
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, "App 4", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)
	require.Equal(t, "App 5", statusResp.Results.Software[2].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[2].Status)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID)
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

	statusResp = getOrbitSetupExperienceStatusResponse{}
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
	// App 4 has no licenses available, so it should fail and because we have "requre_all_software_macos" set,
	// the other software and the script should be marked as failed too.
	require.Equal(t, "App 4", statusResp.Results.Software[1].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[1].Status)
	require.Equal(t, "App 5", statusResp.Results.Software[2].Name)
	require.Equal(t, fleet.SetupExperienceStatusFailure, statusResp.Results.Software[2].Status)

	// Reset the setup experience items.
	statusResp = getOrbitSetupExperienceStatusResponse{}
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
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)
	require.Equal(t, "App 5", statusResp.Results.Software[2].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[2].Status)
}

func (s *integrationMDMTestSuite) TestSetupExperienceGetPutSoftware() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	s.setVPPTokenForTeam(0)

	// add a macOS software installer
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp",
		TeamID:        nil,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")

	// add a VPP app only available on macos
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: nil, Platform: "darwin", AppStoreID: "1", SelfService: true}, http.StatusOK)
	// add a VPP app available on all macos and ios
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: nil, Platform: "darwin", AppStoreID: "2", SelfService: false}, http.StatusOK)
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: nil, Platform: "ios", AppStoreID: "2", SelfService: false}, http.StatusOK)

	// add an iDevice ipa installer
	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{Filename: "ipa_test.ipa"}, http.StatusOK, "")

	// get the macos title IDs
	var listSoftware listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftware, "team_id", "0", "platform", "darwin", "order_key", "name")
	require.Len(t, listSoftware.SoftwareTitles, 3)
	app1MacTitleID := listSoftware.SoftwareTitles[0].ID
	app2MacTitleID := listSoftware.SoftwareTitles[1].ID
	pkgTitleID := listSoftware.SoftwareTitles[2].ID

	// get the ios title IDs
	listSoftware = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &listSoftware, "team_id", "0", "platform", "ios", "order_key", "name")
	require.Len(t, listSoftware.SoftwareTitles, 2)
	app2IOSTitleID := listSoftware.SoftwareTitles[0].ID
	ipaTitleID := listSoftware.SoftwareTitles[1].ID

	// list software for setup experience macos
	var listSetupSoftware getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{},
		http.StatusOK, &listSetupSoftware, "platform", "macos", "team_id", "0", "order_key", "name")

	require.Len(t, listSetupSoftware.SoftwareTitles, 3)
	require.Equal(t, "App 1", listSetupSoftware.SoftwareTitles[0].Name)
	require.NotNil(t, listSetupSoftware.SoftwareTitles[0].AppStoreApp)
	require.Equal(t, "1", listSetupSoftware.SoftwareTitles[0].AppStoreApp.AppStoreID)
	require.Equal(t, "App 2", listSetupSoftware.SoftwareTitles[1].Name)
	require.NotNil(t, listSetupSoftware.SoftwareTitles[1].AppStoreApp)
	require.Equal(t, "2", listSetupSoftware.SoftwareTitles[1].AppStoreApp.AppStoreID)
	require.Equal(t, "DummyApp", listSetupSoftware.SoftwareTitles[2].Name)
	require.NotNil(t, listSetupSoftware.SoftwareTitles[2].SoftwarePackage)
	require.Equal(t, "dummy_installer.pkg", listSetupSoftware.SoftwareTitles[2].SoftwarePackage.Name)

	// list software for setup experience ios
	listSetupSoftware = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{},
		http.StatusOK, &listSetupSoftware, "platform", "ios", "team_id", "0", "order_key", "name")

	// only 1 installer, the VPP app, the ipa is filtered out because unsupported
	require.Len(t, listSetupSoftware.SoftwareTitles, 1)
	require.Equal(t, "App 2", listSetupSoftware.SoftwareTitles[0].Name)

	// put software for setup experience macos with an unknown one
	res := s.Do("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "macos",
		TeamID:   0,
		TitleIDs: []uint{pkgTitleID, 9999},
	}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one selected software title does not exist or is not available for setup experience")

	// put software for setup experience macos with an invalid one (the ipa)
	res = s.Do("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "macos",
		TeamID:   0,
		TitleIDs: []uint{app1MacTitleID, ipaTitleID},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one selected software title does not exist or is not available for setup experience")

	// put software for setup experience macos with valid ones
	var putSetupSoftware putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "macos",
		TeamID:   0,
		TitleIDs: []uint{app1MacTitleID, app2MacTitleID},
	}, http.StatusOK, &putSetupSoftware)

	// put software for setup experience ios with an unknown one
	res = s.Do("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "ios",
		TeamID:   0,
		TitleIDs: []uint{app2IOSTitleID, 9999},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one selected software title does not exist or is not available for setup experience")

	// put software for setup experience ios with an invalid one (ipa)
	res = s.Do("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "ios",
		TeamID:   0,
		TitleIDs: []uint{ipaTitleID},
	}, http.StatusBadRequest)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "at least one selected software title does not exist or is not available for setup experience")

	// put software for setup experience ios with valid ones
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "ios",
		TeamID:   0,
		TitleIDs: []uint{app2IOSTitleID},
	}, http.StatusOK, &putSetupSoftware)
}

func (s *integrationMDMTestSuite) TestSetupExperienceAndroid() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	s.setVPPTokenForTeam(0)
	enterpriseID := s.enableAndroidMDM(t)

	// add a macOS software installer
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "dummy_installer.pkg",
		Title:         "DummyApp",
		TeamID:        nil,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")

	// add a VPP app available on macos and ios
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: nil, Platform: "darwin", AppStoreID: "2", SelfService: false}, http.StatusOK)
	s.Do("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: nil, Platform: "ios", AppStoreID: "2", SelfService: false}, http.StatusOK)

	// add 2 android apps
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

	// should be called twice - once with the 2 Android apps to make available for self-install,
	// and once for the setup experience with only the app to install at setup (and install type
	// PREINSTALLED)
	var patchAppsCallCount int // no need for mutex, protected via runWorkerUntilDone
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		switch patchAppsCallCount {
		case 1:
			// first call to make apps available for self-install
			require.Len(t, appPolicies, 2, "initial call to make apps available for self-install should have 2 apps")
			require.Equal(t, appPolicies[0].InstallType, "AVAILABLE")
			require.Equal(t, appPolicies[0].PackageName, app1.VPPAppID.AdamID)
			require.Equal(t, appPolicies[1].InstallType, "AVAILABLE")
			require.Equal(t, appPolicies[1].PackageName, app2.VPPAppID.AdamID)
		case 2:
			// second call for setup experience, should have only app1 with PREINSTALLED
			require.Len(t, appPolicies, 1, "second call for setup experience should have only 1 app")
			require.Equal(t, appPolicies[0].InstallType, "PREINSTALLED")
			require.Equal(t, appPolicies[0].PackageName, app1.VPPAppID.AdamID)
		default:
			t.Fatalf("unexpected call count %d to EnterprisesPoliciesModifyPolicyApplications", patchAppsCallCount)
		}

		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
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

	// Run worker to flush out no-op "make_android_app_available" tasks from adding the apps above
	s.runWorkerUntilDoneWithChecks(true)

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

	host, deviceInfo, pubSubToken := s.createAndEnrollAndroidDevice(t, "test-android", nil)

	// Google AMAPI hasn't been hit yet
	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	// run worker, should run the job that assigns the app to the host's MDM policy
	s.runWorkerUntilDoneWithChecks(true)
	// should have hit the android API endpoint
	require.True(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)

	var count int
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, tx, &count,
			`SELECT COUNT(*) FROM android_policy_requests WHERE policy_id = ?`,
			host.UUID)
	})
	// 1. The default enrollment policy
	// 2. The Fleet-enforced per-device policy
	// 3. The patch applications to make apps available for self-service
	// 4. The patch applications to force install at setup experience
	require.Equal(t, 4, count)

	// the pending install should show up in the host software
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	app1CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// send a pub-sub with the software installed, to make it verified
	policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host.UUID)
	reportMsg := statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo.Name,
			EnrollmentTokenData:  deviceInfo.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 2,
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host.UUID,
	)
	req := android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))
	s.lastActivityOfTypeMatches(fleet.ActivityInstalledAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"app_store_id":%q,
		"command_uuid":%q, "host_display_name":%q, "host_id":%d, "host_platform":%q, "policy_id":null, "policy_name":null, "self_service":false, "software_title":%q,
		"status":%q}`, app1.AdamID, app1CmdUUID, host.DisplayName(), host.ID, host.Platform, app1.Name, fleet.SoftwareInstalled), 0)

	// the pending install should now be verified
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// the software now shows up in the host inventory
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host.ID), nil, http.StatusOK, &getHostSw, "order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// add app 2 to Android setup experience
	putResp = putSetupExperienceSoftwareResponse{}
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID, app2TitleID},
	}, http.StatusOK, &putResp)

	// enroll another Android device to test 2 apps that install on enroll
	host2, _, _ := s.createAndEnrollAndroidDevice(t, "test-android-2", nil)

	patchAppsCallCount = 0
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		switch patchAppsCallCount {
		case 1:
			// first call to make apps available for self-install
			require.Len(t, appPolicies, 2, "initial call to make apps available for self-install should have 2 apps")
			require.Equal(t, appPolicies[0].InstallType, "AVAILABLE")
			require.Equal(t, appPolicies[0].PackageName, app1.VPPAppID.AdamID)
			require.Equal(t, appPolicies[1].InstallType, "AVAILABLE")
			require.Equal(t, appPolicies[1].PackageName, app2.VPPAppID.AdamID)
		case 2:
			// second call for setup experience, should have both apps
			require.Len(t, appPolicies, 2, "second call for setup experience should have 2 apps")
			require.Equal(t, appPolicies[0].InstallType, "PREINSTALLED")
			require.Equal(t, appPolicies[0].PackageName, app1.VPPAppID.AdamID)
			require.Equal(t, appPolicies[1].InstallType, "PREINSTALLED")
			require.Equal(t, appPolicies[1].PackageName, app2.VPPAppID.AdamID)
		default:
			t.Fatalf("unexpected call count %d to EnterprisesPoliciesModifyPolicyApplications", patchAppsCallCount)
		}

		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
	}
	s.runWorkerUntilDoneWithChecks(true)

	// the pending installs should show up in the host software
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[1].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[1].Status)
}

func (s *integrationMDMTestSuite) TestSetupExperienceAndroidCancelOnUnenroll() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)
	enterpriseID := s.enableAndroidMDM(t)

	// add 2 android apps
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
	s.androidAPIClient.EnterprisesDevicesDeleteFunc = func(ctx context.Context, deviceName string) error {
		return nil
	}

	var patchAppsCallCount int // no need for mutex, protected via runWorkerUntilDone
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
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

	// enroll a few Android devices, will get app1 at setup
	host1, deviceInfo1, pubSubToken := s.createAndEnrollAndroidDevice(t, "test-1", nil)
	host2, _, _ := s.createAndEnrollAndroidDevice(t, "test-2", nil)
	host3, _, _ := s.createAndEnrollAndroidDevice(t, "test-3", nil)

	require.False(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)
	s.runWorkerUntilDoneWithChecks(true)
	require.True(t, s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFuncInvoked)

	// app install is pending
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.EqualValues(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	// turn off MDM for that host, should fail the pending install
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/mdm", host1.ID), nil, http.StatusNoContent)

	// app install is still pending as the device hasn't reported back its unenrollment yet
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.EqualValues(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	// send a pub-sub with the status repored as deleted
	policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg := statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 2,
			LastPolicySyncTime:   time.Now().Format(time.RFC3339Nano),
			AppliedState:         "DELETED",
		},
		host1.UUID,
	)
	req := android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	// app install is now failed
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.EqualValues(t, fleet.SoftwareInstallFailed, *getHostSw.Software[0].Status)
	app1CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID

	// activities got created as expected
	s.lastActivityOfTypeMatches(fleet.ActivityTypeMDMUnenrolled{}.ActivityName(), fmt.Sprintf(`
	{"enrollment_id": null, "host_display_name": %q, "host_serial": %q, "installed_from_dep": false, "platform": %q}`,
		host1.DisplayName(), "", host1.Platform), 0) // for some reason the serial is force-set to empty string when we create this activity
	s.lastActivityOfTypeMatches(fleet.ActivityInstalledAppStoreApp{}.ActivityName(), fmt.Sprintf(`{"app_store_id":%q,
		"command_uuid":%q, "host_display_name":%q, "host_id":%d, "host_platform":%q, "policy_id":null, "policy_name":null, "self_service":false, "software_title":%q,
		"status":%q}`, app1.AdamID, app1CmdUUID, host1.DisplayName(), host1.ID, host1.Platform, app1.Name, fleet.SoftwareInstallFailed), 0)

	// host2 and host3 haven't been unenrolled, app install is still pending
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.EqualValues(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host3.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"query", app1.Name, "order_key", "name")
	require.Len(t, getHostSw.Software, 1)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.EqualValues(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)

	// turn off MDM for Android globally
	var deleteEnterResp android.DefaultResponse
	s.DoJSON("DELETE", "/api/latest/fleet/android_enterprise", nil, http.StatusOK, &deleteEnterResp)

	// host2 and host3 app install is now failed, but because Android MDM is now off globally,
	// we can't list software available for install anymore, we have to use adhoc sql to confirm.
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true")
	require.Len(t, getHostSw.Software, 0)

	var countFailed, countOther int
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		err := sqlx.GetContext(ctx, tx, &countFailed, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id IN (?, ?) AND verification_failed_at IS NOT NULL`,
			host2.ID, host3.ID)
		if err != nil {
			return err
		}
		err = sqlx.GetContext(ctx, tx, &countOther, `SELECT COUNT(*) FROM host_vpp_software_installs WHERE host_id IN (?, ?) AND verification_failed_at IS NULL`,
			host2.ID, host3.ID)
		return err
	})
	require.Equal(t, 2, countFailed)
	require.Equal(t, 0, countOther)
}

func (s *integrationMDMTestSuite) TestAndroidAppConfiguration() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	s.enableAndroidMDM(t)

	// add some android apps
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
	app3 := &fleet.VPPApp{
		VPPAppTeam: fleet.VPPAppTeam{
			VPPAppID: fleet.VPPAppID{
				AdamID:   "com.test3",
				Platform: fleet.AndroidPlatform,
			},
		},
		Name:             "Test3",
		BundleIdentifier: "com.test3",
		IconURL:          "https://example.com/3",
	}

	androidApps := []*fleet.VPPApp{app1, app2, app3}
	s.androidAPIClient.EnterprisesApplicationsFunc = func(ctx context.Context, enterpriseName string, packageName string) (*androidmanagement.Application, error) {
		for _, app := range androidApps {
			if app.AdamID == packageName {
				return &androidmanagement.Application{IconUrl: app.IconURL, Title: app.Name}, nil
			}
		}
		return nil, &notFoundError{}
	}

	// vars have no need for a mutex, protected via runWorkerUntilDone
	var (
		// records the appPolicies received in the ModifyPolicyApplications calls
		patchAppsPolicies  [][]*androidmanagement.ApplicationPolicy
		patchAppsCallCount int
	)
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		patchAppsPolicies = append(patchAppsPolicies, appPolicies)

		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
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

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have done nothing (no host to add apps to yet)
	require.Len(t, patchAppsPolicies, 0)
	patchAppsPolicies = nil

	var patchAppResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 1}`),
	}, http.StatusOK, &patchAppResp)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app2TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 2}`),
	}, http.StatusOK, &patchAppResp)

	// add app 1 and 2 to Android setup experience
	var putResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", &putSetupExperienceSoftwareRequest{
		Platform: string(fleet.AndroidPlatform),
		TeamID:   0,
		TitleIDs: []uint{app1TitleID, app2TitleID},
	}, http.StatusOK, &putResp)

	s.createAndEnrollAndroidDevice(t, "test-android", nil)

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made each app available to the included hosts (for self-service), so 2 entries for that (from the PATCH apps to set the config)
	// (this is because I made the worker run after host enrollment, if there were no host, the task would have nothing to do)
	// 2. made all apps available to the enrolled host (for self-service), from the host enrollment
	// 3. installed the apps, from the host enrollment
	require.Len(t, patchAppsPolicies, 4)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app1.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage(`1`)},
	}, patchAppsPolicies[0])
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app2.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage(`2`)},
	}, patchAppsPolicies[1])
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app1.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage(`1`)},
		{PackageName: app2.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage(`2`)},
	}, patchAppsPolicies[2])
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app1.VPPAppID.AdamID, InstallType: "PREINSTALLED", ManagedConfiguration: googleapi.RawMessage(`1`)},
		{PackageName: app2.VPPAppID.AdamID, InstallType: "PREINSTALLED", ManagedConfiguration: googleapi.RawMessage(`2`)},
	}, patchAppsPolicies[3])

	patchAppsPolicies = nil

	// add app3 to Fleet
	addAppResp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		AppStoreID: app3.AdamID,
		Platform:   fleet.AndroidPlatform,
	}, http.StatusOK, &addAppResp)
	app3TitleID := addAppResp.TitleID

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made the apps available to the host (for self-service), without any config provided
	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app3.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage{}, WorkProfileWidgets: "WORK_PROFILE_WIDGETS_UNSPECIFIED"},
	}, patchAppsPolicies[0])

	patchAppsPolicies = nil

	// set a configuration for the app3
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 3}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made the app available with its config
	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app3.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage(`3`)},
	}, patchAppsPolicies[0])

	patchAppsPolicies = nil

	// patch but no change to the configuration for the app3
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 3}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	require.Len(t, patchAppsPolicies, 0)

	// patch with a different config just to trigger the worker
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app3TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{}`),
	}, http.StatusOK, &patchAppResp)

	// delete directly the config from the DB (it seems like to clear the config from
	// the API, an empty object needs to be passed, but that won't clear the config,
	// it just won't change any managedConfig/widgets - to really clear the config
	// from the API, the user would have to send something like:
	// {
	//   "managedConfiguration": null,
	//   "workProfileWidgets": "WORK_PROFILE_WIDGETS_UNSPECIFIED"
	// }
	//
	// Is that how we want it to work?
	mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(t.Context(), `DELETE FROM android_app_configurations WHERE application_id = ?`, app3.VPPAppID.AdamID)
		return err
	})

	s.runWorkerUntilDoneWithChecks(true)

	// worker should have:
	// 1. made the app available with its config cleared
	require.Len(t, patchAppsPolicies, 1)
	require.ElementsMatch(t, []*androidmanagement.ApplicationPolicy{
		{PackageName: app3.VPPAppID.AdamID, InstallType: "AVAILABLE", ManagedConfiguration: googleapi.RawMessage{}, WorkProfileWidgets: "WORK_PROFILE_WIDGETS_UNSPECIFIED"},
	}, patchAppsPolicies[0])
}

func (s *integrationMDMTestSuite) TestSetupExperienceAndroidWithConfiguration() {
	t := s.T()
	ctx := t.Context()
	s.setSkipWorkerJobs(t)

	enterpriseID := s.enableAndroidMDM(t)

	// add 2 android apps
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

	var patchAppsCallCount int // no need for mutex, protected via runWorkerUntilDone
	s.androidAPIClient.EnterprisesPoliciesModifyPolicyApplicationsFunc = func(ctx context.Context, policyName string, appPolicies []*androidmanagement.ApplicationPolicy) (*androidmanagement.Policy, error) {
		patchAppsCallCount++
		return &androidmanagement.Policy{Version: int64(patchAppsCallCount)}, nil
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

	s.runWorkerUntilDoneWithChecks(true)

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "test team", Secrets: []*fleet.EnrollSecret{{Secret: uuid.NewString()}}})
	require.NoError(t, err)

	// enroll a couple android devices on no-team and one on a team
	host1, deviceInfo1, pubSubToken := s.createAndEnrollAndroidDevice(t, "test-android1", nil)
	host2, deviceInfo2, _ := s.createAndEnrollAndroidDevice(t, "test-android2", nil)
	host3, _, _ := s.createAndEnrollAndroidDevice(t, "test-android3", &tm.ID)

	s.runWorkerUntilDoneWithChecks(true)

	// hosts 1 and 2 have app1 pending install
	getHostSw := getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	app1Host1CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)
	require.NotEmpty(t, app1Host1CmdUUID)

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	app1Host2CmdUUID := getHostSw.Software[0].AppStoreApp.LastInstall.CommandUUID
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)
	require.NotEmpty(t, app1Host2CmdUUID)

	// host 3 has no apps available to install
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host3.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 0)

	// make the software installed on host1, failed on host2
	policyName := fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg := statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 10,
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host1.UUID,
	)
	req := android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	policyName = fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host2.UUID)
	reportMsg = statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo2.Name,
			EnrollmentTokenData:  deviceInfo2.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: 20,
			NonComplianceDetails: []*androidmanagement.NonComplianceDetail{
				{PackageName: app1.AdamID, NonComplianceReason: "APP_NOT_INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host2.UUID,
	)
	req = android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	// the pending install should now be verified for host1
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// the pending install should now be failed for host2
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// set a configuration for app1
	var patchAppResp updateAppStoreAppResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 1}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// the verified install should now be back to pending for host1
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// the failed install is still failed for host2
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host2.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallFailed, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// and still no apps for host3
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host3.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 0)

	getPolicyVersion := func(hostID uint, appID string) int64 {
		var version int64
		mysql.ExecAdhocSQL(t, s.ds, func(tx sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, tx, &version, `
				SELECT CAST(associated_event_id AS SIGNED) FROM host_vpp_software_installs WHERE host_id = ? AND adam_id = ?`,
				hostID, appID)
		})
		return version
	}
	versionBefore := getPolicyVersion(host1.ID, app1.AdamID)

	// update the configuration for app1
	patchAppResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 2}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// install for host1 will still be pending, but will now be verified only when that
	// latest policy version will get reported
	versionAfter := getPolicyVersion(host1.ID, app1.AdamID)
	require.Greater(t, versionAfter, versionBefore)

	// the pending install is still pending for host1
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// reporting it as installed with the previous policy version does not make it verified
	policyName = fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg = statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: versionBefore,
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host1.UUID,
	)
	req = android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	// still pending
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstallPending, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// reporting it as installed with the latest policy version makes it verified
	policyName = fmt.Sprintf("enterprises/%s/policies/%s", enterpriseID, host1.UUID)
	reportMsg = statusReportMessageWithEnterpriseSpecificID(
		t,
		androidmanagement.Device{
			Name:                 deviceInfo1.Name,
			EnrollmentTokenData:  deviceInfo1.EnrollmentTokenData,
			AppliedPolicyName:    policyName,
			AppliedPolicyVersion: versionAfter,
			ApplicationReports: []*androidmanagement.ApplicationReport{
				{PackageName: app1.AdamID, State: "INSTALLED"},
			},
			LastPolicySyncTime: time.Now().Format(time.RFC3339Nano),
		},
		host1.UUID,
	)
	req = android_service.PubSubPushRequest{PubSubMessage: *reportMsg}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubSubToken.Value))

	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)

	// update app1 again, but configuration stays the same
	patchAppResp = updateAppStoreAppResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", app1TitleID), &updateAppStoreAppRequest{
		TeamID:        nil,
		Configuration: json.RawMessage(`{"managedConfiguration": 2}`),
	}, http.StatusOK, &patchAppResp)

	s.runWorkerUntilDoneWithChecks(true)

	// status stays installed
	getHostSw = getHostSoftwareResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/software", host1.ID), nil, http.StatusOK, &getHostSw, "available_for_install", "true",
		"order_key", "name")
	require.Len(t, getHostSw.Software, 2)
	require.NotNil(t, getHostSw.Software[0].AppStoreApp)
	require.Equal(t, app1.AdamID, getHostSw.Software[0].AppStoreApp.AppStoreID)
	require.NotNil(t, getHostSw.Software[0].Status)
	require.Equal(t, fleet.SoftwareInstalled, *getHostSw.Software[0].Status)
	require.NotNil(t, getHostSw.Software[1].AppStoreApp)
	require.Equal(t, app2.AdamID, getHostSw.Software[1].AppStoreApp.AppStoreID)
	require.Nil(t, getHostSw.Software[1].Status)
}

func (s *integrationMDMTestSuite) createAndEnrollAndroidDevice(t *testing.T, name string, teamID *uint) (host *fleet.Host, deviceInfo androidmanagement.Device, pubSubToken fleet.MDMConfigAsset) {
	ctx := t.Context()

	// get the required secrets to enroll an Android device
	secrets, err := s.ds.GetEnrollSecrets(ctx, teamID)
	require.NoError(t, err)
	require.Len(t, secrets, 1)

	assets, err := s.ds.GetAllMDMConfigAssetsByName(ctx, []fleet.MDMAssetName{fleet.MDMAssetAndroidPubSubToken}, nil)
	require.NoError(t, err)
	pubsubToken := assets[fleet.MDMAssetAndroidPubSubToken]
	require.NotEmpty(t, pubsubToken.Value)

	// enroll an Android device
	deviceID := createAndroidDeviceID(name)
	enterpriseSpecificID := strings.ToUpper(uuid.New().String())
	deviceInfo = androidmanagement.Device{
		Name:                deviceID,
		EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, secrets[0].Secret),
	}
	enrollmentMessage := enrollmentMessageWithEnterpriseSpecificID(t, deviceInfo, enterpriseSpecificID)

	req := android_service.PubSubPushRequest{PubSubMessage: *enrollmentMessage}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	var hosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts, "query", enterpriseSpecificID)
	require.Len(t, hosts.Hosts, 1)
	hostResp := hosts.Hosts[0]
	require.EqualValues(t, fleet.AndroidPlatform, hostResp.Host.Platform)

	return hostResp.Host, deviceInfo, pubsubToken
}
