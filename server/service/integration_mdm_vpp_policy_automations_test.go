package service

// VPP app policy automation tests for the MDM suite.
//
// Belongs here: policies that install App Store apps when they fail, including label scoping,
// continuous mode, retries, and automations configured through app config.
//
// Does not belong here: VPP app CRUD and installs requested directly (integration_mdm_vpp_test.go)
// and policy automations for software packages and scripts
// (integration_enterprise_policy_automations_test.go).

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/fleetdm/fleet/v4/server/dev_mode"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestVPPAppPolicyAutomation() {
	t := s.T()
	ctx := context.Background()

	activitiesToString := func(activities []*fleet.UpcomingActivity) []string {
		var res []string
		for _, activity := range activities {
			res = append(res, fmt.Sprintf("%+v", activity))
		}
		return res
	}

	// Set up VPP token
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
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

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Create a couple of hosts
	orbitHost := createOrbitEnrolledHost(t, "darwin", "nonmdm", s.ds)
	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)
	mdmHost2, mdmDevice2 := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice2, true)
	key := setOrbitEnrollment(t, mdmHost2, s.ds)
	mdmHost2.OrbitNodeKey = &key
	selfServiceHost, selfServiceDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, selfServiceHost, s.ds)
	selfServiceToken := "selfservicetoken"
	updateDeviceTokenForHost(t, s.ds, selfServiceHost.ID, selfServiceToken)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, selfServiceDevice.SerialNumber)

	// Add serial number to our fake Apple server
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial, mdmHost2.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID, mdmHost2.ID, orbitHost.ID, selfServiceHost.ID}, TeamID: &team.ID}, http.StatusOK)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	var createLabelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "label1" + t.Name(), Hosts: []string{mdmHost.HardwareSerial}}, http.StatusOK, &createLabelResp)
	l1 := createLabelResp.Label
	require.NotNil(t, l1)

	s.DoJSON("POST", "/api/latest/fleet/labels", &fleet.LabelPayload{Name: "label2" + t.Name(), Hosts: []string{mdmHost2.HardwareSerial}}, http.StatusOK, &createLabelResp)
	l2 := createLabelResp.Label

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
		IconURL:          "https://example.com/images/1/512x512.png",
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
		IconURL:          "https://example.com/images/2/512x512.png",
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
		IconURL:          "https://example.com/images/2/512x512.png",
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
			IconURL:          "https://example.com/images/2/512x512.png",
			LatestVersion:    "2.0.1", // different version for macOS
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
			IconURL:          "https://example.com/images/3/512x512.png",
			LatestVersion:    "3.0.0",
		},
	}
	assert.ElementsMatch(t, expectedApps, appResp.AppStoreApps)

	getSoftwareTitleIDFromApp := func(app *fleet.VPPApp) uint {
		var titleID uint
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &titleID, `SELECT title_id FROM vpp_apps WHERE adam_id = ? AND platform = ?;`, app.AdamID, app.Platform)
		})

		return titleID
	}

	// Insert/deletion flow for macOS app
	// Add an app store app to team 1
	addedApp := expectedApps[0]
	var addedMacOSApp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &team.ID, Platform: addedApp.Platform, AppStoreID: addedApp.AdamID, SelfService: true, LabelsIncludeAny: []string{l1.Name}}, http.StatusOK, &addedMacOSApp)

	// list the software titles for that team, to get the title id of the VPP app
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	require.True(t, *listSw.SoftwareTitles[0].AppStoreApp.SelfService)
	macOSTitleID := listSw.SoftwareTitles[0].ID

	// Insert iPadOS app
	addedApp = expectedApps[1]
	addedIOSApp := addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
		&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: addedApp.AdamID, Platform: addedApp.Platform},
		http.StatusOK, &addedIOSApp)

	// Add all apps to the team, app 1 is already added as self-service and app 2 as non-self-service
	// Add remaining as non-self-service
	for _, app := range expectedApps[2:] {
		addedMacOSApp = addAppStoreAppResponse{}
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps",
			&addAppStoreAppRequest{TeamID: &team.ID, AppStoreID: app.AdamID, Platform: app.Platform},
			http.StatusOK, &addedMacOSApp)
		s.lastActivityMatches(
			fleet.ActivityAddedAppStoreApp{}.ActivityName(),
			fmt.Sprintf(`{"version_name": "Default version", "team_name": "%s", "fleet_name": "%s", "software_title": "%s", "software_title_id": %d, "app_store_id": "%s", "team_id": %d, "fleet_id": %d, "platform": "%s", "self_service": false}`, team.Name, team.Name,
				app.Name, getSoftwareTitleIDFromApp(app), app.AdamID, team.ID, team.ID, app.Platform),
			0,
		)
		listSw = listSoftwareTitlesResponse{}
		s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID),
			"available_for_install", "true")
	}

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(team.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, len(expectedApps))
	iOSTitleID := uint(99999)

	policy0Team1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "policy0Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	policy1Team1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "policy1Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	policy2Team1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{ // will be set up with same VPP app
		Name:     "policy2Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	policy3Team1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{ // will be set up with same VPP app + a script
		Name:     "policy3Team1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	savedTmScript, err := s.ds.NewScript(ctx, &fleet.Script{
		TeamID:         &team.ID,
		Name:           "team_script.sh",
		ScriptContents: "echo 'team'",
	})
	require.NoError(t, err)

	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: iOSTitleID},
		},
	}, http.StatusBadRequest, &mtplr)
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy1Team1.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: macOSTitleID},
		},
	}, http.StatusOK, &mtplr)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy3Team1.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: macOSTitleID},
			ScriptID:        optjson.Any[uint]{Set: true, Valid: true, Value: savedTmScript.ID},
		},
	}, http.StatusOK, &mtplr)

	titleResponse := getSoftwareTitleResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{
		TeamID: &team.ID,
	}, http.StatusOK, &titleResponse)
	require.Len(t, titleResponse.SoftwareTitle.AppStoreApp.AutomaticInstallPolicies, 2)
	require.Equal(t, titleResponse.SoftwareTitle.AppStoreApp.AutomaticInstallPolicies[0].ID, policy1Team1.ID)

	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy2Team1.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: macOSTitleID},
		},
	}, http.StatusOK, &mtplr)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", macOSTitleID), getSoftwareTitleRequest{
		TeamID: &team.ID,
	}, http.StatusOK, &titleResponse)
	require.Len(t, titleResponse.SoftwareTitle.AppStoreApp.AutomaticInstallPolicies, 3)

	// add a non-macOS host
	newHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			DetailUpdatedAt: time.Now(),
			LabelUpdatedAt:  time.Now(),
			PolicyUpdatedAt: time.Now(),
			SeenTime:        time.Now().Add(-1 * time.Minute),
			OsqueryHostID:   ptr.String(t.Name() + name),
			NodeKey:         ptr.String(t.Name() + name),
			UUID:            uuid.New().String(),
			Hostname:        fmt.Sprintf("%s.%s.local", name, t.Name()),
			Platform:        platform,
			TeamID:          teamID,
		})
		require.NoError(t, err)
		return h
	}
	newFleetdHost := func(name string, teamID *uint, platform string) *fleet.Host {
		h := newHost(name, teamID, platform)
		orbitKey := setOrbitEnrollment(t, h, s.ds)
		h.OrbitNodeKey = &orbitKey
		return h
	}

	host3 := newFleetdHost("host3", &team.ID, "ubuntu")

	// setting/unsetting is covered by the other software automation integration test

	// We use DoJSONWithoutAuth for distributed/write because we want the requests to not have the
	// current user's "Authorization: Bearer <API_TOKEN>" header.

	// host3 failure should not queue an install due to mismatched platform
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host3,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	var countResp countHostsResponse
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// non-MDM host failure should not queue an install due to lack of MDM
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		orbitHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// MDM host passing policy should not queue install, nor should failing non-VPP policy
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy0Team1.ID: ptr.Bool(false),
			policy1Team1.ID: ptr.Bool(true),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)

	// MDM host failing policy should queue install (adam ID 1 for macOS) and it will be "activated"
	// in the queue as there is nothing pending for that host.
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// App is out of scope for mdmHost2, so we do not enqueue a software install
	// but we do enqueue a script (will be "activated" in the queue as there is nothing
	// pending for that host).
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost2,
		map[uint]*bool{
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost2.ID), nil, http.StatusOK, &getHostResp, "exclude_software", "true")
	require.Equal(t, getHostResp.Host.ID, mdmHost2.ID)
	require.NotNil(t, getHostResp.Host.Policies)
	require.Len(t, *getHostResp.Host.Policies, 4)
	policySeen := false
	for _, p := range *getHostResp.Host.Policies {
		if p.Name == policy3Team1.Name {
			require.Empty(t, p.Response)
			policySeen = true
		}
	}
	require.True(t, policySeen)

	// Validate that orbit got a notif (and get the exec ID for the script)
	var orbitResp fleet.OrbitGetConfigResponse
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *mdmHost2.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Len(t, orbitResp.Notifications.PendingScriptExecutionIDs, 1)
	scriptExecID := orbitResp.Notifications.PendingScriptExecutionIDs[0]

	var hostActivitiesResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
	assert.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityTypeRanScript{}.ActivityName())
	assert.True(t, hostActivitiesResp.Activities[0].FleetInitiated)
	assert.EqualValues(t, 1, hostActivitiesResp.Count)
	assert.JSONEq(
		t,
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "script_name": "%s", "async": true, "policy_id": %d, "policy_name": "%s", "script_execution_id": "%s", "batch_execution_id": null}`,
			mdmHost2.ID,
			mdmHost2.DisplayName(),
			savedTmScript.Name,
			policy3Team1.ID,
			policy3Team1.Name,
			scriptExecID,
		),
		string(*hostActivitiesResp.Activities[0].Details),
	)

	var orbitPostScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *mdmHost2.OrbitNodeKey, scriptExecID)),
		http.StatusOK, &orbitPostScriptResp)

	s.lastActivityMatchesExtended(
		fleet.ActivityTypeRanScript{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "script_name": %q, "script_execution_id": %q, "async": true, "policy_id": %d, "policy_name": "%s", "batch_execution_id": null, "from_setup_experience": false}`,
			mdmHost2.ID, mdmHost2.DisplayName(), savedTmScript.Name, scriptExecID, policy3Team1.ID, policy3Team1.Name,
		),
		0,
		ptr.Bool(true),
	)

	// Update the app to exclude any with l2. We should not enqueue an install here because mdmHost2
	// has l2 (but it will re-enqueue the script for execution, immediately "activated")
	updateAppReq := &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), LabelsExcludeAny: []string{l2.Name}}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", macOSTitleID), updateAppReq, http.StatusOK)

	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost2,
		map[uint]*bool{
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 1, countResp.Count)

	// Check that the policy shows up as "nil" status
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost2.ID), nil, http.StatusOK, &getHostResp, "exclude_software", "true")
	require.Equal(t, getHostResp.Host.ID, mdmHost2.ID)
	require.NotNil(t, getHostResp.Host.Policies)
	require.Len(t, *getHostResp.Host.Policies, 4)
	policySeen = false
	for _, p := range *getHostResp.Host.Policies {
		if p.Name == policy3Team1.Name {
			require.Empty(t, p.Response)
			policySeen = true
		}
	}
	require.True(t, policySeen)

	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *mdmHost2.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Len(t, orbitResp.Notifications.PendingScriptExecutionIDs, 1)
	scriptExecID = orbitResp.Notifications.PendingScriptExecutionIDs[0]

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
	assert.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityTypeRanScript{}.ActivityName())
	assert.EqualValues(t, 1, hostActivitiesResp.Count)
	assert.JSONEq(
		t,
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "script_name": "%s", "async": true, "policy_id": %d, "policy_name": "%s", "script_execution_id": "%s", "batch_execution_id": null}`,
			mdmHost2.ID,
			mdmHost2.DisplayName(),
			savedTmScript.Name,
			policy3Team1.ID,
			policy3Team1.Name,
			scriptExecID,
		),
		string(*hostActivitiesResp.Activities[0].Details),
	)

	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *mdmHost2.OrbitNodeKey, scriptExecID)),
		http.StatusOK, &orbitPostScriptResp)

	s.lastActivityMatchesExtended(
		fleet.ActivityTypeRanScript{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "script_name": %q, "script_execution_id": %q, "async": true, "policy_id": %d, "policy_name": "%s", "batch_execution_id": null, "from_setup_experience": false}`,
			mdmHost2.ID, mdmHost2.DisplayName(), savedTmScript.Name, scriptExecID, policy3Team1.ID, policy3Team1.Name,
		),
		0,
		ptr.Bool(true),
	)

	// Update the app to include any with l1. We should now enqueue an install as the app is in scope
	// (in addition to the script execution, which will be the only one "activated").
	updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), LabelsIncludeAny: []string{l1.Name, l2.Name}}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", macOSTitleID), updateAppReq, http.StatusOK)

	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost2,
		map[uint]*bool{
			policy3Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 2, countResp.Count)

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", mdmHost2.ID), nil, http.StatusOK, &getHostResp, "exclude_software", "true")
	require.Equal(t, getHostResp.Host.ID, mdmHost2.ID)
	require.NotNil(t, getHostResp.Host.Policies)
	require.Len(t, *getHostResp.Host.Policies, 4)
	policySeen = false
	for _, p := range *getHostResp.Host.Policies {
		if p.Name == policy3Team1.Name {
			require.Equal(t, "fail", p.Response)
			policySeen = true
		}
	}
	require.True(t, policySeen)

	// MDM host failing policy should not queue another install while install is pending
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	var countPendingInstalls uint
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &countPendingInstalls, `SELECT COUNT(*)
			FROM upcoming_activities
			WHERE activity_type = 'vpp_app_install'
			AND host_id = ?`, mdmHost.ID)
	})
	require.Equal(t, uint(1), countPendingInstalls)

	// send an idle request to grab the command uuid
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	var cmdUUID string
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	var fullCmd micromdm.CommandPayload
	require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
	cmdUUID = cmd.CommandUUID

	// Get pending activity, confirm one pending activity
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost.ID),
		nil, http.StatusOK, &hostActivitiesResp)
	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
	assert.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityInstalledAppStoreApp{}.ActivityName())
	assert.EqualValues(t, 1, hostActivitiesResp.Count)
	assert.JSONEq(
		t,
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "software_display_name": null, "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "host_platform": "%s"}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			macOSApp.Name,
			macOSApp.AdamID,
			cmdUUID,
			fleet.SoftwareInstallPending,
			false,
			mdmHost.Platform,
		),
		string(*hostActivitiesResp.Activities[0].Details),
	)

	_, err = mdmDevice.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	s.runWorker()

	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, []fleet.Software{{Name: macOSApp.Name, BundleIdentifier: macOSApp.BundleIdentifier, Version: macOSApp.LatestVersion, Installed: true}})
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected command type", cmd.Command.RequestType)
		}
	}

	s.runWorker()

	// Shouldn't be any more pending installs
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &countPendingInstalls, `SELECT COUNT(*)
			FROM upcoming_activities
			WHERE activity_type = 'vpp_app_install'
			AND host_id = ?`, mdmHost.ID)
	})
	require.Zero(t, countPendingInstalls)

	s.lastActivityMatchesExtended(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"version_name": "Default version", "host_id": %d, "host_display_name": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "policy_id": %d, "policy_name": "%s", "host_platform": "%s", "from_setup_experience": false, "from_auto_update": false, "from_configuration_resend": false}`,
			mdmHost.ID,
			mdmHost.DisplayName(),
			macOSApp.Name,
			macOSApp.AdamID,
			cmdUUID,
			fleet.SoftwareInstalled,
			false,
			policy1Team1.ID,
			policy1Team1.Name,
			mdmHost.Platform,
		),
		0,
		ptr.Bool(true),
	)

	// Process script execution
	s.DoJSON("POST", "/api/fleet/orbit/config",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q}`, *mdmHost2.OrbitNodeKey)),
		http.StatusOK, &orbitResp)
	require.Len(t, orbitResp.Notifications.PendingScriptExecutionIDs, 1)
	scriptExecID = orbitResp.Notifications.PendingScriptExecutionIDs[0]

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	// app install + script exec
	require.Len(t, hostActivitiesResp.Activities, 2, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
	assert.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityTypeRanScript{}.ActivityName())
	assert.EqualValues(t, 2, hostActivitiesResp.Count)
	assert.JSONEq(
		t,
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "script_name": "%s", "async": true, "policy_id": %d, "policy_name": "%s", "script_execution_id": "%s", "batch_execution_id": null}`,
			mdmHost2.ID,
			mdmHost2.DisplayName(),
			savedTmScript.Name,
			policy3Team1.ID,
			policy3Team1.Name,
			scriptExecID,
		),
		string(*hostActivitiesResp.Activities[0].Details),
	)

	// setting a script result will activate the vpp install
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *mdmHost2.OrbitNodeKey, scriptExecID)),
		http.StatusOK, &orbitPostScriptResp)

	s.lastActivityMatchesExtended(
		fleet.ActivityTypeRanScript{}.ActivityName(),
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": %q, "script_name": %q, "script_execution_id": %q, "async": true, "policy_id": %d, "policy_name": "%s", "batch_execution_id": null, "from_setup_experience": false}`,
			mdmHost2.ID, mdmHost2.DisplayName(), savedTmScript.Name, scriptExecID, policy3Team1.ID, policy3Team1.Name,
		),
		0,
		ptr.Bool(true),
	)

	// Process mdmHost2's vpp installation
	s.runWorker()
	cmd, err = mdmDevice2.Idle()
	require.NoError(t, err)
	require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
	cmdUUID = cmd.CommandUUID

	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", mdmHost2.ID),
		nil, http.StatusOK, &hostActivitiesResp)

	require.Len(t, hostActivitiesResp.Activities, 1, "got activities: %v", activitiesToString(hostActivitiesResp.Activities))
	assert.Equal(t, hostActivitiesResp.Activities[0].Type, fleet.ActivityInstalledAppStoreApp{}.ActivityName())
	assert.EqualValues(t, 1, hostActivitiesResp.Count)
	assert.JSONEq(
		t,
		fmt.Sprintf(
			`{"host_id": %d, "host_display_name": "%s", "software_title": "%s", "software_display_name": null, "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "host_platform": "%s"}`,
			mdmHost2.ID,
			mdmHost2.DisplayName(),
			macOSApp.Name,
			macOSApp.AdamID,
			cmdUUID,
			fleet.SoftwareInstallPending,
			false,
			mdmHost2.Platform,
		),
		string(*hostActivitiesResp.Activities[0].Details),
	)

	_, err = mdmDevice2.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	s.runWorker()

	cmd, err = mdmDevice2.Idle()
	require.NoError(t, err)
	for cmd != nil {
		var fullCmd micromdm.CommandPayload
		switch cmd.Command.RequestType { //nolint:gocritic // ignore singleCaseSwitch
		case "InstalledApplicationList":
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
			cmd, err = mdmDevice2.AcknowledgeInstalledApplicationList(mdmDevice2.UUID, cmd.CommandUUID, []fleet.Software{{Name: macOSApp.Name, BundleIdentifier: macOSApp.BundleIdentifier, Version: macOSApp.LatestVersion, Installed: true}})
			require.NoError(t, err)
		default:
			require.Fail(t, "unexpected command type", cmd.Command.RequestType)
		}
	}

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &countPendingInstalls, `SELECT COUNT(*)
			FROM upcoming_activities
			WHERE activity_type = 'vpp_app_install'
			AND host_id = ?`, mdmHost2.ID)
	})
	require.Zero(t, countPendingInstalls)

	s.lastActivityMatchesExtended(
		fleet.ActivityInstalledAppStoreApp{}.ActivityName(),
		fmt.Sprintf(
			`{"version_name": "Default version", "host_id": %d, "host_display_name": "%s", "host_platform": "%s", "software_title": "%s", "app_store_id": "%s", "command_uuid": "%s", "status": "%s", "self_service": %v, "policy_id": %d, "policy_name": "%s", "from_setup_experience": false, "from_auto_update": false, "from_configuration_resend": false}`,
			mdmHost2.ID,
			mdmHost2.DisplayName(),
			mdmHost2.Platform,
			macOSApp.Name,
			macOSApp.AdamID,
			cmdUUID,
			fleet.SoftwareInstalled,
			false,
			policy3Team1.ID,
			policy3Team1.Name,
		),
		0,
		ptr.Bool(true),
	)

	// MDM host failing already-failing policies should not trigger any installs
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		mdmHost,
		map[uint]*bool{
			policy1Team1.ID: ptr.Bool(false),
			policy2Team1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	s.DoJSON("GET", "/api/latest/fleet/hosts/count", nil, http.StatusOK, &countResp, "software_status", "pending", "team_id",
		fmt.Sprint(team.ID), "software_title_id", fmt.Sprint(macOSTitleID))
	require.Equal(t, 0, countResp.Count)
}

