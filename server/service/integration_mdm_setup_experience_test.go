package service

// macOS and Linux setup experience tests for the MDM suite.
//
// Belongs here: setup experience scripts and software (packages and Fleet-maintained apps) on macOS
// and Linux, the end-to-end ADE flow with automatic or forced release, required software, editing or
// deleting installers that setup experience uses, custom display names and icons, and the setup
// experience endpoints themselves (platform handling and isolation).
//
// Does not belong here: VPP apps in setup experience (integration_mdm_setup_experience_vpp_test.go),
// iOS/iPadOS (integration_mdm_setup_experience_ios_ipados_test.go), Android
// (integration_mdm_setup_experience_android_apps_test.go), and Windows/Linux setup experience on the
// enterprise suite (integration_enterprise_setup_experience_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/android"
	android_service "github.com/fleetdm/fleet/v4/server/mdm/android/service"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/push"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/androidmanagement/v1"
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

	// creating a team script generates a created_setup_experience_script activity
	s.lastActivityOfTypeMatches(
		fleet.ActivityCreatedSetupExperienceScript{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "script_name": "script42.sh"}`, tm.ID, tm.Name),
		0)

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

	// record the latest activity id before a no-op re-upload so we can assert nothing new is logged
	lastActID := s.lastActivityMatches("", "", 0)

	// try to update script with same name, should not fail because this is allowed
	body, headers = generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)

	// re-uploading identical content is a no-op and must NOT generate a new activity (GitOps re-applies every run)
	require.Equal(t, lastActID, s.lastActivityMatches("", "", 0))

	// update with a different name and contents via PUT endpoint, should suceed
	body, headers = generateNewScriptMultipartRequest(t,
		"different.sh", []byte(`echo "hello2"`), s.token, map[string][]string{"team_id": {fmt.Sprintf("%d", tm.ID)}})
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)

	// replacing the script content generates a new created_setup_experience_script activity
	s.lastActivityOfTypeMatches(
		fleet.ActivityCreatedSetupExperienceScript{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "script_name": "different.sh"}`, tm.ID, tm.Name),
		0)

	// create no-team script
	body, headers = generateNewScriptMultipartRequest(t,
		"script42.sh", []byte(`echo "hello"`), s.token, nil)
	res = s.DoRawWithHeaders("POST", "/api/latest/fleet/setup_experience/script", body.Bytes(), http.StatusOK, headers)
	err = json.NewDecoder(res.Body).Decode(&newScriptResp)
	require.NoError(t, err)

	// creating the no-team script generates a created_setup_experience_script activity with null fleet
	s.lastActivityOfTypeMatches(
		fleet.ActivityCreatedSetupExperienceScript{}.ActivityName(),
		`{"fleet_id": null, "fleet_name": null, "script_name": "script42.sh"}`,
		0)
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

	// deleting the no-team script generates a deleted_setup_experience_script activity with null fleet
	s.lastActivityOfTypeMatches(
		fleet.ActivityDeletedSetupExperienceScript{}.ActivityName(),
		`{"fleet_id": null, "fleet_name": null, "script_name": "script42.sh"}`,
		0)

	// try get the no-team script
	s.Do("GET", "/api/latest/fleet/setup_experience/script", nil, http.StatusNotFound)

	// try deleting the no-team script again, which is a no-op and must not generate a new activity
	lastActID = s.lastActivityMatches("", "", 0)
	s.Do("DELETE", "/api/latest/fleet/setup_experience/script", nil, http.StatusOK) // TODO: confirm if we want to return not found
	require.Equal(t, lastActID, s.lastActivityMatches("", "", 0))

	// // TODO: confirm if we will allow team_id=0 requests
	// s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script/?team_id=%d", noTeamID), nil, http.StatusOK)

	// delete the team script
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK)

	// deleting the team script generates a deleted_setup_experience_script activity naming the current script
	s.lastActivityOfTypeMatches(
		fleet.ActivityDeletedSetupExperienceScript{}.ActivityName(),
		fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "script_name": "different.sh"}`, tm.ID, tm.Name),
		0)

	// try get the team script
	s.Do("GET", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusNotFound)

	// try deleting the team script again, which is a no-op and must not generate a new activity
	lastActID = s.lastActivityMatches("", "", 0)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/setup_experience/script?team_id=%d", tm.ID), nil, http.StatusOK) // TODO: confirm if we want to return not found
	require.Equal(t, lastActID, s.lastActivityMatches("", "", 0))
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
		fmt.Sprintf(`{"platform": "darwin", "fleet_id": %d, "fleet_name": "%s", "team_id": %d, "team_name": "%s"}`, tm.ID, tm.Name, tm.ID, tm.Name), 0)

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
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)

	// The /setup_experience/status endpoint doesn't return the various IDs for executions, so pull
	// it out manually
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, *enrolledHost.TeamID)
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
		"software_display_name": null,
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

	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.NotZero(t, *statusResp.Results.Software[0].SoftwareTitleID)
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Script.Status)

	// Get script exec ID
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, *enrolledHost.TeamID)
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
  "hash_sha256": "%s",
  "self_service": false,
  "install_uuid": "%s",
  "status": "installed",
  "source": "apps",
  "policy_id": null,
  "policy_name": null,
  "from_setup_experience": true,
  "patch_when_closed": false
}
	`, enrolledHost.ID, getHostResp.Host.DisplayName, statusResp.Results.Software[0].Name, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage.Name, getSoftwareTitleResp.SoftwareTitle.SoftwarePackage.StorageID, installUUID)

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
	var scriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *enrolledHost.OrbitNodeKey, execID)),
		http.StatusOK, &scriptResp)

	// Get status again, now the script should be complete. This should also trigger the automatic
	// release of the device, as all setup experience steps are now complete.
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	"batch_execution_id": null,
	"from_setup_experience": true
}
	`, enrolledHost.ID, statusResp.Results.Script.Name, getHostResp.Host.DisplayName, execID)

	s.lastActivityMatches(fleet.ActivityTypeRanScript{}.ActivityName(), expectedActivityDetail, 0)
}

// TestSetupExperienceFlowWithFMAAndVersionRollback tests the full setup
// experience flow using a Fleet Maintained App (FMA) as the software to
// install, and exercises the FMA version rollback functionality: the FMA is
// added at v1.0, upgraded to v2.0 (so both are cached), then rolled back to
// v1.0 via the batch-set endpoint. The setup experience is then driven to
// completion using the rolled-back installer and the device is auto-released.
func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithFMAAndVersionRollback() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	// -------------------------------------------------------------------------
	// Set up manifest + installer mock servers for a darwin FMA using the
	// shared startFMAServers helper.
	// We use dummy_installer.pkg as the on-disk bytes so the server can parse
	// it as a real macOS pkg. The SHA is computed by fmaTestState.ComputeSHA.
	// -------------------------------------------------------------------------
	pkgBytes, err := os.ReadFile("testdata/software-installers/dummy_installer.pkg")
	require.NoError(t, err)

	// v2 uses slightly different bytes so it gets a distinct SHA and storage ID.
	v2Bytes := fmt.Append(pkgBytes, []byte("v2"))

	// fmaState is the single mutable state object the manifest server reads.
	// startFMAServers calls ComputeSHA on the initial installerBytes.
	fmaState := &fmaTestState{
		version:        "1.0",
		installerBytes: pkgBytes,
		installerPath:  "/1password.pkg",
	}

	startFMAServers(t, s.ds, map[string]*fmaTestState{
		"/1password/darwin.json": fmaState,
	})

	// -------------------------------------------------------------------------
	// Helper: issue a batch-set request and wait for completion.
	// -------------------------------------------------------------------------
	batchSet := func(tm fleet.Team, software []*fleet.SoftwareInstallerPayload) []fleet.SoftwarePackageResponse {
		var resp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{Software: software, TeamName: tm.Name},
			http.StatusAccepted, &resp,
			"team_name", tm.Name, "team_id", fmt.Sprint(tm.ID),
		)
		return waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, tm.Name, resp.RequestUUID)
	}

	// -------------------------------------------------------------------------
	// Create the team and DEP-enroll a device into it (same as the Auto-Release
	// test, but we inject the FMA instead of a custom .pkg).
	// -------------------------------------------------------------------------
	s.enableABM("fleet-setup-experience-fma")
	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "-team"})
	require.NoError(t, err)

	teamDevice := godep.Device{
		SerialNumber: uuid.New().String(),
		Model:        "MacBook Pro",
		OS:           "osx",
		OpType:       "added",
	}

	// Add a team MDM profile so we can assert on the profile install commands.
	teamProfile := mobileconfigForTest("N1", "I1")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch",
		batchSetMDMAppleProfilesRequest{Profiles: [][]byte{teamProfile}},
		http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	// -------------------------------------------------------------------------
	// Section 1: add the FMA at v1.0 via batch-set, then upgrade to v2.0 so
	// that two versions are cached, then roll back to v1.0.
	// -------------------------------------------------------------------------

	// Add v1.0.
	packages := batchSet(*tm, []*fleet.SoftwareInstallerPayload{
		{Slug: ptr.String("1password/darwin")},
	})
	require.Len(t, packages, 1)
	require.NotNil(t, packages[0].TitleID)
	fmaTitleID := *packages[0].TitleID

	// Verify the active version is v1.0.
	var titlesResp listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{}, http.StatusOK, &titlesResp,
		"available_for_install", "true",
		"team_id", fmt.Sprint(tm.ID),
	)
	require.Len(t, titlesResp.SoftwareTitles, 1)
	require.Equal(t, "1.0", titlesResp.SoftwareTitles[0].SoftwarePackage.Version)
	require.Len(t, titlesResp.SoftwareTitles[0].SoftwarePackage.FleetMaintainedVersions, 1)

	// Advance to v2.0 — both v1 and v2 are now cached.
	fmaState.version = "2.0"
	fmaState.installerBytes = v2Bytes
	fmaState.ComputeSHA(v2Bytes)
	packages = batchSet(*tm, []*fleet.SoftwareInstallerPayload{
		{Slug: ptr.String("1password/darwin")},
	})
	require.Len(t, packages, 2, "both v1.0 and v2.0 should be cached")

	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{}, http.StatusOK, &titlesResp,
		"available_for_install", "true",
		"team_id", fmt.Sprint(tm.ID),
	)
	require.Len(t, titlesResp.SoftwareTitles, 1)
	require.Equal(t, "2.0", titlesResp.SoftwareTitles[0].SoftwarePackage.Version)
	require.Len(t, titlesResp.SoftwareTitles[0].SoftwarePackage.FleetMaintainedVersions, 2)

	// Roll back to v1.0 by specifying RollbackVersion in the batch request
	// (simulating a GitOps yaml that pins fleet_maintained_app_version: "1.0").
	// The manifest server still advertises v2.0, but the rollback tells the
	// batch-set endpoint to activate the already-cached v1.0 installer instead
	// of downloading again.
	packages = batchSet(*tm, []*fleet.SoftwareInstallerPayload{
		{Slug: ptr.String("1password/darwin"), RollbackVersion: "1.0"},
	})
	require.Len(t, packages, 2, "both versions should still be cached after rollback")

	titlesResp = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{}, http.StatusOK, &titlesResp,
		"available_for_install", "true",
		"team_id", fmt.Sprint(tm.ID),
	)
	require.Len(t, titlesResp.SoftwareTitles, 1)
	require.Equal(t, "1.0", titlesResp.SoftwareTitles[0].SoftwarePackage.Version,
		"active version must be v1.0 after rollback")

	// -------------------------------------------------------------------------
	// Section 2: configure setup experience to install the (rolled-back) FMA,
	// then drive a full DEP enrollment through to auto-release.
	// -------------------------------------------------------------------------

	// Mark the FMA title as a setup experience install.
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software",
		putSetupExperienceSoftwareRequest{TeamID: tm.ID, TitleIDs: []uint{fmaTitleID}},
		http.StatusOK, &swInstallResp)

	s.lastActivityOfTypeMatches(fleet.ActivityEditedSetupExperienceSoftware{}.ActivityName(),
		fmt.Sprintf(`{"platform": "darwin", "fleet_id": %d, "fleet_name": "%s", "team_id": %d, "team_name": "%s"}`,
			tm.ID, tm.Name, tm.ID, tm.Name), 0)

	s.pushProvider.PushFunc = func(_ context.Context, pushes []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	s.mockDEPResponse("fleet-setup-experience-fma", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			require.NoError(t, encoder.Encode(map[string]string{"auth_session_token": "xyz"}))
		case "/profile":
			require.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()}))
		case "/server/devices":
			require.NoError(t, encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}}))
		case "/devices/sync":
			require.NoError(t, encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}, Cursor: "foo"}))
		case "/profile/devices":
			b, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var prof profileAssignmentReq
			require.NoError(t, json.Unmarshal(b, &prof))
			resp := godep.ProfileResponse{ProfileUUID: prof.ProfileUUID}
			resp.Devices = make(map[string]string, len(prof.Devices))
			for _, d := range prof.Devices {
				resp.Devices[d] = string(fleet.DEPAssignProfileResponseSuccess)
			}
			require.NoError(t, encoder.Encode(resp))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))

	s.runDEPSchedule()

	listHostsRes := listHostsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
	require.Len(t, listHostsRes.Hosts, 1)
	require.Equal(t, teamDevice.SerialNumber, listHostsRes.Hosts[0].HardwareSerial)
	enrolledHost := listHostsRes.Hosts[0].Host
	enrolledHost.TeamID = &tm.ID

	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tm.ID, HostIDs: []uint{enrolledHost.ID}}, http.StatusOK)

	// DEP enroll the MDM device.
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = teamDevice.SerialNumber
	require.NoError(t, mdmDevice.Enroll())

	// Ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()

	// Drain the initial MDM commands (InstallProfile × 3 + InstallEnterpriseApplication × 1).
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
	require.Len(t, cmds, 4) // 3 InstallProfile + 1 InstallEnterpriseApplication (fleetd)

	// Orbit-enroll the host (simulates fleetd being installed).
	enrolledHost.OsqueryHostID = ptr.String(mdmDevice.UUID)
	enrolledHost.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, enrolledHost, s.ds)
	enrolledHost.OrbitNodeKey = &orbitKey

	// No pending Release Device job yet.
	pending, err := s.ds.GetQueuedJobs(ctx, 1, time.Now().UTC().Add(time.Minute))
	require.NoError(t, err)
	require.Len(t, pending, 0)

	// First /status call: software pending, no script involved.
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)),
		http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)
	require.Nil(t, statusResp.Results.AccountConfiguration)
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3)
	require.Nil(t, statusResp.Results.Script)

	require.Len(t, statusResp.Results.Software, 1)

	fmaResult := statusResp.Results.Software[0]
	require.Equal(t, "1Password", fmaResult.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, fmaResult.Status)
	require.NotNil(t, fmaResult.SoftwareTitleID)
	require.Equal(t, fmaTitleID, *fmaResult.SoftwareTitleID)

	// Pull the execution ID out of the DB (the status endpoint doesn't surface it).
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, tm.ID)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.NotNil(t, results[0].HostSoftwareInstallsExecutionID)
	installUUID := *results[0].HostSoftwareInstallsExecutionID
	require.NotEmpty(t, installUUID)

	// Retrieve the title so we can read the active package name (should be v1.0).
	var titleDetail getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", fmaTitleID),
		nil, http.StatusOK, &titleDetail,
		"team_id", fmt.Sprint(tm.ID))
	require.NotNil(t, titleDetail.SoftwareTitle)
	require.NotNil(t, titleDetail.SoftwareTitle.SoftwarePackage)
	require.Equal(t, "1.0", titleDetail.SoftwareTitle.SoftwarePackage.Version,
		"the installed version during setup experience must be the rolled-back v1.0")

	// No MDM command was enqueued just from the /status call (device not released yet).
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// Second /status call: software transitions to running.
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)),
		http.StatusOK, &statusResp)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "1Password", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[0].Status)

	// Verify the upcoming activity references the rolled-back v1.0 package.
	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", enrolledHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1)
	require.NotNil(t, hostActivitiesResp.Activities[0].Details)
	var activityDetails map[string]any
	require.NoError(t, json.Unmarshal(*hostActivitiesResp.Activities[0].Details, &activityDetails))
	require.Equal(t, installUUID, activityDetails["install_uuid"])
	require.Equal(t, "1Password", activityDetails["software_title"])
	// The package name must come from the v1.0 installer, not v2.0.
	require.Equal(t, titleDetail.SoftwareTitle.SoftwarePackage.Name, activityDetails["software_package"])

	// Post a successful install result for the FMA.
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *enrolledHost.OrbitNodeKey, installUUID)), http.StatusNoContent)

	// /status call after success: software is now complete.
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)),
		http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)
	require.Nil(t, statusResp.Results.AccountConfiguration)
	require.Nil(t, statusResp.Results.Script)
	require.Len(t, statusResp.Results.Software, 1)
	require.Equal(t, "1Password", statusResp.Results.Software[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusSuccess, statusResp.Results.Software[0].Status)

	// The device should now receive a DeviceConfigured command (auto-release).
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

	// Verify the installed-software activity references the rolled-back v1.0 package.
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", enrolledHost.ID), nil, http.StatusOK, &getHostResp)

	expectedActivityDetail := fmt.Sprintf(`
{
  "host_id": %d,
  "host_display_name": "%s",
  "software_title": "1Password",
  "software_package": "%s",
  "hash_sha256": "%s",
  "self_service": false,
  "install_uuid": "%s",
  "status": "installed",
  "source": "apps",
  "policy_id": null,
  "policy_name": null,
  "from_setup_experience": true,
  "patch_when_closed": false
}
	`, enrolledHost.ID, getHostResp.Host.DisplayName, titleDetail.SoftwareTitle.SoftwarePackage.Name, titleDetail.SoftwareTitle.SoftwarePackage.StorageID, installUUID)
	s.lastActivityMatchesExtended(fleet.ActivityTypeInstalledSoftware{}.ActivityName(), expectedActivityDetail, 0, ptr.Bool(true))
}

func (s *integrationMDMTestSuite) TestSetupExperienceFlowWithSoftwareAndScriptForceRelease() {
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

func (s *integrationMDMTestSuite) TestSetupExperienceFlowUpdateScript() {
	t := s.T()
	ctx := context.Background()

	s.setSkipWorkerJobs(t)

	device, host, tm := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = device.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// run the worker to assign fleet profiles
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

	// simulate fleetd being installed and the host being orbit-enrolled now
	host.OsqueryHostID = ptr.String(mdmDevice.UUID)
	host.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// call the /status endpoint, the software and script should be pending
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
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
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, tm.ID)
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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, tm.ID)
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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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

	s.setSkipWorkerJobs(t)

	device, host, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = device.SerialNumber
	err := mdmDevice.Enroll()
	require.NoError(t, err)

	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)
	// run the worker to process the DEP enroll request
	s.awaitRunAppleMDMWorkerSchedule()

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

	// simulate fleetd being installed and the host being orbit-enrolled now
	host.OsqueryHostID = ptr.String(mdmDevice.UUID)
	host.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// call the /status endpoint, the software and script should be pending
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
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
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
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
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
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
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
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
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, *enrolledHost.TeamID)
	require.NoError(t, err)
	require.Len(t, results, 4)
	var installUUIDs []string
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			installUUIDs = append(installUUIDs, *r.HostSoftwareInstallsExecutionID)
		}
	}
	require.Len(t, installUUIDs, 1)

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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *enrolledHost.OrbitNodeKey)), http.StatusOK, &statusResp)
	// First software is now running, other software and script are still pending
	require.Equal(t, len(statusResp.Results.Software), 3)
	for i, softwareResult := range statusResp.Results.Software {
		if i == 0 {
			require.Equal(t, fleet.SetupExperienceStatusRunning, softwareResult.Status)
			continue
		}
		require.Equal(t, fleet.SetupExperienceStatusPending, softwareResult.Status)
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

	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, enrolledHost.UUID, *enrolledHost.TeamID)
	require.NoError(t, err)
	require.Len(t, results, 4)
	installUUIDs = []string{}
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			installUUIDs = append(installUUIDs, *r.HostSoftwareInstallsExecutionID)
		}
	}
	require.Len(t, installUUIDs, 2)

	// status still shows script as pending
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	require.Equal(t, fleet.SetupExperienceStatusRunning, statusResp.Results.Software[1].Status)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[2].Status)

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
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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
	require.Empty(t, hostActivitiesResp.Activities)

	// Reset the setup experience items.
	statusResp = fleet.GetOrbitSetupExperienceStatusResponse{}
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

	// the VPP app and the .ipa in-house app, which is supported for iOS/iPadOS setup experience
	require.Len(t, listSetupSoftware.SoftwareTitles, 2)
	require.Equal(t, "App 2", listSetupSoftware.SoftwareTitles[0].Name)
	require.Equal(t, "ipa_test", listSetupSoftware.SoftwareTitles[1].Name)
	require.NotNil(t, listSetupSoftware.SoftwareTitles[1].SoftwarePackage)
	require.NotNil(t, listSetupSoftware.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)
	require.False(t, *listSetupSoftware.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)

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

	// put software for setup experience ios with valid ones, including the ipa
	s.DoJSON("PUT", "/api/latest/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		Platform: "ios",
		TeamID:   0,
		TitleIDs: []uint{app2IOSTitleID, ipaTitleID},
	}, http.StatusOK, &putSetupSoftware)

	// the ipa selection round-trips on a re-GET
	listSetupSoftware = getSetupExperienceSoftwareResponse{}
	s.DoJSON("GET", "/api/latest/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{},
		http.StatusOK, &listSetupSoftware, "platform", "ios", "team_id", "0", "order_key", "name")
	require.Len(t, listSetupSoftware.SoftwareTitles, 2)
	require.Equal(t, "ipa_test", listSetupSoftware.SoftwareTitles[1].Name)
	require.NotNil(t, listSetupSoftware.SoftwareTitles[1].SoftwarePackage)
	require.NotNil(t, listSetupSoftware.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)
	require.True(t, *listSetupSoftware.SoftwareTitles[1].SoftwarePackage.InstallDuringSetup)
}

func (s *integrationMDMTestSuite) TestSetupExperienceMacOSCustomDisplayNameIcon() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	device, host, tm := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()
	token := "token_test_setup"
	createDeviceTokenForHost(t, s.ds, host.ID, token)

	// get the created setup experience software title id
	var setupExpSw getSetupExperienceSoftwareResponse
	s.DoJSON("GET", "/api/v1/fleet/setup_experience/software", getSetupExperienceSoftwareRequest{Platforms: "macos"}, http.StatusOK, &setupExpSw, "team_id", fmt.Sprint(tm.ID))
	require.Len(t, setupExpSw.SoftwareTitles, 1)
	dummyTitleID := setupExpSw.SoftwareTitles[0].ID

	// add an additional macOS software to install during setup experience
	installer := &fleet.UploadSoftwareInstallerPayload{
		InstallScript: "install",
		Filename:      "EchoApp.pkg",
		Title:         "EchoApp",
		TeamID:        &tm.ID,
	}
	s.uploadSoftwareInstaller(t, installer, http.StatusOK, "")
	echoTitleID := getSoftwareTitleID(t, s.ds, installer.Title, "apps")
	var swInstallResp putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{TeamID: tm.ID, TitleIDs: []uint{echoTitleID, dummyTitleID}}, http.StatusOK, &swInstallResp)

	// set a custom icon and custom display name for that app
	s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
		TitleID:     echoTitleID,
		TeamID:      &tm.ID,
		DisplayName: new("My Custom EchoApp"),
	}, http.StatusOK, "")

	iconBytes, err := os.ReadFile("testdata/icons/valid-icon.png")
	require.NoError(t, err)
	body, headers := generateMultipartRequest(t, "icon", "icon.png", iconBytes, s.token, nil)
	s.DoRawWithHeaders("PUT", fmt.Sprintf("/api/latest/fleet/software/titles/%d/icon?team_id=%d", echoTitleID, tm.ID),
		body.Bytes(), http.StatusOK, headers)

	// enroll the host
	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = device.SerialNumber
	err = mdmDevice.Enroll()
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

	// simulate fleetd being installed and the host being orbit-enrolled now
	host.OsqueryHostID = new(mdmDevice.UUID)
	host.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	// call the /status endpoint, the 2 software and script should be pending
	var statusResp fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	require.Nil(t, statusResp.Results.BootstrapPackage)         // no bootstrap package involved
	require.Nil(t, statusResp.Results.AccountConfiguration)     // no SSO involved
	require.Len(t, statusResp.Results.ConfigurationProfiles, 3) // fleetd config, root CA, custom profile

	// the 2 software and script are pending
	require.NotNil(t, statusResp.Results.Script)
	require.Equal(t, "script.sh", statusResp.Results.Script.Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Script.Status)
	require.Len(t, statusResp.Results.Software, 2)
	require.Equal(t, "DummyApp", statusResp.Results.Software[0].Name)
	require.Empty(t, statusResp.Results.Software[0].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[0].Status)
	require.NotNil(t, statusResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, dummyTitleID, *statusResp.Results.Software[0].SoftwareTitleID)
	require.Empty(t, statusResp.Results.Software[0].IconURL)
	require.Equal(t, "EchoApp", statusResp.Results.Software[1].Name)
	require.Equal(t, "My Custom EchoApp", statusResp.Results.Software[1].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusPending, statusResp.Results.Software[1].Status)
	require.NotNil(t, statusResp.Results.Software[1].SoftwareTitleID)
	require.Equal(t, echoTitleID, *statusResp.Results.Software[1].SoftwareTitleID)
	require.NotEmpty(t, statusResp.Results.Software[1].IconURL)

	// since this was the call for the orbit endpoint, and not the device-authenticated
	// one, the URL was not transformed for device-authenticated
	require.NotContains(t, statusResp.Results.Software[1].IconURL, "/device/")

	// requesting the setup experience status via the device-authenticated endpoint
	// returns the custom icon URL ready to be called via device-auth.
	var deviceResp getDeviceSetupExperienceStatusResponse
	res := s.DoRawNoAuth("POST", "/api/latest/fleet/device/"+token+"/setup_experience/status", json.RawMessage{}, http.StatusOK)
	require.NoError(t, json.NewDecoder(res.Body).Decode(&deviceResp))
	require.NoError(t, res.Body.Close())
	require.NoError(t, deviceResp.Err)

	// Software is installed one at a time in alphabetical order by display
	// name (falling back to name). The previous call to /status kickstarted
	// the first item; the second remains pending until the first finishes.
	require.Len(t, deviceResp.Results.Scripts, 1)
	require.Equal(t, "script.sh", deviceResp.Results.Scripts[0].Name)
	require.Equal(t, fleet.SetupExperienceStatusPending, deviceResp.Results.Scripts[0].Status)
	require.Len(t, deviceResp.Results.Software, 2)
	require.Equal(t, "DummyApp", deviceResp.Results.Software[0].Name)
	require.Empty(t, deviceResp.Results.Software[0].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusRunning, deviceResp.Results.Software[0].Status)
	require.NotNil(t, deviceResp.Results.Software[0].SoftwareTitleID)
	require.Equal(t, dummyTitleID, *deviceResp.Results.Software[0].SoftwareTitleID)
	require.Empty(t, deviceResp.Results.Software[0].IconURL)
	require.Equal(t, "EchoApp", deviceResp.Results.Software[1].Name)
	require.Equal(t, "My Custom EchoApp", deviceResp.Results.Software[1].DisplayName)
	require.Equal(t, fleet.SetupExperienceStatusPending, deviceResp.Results.Software[1].Status)
	require.NotNil(t, deviceResp.Results.Software[1].SoftwareTitleID)
	require.Equal(t, echoTitleID, *deviceResp.Results.Software[1].SoftwareTitleID)
	require.NotEmpty(t, deviceResp.Results.Software[1].IconURL)

	require.Contains(t, deviceResp.Results.Software[1].IconURL, "/device/"+token)
}

func (s *integrationMDMTestSuite) createAndEnrollAndroidDevice(t *testing.T, name string, teamID *uint, companyOwned bool) (host *fleet.Host, deviceInfo androidmanagement.Device, pubSubToken fleet.MDMConfigAsset) {
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
	identifier := strings.ToUpper(uuid.New().String())
	deviceInfo = androidmanagement.Device{
		Name:                deviceID,
		EnrollmentTokenData: fmt.Sprintf(`{"EnrollSecret": "%s"}`, secrets[0].Secret),
	}
	var enrollmentMessage *android.PubSubMessage
	if companyOwned {
		enrollmentMessage = enrollmentMessageWithSerialNumber(t, deviceInfo, identifier)
	} else {
		enrollmentMessage = enrollmentMessageWithEnterpriseSpecificID(t, deviceInfo, identifier)
	}

	req := android_service.PubSubPushRequest{PubSubMessage: *enrollmentMessage}
	s.Do("POST", "/api/v1/fleet/android_enterprise/pubsub", &req, http.StatusOK, "token", string(pubsubToken.Value))

	var hosts listHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &hosts, "query", identifier)
	require.Len(t, hosts.Hosts, 1)
	hostResp := hosts.Hosts[0]
	require.EqualValues(t, fleet.AndroidPlatform, hostResp.Host.Platform)

	if teamID != nil {
		require.NotNil(t, hostResp.Host.TeamID)
		require.EqualValues(t, *teamID, *hostResp.Host.TeamID)
	} else {
		require.Nil(t, hostResp.Host.TeamID)
	}

	return hostResp.Host, deviceInfo, pubsubToken
}

func (s *integrationMDMTestSuite) TestLinuxSetupExperienceEnqueueSoftwareInstalls() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	// add a macOS software to install
	payloadDummy := &fleet.UploadSoftwareInstallerPayload{
		Filename: "dummy_installer.pkg",
		Title:    "DummyApp",
		TeamID:   nil,
	}
	s.uploadSoftwareInstaller(t, payloadDummy, http.StatusOK, "")
	macTitleID := getSoftwareTitleID(t, s.ds, payloadDummy.Title, "apps")
	require.NotZero(t, macTitleID)

	// add a .deb custom package
	payloadRuby := &fleet.UploadSoftwareInstallerPayload{
		Filename: "ruby.deb",
		Title:    "ruby",
		Platform: "linux",
		TeamID:   nil,
	}
	s.uploadSoftwareInstaller(t, payloadRuby, http.StatusOK, "")
	debTitleID := getSoftwareTitleID(t, s.ds, payloadRuby.Title, "deb_packages")
	require.NotZero(t, debTitleID)

	// add a .sh script-only package
	payloadSh := &fleet.UploadSoftwareInstallerPayload{
		Filename: "script.sh",
		Title:    "script",
		Platform: "linux",
		TeamID:   nil,
	}
	s.uploadSoftwareInstaller(t, payloadSh, http.StatusOK, "")
	shTitleID := getSoftwareTitleID(t, s.ds, payloadSh.Title, "sh_packages")
	require.NotZero(t, shTitleID)

	var putSetupExpResponse putSetupExperienceSoftwareResponse
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		TeamID: 0, Platform: "linux", TitleIDs: []uint{debTitleID, shTitleID},
	}, http.StatusOK, &putSetupExpResponse)
	s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
		TeamID: 0, Platform: "macos", TitleIDs: []uint{macTitleID},
	}, http.StatusOK, &putSetupExpResponse)

	// create an arch linux host (platform_like is empty)
	hostArch := createOrbitEnrolledHost(t, "arch", "host1", s.ds)
	createDeviceTokenForHost(t, s.ds, hostArch.ID, uuid.NewString())

	// create a ubuntu host (platform_like is "debian")
	hostUbuntu := createOrbitEnrolledHost(t, "ubuntu", "host2", s.ds)
	hostUbuntu.PlatformLike = "debian"
	require.NoError(t, s.ds.UpdateHost(ctx, hostUbuntu))
	createDeviceTokenForHost(t, s.ds, hostUbuntu.ID, uuid.NewString())

	// trigger setup experience for arch
	var orbitInitResponse fleet.OrbitSetupExperienceInitResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
		OrbitNodeKey: *hostArch.OrbitNodeKey,
	}, http.StatusOK, &orbitInitResponse)
	require.True(t, orbitInitResponse.Result.Enabled)

	// get status of the "Setup experience", only the .sh is compatible
	var orbitStatusResponse fleet.GetOrbitSetupExperienceStatusResponse
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", fleet.GetOrbitSetupExperienceStatusRequest{
		OrbitNodeKey: *hostArch.OrbitNodeKey,
	}, http.StatusOK, &orbitStatusResponse)
	require.Nil(t, orbitStatusResponse.Results.Script)
	require.Nil(t, orbitStatusResponse.Results.BootstrapPackage)
	require.Len(t, orbitStatusResponse.Results.ConfigurationProfiles, 0)
	require.Nil(t, orbitStatusResponse.Results.AccountConfiguration)
	require.False(t, orbitStatusResponse.Results.RequireAllSoftware)

	require.Len(t, orbitStatusResponse.Results.Software, 1)
	require.Equal(t, payloadSh.Title, orbitStatusResponse.Results.Software[0].Name)
	require.NotNil(t, orbitStatusResponse.Results.Software[0].SoftwareTitleID)
	require.Equal(t, shTitleID, *orbitStatusResponse.Results.Software[0].SoftwareTitleID)

	// trigger setup experience for ubuntu
	orbitInitResponse = fleet.OrbitSetupExperienceInitResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/init", fleet.OrbitSetupExperienceInitRequest{
		OrbitNodeKey: *hostUbuntu.OrbitNodeKey,
	}, http.StatusOK, &orbitInitResponse)
	require.True(t, orbitInitResponse.Result.Enabled)

	// get status of the "Setup experience", both the .sh and .deb are enqueued
	orbitStatusResponse = fleet.GetOrbitSetupExperienceStatusResponse{}
	s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status", fleet.GetOrbitSetupExperienceStatusRequest{
		OrbitNodeKey: *hostUbuntu.OrbitNodeKey,
	}, http.StatusOK, &orbitStatusResponse)
	require.Nil(t, orbitStatusResponse.Results.Script)
	require.Nil(t, orbitStatusResponse.Results.BootstrapPackage)
	require.Len(t, orbitStatusResponse.Results.ConfigurationProfiles, 0)
	require.Nil(t, orbitStatusResponse.Results.AccountConfiguration)
	require.False(t, orbitStatusResponse.Results.RequireAllSoftware)

	require.Len(t, orbitStatusResponse.Results.Software, 2)
	sort.Slice(orbitStatusResponse.Results.Software, func(i, j int) bool {
		return orbitStatusResponse.Results.Software[i].Name < orbitStatusResponse.Results.Software[j].Name
	})
	require.Equal(t, payloadRuby.Title, orbitStatusResponse.Results.Software[0].Name)
	require.NotNil(t, orbitStatusResponse.Results.Software[0].SoftwareTitleID)
	require.Equal(t, debTitleID, *orbitStatusResponse.Results.Software[0].SoftwareTitleID)
	require.Equal(t, payloadSh.Title, orbitStatusResponse.Results.Software[1].Name)
	require.NotNil(t, orbitStatusResponse.Results.Software[1].SoftwareTitleID)
	require.Equal(t, shTitleID, *orbitStatusResponse.Results.Software[1].SoftwareTitleID)
}

func (s *integrationMDMTestSuite) TestSetupExperienceInstallerEditAndDelete() {
	tOuter := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(tOuter)

	s.pushProvider.PushFunc = func(_ context.Context, _ []*mdm.Push) (map[string]*push.Response, error) {
		return map[string]*push.Response{}, nil
	}

	enrollHostWithSEInstallers := func(t *testing.T, installers []struct {
		Filename string
		Title    string
	},
	) (*fleet.Host, *mdmtest.TestAppleMDMClient, map[string]uint) {
		// unique per-subtest team name and ABM org so subtests don't collide
		isoName := strings.ReplaceAll(t.Name(), "/", "_")
		s.enableABM(isoName)
		tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: isoName})
		require.NoError(t, err)

		teamDevice := godep.Device{SerialNumber: uuid.New().String(), Model: "MacBook Pro", OS: "osx", OpType: "added"}

		titleIDs := make(map[string]uint, len(installers))
		var titleIDList []uint
		for _, inst := range installers {
			s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
				InstallScript: "install script for " + inst.Title,
				Filename:      inst.Filename,
				Title:         inst.Title,
				TeamID:        &tm.ID,
			}, http.StatusOK, "")
			titleIDs[inst.Title] = getSoftwareTitleID(t, s.ds, inst.Title, "apps")
			titleIDList = append(titleIDList, titleIDs[inst.Title])
		}
		var swInstallResp putSetupExperienceSoftwareResponse
		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software",
			putSetupExperienceSoftwareRequest{TeamID: tm.ID, TitleIDs: titleIDList},
			http.StatusOK, &swInstallResp)

		s.mockDEPResponse(isoName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// assert (not require) in HTTP handlers per testifylint go-require
			encoder := json.NewEncoder(w)
			switch r.URL.Path {
			case "/session":
				assert.NoError(t, encoder.Encode(map[string]string{"auth_session_token": "xyz"}))
			case "/profile":
				assert.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()}))
			case "/server/devices":
				assert.NoError(t, encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}}))
			case "/devices/sync":
				assert.NoError(t, encoder.Encode(godep.DeviceResponse{Devices: []godep.Device{teamDevice}, Cursor: "foo"}))
			case "/profile/devices":
				b, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				var prof profileAssignmentReq
				assert.NoError(t, json.Unmarshal(b, &prof))
				resp := godep.ProfileResponse{ProfileUUID: prof.ProfileUUID, Devices: make(map[string]string, len(prof.Devices))}
				for _, device := range prof.Devices {
					resp.Devices[device] = string(fleet.DEPAssignProfileResponseSuccess)
				}
				assert.NoError(t, encoder.Encode(resp))
			default:
				_, _ = w.Write([]byte(`{}`))
			}
		}))

		s.runDEPSchedule()

		listHostsRes := listHostsResponse{}
		s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsRes)
		var host *fleet.Host
		for _, h := range listHostsRes.Hosts {
			if h.HardwareSerial == teamDevice.SerialNumber {
				host = h.Host
				break
			}
		}
		require.NotNil(t, host, "could not find host by serial after DEP schedule")
		host.TeamID = &tm.ID
		s.Do("POST", "/api/v1/fleet/hosts/transfer",
			addHostsToTeamRequest{TeamID: &tm.ID, HostIDs: []uint{host.ID}}, http.StatusOK)

		depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
		mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
		mdmDevice.SerialNumber = teamDevice.SerialNumber
		require.NoError(t, mdmDevice.Enroll())

		s.awaitTriggerProfileSchedule(t)
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()

		// drain MDM commands
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}

		host.OsqueryHostID = ptr.String(mdmDevice.UUID)
		host.UUID = mdmDevice.UUID
		orbitKey := setOrbitEnrollment(t, host, s.ds)
		host.OrbitNodeKey = &orbitKey

		return host, mdmDevice, titleIDs
	}

	pollOrbitSetupStatus := func(t *testing.T, host *fleet.Host) fleet.GetOrbitSetupExperienceStatusResponse {
		var statusResp fleet.GetOrbitSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)),
			http.StatusOK, &statusResp)
		return statusResp
	}

	// edits an installer while its SE row is pending and again while running, under
	// require_all_software_macos=true. both edits must leave the SE rows as-is and
	// must not flip the running row to failure on the next orbit poll.
	tOuter.Run("edit installer leaves setup experience intact", func(t *testing.T) {
		host, _, titleIDs := enrollHostWithSEInstallers(t, []struct{ Filename, Title string }{
			{"dummy_installer.pkg", "DummyApp"},
			{"EchoApp.pkg", "EchoApp"},
			{"no_version.pkg", "NoVersion"},
		})

		s.Do("PATCH", "/api/latest/fleet/setup_experience",
			json.RawMessage(fmt.Sprintf(`{"require_all_software_macos": true, "team_id": %d}`, *host.TeamID)),
			http.StatusNoContent)

		// first poll: DummyApp runs (alphabetically first), EchoApp and NoVersion stay pending
		pollOrbitSetupStatus(t, host)

		// edit NoVersion while its SE row is pending. must not cancel it.
		s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
			TitleID:       titleIDs["NoVersion"],
			TeamID:        host.TeamID,
			InstallScript: ptr.String("updated install script for NoVersion"),
		}, http.StatusOK, "")

		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var dummyInstallUUID string
		var noVersion *fleet.SetupExperienceStatusResult
		for _, r := range results {
			switch r.Name {
			case "DummyApp":
				if r.HostSoftwareInstallsExecutionID != nil {
					dummyInstallUUID = *r.HostSoftwareInstallsExecutionID
				}
			case "NoVersion":
				noVersion = r
			}
		}
		require.NotEmpty(t, dummyInstallUUID)
		require.NotNil(t, noVersion)
		require.Equal(t, fleet.SetupExperienceStatusPending, noVersion.Status,
			"pending SE row must not be cancelled by an installer edit")

		// complete DummyApp so EchoApp advances to running
		s.Do("POST", "/api/fleet/orbit/software_install/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "install_uuid": %q, "install_script_exit_code": 0, "install_script_output": "ok"}`,
				*host.OrbitNodeKey, dummyInstallUUID)),
			http.StatusNoContent)
		pollOrbitSetupStatus(t, host)

		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var echoInstallUUIDBefore string
		for _, r := range results {
			if r.Name == "EchoApp" {
				require.Equal(t, fleet.SetupExperienceStatusRunning, r.Status, "precondition: EchoApp should be running")
				require.NotNil(t, r.HostSoftwareInstallsExecutionID)
				echoInstallUUIDBefore = *r.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, echoInstallUUIDBefore)

		// edit EchoApp while its SE row is running. must not cancel it, hsi row must survive.
		s.updateSoftwareInstaller(t, &fleet.UpdateSoftwareInstallerPayload{
			TitleID:       titleIDs["EchoApp"],
			TeamID:        host.TeamID,
			InstallScript: ptr.String("updated install script for EchoApp"),
		}, http.StatusOK, "")

		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var echo *fleet.SetupExperienceStatusResult
		for _, r := range results {
			if r.Name == "EchoApp" {
				echo = r
			}
		}
		require.NotNil(t, echo)
		require.Equal(t, fleet.SetupExperienceStatusRunning, echo.Status,
			"running SE row must not be cancelled by an installer edit")
		require.NotNil(t, echo.HostSoftwareInstallsExecutionID)
		require.Equal(t, echoInstallUUIDBefore, *echo.HostSoftwareInstallsExecutionID,
			"execution id must survive the edit so the eventual result can land on the right row")

		hsi, err := s.ds.GetSoftwareInstallResults(ctx, echoInstallUUIDBefore)
		require.NoError(t, err, "host_software_installs row must survive the edit")
		require.Equal(t, fleet.SoftwareInstallPending, hsi.Status)

		// orbit endpoint must show the expected final state: DummyApp success, EchoApp running, NoVersion pending
		statusAfter := pollOrbitSetupStatus(t, host)
		require.Len(t, statusAfter.Results.Software, 3)
		var dummyOrbit, echoOrbit, noVersionOrbit *fleet.SetupExperienceStatusResult
		for _, r := range statusAfter.Results.Software {
			switch r.Name {
			case "DummyApp":
				dummyOrbit = r
			case "EchoApp":
				echoOrbit = r
			case "NoVersion":
				noVersionOrbit = r
			}
		}
		require.NotNil(t, dummyOrbit)
		require.NotNil(t, echoOrbit)
		require.NotNil(t, noVersionOrbit)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, dummyOrbit.Status)
		require.Equal(t, fleet.SetupExperienceStatusRunning, echoOrbit.Status,
			"running SE row must not flip to failure after an installer edit")
		require.Equal(t, fleet.SetupExperienceStatusPending, noVersionOrbit.Status)
	})

	// covers delete-while-pending and delete-while-running on the same host.
	// The running case verifies the hsi row is preserved as canceled with its installer id nulled.
	tOuter.Run("delete installer removes setup experience row", func(t *testing.T) {
		host, _, titleIDs := enrollHostWithSEInstallers(t, []struct{ Filename, Title string }{
			{"dummy_installer.pkg", "DummyApp"},
			{"EchoApp.pkg", "EchoApp"},
			{"no_version.pkg", "NoVersion"},
		})

		// drive DummyApp to success, EchoApp to running. NoVersion stays pending.
		pollOrbitSetupStatus(t, host)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var dummyInstallUUID string
		for _, r := range results {
			if r.Name == "DummyApp" && r.HostSoftwareInstallsExecutionID != nil {
				dummyInstallUUID = *r.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, dummyInstallUUID)
		s.Do("POST", "/api/fleet/orbit/software_install/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "install_uuid": %q, "install_script_exit_code": 0, "install_script_output": "ok"}`,
				*host.OrbitNodeKey, dummyInstallUUID)),
			http.StatusNoContent)
		pollOrbitSetupStatus(t, host)

		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var echoInstallUUID string
		for _, r := range results {
			if r.Name == "EchoApp" {
				require.Equal(t, fleet.SetupExperienceStatusRunning, r.Status, "precondition: EchoApp should be running")
				require.NotNil(t, r.HostSoftwareInstallsExecutionID)
				echoInstallUUID = *r.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, echoInstallUUID)

		// remove both EchoApp and NoVersion from the SE list so both installers can be deleted
		var swInstallResp putSetupExperienceSoftwareResponse
		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software",
			putSetupExperienceSoftwareRequest{TeamID: *host.TeamID, TitleIDs: []uint{titleIDs["DummyApp"]}},
			http.StatusOK, &swInstallResp)

		// delete EchoApp (was running) and NoVersion (was pending)
		s.Do("DELETE",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install?team_id=%d", titleIDs["EchoApp"], *host.TeamID),
			nil, http.StatusNoContent)
		s.Do("DELETE",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install?team_id=%d", titleIDs["NoVersion"], *host.TeamID),
			nil, http.StatusNoContent)

		// both SE rows must be removed by the FK ON DELETE CASCADE
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		for _, r := range results {
			require.NotEqual(t, "EchoApp", r.Name, "EchoApp SE row must be removed after installer delete")
			require.NotEqual(t, "NoVersion", r.Name, "NoVersion SE row must be removed after installer delete")
		}

		// the running install is marked canceled with software_installer_id nulled, so it's filtered out of
		// GetSoftwareInstallResults but still available for a late result.
		_, err = s.ds.GetSoftwareInstallResults(ctx, echoInstallUUID)
		require.True(t, fleet.IsNotFound(err), "canceled install must not be returned by GetSoftwareInstallResults")

		var echoRow struct {
			InstallerID *uint `db:"software_installer_id"`
			Canceled    bool  `db:"canceled"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &echoRow, `SELECT software_installer_id, canceled FROM host_software_installs WHERE execution_id = ?`, echoInstallUUID)
		})
		require.Nil(t, echoRow.InstallerID, "installer id must be nulled by the FK on installer delete")
		require.True(t, echoRow.Canceled, "running install must be preserved as canceled, not deleted")

		// orbit endpoint must show only DummyApp now, still successful
		statusAfter := pollOrbitSetupStatus(t, host)
		require.Len(t, statusAfter.Results.Software, 1)
		require.Equal(t, "DummyApp", statusAfter.Results.Software[0].Name)
		require.Equal(t, fleet.SetupExperienceStatusSuccess, statusAfter.Results.Software[0].Status)

		// a late result for the deleted install must not 500: the canceled row still lets
		// CreateIntermediateInstallFailureRecord find the original install details.
		s.Do("POST", "/api/fleet/orbit/software_install/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "install_uuid": %q, "install_script_exit_code": 1, "install_script_output": "boom", "retries_remaining": 1}`,
				*host.OrbitNodeKey, echoInstallUUID)),
			http.StatusNoContent)

		// the intermediate failure record was written as a NEW row (distinct from the canceled original),
		// carrying the reported failure output and the denormalized installer details from the original.
		var failureCount int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &failureCount,
				`SELECT COUNT(*) FROM host_software_installs
				 WHERE host_id = ? AND execution_id != ? AND install_script_output = ?
				 AND software_title_name = ? AND installer_filename = ?`,
				host.ID, echoInstallUUID, "boom", "EchoApp", "EchoApp.pkg")
		})
		require.Equal(t, 1, failureCount)

		// a completed install is marked removed (not canceled) when its installer is deleted.
		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software",
			putSetupExperienceSoftwareRequest{TeamID: *host.TeamID, TitleIDs: []uint{}},
			http.StatusOK, &swInstallResp)
		s.Do("DELETE",
			fmt.Sprintf("/api/latest/fleet/software/titles/%d/available_for_install?team_id=%d", titleIDs["DummyApp"], *host.TeamID),
			nil, http.StatusNoContent)

		var dummyRow struct {
			InstallerID *uint `db:"software_installer_id"`
			Removed     bool  `db:"removed"`
			Canceled    bool  `db:"canceled"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &dummyRow, `SELECT software_installer_id, removed, canceled FROM host_software_installs WHERE execution_id = ?`, dummyInstallUUID)
		})
		require.Nil(t, dummyRow.InstallerID)
		require.True(t, dummyRow.Removed, "completed install must be marked removed when its installer is deleted")
		require.False(t, dummyRow.Canceled, "completed install must not be canceled")
	})

	// covers the GitOps batch endpoint for both an installer edit (the
	// runInstallerUpdateSideEffectsInTransaction isEdit=true path) and a not-in-list delete (the
	// cancelSetupExperienceStatusForDeletedSoftwareInstalls + cancelPendingSoftwareInstallsNotInListHSI path).
	tOuter.Run("gitops batch edit then delete via /software/batch", func(t *testing.T) {
		host, _, _ := enrollHostWithSEInstallers(t, []struct{ Filename, Title string }{
			{"dummy_installer.pkg", "DummyApp"},
			{"EchoApp.pkg", "EchoApp"},
		})

		// drive DummyApp to success, EchoApp to running
		pollOrbitSetupStatus(t, host)
		results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var dummyInstallUUID string
		for _, r := range results {
			if r.Name == "DummyApp" && r.HostSoftwareInstallsExecutionID != nil {
				dummyInstallUUID = *r.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, dummyInstallUUID)
		s.Do("POST", "/api/fleet/orbit/software_install/result",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "install_uuid": %q, "install_script_exit_code": 0, "install_script_output": "ok"}`,
				*host.OrbitNodeKey, dummyInstallUUID)),
			http.StatusNoContent)
		pollOrbitSetupStatus(t, host)

		// snapshot EchoApp's running install execution id
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var echoInstallUUIDBefore string
		for _, r := range results {
			if r.Name == "EchoApp" {
				require.Equal(t, fleet.SetupExperienceStatusRunning, r.Status, "precondition: EchoApp should be running")
				require.NotNil(t, r.HostSoftwareInstallsExecutionID)
				echoInstallUUIDBefore = *r.HostSoftwareInstallsExecutionID
			}
		}
		require.NotEmpty(t, echoInstallUUIDBefore)

		// HTTP server to serve the installer files for the batch endpoint
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			file, err := os.Open(filepath.Join("testdata", "software-installers", strings.TrimPrefix(r.URL.Path, "/")))
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			defer file.Close()
			_, _ = io.Copy(w, file)
		}))
		t.Cleanup(srv.Close)

		teamName := strings.ReplaceAll(t.Name(), "/", "_")

		// GitOps edit: batch with both installers, EchoApp's install script changed.
		var batchResp batchSetSoftwareInstallersResponse
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{
				Software: []*fleet.SoftwareInstallerPayload{
					{URL: srv.URL + "/dummy_installer.pkg", InstallScript: "install", InstallDuringSetup: new(true)},
					{URL: srv.URL + "/EchoApp.pkg", InstallScript: "updated install script for EchoApp", InstallDuringSetup: new(true)},
				},
			}, http.StatusAccepted, &batchResp, "team_name", teamName)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, batchResp.RequestUUID)

		// SE row for EchoApp must still be running, hsi row preserved
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		var echo *fleet.SetupExperienceStatusResult
		for _, r := range results {
			if r.Name == "EchoApp" {
				echo = r
			}
		}
		require.NotNil(t, echo)
		require.Equal(t, fleet.SetupExperienceStatusRunning, echo.Status,
			"running SE row must not be cancelled by GitOps batch-edit")
		require.NotNil(t, echo.HostSoftwareInstallsExecutionID)
		require.Equal(t, echoInstallUUIDBefore, *echo.HostSoftwareInstallsExecutionID)
		hsi, err := s.ds.GetSoftwareInstallResults(ctx, echoInstallUUIDBefore)
		require.NoError(t, err, "host_software_installs row must survive the batch-edit")
		require.Equal(t, fleet.SoftwareInstallPending, hsi.Status)

		// GitOps delete: batch with only DummyApp (EchoApp dropped from list).
		batchResp = batchSetSoftwareInstallersResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/batch",
			batchSetSoftwareInstallersRequest{
				Software: []*fleet.SoftwareInstallerPayload{
					{URL: srv.URL + "/dummy_installer.pkg", InstallScript: "install", InstallDuringSetup: new(true)},
				},
			}, http.StatusAccepted, &batchResp, "team_name", teamName)
		waitBatchSetSoftwareInstallersCompleted(t, &s.withServer, teamName, batchResp.RequestUUID)

		// EchoApp SE row gone; its running install is canceled with its installer id nulled, so it's
		// filtered out of GetSoftwareInstallResults but still on the host.
		results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
		require.NoError(t, err)
		for _, r := range results {
			require.NotEqual(t, "EchoApp", r.Name, "EchoApp SE row must be removed after GitOps delete")
		}
		_, err = s.ds.GetSoftwareInstallResults(ctx, echoInstallUUIDBefore)
		require.True(t, fleet.IsNotFound(err), "canceled install must not be returned by GetSoftwareInstallResults")

		var echoBatchRow struct {
			InstallerID *uint `db:"software_installer_id"`
			Canceled    bool  `db:"canceled"`
		}
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &echoBatchRow, `SELECT software_installer_id, canceled FROM host_software_installs WHERE execution_id = ?`, echoInstallUUIDBefore)
		})
		require.Nil(t, echoBatchRow.InstallerID)
		require.True(t, echoBatchRow.Canceled, "running install must be preserved as canceled after GitOps delete")
	})
}

func (s *integrationMDMTestSuite) TestSetupExperienceMacOSScriptOnlyPackage() {
	t := s.T()
	ctx := context.Background()

	team, err := s.ds.NewTeam(ctx, &fleet.Team{Name: "macos-sh-test"})
	require.NoError(t, err)

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		Filename: "script.sh",
		Platform: "linux",
		TeamID:   &team.ID,
	}, http.StatusOK, "")
	shTitleID := getSoftwareTitleID(t, s.ds, "script", "sh_packages")

	s.uploadSoftwareInstaller(t, &fleet.UploadSoftwareInstallerPayload{
		Filename: "dummy_installer.pkg",
		TeamID:   &team.ID,
	}, http.StatusOK, "")
	macosTitleID := getSoftwareTitleID(t, s.ds, "DummyApp", "apps")

	t.Run("sh appears in macOS listing", func(t *testing.T) {
		var resp getSetupExperienceSoftwareResponse
		s.DoJSON("GET", "/api/v1/fleet/setup_experience/software", nil, http.StatusOK, &resp,
			"platform", "macos", "team_id", fmt.Sprint(team.ID))
		names := make([]string, 0, len(resp.SoftwareTitles))
		for _, title := range resp.SoftwareTitles {
			if title.SoftwarePackage != nil {
				names = append(names, title.SoftwarePackage.Name)
			}
		}
		require.Contains(t, names, "script.sh")
		require.Contains(t, names, "dummy_installer.pkg")
	})

	t.Run("saving sh for macOS succeeds", func(t *testing.T) {
		var resp putSetupExperienceSoftwareResponse
		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
			Platform: "macos",
			TeamID:   team.ID,
			TitleIDs: []uint{shTitleID, macosTitleID},
		}, http.StatusOK, &resp)
		require.NoError(t, resp.Err)
	})

	t.Run("macOS listing shows sh as selected after save", func(t *testing.T) {
		var resp getSetupExperienceSoftwareResponse
		s.DoJSON("GET", "/api/v1/fleet/setup_experience/software", nil, http.StatusOK, &resp,
			"platform", "macos", "team_id", fmt.Sprint(team.ID))
		var shSelected, macosSelected bool
		for _, title := range resp.SoftwareTitles {
			if title.SoftwarePackage == nil {
				continue
			}
			if title.SoftwarePackage.Name == "script.sh" && title.SoftwarePackage.InstallDuringSetup != nil {
				shSelected = *title.SoftwarePackage.InstallDuringSetup
			}
			if title.SoftwarePackage.Name == "dummy_installer.pkg" && title.SoftwarePackage.InstallDuringSetup != nil {
				macosSelected = *title.SoftwarePackage.InstallDuringSetup
			}
		}
		require.True(t, shSelected)
		require.True(t, macosSelected)
	})

	t.Run("linux tab selection is independent", func(t *testing.T) {
		var resp putSetupExperienceSoftwareResponse
		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
			Platform: "linux",
			TeamID:   team.ID,
			TitleIDs: []uint{shTitleID},
		}, http.StatusOK, &resp)
		require.NoError(t, resp.Err)

		s.DoJSON("PUT", "/api/v1/fleet/setup_experience/software", putSetupExperienceSoftwareRequest{
			Platform: "macos",
			TeamID:   team.ID,
			TitleIDs: []uint{},
		}, http.StatusOK, &resp)
		require.NoError(t, resp.Err)

		var linuxResp getSetupExperienceSoftwareResponse
		s.DoJSON("GET", "/api/v1/fleet/setup_experience/software", nil, http.StatusOK, &linuxResp,
			"platform", "linux", "team_id", fmt.Sprint(team.ID))
		var linuxShSelected bool
		for _, title := range linuxResp.SoftwareTitles {
			if title.SoftwarePackage != nil && title.SoftwarePackage.Name == "script.sh" && title.SoftwarePackage.InstallDuringSetup != nil {
				linuxShSelected = *title.SoftwarePackage.InstallDuringSetup
			}
		}
		require.True(t, linuxShSelected)
	})
}