func (s *integrationMDMTestSuite) TestVPPPolicyAutomationLabelScopingRetrigger() {
	t := s.T()
	ctx := context.Background()
	batchURL := "/api/latest/fleet/software/app_store_apps/batch"

	// Set up VPP token
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
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

	// Create a team
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: ptr.String("Team 1")}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	// Create a host and add to team
	host, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	nodeKey := setOrbitEnrollment(t, host, s.ds)
	host.OrbitNodeKey = &nodeKey
	s.Do("POST", "/api/latest/fleet/hosts/transfer", &addHostsToTeamRequest{HostIDs: []uint{host.ID}, TeamID: &team.ID}, http.StatusOK)

	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, host.HardwareSerial)

	// Associate team to the VPP token.
	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Create a few labels
	var newLabelResp fleet.CreateLabelResponse
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 1",
	}, http.StatusOK, &newLabelResp)
	label1 := newLabelResp.Label

	newLabelResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 2",
	}, http.StatusOK, &newLabelResp)
	label2 := newLabelResp.Label

	newLabelResp = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/v1/fleet/labels", fleet.LabelPayload{
		Name:  uuid.NewString(),
		Query: "SELECT 3",
	}, http.StatusOK, &newLabelResp)
	label3 := newLabelResp.Label

	// Sleep to guarantee that host label updated at timestamp is greater than the label creation timestamps
	time.Sleep(1 * time.Second)

	// Add label1 and label2 to the host
	host.LabelUpdatedAt = time.Now()
	err := s.ds.RecordLabelQueryExecutions(context.Background(), host, map[uint]*bool{label1.ID: ptr.Bool(true), label2.ID: ptr.Bool(true)}, host.LabelUpdatedAt, false)
	require.NoError(t, err)

	// add VPP app. Add label1 and label3 as "exclude any" labels.
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id",
		fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	addedApp := appResp.AppStoreApps[0]
	var addedMacOSApp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:           &team.ID,
		Platform:         addedApp.Platform,
		AppStoreID:       addedApp.AdamID,
		SelfService:      true,
		LabelsExcludeAny: []string{label1.Name, label3.Name},
	}, http.StatusOK, &addedMacOSApp)

	// Get software title ID of the added VPP app
	listSWTitlesResp := listSoftwareTitlesResponse{}
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles",
		listSoftwareTitlesRequest{},
		http.StatusOK, &listSWTitlesResp,
		"query", addedApp.Name,
		"team_id", fmt.Sprintf("%d", team.ID),
	)
	require.Len(t, listSWTitlesResp.SoftwareTitles, 1)
	require.NotNil(t, listSWTitlesResp.SoftwareTitles[0].AppStoreApp)
	vppAppTitleID := listSWTitlesResp.SoftwareTitles[0].ID

	var vppAppDetail getSoftwareTitleResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", vppAppTitleID), nil, http.StatusOK, &vppAppDetail, "team_id", fmt.Sprintf("%d", team.ID))
	require.NotNil(t, vppAppDetail.SoftwareTitle)
	require.NotNil(t, vppAppDetail.SoftwareTitle.AppStoreApp)

	policy1, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "policy1",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy1.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: vppAppTitleID},
		},
	}, http.StatusOK, &mtplr)

	// Send back a failed result for the policy.
	distributedResp := submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)

	// Note: host has label1, label2, and at this point the app has exclude_any: label1, label3
	// Because the VPP app is out of scope, the policy doesn't have a status
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(0), policy1.FailingHostCount)

	// Update the include any labels. The host has label2, so this means that the software
	// moved in scope.
	updateAppReq := &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), LabelsIncludeAny: []string{label2.Name}}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", vppAppTitleID), updateAppReq, http.StatusOK)

	// Send back a failed result for the policy.
	distributedResp = submitDistributedQueryResultsResponse{}
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)

	// Note: host has label1, label2, and at this point the app has include_any: label2
	// Because the VPP app is in scope, the policy is failing
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)

	// Update to exclude_any: label 2. This moves the software out of scope. The policy is still failing.
	updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), LabelsExcludeAny: []string{label2.Name}}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", vppAppTitleID), updateAppReq, http.StatusOK)

	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)

	// Update to exclude_any: label3. Host has label1, label2, so the app is in scope again and
	// status should clear.

	var batchAssociateResponse batchAssociateAppStoreAppsResponse
	s.DoJSON("POST", batchURL, batchAssociateAppStoreAppsRequest{Apps: []fleet.VPPBatchPayload{
		{AppStoreID: addedApp.AdamID, LabelsExcludeAny: []string{label3.Name}},
	}}, http.StatusOK, &batchAssociateResponse, "team_name", team.Name)
	require.Len(t, batchAssociateResponse.Apps, 1)

	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(0), policy1.FailingHostCount)

	// Sending back another failed response
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	// Because the app is in scope, the policy should be failing again.
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)

	// Update to exclude_any: label 2. This moves the software out of scope. The policy is still failing.
	updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), LabelsExcludeAny: []string{label2.Name}}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", vppAppTitleID), updateAppReq, http.StatusOK)

	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)

	// Update to exclude_any: label3. Host has label1, label2, so the app is in scope again and
	// status should clear.
	updateAppReq = &updateAppStoreAppRequest{TeamID: &team.ID, SelfService: ptr.Bool(false), LabelsExcludeAny: []string{label3.Name}}
	s.Do("PATCH", fmt.Sprintf("/api/latest/fleet/software/titles/%d/app_store_app", vppAppTitleID), updateAppReq, http.StatusOK)

	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(0), policy1.FailingHostCount)

	// Sending back another failed response
	s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
		host,
		map[uint]*bool{
			policy1.ID: ptr.Bool(false),
		},
	), http.StatusOK, &distributedResp)
	err = s.ds.UpdateHostPolicyCounts(ctx)
	require.NoError(t, err)
	policy1, err = s.ds.Policy(ctx, policy1.ID)
	require.NoError(t, err)
	// Because the app is in scope, the policy should be failing again.
	require.Equal(t, uint(0), policy1.PassingHostCount)
	require.Equal(t, uint(1), policy1.FailingHostCount)
}

// TestPolicyAutomationsContinuousVPPApp mirrors
// TestPolicyAutomationsContinuousSoftwareInstaller but for a VPP app
// automation: continuous_automations_enabled=true must re-trigger an
// install on every failing policy result (not only on pass→fail), and
// passing results must never trigger an install.
func (s *integrationMDMTestSuite) TestPolicyAutomationsContinuousVPPApp() {
	t := s.T()
	ctx := context.Background()

	// VPP token setup.
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
		[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))),
		http.StatusAccepted, "", &validToken)

	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)

	// Team and MDM-enrolled host.
	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: new(t.Name())}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID}, TeamID: &team.ID}, http.StatusOK)

	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Pick two macOS VPP apps and add them to the team. Each policy gets its
	// own VPP app: a queued VPP install blocks further queueing for the same
	// adam_id (MapAdamIDsPendingInstall gate), so sharing one app between the
	// two policies would hide the behavior being tested.
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	var macOSApps []*fleet.VPPApp
	for _, app := range appResp.AppStoreApps {
		if app.Platform == fleet.MacOSPlatform {
			macOSApps = append(macOSApps, app)
		}
	}
	require.GreaterOrEqual(t, len(macOSApps), 2, "expected at least two macOS VPP apps in the mock catalog")
	continuousApp, transitionApp := macOSApps[0], macOSApps[1]

	addTitleID := func(app *fleet.VPPApp) uint {
		var addAppResp addAppStoreAppResponse
		s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
			TeamID:     &team.ID,
			Platform:   app.Platform,
			AppStoreID: app.AdamID,
		}, http.StatusOK, &addAppResp)
		var listSw listSoftwareTitlesResponse
		s.DoJSON(
			"GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw,
			"team_id", fmt.Sprint(team.ID),
			"available_for_install", "true",
			"query", app.Name,
		)
		require.NotEmpty(t, listSw.SoftwareTitles)
		require.NotNil(t, listSw.SoftwareTitles[0].AppStoreApp)
		return listSw.SoftwareTitles[0].ID
	}
	continuousTitleID := addTitleID(continuousApp)
	transitionTitleID := addTitleID(transitionApp)

	// Two policies attached to the same VPP app: one continuous, one default.
	continuousPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:                         "continuous",
		Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	})
	require.NoError(t, err)
	transitionPolicy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:     "transition-only",
		Query:    "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform: "darwin",
	})
	require.NoError(t, err)

	attach := func(policyID, titleID uint) {
		var mtplr fleet.ModifyTeamPolicyResponse
		s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policyID), fleet.ModifyTeamPolicyRequest{
			ModifyPolicyPayload: fleet.ModifyPolicyPayload{
				SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: titleID},
			},
		}, http.StatusOK, &mtplr)
	}
	attach(continuousPolicy.ID, continuousTitleID)
	attach(transitionPolicy.ID, transitionTitleID)

	// Distributed writes carry a result for every policy in scope for the host,
	// mirroring how osquery reports: it runs all distributed queries from a read
	// and reports them together in one write. (Submitting a subset would make
	// the missing policies look out-of-scope and get their membership cleaned up.)
	submitPolicyResults := func(results map[uint]*bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			mdmHost,
			results,
		), http.StatusOK, &distributedResp)
	}

	// host_vpp_software_installs grows by one row per queued install; a queued
	// install also blocks further queueing for the same adam_id until the MDM
	// install command is acknowledged.
	countInstallsFor := func(policyID uint) int {
		var count int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &count, `
				SELECT COUNT(*) FROM host_vpp_software_installs
				WHERE host_id = ? AND policy_id = ?
			`, mdmHost.ID, policyID)
		})
		return count
	}

	completeVPPInstall := func() {
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()
		// First drain: acknowledge the InstallApplication command.
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			switch cmd.Command.RequestType {
			case "InstallApplication":
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			case "InstalledApplicationList":
				cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, []fleet.Software{
					{Name: continuousApp.Name, BundleIdentifier: continuousApp.BundleIdentifier, Version: continuousApp.LatestVersion, Installed: true},
					{Name: transitionApp.Name, BundleIdentifier: transitionApp.BundleIdentifier, Version: transitionApp.LatestVersion, Installed: true},
				})
				require.NoError(t, err)
			default:
				require.Fail(t, "unexpected command type", cmd.Command.RequestType)
			}
		}
		// Second drain: the post-install InstalledApplicationList verification
		// command is queued by a worker after the Acknowledge above, so we have
		// to runWorker again and re-Idle to pick it up.
		s.runWorker()
		cmd, err = mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			switch cmd.Command.RequestType {
			case "InstalledApplicationList":
				cmd, err = mdmDevice.AcknowledgeInstalledApplicationList(mdmDevice.UUID, cmd.CommandUUID, []fleet.Software{
					{Name: continuousApp.Name, BundleIdentifier: continuousApp.BundleIdentifier, Version: continuousApp.LatestVersion, Installed: true},
					{Name: transitionApp.Name, BundleIdentifier: transitionApp.BundleIdentifier, Version: transitionApp.LatestVersion, Installed: true},
				})
				require.NoError(t, err)
			default:
				require.Fail(t, "unexpected command type", cmd.Command.RequestType)
			}
		}
		s.runWorker()
	}

	continuousCount := func() int { return countInstallsFor(continuousPolicy.ID) }
	transitionCount := func() int { return countInstallsFor(transitionPolicy.ID) }
	bothFail := map[uint]*bool{continuousPolicy.ID: new(false), transitionPolicy.ID: new(false)}

	// Age a policy's verified VPP install so it sits outside the policy update interval.
	// The continuous cooldown is keyed on verification_at.
	ageVPPInstall := func(policyID uint) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			_, err := q.ExecContext(ctx, `
				UPDATE host_vpp_software_installs SET verification_at = NOW() - INTERVAL 2 HOUR
				WHERE host_id = ? AND policy_id = ?
			`, mdmHost.ID, policyID)
			return err
		})
	}

	// First fail for the continuous policy (transition still passing): the
	// pass→fail transition queues an install. The first failures are staggered
	// across writes because only one upcoming activity is activated at a time
	// per host: a second install queued in the same write would only become
	// visible after the first completes.
	submitPolicyResults(map[uint]*bool{continuousPolicy.ID: new(false), transitionPolicy.ID: new(true)})
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 1, continuousCount(), "first install for continuous policy")
	}, 5*time.Second, 100*time.Millisecond)
	completeVPPInstall()

	// Transition policy now fails for the first time (pass→fail): queues its
	// install. The continuous policy keeps failing (fail→fail) within the
	// interval: it is throttled. A successful VPP install requests a host
	// refetch that re-runs policies immediately, so without throttling this
	// would be a tight loop. It must NOT re-queue.
	submitPolicyResults(bothFail)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 1, transitionCount(), "first install for transition policy")
	}, 5*time.Second, 100*time.Millisecond)
	require.Never(t, func() bool {
		return continuousCount() != 1
	}, 2*time.Second, 100*time.Millisecond, "continuous policy must be throttled within the policy update interval")
	completeVPPInstall()

	// After the interval elapses, the continuous policy re-fires; the default
	// (transition-only) policy still does not.
	ageVPPInstall(continuousPolicy.ID)
	submitPolicyResults(bothFail)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		assert.Equal(t, 2, continuousCount(), "continuous policy must re-fire after the cooldown elapses")
	}, 5*time.Second, 100*time.Millisecond)
	completeVPPInstall()
	require.Never(t, func() bool {
		return transitionCount() != 1
	}, 2*time.Second, 100*time.Millisecond, "default policy must not re-trigger on fail→fail")

	// Final: passing results never trigger an install, regardless of mode.
	continuousBefore := continuousCount()
	transitionBefore := transitionCount()
	submitPolicyResults(map[uint]*bool{continuousPolicy.ID: new(true), transitionPolicy.ID: new(true)})
	require.Never(t, func() bool {
		return continuousCount() != continuousBefore
	}, 2*time.Second, 100*time.Millisecond, "continuous policy must not trigger install on passing result")
	require.Never(t, func() bool {
		return transitionCount() != transitionBefore
	}, 2*time.Second, 100*time.Millisecond, "transition policy must not trigger install on passing result")
}