func (s *integrationMDMTestSuite) TestSetupExperienceStatusPollsSendOneDeviceConfigured() {
	t := s.T()
	ctx := t.Context()

	s.setSkipWorkerJobs(t)

	device, host, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = device.SerialNumber
	require.NoError(t, mdmDevice.Enroll())

	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	ackAll := func() []string {
		var requestTypes []string
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType != "DeclarativeManagement" {
				requestTypes = append(requestTypes, cmd.Command.RequestType)
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		return requestTypes
	}
	ackAll()

	host.OsqueryHostID = new(mdmDevice.UUID)
	host.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	pollStatus := func() {
		var statusResp fleet.GetOrbitSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
	}
	countDeviceConfigured := func() int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `
				SELECT COUNT(*) FROM nano_enrollment_queue nq
				JOIN nano_commands nc ON nc.command_uuid = nq.command_uuid
				WHERE nq.id = ? AND nc.request_type = 'DeviceConfigured'`, host.UUID)
		})
		return count
	}

	// Cancel the software install and then the script so that every setup step is done.
	pollStatus()
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
	require.NoError(t, err)
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", host.ID, *r.HostSoftwareInstallsExecutionID), nil, http.StatusNoContent)
		}
	}
	pollStatus()
	results, err = s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
	require.NoError(t, err)
	for _, r := range results {
		if r.ScriptExecutionID != nil {
			s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", host.ID, *r.ScriptExecutionID), nil, http.StatusNoContent)
		}
	}

	// The device does not check in between polls, so the first DeviceConfigured stays unanswered.
	for range 5 {
		pollStatus()
	}
	require.Equal(t, 1, countDeviceConfigured())

	// The release clears the flag when it's sent, without waiting for the device's result.
	awaiting, err := s.ds.GetHostAwaitingConfiguration(ctx, host.UUID)
	require.NoError(t, err)
	require.False(t, awaiting)

	require.Equal(t, []string{"DeviceConfigured"}, ackAll())
	pollStatus()
	require.Equal(t, 1, countDeviceConfigured())
	require.Empty(t, ackAll())
}

func (s *integrationMDMTestSuite) TestSetupExperienceScriptResultSavedWithoutStepUpdate() {
	t := s.T()
	ctx := t.Context()

	s.setSkipWorkerJobs(t)

	device, host, _ := s.createTeamDeviceForSetupExperienceWithProfileSoftwareAndScript()

	depURLToken := loadEnrollmentProfileDEPToken(t, s.ds)
	mdmDevice := mdmtest.NewTestMDMClientAppleDEP(s.server.URL, depURLToken)
	mdmDevice.SerialNumber = device.SerialNumber
	require.NoError(t, mdmDevice.Enroll())

	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	ackAll := func() []string {
		var requestTypes []string
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType != "DeclarativeManagement" {
				requestTypes = append(requestTypes, cmd.Command.RequestType)
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		return requestTypes
	}
	ackAll()

	host.OsqueryHostID = new(mdmDevice.UUID)
	host.UUID = mdmDevice.UUID
	orbitKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &orbitKey

	pollStatus := func() *fleet.SetupExperienceStatusPayload {
		var statusResp fleet.GetOrbitSetupExperienceStatusResponse
		s.DoJSON("POST", "/api/fleet/orbit/setup_experience/status",
			json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *host.OrbitNodeKey)), http.StatusOK, &statusResp)
		return statusResp.Results
	}

	// Cancel the software install so the script is the only remaining step and gets queued.
	pollStatus()
	results, err := s.ds.ListSetupExperienceResultsByHostUUID(ctx, host.UUID, *host.TeamID)
	require.NoError(t, err)
	for _, r := range results {
		if r.HostSoftwareInstallsExecutionID != nil {
			s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming/%s", host.ID, *r.HostSoftwareInstallsExecutionID), nil, http.StatusNoContent)
		}
	}
	pollStatus()
	// The status is built before the next step runs, so the script shows as running from the following poll.
	payload := pollStatus()
	require.NotNil(t, payload.Script)
	require.Equal(t, fleet.SetupExperienceStatusRunning, payload.Script.Status)
	require.NotNil(t, payload.Script.ScriptExecutionID)

	// Record the result without updating the setup experience step, as when the result request fails after the
	// result is saved.
	_, _, err = s.ds.SetHostScriptExecutionResult(ctx, &fleet.HostScriptResultPayload{
		HostID:      host.ID,
		ExecutionID: *payload.Script.ScriptExecutionID,
		Output:      "done",
		ExitCode:    0,
	}, nil)
	require.NoError(t, err)

	pollStatus()
	payload = pollStatus()
	require.Equal(t, fleet.SetupExperienceStatusSuccess, payload.Script.Status)
	require.Equal(t, []string{"DeviceConfigured"}, ackAll())
}