// TestPolicyAutomationsContinuousVPPAppRetryReset is the VPP analog of the
// script and software-installer retry-reset tests, but the assertion is
// different. VPP retry tracking is per-row (host_vpp_software_installs.retry_count
// is bumped in-place by RetryVPPInstall, gated against MaxSoftwareInstallAttempts
// on that single row), so a continuous re-fire produces a *brand new* row whose
// retry_count starts at 0 — no reset of the old row is needed for the new one to
// have a fresh retry budget. This test pins that behavior so future refactors
// don't quietly break it.
func (s *integrationMDMTestSuite) TestPolicyAutomationsContinuousVPPAppRetryReset() {
	t := s.T()
	ctx := context.Background()

	// VPP token setup.
	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken",
		[]byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))),
		http.StatusAccepted, "", &validToken)

	var resp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &resp)
	require.NoError(t, resp.Err)

	var newTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{TeamPayload: fleet.TeamPayload{Name: new(t.Name())}}, http.StatusOK, &newTeamResp)
	team := newTeamResp.Team

	mdmHost, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, mdmHost, s.ds)
	s.awaitRunAppleMDMWorkerSchedule()
	s.runWorker()
	checkInstallFleetdCommandSent(t, mdmDevice, true)
	s.appleVPPConfigSrvConfig.SerialNumbers = append(s.appleVPPConfigSrvConfig.SerialNumbers, mdmHost.HardwareSerial)
	s.Do("POST", "/api/latest/fleet/hosts/transfer",
		&addHostsToTeamRequest{HostIDs: []uint{mdmHost.ID}, TeamID: &team.ID}, http.StatusOK)

	var resPatchVPP patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", resp.Tokens[0].ID),
		patchVPPTokensTeamsRequest{TeamIDs: []uint{team.ID}}, http.StatusOK, &resPatchVPP)

	// Pick a macOS VPP app and add it to the team.
	var appResp getAppStoreAppsResponse
	s.DoJSON("GET", "/api/latest/fleet/software/app_store_apps", &getAppStoreAppsRequest{}, http.StatusOK, &appResp, "team_id", fmt.Sprint(team.ID))
	require.NoError(t, appResp.Err)
	var addedApp *fleet.VPPApp
	for _, app := range appResp.AppStoreApps {
		if app.Platform == fleet.MacOSPlatform {
			addedApp = app
			break
		}
	}
	require.NotNil(t, addedApp)

	var addAppResp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{
		TeamID:     &team.ID,
		Platform:   addedApp.Platform,
		AppStoreID: addedApp.AdamID,
	}, http.StatusOK, &addAppResp)

	var listSw listSoftwareTitlesResponse
	s.DoJSON(
		"GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw,
		"team_id", fmt.Sprint(team.ID),
		"available_for_install", "true",
		"query", addedApp.Name,
	)
	require.Len(t, listSw.SoftwareTitles, 1)
	require.NotNil(t, listSw.SoftwareTitles[0].AppStoreApp)
	vppTitleID := listSw.SoftwareTitles[0].ID

	policy, err := s.ds.NewTeamPolicy(ctx, team.ID, nil, fleet.PolicyPayload{
		Name:                         "continuous-vpp-retry-reset",
		Query:                        "SELECT 1 FROM osquery_info WHERE start_time < 0;",
		Platform:                     "darwin",
		ContinuousAutomationsEnabled: true,
	})
	require.NoError(t, err)

	var mtplr fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", team.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		ModifyPolicyPayload: fleet.ModifyPolicyPayload{
			SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: vppTitleID},
		},
	}, http.StatusOK, &mtplr)

	submitPolicyResult := func(passes bool) {
		var distributedResp submitDistributedQueryResultsResponse
		s.DoJSONWithoutAuth("POST", "/api/osquery/distributed/write", genDistributedReqWithPolicyResults(
			mdmHost,
			map[uint]*bool{policy.ID: new(passes)},
		), http.StatusOK, &distributedResp)
	}

	// errorOnInstallApplicationCommand drains the MDM queue for an
	// InstallApplication command and responds with an MDM error. apple_mdm.go's
	// command-result handler reacts to that by calling RetryVPPInstall (in place
	// on the same host_vpp_software_installs row, incrementing retry_count) up
	// to MaxSoftwareInstallAttempts.
	errorOnInstallApplicationCommand := func() {
		s.awaitRunAppleMDMWorkerSchedule()
		s.runWorker()
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		require.NotNil(t, cmd, "expected an InstallApplication command on the MDM queue")
		for cmd != nil {
			switch cmd.Command.RequestType {
			case "InstallApplication":
				cmd, err = mdmDevice.Err(cmd.CommandUUID, []mdm.ErrorChain{{ErrorCode: 1234}})
				require.NoError(t, err)
			default:
				cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
				require.NoError(t, err)
			}
		}
		s.runWorker()
	}

	type vppRow struct {
		ID          uint   `db:"id"`
		CommandUUID string `db:"command_uuid"`
		RetryCount  int    `db:"retry_count"`
	}
	listVPPRows := func() []vppRow {
		var rows []vppRow
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.SelectContext(ctx, q, &rows, `
				SELECT id, command_uuid, retry_count
				FROM host_vpp_software_installs
				WHERE host_id = ? AND policy_id = ?
				ORDER BY id ASC
			`, mdmHost.ID, policy.ID)
		})
		return rows
	}

	// Phase 1: pass→fail queues a single row. Fail it through every retry until
	// the per-row cap is reached. Each RetryVPPInstall reuses the same row and
	// gives it a new command_uuid, so we still have exactly one row at the end.
	// The gate is retry_count < MaxSoftwareInstallAttempts, so retries fire at
	// retry_count 0,1,2 (→ ends at MaxSoftwareInstallAttempts); the final
	// failure at the cap produces no further retry. That's
	// MaxSoftwareInstallAttempts+1 failures total.
	submitPolicyResult(false)
	for i := 0; i <= fleet.MaxSoftwareInstallAttempts; i++ {
		errorOnInstallApplicationCommand()
	}
	rows := listVPPRows()
	require.Len(t, rows, 1, "single row tracks retry_count in place")
	require.Equal(t, fleet.MaxSoftwareInstallAttempts, rows[0].RetryCount,
		"retry_count incremented on each RetryVPPInstall until it reaches MaxSoftwareInstallAttempts")
	firstRowID := rows[0].ID

	// No more retries should fire once the cap is reached, even though the
	// install never succeeded.
	require.Never(t, func() bool {
		r := listVPPRows()
		return len(r) != 1 || r[0].ID != firstRowID || r[0].RetryCount != fleet.MaxSoftwareInstallAttempts
	}, 2*time.Second, 100*time.Millisecond, "no further VPP retries after cap is hit")

	// Phase 2: continuous re-fire. processVPPForNewlyFailingPolicies should
	// queue a brand-new install, producing a *second* row in
	// host_vpp_software_installs with retry_count = 0 (fresh budget).
	submitPolicyResult(false)
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		r := listVPPRows()
		if !assert.Len(t, r, 2, "continuous re-fire should insert a new row") {
			return
		}
		assert.Equal(t, firstRowID, r[0].ID)
		assert.Equal(t, fleet.MaxSoftwareInstallAttempts, r[0].RetryCount, "old row's retry_count is left untouched")
		assert.NotEqual(t, firstRowID, r[1].ID, "second row is a brand new install")
		assert.Equal(t, 0, r[1].RetryCount, "new row starts with retry_count = 0 (fresh budget)")
	}, 5*time.Second, 100*time.Millisecond)

	// And the new row's retry budget is genuinely fresh — failing its first
	// InstallApplication should trigger RetryVPPInstall, taking retry_count to 1.
	errorOnInstallApplicationCommand()
	require.EventuallyWithT(t, func(t *assert.CollectT) {
		r := listVPPRows()
		if !assert.Len(t, r, 2) {
			return
		}
		assert.Equal(t, 1, r[1].RetryCount, "new row's retries are eligible (now at 1)")
	}, 5*time.Second, 100*time.Millisecond)
}

func (s *integrationMDMTestSuite) TestVPPAutomationsOnAppConfigApply() {
	t := s.T()
	ctx := t.Context()

	orgName := "Fleet Device Management Inc."
	token := "mycooltoken"
	expTime := time.Now().Add(200 * time.Hour).UTC().Round(time.Second)
	expDate := expTime.Format(fleet.VPPTimeFormat)
	tokenJSON := fmt.Sprintf(`{"expDate":"%s","token":"%s","orgName":"%s"}`, expDate, token, orgName)
	dev_mode.SetOverride("FLEET_DEV_VPP_URL", s.appleVPPConfigSrv.URL, t)
	var validToken uploadVPPTokenResponse
	s.uploadDataViaForm("/api/latest/fleet/vpp_tokens", "token", "token.vpptoken", []byte(base64.StdEncoding.EncodeToString([]byte(tokenJSON))), http.StatusAccepted, "", &validToken)

	var tokensResp getVPPTokensResponse
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &tokensResp)
	require.NoError(t, tokensResp.Err)
	require.Len(t, tokensResp.Tokens, 1)
	location := tokensResp.Tokens[0].Location

	var fleetAResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Fleet A")}, http.StatusOK, &fleetAResp)
	fleetA := fleetAResp.Team
	var fleetBResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", &createTeamRequest{Name: new("Fleet B")}, http.StatusOK, &fleetBResp)
	fleetB := fleetBResp.Team

	var patchVPPResp patchVPPTokensTeamsResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", tokensResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{fleetA.ID}}, http.StatusOK, &patchVPPResp)

	var addedApp addAppStoreAppResponse
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &fleetA.ID, Platform: fleet.MacOSPlatform, AppStoreID: "1"}, http.StatusOK, &addedApp)
	var listSw listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(fleetA.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	titleID := listSw.SoftwareTitles[0].ID

	policy, err := s.ds.NewTeamPolicy(ctx, fleetA.ID, nil, fleet.PolicyPayload{
		Name:     "policyFleetA",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	var modifyResp fleet.ModifyTeamPolicyResponse
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetA.ID, policy.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: titleID},
	}, http.StatusOK, &modifyResp)

	// apply the app config with the token on fleet A, the automation on fleet A should be kept
	var acResp appConfigResponse
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": [ "%s" ]} ] }
	}`, location, fleetA.Name)), http.StatusOK, &acResp)

	var getPolicyResp fleet.GetPolicyByIDResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetA.ID, policy.ID), nil, http.StatusOK, &getPolicyResp)
	require.NotNil(t, getPolicyResp.Policy.InstallSoftware)
	require.Equal(t, titleID, getPolicyResp.Policy.InstallSoftware.SoftwareTitleID)

	// assign the token to fleets A and B, the automation on fleet A should be kept
	patchVPPResp = patchVPPTokensTeamsResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", tokensResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{fleetA.ID, fleetB.ID}}, http.StatusOK, &patchVPPResp)

	getPolicyResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetA.ID, policy.ID), nil, http.StatusOK, &getPolicyResp)
	require.NotNil(t, getPolicyResp.Policy.InstallSoftware)
	require.Equal(t, titleID, getPolicyResp.Policy.InstallSoftware.SoftwareTitleID)

	// apply the app config with the token on fleets B and A, the automation on fleet A should be kept
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": [ "%s", "%s" ]} ] }
	}`, location, fleetB.Name, fleetA.Name)), http.StatusOK, &acResp)

	getPolicyResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetA.ID, policy.ID), nil, http.StatusOK, &getPolicyResp)
	require.NotNil(t, getPolicyResp.Policy.InstallSoftware)
	require.Equal(t, titleID, getPolicyResp.Policy.InstallSoftware.SoftwareTitleID)

	// assign the token to all fleets, the app and the automation on fleet A should be kept
	patchVPPResp = patchVPPTokensTeamsResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/vpp_tokens/%d/teams", tokensResp.Tokens[0].ID), patchVPPTokensTeamsRequest{TeamIDs: []uint{}}, http.StatusOK, &patchVPPResp)

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(fleetA.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	getPolicyResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetA.ID, policy.ID), nil, http.StatusOK, &getPolicyResp)
	require.NotNil(t, getPolicyResp.Policy.InstallSoftware)
	require.Equal(t, titleID, getPolicyResp.Policy.InstallSoftware.SoftwareTitleID)

	// apply the app config with the token on All fleets, the app and the automation on fleet A should be kept
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": [ "%s" ]} ] }
	}`, location, fleet.DisplayNameAllTeams)), http.StatusOK, &acResp)

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(fleetA.ID), "available_for_install", "true")
	require.Len(t, listSw.SoftwareTitles, 1)
	getPolicyResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetA.ID, policy.ID), nil, http.StatusOK, &getPolicyResp)
	require.NotNil(t, getPolicyResp.Policy.InstallSoftware)
	require.Equal(t, titleID, getPolicyResp.Policy.InstallSoftware.SoftwareTitleID)

	// apply the app config with the token on fleet B, the automation on fleet A should be cleared
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": [ "%s" ]} ] }
	}`, location, fleetB.Name)), http.StatusOK, &acResp)

	getPolicyResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetA.ID, policy.ID), nil, http.StatusOK, &getPolicyResp)
	require.Nil(t, getPolicyResp.Policy.InstallSoftware)

	// add the app to fleet B
	addedApp = addAppStoreAppResponse{}
	s.DoJSON("POST", "/api/latest/fleet/software/app_store_apps", &addAppStoreAppRequest{TeamID: &fleetB.ID, Platform: fleet.MacOSPlatform, AppStoreID: "1"}, http.StatusOK, &addedApp)

	// set an automation on fleet B
	policyFleetB, err := s.ds.NewTeamPolicy(ctx, fleetB.ID, nil, fleet.PolicyPayload{
		Name:     "policyFleetB",
		Query:    "SELECT 1;",
		Platform: "darwin",
	})
	require.NoError(t, err)
	modifyResp = fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetB.ID, policyFleetB.ID), fleet.ModifyTeamPolicyRequest{
		SoftwareTitleID: optjson.Any[uint]{Set: true, Valid: true, Value: titleID},
	}, http.StatusOK, &modifyResp)

	// apply the app config with the token on no fleets, the app and the automation on fleet B should be removed
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": []} ] }
	}`, location)), http.StatusOK, &acResp)

	listSw = listSoftwareTitlesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/software/titles", nil, http.StatusOK, &listSw, "team_id", fmt.Sprint(fleetB.ID), "available_for_install", "true")
	require.Empty(t, listSw.SoftwareTitles)
	getPolicyResp = fleet.GetPolicyByIDResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/teams/%d/policies/%d", fleetB.ID, policyFleetB.ID), nil, http.StatusOK, &getPolicyResp)
	require.Nil(t, getPolicyResp.Policy.InstallSoftware)

	// apply the app config with the token on All fleets, the token should be assigned to all fleets
	acResp = appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"mdm": { "volume_purchasing_program": [ {"location": "%s", "teams": [ "%s" ]} ] }
	}`, location, fleet.DisplayNameAllTeams)), http.StatusOK, &acResp)

	tokensResp = getVPPTokensResponse{}
	s.DoJSON("GET", "/api/latest/fleet/vpp_tokens", &getVPPTokensRequest{}, http.StatusOK, &tokensResp)
	require.Len(t, tokensResp.Tokens, 1)
	require.NotNil(t, tokensResp.Tokens[0].Teams)
	require.Empty(t, tokensResp.Tokens[0].Teams)
}
