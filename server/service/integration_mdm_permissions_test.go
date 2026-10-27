package service

// Role permission tests for the MDM suite.
//
// Belongs here: what technician and GitOps users can and cannot do across MDM endpoints, per the
// access documentation.
//
// Does not belong here: permission checks specific to a single feature, which live with that
// feature's tests.

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/optjson"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	testify_mock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestGitOpsUserActions tests the MDM permissions listed in ../../docs/Using\ Fleet/manage-access.md
func (s *integrationMDMTestSuite) TestGitOpsUserActions() {
	t := s.T()
	ctx := context.Background()

	//
	// Setup test data.
	// All setup actions are authored by a global admin.
	//

	t1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foo",
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Bar",
	})
	require.NoError(t, err)
	t3, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Zoo",
	})
	require.NoError(t, err)
	// Create the global GitOps user we'll use in tests.
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops1-mdm@example.com",
		GlobalRole: new(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	// Create a GitOps user for team t1 we'll use in tests.
	u2 := &fleet.User{
		Name:       "GitOps 2",
		Email:      "gitops2-mdm@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleGitOps,
			},
			{
				Team: *t3,
				Role: fleet.RoleGitOps,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)

	//
	// Start running permission tests with user gitops1-mdm.
	//
	s.setTokenForTest(t, "gitops1-mdm@example.com", test.GoodPassword)

	// Attempt to edit global MDM settings, should allow (also ensure the IdP settings are cleared).
	acResp := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"mdm": {
			"macos_setup": {
				"enable_end_user_authentication": false
			},
			"enable_disk_encryption": true,
			"end_user_authentication": {
				"entity_id": "",
				"idp_name": "",
				"metadata_url": ""
			}
		}
  }`), http.StatusOK, &acResp)
	assert.True(t, acResp.MDM.EnableDiskEncryption.Value)

	// Attempt to setup Apple MDM, will fail but the important thing is that it
	// fails with 422 (cannot enable end user auth because no IdP is configured)
	// and not 403 forbidden.
	s.Do("PATCH", "/api/latest/fleet/setup_experience",
		fleet.MDMAppleSetupPayload{TeamID: new(uint(0)), EnableEndUserAuthentication: new(true)}, http.StatusUnprocessableEntity)

	// Attempt to update the Apple MDM settings but with no change, just to
	// validate the access.
	s.Do("PATCH", "/api/latest/fleet/mdm/apple/settings",
		fleet.MDMAppleSettingsPayload{}, http.StatusNoContent)

	// Attempt to set profile batch globally, should allow.
	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)

	// Attempt to edit team MDM settings, should allow.
	teamSpecs := applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: t1.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
			MacOSSettings: map[string]any{
				"custom_settings": []any{"foo", "bar"},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)

	// Attempt to set profile batch for team t1, should allow.
	teamProfiles := [][]byte{
		mobileconfigForTest("N3", "I3"),
		mobileconfigForTest("N4", "I4"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{
		Profiles: teamProfiles,
	}, http.StatusNoContent, "fleet_id", fmt.Sprint(t1.ID))

	//
	// Start running permission tests with user gitops2-mdm,
	// which is GitOps for teams t1 and t3.
	//
	s.setTokenForTest(t, "gitops2-mdm@example.com", test.GoodPassword)

	// Attempt to edit team t1 MDM settings, should allow.
	teamSpecs = applyTeamSpecsRequest{Specs: []*fleet.TeamSpec{{
		Name: t1.Name,
		MDM: fleet.TeamSpecMDM{
			EnableDiskEncryption: optjson.SetBool(true),
			MacOSSettings: map[string]any{
				"custom_settings": []any{"foo", "bar"},
			},
		},
	}}}
	s.Do("POST", "/api/latest/fleet/spec/fleets", teamSpecs, http.StatusOK)

	// Attempt to set profile batch for team t1, should allow.
	teamProfiles = [][]byte{
		mobileconfigForTest("N5", "I5"),
		mobileconfigForTest("N6", "I6"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{
		Profiles: teamProfiles,
	}, http.StatusNoContent, "fleet_id", fmt.Sprint(t1.ID))

	// Attempt to set profile batch for team t2, should not allow.
	teamProfiles = [][]byte{
		mobileconfigForTest("N7", "I7"),
		mobileconfigForTest("N8", "I8"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{
		Profiles: teamProfiles,
	}, http.StatusForbidden, "fleet_id", fmt.Sprint(t2.ID))

	// Attempt to retrieve host profiles fails if the host doesn't belong to the team
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  new(t.Name() + "1"),
		UUID:     t.Name() + "1",
		Hostname: t.Name() + "foo.local",
	})
	require.NoError(t, err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/profiles", h1.ID), getHostRequest{}, http.StatusForbidden, &getHostResponse{})

	err = s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&t1.ID, []uint{h1.ID}))
	require.NoError(t, err)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/mdm/hosts/%d/profiles", h1.ID), getHostRequest{}, http.StatusOK, &getHostResponse{})
}

// TestTechnicianPermissions tests the permissions listed in ../../docs/Using\ Fleet/manage-access.md
func (s *integrationMDMTestSuite) TestTechnicianPermissions() {
	t := s.T()
	ctx := t.Context()
	s.lq.SetQueryResultsCountOverride = func(queryID uint, count int) error {
		return nil
	}

	//
	// Setup test data.
	// All setup actions are authored by a global admin.
	//

	admin, err := s.ds.UserByEmail(ctx, "admin1@example.com")
	require.NoError(t, err)
	h1, err := s.ds.NewHost(ctx, &fleet.Host{
		NodeKey:  new(t.Name() + "1"),
		UUID:     t.Name() + "1",
		Hostname: strings.ReplaceAll(t.Name()+"foo.local", "/", "_"),
	})
	require.NoError(t, err)
	t1, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Foo",
		Secrets: []*fleet.EnrollSecret{
			{
				Secret: "super secret",
			},
		},
	})
	require.NoError(t, err)
	t2, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Bar",
		Secrets: []*fleet.EnrollSecret{
			{
				Secret: "super secret 2",
			},
		},
	})
	require.NoError(t, err)
	t3, err := s.ds.NewTeam(ctx, &fleet.Team{
		Name: "Zoo",
		Secrets: []*fleet.EnrollSecret{
			{
				Secret: "super secret 3",
			},
		},
	})
	require.NoError(t, err)
	team1Host, err := s.ds.NewHost(ctx, &fleet.Host{
		OsqueryHostID: new(t.Name() + "2"),
		NodeKey:       new(t.Name() + "2"),
		UUID:          t.Name() + "2",
		Hostname:      strings.ReplaceAll(t.Name()+"zoo.local", "/", "_"),
		TeamID:        &t1.ID,
		Platform:      "ubuntu",
	})
	require.NoError(t, err)
	team1Host.OrbitNodeKey = new(setOrbitEnrollment(t, team1Host, s.ds))
	globalHost := createOrbitEnrolledHost(t, "ubuntu", "3", s.ds)
	// Hosts that only exist to be deleted by the technician users below.
	newHostToDelete := func(suffix string, teamID *uint) *fleet.Host {
		h, err := s.ds.NewHost(ctx, &fleet.Host{
			NodeKey:  new(t.Name() + suffix),
			UUID:     t.Name() + suffix,
			Hostname: strings.ReplaceAll(t.Name()+suffix+".local", "/", "_"),
			TeamID:   teamID,
		})
		require.NoError(t, err)
		return h
	}
	globalHostToDelete := newHostToDelete("del-global", nil)
	globalHostToBulkDelete := newHostToDelete("del-global-bulk", nil)
	team1HostToDelete := newHostToDelete("del-t1", &t1.ID)
	team1HostToBulkDelete := newHostToDelete("del-t1-bulk", &t1.ID)
	team2HostToDelete := newHostToDelete("del-t2", &t2.ID)
	q1, err := s.ds.NewQuery(ctx, &fleet.Query{
		Name:    "Foo",
		Query:   "SELECT * from time;",
		Logging: fleet.LoggingSnapshot,
	})
	require.NoError(t, err)
	_, globalMacOSMDMClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	team1MacOSHost, team1MacOSMDMClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &t1.ID, HostIDs: []uint{team1MacOSHost.ID}}, http.StatusOK)
	// iOS hosts to test clearing passcodes (MDM enrollment sends the unlock token).
	globalIOSHost, _ := s.createAppleMobileHostThenEnrollMDM("ios")
	team1IOSHost, _ := s.createAppleMobileHostThenEnrollMDM("ios")
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &t1.ID, HostIDs: []uint{team1IOSHost.ID}}, http.StatusOK)
	team2IOSHost, _ := s.createAppleMobileHostThenEnrollMDM("ios")
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &t2.ID, HostIDs: []uint{team2IOSHost.ID}}, http.StatusOK)
	// Add a configuration profile to t1.
	mcUUID := "a" + uuid.NewString()
	prof := mcBytesForTest("name-"+mcUUID, "identifier-"+mcUUID, mcUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(t.Context(), stmt, mcUUID, t1.ID, "name-"+mcUUID, "identifier-"+mcUUID, prof, test.MakeTestBytes())
		return err
	})
	s.awaitTriggerProfileSchedule(t)
	checkNextPayloads(t, team1MacOSMDMClient, false)
	// Add a configuration profile to "No team".
	mcUUID2 := "a" + uuid.NewString()
	prof2 := mcBytesForTest("name-"+mcUUID2, "identifier-"+mcUUID2, mcUUID2)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(t.Context(), stmt, mcUUID2, 0, "name-"+mcUUID2, "identifier-"+mcUUID2, prof2, test.MakeTestBytes())
		return err
	})
	s.awaitTriggerProfileSchedule(t)
	checkNextPayloads(t, globalMacOSMDMClient, false)
	team1Host.OrbitNodeKey = new(setOrbitEnrollment(t, team1Host, s.ds))
	ggsr := getGlobalScheduleResponse{}
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &ggsr)
	require.NoError(t, ggsr.Err)
	cpar := createPackResponse{}
	var userPackID uint
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		Name:     new("Foobar"),
		Disabled: new(false),
	}, http.StatusOK, &cpar)
	userPackID = cpar.Pack.Pack.ID
	require.NotZero(t, userPackID)
	cur := createUserResponse{}
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		Email:      new("foo42@example.com"),
		Password:   new("p4ssw0rd.123"),
		Name:       new("foo42"),
		GlobalRole: new("maintainer"),
	}, http.StatusOK, &cur)
	maintainer := cur.User
	var carveBeginResp fleet.CarveBeginResponse
	s.DoJSON("POST", "/api/osquery/carve/begin", fleet.CarveBeginRequest{
		NodeKey:    *h1.NodeKey,
		BlockCount: 3,
		BlockSize:  3,
		CarveSize:  8,
		CarveId:    "c1",
		RequestId:  "r1",
	}, http.StatusOK, &carveBeginResp)
	require.NotEmpty(t, carveBeginResp.SessionId)
	lcr := fleet.ListCarvesResponse{}
	s.DoJSON("GET", "/api/latest/fleet/carves", fleet.ListCarvesRequest{}, http.StatusOK, &lcr)
	require.NotEmpty(t, lcr.Carves)
	carveID := lcr.Carves[0].ID
	// Create the global Technician user we'll use in tests.
	u := &fleet.User{
		Name:       "Global technician",
		Email:      "global-technician@example.com",
		GlobalRole: new(fleet.RoleTechnician),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	// Create a Technician user for team t1 and t3 we'll use in tests.
	u2 := &fleet.User{
		Name:       "Team t1 and t3 technician",
		Email:      "t1-t3-technician@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleTechnician,
			},
			{
				Team: *t3,
				Role: fleet.RoleTechnician,
			},
		},
	}
	require.NoError(t, u2.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u2)
	require.NoError(t, err)
	gp2, err := s.ds.NewGlobalPolicy(ctx, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo",
		Query: "SELECT 0;",
	})
	require.NoError(t, err)
	t1p, err := s.ds.NewTeamPolicy(ctx, t1.ID, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo1",
		Query: "SELECT 1;",
	})
	require.NoError(t, err)
	t2p, err := s.ds.NewTeamPolicy(ctx, t2.ID, &admin.ID, fleet.PolicyPayload{
		Name:  "Zoo2",
		Query: "SELECT 2;",
	})
	require.NoError(t, err)
	// Create some test user to test moving from/to teams.
	u3 := &fleet.User{
		Name:       "Test Foo Observer",
		Email:      "test-foo-observer@example.com",
		GlobalRole: nil,
		Teams: []fleet.UserTeam{
			{
				Team: *t1,
				Role: fleet.RoleObserver,
			},
		},
	}
	require.NoError(t, u3.SetPassword(test.GoodPassword, 10, 10))
	_, err = s.ds.NewUser(context.Background(), u3)
	require.NoError(t, err)
	manualLabel1, err := s.ds.NewLabel(ctx, &fleet.Label{
		Name:                "manualLabel1",
		Query:               "SELECT 2;",
		LabelMembershipType: fleet.LabelMembershipTypeManual,
	})
	require.NoError(t, err)
	// Create a software installer in "No team".
	payloadNoTeam := &fleet.UploadSoftwareInstallerPayload{
		TeamID:            nil,
		InstallScript:     "another install script",
		PreInstallQuery:   "another pre install query",
		PostInstallScript: "another post install script",
		Filename:          "ruby.deb",
		// additional fields below are pre-populated so we can re-use the payload later for the test assertions
		Title:       "ruby",
		Version:     "1:2.5.1",
		Source:      "deb_packages",
		StorageID:   "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
		Platform:    "linux",
		SelfService: true,
	}
	s.uploadSoftwareInstaller(t, payloadNoTeam, http.StatusOK, "")
	_, softwareTitleIDNoTeam := checkSoftwareInstaller(t, s.ds, payloadNoTeam)
	// Create a software installer on t1.
	payload := &fleet.UploadSoftwareInstallerPayload{
		TeamID:            &t1.ID,
		InstallScript:     "another install script",
		PreInstallQuery:   "another pre install query",
		PostInstallScript: "another post install script",
		Filename:          "ruby.deb",
		// additional fields below are pre-populated so we can re-use the payload later for the test assertions
		Title:       "ruby",
		Version:     "1:2.5.1",
		Source:      "deb_packages",
		StorageID:   "df06d9ce9e2090d9cb2e8cd1f4d7754a803dc452bf93e3204e3acd3b95508628",
		Platform:    "linux",
		SelfService: true,
	}
	s.uploadSoftwareInstaller(t, payload, http.StatusOK, "")
	_, softwareTitleID := checkSoftwareInstaller(t, s.ds, payload)
	// Create a software installer on t2.
	payload2 := &fleet.UploadSoftwareInstallerPayload{
		TeamID:            &t2.ID,
		InstallScript:     "install script",
		PreInstallQuery:   "pre install query",
		PostInstallScript: "post install script",
		Filename:          "ruby.rpm",
		Title:             "ruby",
	}
	s.uploadSoftwareInstaller(t, payload2, http.StatusOK, "")
	softwareTitleID2 := getSoftwareTitleID(t, s.ds, payload2.Title, "rpm_packages")

	// Set software on team1Host.
	software1 := []fleet.Software{
		{Name: "foo", Version: "0.0.1", Source: "chrome_extensions"},
	}
	_, err = s.ds.UpdateHostSoftware(t.Context(), team1Host.ID, software1)
	require.NoError(t, err)
	err = s.ds.SyncHostsSoftware(t.Context(), time.Now())
	require.NoError(t, err)
	err = s.ds.SyncHostsSoftwareTitles(t.Context(), time.Now())
	require.NoError(t, err)
	s.lq.On("QueriesForHost", team1Host.ID).Return(map[string]string{fmt.Sprintf("%d", team1Host.ID): "SELECT 1 FROM osquery_info;"}, nil)
	s.lq.On("RunQuery", testify_mock.Anything, testify_mock.Anything, testify_mock.Anything).Return(nil)
	s.lq.On("StopQuery", testify_mock.Anything).Return(nil)
	// Create a script on t1.
	scriptT1, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "exit_0.sh",
		ScriptContents: "exit 0",
		TeamID:         &t1.ID,
	})
	require.NoError(t, err)
	// Create a script on "No team".
	scriptNoTeam, err := s.ds.NewScript(ctx, &fleet.Script{
		Name:           "no_team_exit_0.sh",
		ScriptContents: "exit 0",
		TeamID:         nil,
	})
	require.NoError(t, err)

	// Set SMTP, agent options, and SSO settings, and check they are not available for Technicians.
	acSetup := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(fmt.Sprintf(`{
		"smtp_settings": {
			"enable_smtp": true,
			"sender_address": "test@example.com",
			"server": "localhost",
			"port": 1025,
			"authentication_type": "authtype_username_password",
			"user_name": "smtpuser",
			"password": "smtpsecretpassword"
		},
		"sso_settings": {
			"enable_sso": true,
			"entity_id": "https://localhost:8080",
			"idp_name": "SimpleSAML",
			"metadata_url": "%s",
			"enable_jit_provisioning": false
		},
		"agent_options": {
			"config": {
				"options": {
					"pack_delimiter": "/",
					"logger_tls_period": 10,
					"distributed_plugin": "tls",
					"disable_distributed": false,
					"logger_tls_endpoint": "/api/osquery/log",
					"distributed_interval": 10,
					"distributed_tls_max_attempts": 3
				}
			}
		}
	}`, testSAMLIDPMetadataURL)), http.StatusOK, &acSetup)
	t.Cleanup(func() {
		acSetup := appConfigResponse{}
		s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
			"smtp_settings": {},
			"sso_settings": {},
			"agent_options": {}
		}`), http.StatusOK, &acSetup)
	})

	//
	// Start running permission tests with user global-technician@example.com.
	//

	s.setTokenForTest(t, "global-technician@example.com", test.GoodPassword)

	// Attempt to retrieve activities, should allow.
	s.DoJSON("GET", "/api/latest/fleet/activities", nil, http.StatusOK, &listActivitiesResponse{})

	// Attempt to retrieve hosts, should allow.
	s.DoJSON("GET", "/api/latest/fleet/hosts", nil, http.StatusOK, &listHostsResponse{})

	// Attempt to retrieve a host by identifier should succeed
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/identifier/%s", h1.Hostname), hostByIdentifierRequest{}, http.StatusOK, &getHostResponse{})

	// Attempt to filter hosts using labels, should fail (label ID 6 is the builtin label "All Hosts")
	s.DoJSON("GET", "/api/latest/fleet/labels/6/hosts", nil, http.StatusOK, &listHostsResponse{})

	// Attempt to delete a host, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", globalHostToDelete.ID), nil, http.StatusOK, &deleteHostResponse{})
	_, err = s.ds.HostLite(ctx, globalHostToDelete.ID)
	require.True(t, fleet.IsNotFound(err))

	// Attempt to bulk delete hosts, should allow.
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", deleteHostsRequest{IDs: []uint{globalHostToBulkDelete.ID}}, http.StatusOK, &deleteHostsResponse{})
	_, err = s.ds.HostLite(ctx, globalHostToBulkDelete.ID)
	require.True(t, fleet.IsNotFound(err))

	// Attempt to transfer host from global to a team, should allow.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t1.ID,
		HostIDs: []uint{h1.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})
	// Move h1 back to "No team" so the rest of the test (and the team
	// technician section below) uses the original setup.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  nil,
		HostIDs: []uint{h1.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	// Attempt to clear the passcode on an iOS host, should allow.
	var cpResp fleet.ClearPasscodeResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", globalIOSHost.ID), nil, http.StatusOK, &cpResp)

	// Attempt to lock a host, should fail. Lock, unlock and wipe share the same
	// authorization, so lock stands in for all three.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", globalIOSHost.ID), nil, http.StatusForbidden)

	// Attempt to create a global label, should allow.
	clr := fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		Name:  "foo",
		Query: "SELECT 1;",
	}, http.StatusOK, &clr)

	// Attempt to modify a label, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), fleet.ModifyLabelRequest{
		Name: new("foo2"),
	}, http.StatusOK, &fleet.ModifyLabelResponse{})

	// Attempt to get a label, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), fleet.GetLabelRequest{}, http.StatusOK, &fleet.GetLabelResponse{})

	// Attempt to list all labels, should allow.
	s.DoJSON("GET", "/api/latest/fleet/labels", fleet.ListLabelsRequest{}, http.StatusOK, &fleet.ListLabelsResponse{})

	// Attempt to delete a label, should allow.
	s.DoJSON("DELETE", "/api/latest/fleet/labels/foo2", fleet.DeleteLabelRequest{}, http.StatusOK, &fleet.DeleteLabelResponse{})

	// Attempt to list all software, should allow.
	s.DoJSON("GET", "/api/latest/fleet/software/versions", listSoftwareRequest{}, http.StatusOK, &listSoftwareVersionsResponse{})
	var lsr listSoftwareResponse
	s.DoJSON("GET", "/api/latest/fleet/software", listSoftwareRequest{}, http.StatusOK, &lsr, "fleet_id", fmt.Sprint(t1.ID))
	require.Len(t, lsr.Software, 1)
	s.DoJSON("GET", "/api/latest/fleet/software/count", countSoftwareRequest{}, http.StatusOK, &countSoftwareResponse{})
	var lstr listSoftwareTitlesResponse
	s.DoJSON("GET", "/api/latest/fleet/software/titles", listSoftwareTitlesRequest{}, http.StatusOK, &lstr, "fleet_id", fmt.Sprint(t1.ID))
	require.Len(t, lstr.SoftwareTitles, 2)
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d", lstr.SoftwareTitles[0].ID), getSoftwareTitleRequest{}, http.StatusOK, &getSoftwareTitleResponse{}, "fleet_id", fmt.Sprint(t1.ID))

	// Attempt to list a software, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/%d", lsr.Software[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResponse{})
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/software/versions/%d", lsr.Software[0].ID), getSoftwareRequest{}, http.StatusOK, &getSoftwareResponse{})

	// Attempt to read app config, only global admins get SMTP settings.
	var globalTechConfigResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &globalTechConfigResp)
	require.Nil(t, globalTechConfigResp.SMTPSettings)
	require.Nil(t, globalTechConfigResp.SSOSettings)
	require.Nil(t, globalTechConfigResp.AgentOptions)

	// Attempt to write app config, should fail.
	acr := appConfigResponse{}
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{
		"webhook_settings": {
			"vulnerabilities_webhook": {
				"enable_vulnerabilities_webhook": true,
				"destination_url": "https://foobar.example.com"
			}
		}
	}`), http.StatusForbidden, &acr)

	// Attempt to add/remove manual labels to/from a host, should allow.
	var addLabelsToHostResp addLabelsToHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", h1.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	var removeLabelsFromHostResp fleet.RemoveLabelsFromHostResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", h1.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Attempt to add and remove hosts to a manual label, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLabel1.ID), fleet.ModifyLabelRequest{
		HostIDs: []uint{h1.ID},
	}, http.StatusOK, &fleet.ModifyLabelResponse{})
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLabel1.ID), fleet.ModifyLabelRequest{
		HostIDs: []uint{},
	}, http.StatusOK, &fleet.ModifyLabelResponse{})

	// Attempt to run live queries asynchronously (new unsaved query), should allow.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QuerySQL: "SELECT * FROM time;",
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusOK, &runLiveQueryResponse{})

	// Attempt to run live queries asynchronously (saved query), should allow.
	s.DoJSON("POST", "/api/latest/fleet/queries/run", createDistributedQueryCampaignRequest{
		QueryID: new(q1.ID),
		Selected: fleet.HostTargets{
			HostIDs: []uint{h1.ID},
		},
	}, http.StatusOK, &runLiveQueryResponse{})

	// Attempt to create queries, should fail.
	cqr := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		Name:  new("foo4"),
		Query: new("SELECT * from osquery_info;"),
	}, http.StatusForbidden, &cqr)

	// Attempt to edit queries, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", q1.ID), fleet.ModifyQueryRequest{
		Name:  new("foo4"),
		Query: new("SELECT * FROM system_info;"),
	}, http.StatusForbidden, &fleet.ModifyQueryResponse{})

	// Attempt to view a query, should work.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/queries/%d", q1.ID), fleet.GetQueryRequest{}, http.StatusOK, &fleet.GetQueryResponse{})

	// Attempt to list all queries, should work.
	s.DoJSON("GET", "/api/latest/fleet/queries", fleet.ListQueriesRequest{}, http.StatusOK, &fleet.ListQueriesResponse{})

	// Attempt to delete queries, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", q1.ID), fleet.DeleteQueryByIDRequest{}, http.StatusForbidden, &fleet.DeleteQueryByIDResponse{})
	s.DoJSON("POST", "/api/latest/fleet/queries/delete", fleet.DeleteQueriesRequest{IDs: []uint{q1.ID}}, http.StatusForbidden, &fleet.DeleteQueriesResponse{})
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/%s", q1.Name), fleet.DeleteQueryRequest{}, http.StatusForbidden, &fleet.DeleteQueryResponse{})

	// Attempt to add a query to a user pack, should fail.
	sqr := fleet.ScheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", fleet.ScheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  q1.ID,
		Interval: 60,
	}, http.StatusForbidden, &sqr)

	// Attempt to read the global schedule, should allow.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &getGlobalScheduleResponse{})

	// Attempt to create a pack, should fail.
	cpr := createPackResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs", createPackRequest{
		Name: new("foo8"),
	}, http.StatusForbidden, &cpr)

	// Attempt to edit a pack, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/packs/%d", cpr.Pack.ID), modifyPackRequest{
		Name: new("foo9"),
	}, http.StatusForbidden, &modifyPackResponse{})

	// Attempt to read a pack, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/packs/%d", userPackID), nil, http.StatusForbidden, &getPackResponse{})

	// Attempt to delete a pack, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/packs/id/%d", userPackID), deletePackRequest{}, http.StatusForbidden, &deletePackResponse{})

	// Attempt to create a global policy, should fail.
	gplr := fleet.GlobalPolicyResponse{}
	s.DoJSON("POST", "/api/latest/fleet/policies", fleet.GlobalPolicyRequest{
		Name:  "foo9",
		Query: "SELECT * from plist;",
	}, http.StatusForbidden, &gplr)

	// Attempt to edit a global policy, should fail.
	mgplr := fleet.ModifyGlobalPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/policies/%d", gp2.ID), fleet.ModifyGlobalPolicyRequest{
		Query: new("SELECT * from plist WHERE path = 'foo';"),
	}, http.StatusForbidden, &mgplr)

	// Attempt to read a global policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gp2.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK,
		&fleet.GetPolicyByIDResponse{},
	)

	// Attempt to delete a global policy, should fail.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", fleet.DeleteGlobalPoliciesRequest{
		IDs: []uint{gp2.ID},
	}, http.StatusForbidden, &fleet.DeleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should fail.
	tplr := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Name:  "foo10",
		Query: "SELECT * from file;",
	}, http.StatusForbidden, &tplr)

	// Attempt to edit a team policy, should fail.
	mtplr := fleet.ModifyTeamPolicyResponse{}
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", t2.ID, t2p.ID), fleet.ModifyTeamPolicyRequest{
		Query: new("SELECT * from file WHERE path = 'foo';"),
	}, http.StatusForbidden, &mtplr)

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", t2.ID, t2p.ID), fleet.GetTeamPolicyByIDRequest{}, http.StatusOK,
		&fleet.GetTeamPolicyByIDResponse{},
	)

	// Attempt to delete a team policy, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/delete", t2.ID), fleet.DeleteTeamPoliciesRequest{
		IDs: []uint{t2p.ID},
	}, http.StatusForbidden, &fleet.DeleteTeamPoliciesResponse{})

	// Attempt to create a user, should fail.
	s.DoJSON("POST", "/api/latest/fleet/users/admin", createUserRequest{
		Email:      new("foo10@example.com"),
		Name:       new("foo10"),
		GlobalRole: new("admin"),
	}, http.StatusForbidden, &createUserResponse{})

	// Attempt to modify a user, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), modifyUserRequest{
		GlobalRole: new("observer"),
	}, http.StatusForbidden, &modifyUserResponse{})

	// Attempt to view a user, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), getUserRequest{}, http.StatusOK, &getUserResponse{})

	// Attempt to delete a user, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/users/%d", admin.ID), deleteUserRequest{}, http.StatusForbidden, &deleteUserResponse{})

	// Attempt to add users to team, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/fleets/%d/users", t1.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *maintainer,
				Role: "admin",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to create a team, should fail.
	tr := teamResponse{}
	s.DoJSON("POST", "/api/latest/fleet/fleets", createTeamRequest{
		Name: new("foo11"),
	}, http.StatusForbidden, &tr)

	// Attempt to edit a team, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", t1.ID), modifyTeamRequest{
		Name: new("foo12"),
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to edit a team's agent options, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/agent_options", t1.ID), json.RawMessage(`{
		"config": {
			"options": {
				"aws_debug": true
			}
		}
	}`), http.StatusForbidden, &teamResponse{})

	// Attempt to view a team, should allow, but enroll secrets should be masked.
	var teamRes teamResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", t1.ID), getTeamRequest{}, http.StatusOK, &teamRes)
	require.Len(t, teamRes.Team.Secrets, 1)
	require.Equal(t, fleet.MaskedPassword, teamRes.Team.Secrets[0].Secret)

	// Attempt to list teams, should allow, but all enroll secrets should be masked.
	var ltr listTeamsResponse
	s.DoJSON("GET", "/api/latest/fleet/fleets", listTeamsRequest{}, http.StatusOK, &ltr)
	require.Len(t, ltr.Teams, 3)
	for _, team := range ltr.Teams {
		require.Len(t, team.Secrets, 1)
		require.Equal(t, fleet.MaskedPassword, team.Secrets[0].Secret)
	}

	// Attempt to delete a team, should fail.
	dtr := deleteTeamResponse{}
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/fleets/%d", t1.ID), deleteTeamRequest{}, http.StatusForbidden, &dtr)

	// Attempt to create/edit enroll secrets, should fail.
	s.DoJSON("POST", "/api/latest/fleet/spec/enroll_secret", applyEnrollSecretSpecRequest{
		Spec: &fleet.EnrollSecretSpec{
			Secrets: []*fleet.EnrollSecret{
				{
					Secret: "foo400",
					TeamID: nil,
				},
				{
					Secret: "foo500",
					TeamID: new(t1.ID),
				},
			},
		},
	}, http.StatusForbidden, &applyEnrollSecretSpecResponse{})

	// Attempt to get enroll secrets, should fail.
	s.DoJSON("GET", "/api/latest/fleet/spec/enroll_secret", nil, http.StatusForbidden, &getEnrollSecretSpecResponse{})

	// Attempt to get team enroll secret, should allow, but should return enroll secrets masked for technicians.
	var tesr teamEnrollSecretsResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/secrets", t1.ID), teamEnrollSecretsRequest{}, http.StatusOK, &tesr)
	require.Len(t, tesr.Secrets, 1)
	require.Equal(t, fleet.MaskedPassword, tesr.Secrets[0].Secret)

	// Attempt to list carved files, should fail.
	s.DoJSON("GET", "/api/latest/fleet/carves", fleet.ListCarvesRequest{}, http.StatusForbidden, &fleet.ListCarvesResponse{})

	// Attempt to get a carved file, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/carves/%d", carveID), fleet.GetCarveRequest{}, http.StatusForbidden, &fleet.GetCarveResponse{})

	// Attempt to search hosts, should allow.
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusOK, &fleet.SearchTargetsResponse{})

	// Attempt to count target hosts, should allow.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusOK, &fleet.CountTargetsResponse{})

	checkDownloadResponse := func(t *testing.T, r *http.Response, expectedFilename string) {
		require.Equal(t, "application/octet-stream", r.Header.Get("Content-Type"))
		require.Equal(t, fmt.Sprintf(`attachment;filename="%s"`, expectedFilename), r.Header.Get("Content-Disposition"))
		require.NotZero(t, r.ContentLength)
		_, err := io.ReadAll(r.Body)
		require.NoError(t, err)
	}

	// Attempt to download installer, should allow.
	tokenResp := getSoftwareInstallerTokenResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", softwareTitleID), nil, http.StatusOK,
		&tokenResp, "fleet_id", fmt.Sprint(t1.ID))
	require.NotEmpty(t, tokenResp.Token)
	r := s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token/%s", softwareTitleID, tokenResp.Token), nil,
		http.StatusOK)
	checkDownloadResponse(t, r, payload.Filename)

	// Attempt to install software on a host, should allow.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", globalHost.ID, softwareTitleIDNoTeam), &installSoftwareRequest{}, http.StatusAccepted)

	// Record successful installation.
	var listUpcomingAct listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", globalHost.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	var details fleet.ActivityTypeInstalledSoftware
	err = json.Unmarshal([]byte(*listUpcomingAct.Activities[0].Details), &details)
	require.NoError(t, err)
	installID := details.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *globalHost.OrbitNodeKey, installID)),
		http.StatusNoContent)

	// Attempt to uninstall software on a host, should allow.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", globalHost.ID, softwareTitleIDNoTeam), &uninstallSoftwareRequest{}, http.StatusAccepted)

	// Attempt to read all profiles on t1, should allow.
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listMDMConfigProfilesResponse{}, "fleet_id", fmt.Sprint(t1.ID))
	// Attempt to read the summary of profiles on t1, should allow.
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &getMDMProfilesSummaryResponse{}, "fleet_id", fmt.Sprint(t1.ID))
	// Attempt to read a profile on t1, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", mcUUID), getMDMConfigProfileRequest{}, http.StatusOK, &getMDMConfigProfileResponse{})

	// Attempt to create a profile on t1, should fail.
	profileBody, profileHeaders := generateNewProfileMultipartRequest(
		t,
		"test_prof",
		mobileconfigForTest("test_prof", "test_prof"),
		s.token,
		map[string][]string{"fleet_id": {fmt.Sprint(t1.ID)}},
	)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", profileBody.Bytes(), http.StatusForbidden, profileHeaders)

	// Attempt to resend a profile.
	// First set the profile to failed, so that we can resend.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_apple_profiles SET status = ? WHERE profile_uuid = ? AND host_uuid = ?`
		_, err := q.ExecContext(t.Context(), stmt, fleet.MDMDeliveryFailed, mcUUID, team1MacOSHost.UUID)
		return err
	})
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", team1MacOSHost.ID, mcUUID), nil, http.StatusAccepted)

	// Attempt to list scripts, should allow.
	s.DoJSON("GET", "/api/latest/fleet/scripts", fleet.ListScriptsRequest{}, http.StatusOK, &fleet.ListScriptsResponse{}, "fleet_id", fmt.Sprint(t1.ID))
	// Attempt to get a script, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", scriptNoTeam.ID), fleet.GetScriptRequest{}, http.StatusOK, &fleet.GetScriptResponse{})
	// Attempt to run script on a host, should allow.
	var rsr fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{
		HostID:     globalHost.ID,
		ScriptName: "no_team_exit_0.sh",
	}, http.StatusAccepted, &rsr)
	// Attempt to run an ad hoc script on a host, should fail.
	rsr = fleet.RunScriptResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{
		HostID:         globalHost.ID,
		ScriptContents: "echo hello",
	}, http.StatusForbidden, &rsr)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{
		HostID:         team1Host.ID,
		ScriptContents: "echo hello",
	}, http.StatusForbidden, &rsr)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{
		HostID:         globalHost.ID,
		ScriptContents: "echo hello",
	}, http.StatusForbidden, &rsr)
	// Attempt to run script on batch of hosts, should fail.
	var batchRes fleet.BatchScriptRunResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/batch", fleet.BatchScriptRunRequest{
		ScriptID: scriptT1.ID,
		HostIDs:  []uint{team1MacOSHost.ID},
	}, http.StatusForbidden, &batchRes)

	//
	// Start running permission tests with t1-t3-technician@example.com.
	//

	s.setTokenForTest(t, "t1-t3-technician@example.com", test.GoodPassword)

	// Attempt to read app config, should pass and secrets should be obfuscated.
	var teamTechConfigResp appConfigResponse
	s.DoJSON("GET", "/api/latest/fleet/config", nil, http.StatusOK, &teamTechConfigResp)
	require.Nil(t, teamTechConfigResp.SMTPSettings)
	require.Nil(t, teamTechConfigResp.SSOSettings)
	require.Nil(t, teamTechConfigResp.AgentOptions)

	// Attempt to clear the passcode on an iOS host of its team, should allow.
	var teamCpResp fleet.ClearPasscodeResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", team1IOSHost.ID), nil, http.StatusOK, &teamCpResp)

	// Attempt to clear the passcode on an iOS host of another team, should fail
	// masked as not-found so other fleets' host IDs can't be probed.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", team2IOSHost.ID), nil, http.StatusNotFound)

	// Attempt to delete a host of its team, should allow.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", team1HostToDelete.ID), nil, http.StatusOK, &deleteHostResponse{})
	_, err = s.ds.HostLite(ctx, team1HostToDelete.ID)
	require.True(t, fleet.IsNotFound(err))

	// Attempt to delete a host of another team or with no team, should fail
	// masked as not-found.
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", team2HostToDelete.ID), nil, http.StatusNotFound)
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d", h1.ID), nil, http.StatusNotFound)

	// Attempt to bulk delete hosts mixing its team and another team, should
	// fail without deleting any host.
	s.Do("POST", "/api/latest/fleet/hosts/delete", deleteHostsRequest{IDs: []uint{team1HostToBulkDelete.ID, team2HostToDelete.ID}}, http.StatusNotFound)
	_, err = s.ds.HostLite(ctx, team1HostToBulkDelete.ID)
	require.NoError(t, err)
	_, err = s.ds.HostLite(ctx, team2HostToDelete.ID)
	require.NoError(t, err)

	// Attempt to bulk delete hosts of its team only, should allow.
	s.DoJSON("POST", "/api/latest/fleet/hosts/delete", deleteHostsRequest{IDs: []uint{team1HostToBulkDelete.ID}}, http.StatusOK, &deleteHostsResponse{})
	_, err = s.ds.HostLite(ctx, team1HostToBulkDelete.ID)
	require.True(t, fleet.IsNotFound(err))

	// Attempt to create queries in global domain, should allow.
	tcqr := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		Name:  new("foo600"),
		Query: new("SELECT * from orbit_info;"),
	}, http.StatusForbidden, &tcqr)

	// Attempt to create queries in its team, should fail.
	tcqr = fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		Name:   new("foo600"),
		Query:  new("SELECT * from orbit_info;"),
		TeamID: &t1.ID,
	}, http.StatusForbidden, &tcqr)

	// Attempt to edit query, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/queries/%d", q1.ID), fleet.ModifyQueryRequest{
		Name:  new("foo4"),
		Query: new("SELECT * FROM system_info;"),
	}, http.StatusForbidden, &fleet.ModifyQueryResponse{})

	// Attempt to delete query, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/queries/id/%d", q1.ID), fleet.DeleteQueryByIDRequest{}, http.StatusForbidden, &fleet.DeleteQueryByIDResponse{})

	// Attempt to read the global schedule, should allow.
	s.DoJSON("GET", "/api/latest/fleet/schedule", nil, http.StatusOK, &getGlobalScheduleResponse{})

	// Attempt to read the team's schedule, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/schedule", t1.ID), getTeamScheduleRequest{}, http.StatusOK,
		&getTeamScheduleResponse{},
	)

	// Attempt to read other team's schedule, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/schedule", t2.ID), getTeamScheduleRequest{}, http.StatusForbidden, &getTeamScheduleResponse{})

	// Attempt to add a query to a user pack, should fail.
	tsqr := fleet.ScheduleQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/packs/schedule", fleet.ScheduleQueryRequest{
		PackID:   userPackID,
		QueryID:  q1.ID,
		Interval: 60,
	}, http.StatusForbidden, &tsqr)

	// Attempt to add a query to the team's schedule, should fail.
	cqrt1 := fleet.CreateQueryResponse{}
	s.DoJSON("POST", "/api/latest/fleet/queries", fleet.CreateQueryRequest{
		Name:   new("foo8"),
		Query:  new("SELECT * from managed_policies;"),
		TeamID: &t1.ID,
	}, http.StatusForbidden, &cqrt1)
	ttsqr := teamScheduleQueryResponse{}
	// Attempt to add a schedule with the deprecated APIs (by referencing a global query), should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/schedule", t1.ID), teamScheduleQueryRequest{
		QueryID:  new(q1.ID),
		Interval: new(uint(60)),
	}, http.StatusForbidden, &ttsqr)

	// Attempt to remove a query from the team's schedule, should fail. q1 is a
	// global query, so it is not in t1's schedule at all: the fleet_id in the
	// path no longer matches the report, and the response is the same
	// not-found a nonexistent report would get rather than a forbidden that
	// would confirm the report exists outside this fleet.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/fleets/%d/schedule/%d", t1.ID, q1.ID), deleteTeamScheduleRequest{}, http.StatusNotFound, &deleteTeamScheduleResponse{})

	// Attempt to add/remove a manual label from a team host, should allow.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", team1Host.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", team1Host.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusOK, &removeLabelsFromHostResp)

	// Attempt to add/remove a manual label from a global host, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", globalHost.ID), addLabelsToHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &addLabelsToHostResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/hosts/%d/labels", globalHost.ID), fleet.RemoveLabelsFromHostRequest{
		Labels: []string{manualLabel1.Name},
	}, http.StatusForbidden, &removeLabelsFromHostResp)

	// Attempt to create a global label, should allow.
	clr = fleet.CreateLabelResponse{}
	s.DoJSON("POST", "/api/latest/fleet/labels", fleet.CreateLabelRequest{
		Name:  "foo",
		Query: "SELECT 1;",
	}, http.StatusOK, &clr)

	// Attempt to modify a global self-authored label, should allow.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", clr.Label.ID), fleet.ModifyLabelRequest{
		Name: new("foo2"),
	}, http.StatusOK, &fleet.ModifyLabelResponse{})

	// Attempt to modify a non-authored global label, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/labels/%d", manualLabel1.ID), fleet.ModifyLabelRequest{
		Name: new("foo2"),
	}, http.StatusForbidden, &fleet.ModifyLabelResponse{})

	// Attempt to read a global policy, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/policies/%d", gp2.ID), fleet.GetPolicyByIDRequest{}, http.StatusOK, &fleet.GetPolicyByIDResponse{})

	// Attempt to delete a global policy, should fail.
	s.DoJSON("POST", "/api/latest/fleet/policies/delete", fleet.DeleteGlobalPoliciesRequest{
		IDs: []uint{gp2.ID},
	}, http.StatusForbidden, &fleet.DeleteGlobalPoliciesResponse{})

	// Attempt to create a team policy, should fail.
	ttplr := fleet.TeamPolicyResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies", t1.ID), fleet.TeamPolicyRequest{
		Name:  "foo1000",
		Query: "SELECT * from file;",
	}, http.StatusForbidden, &ttplr)

	// Attempt to edit a team policy, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", t1.ID, t1p.ID), fleet.ModifyTeamPolicyRequest{
		Query: new("SELECT * from file WHERE path = 'foobar';"),
	}, http.StatusForbidden, &fleet.ModifyTeamPolicyResponse{})

	// Attempt to edit another team's policy, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", t2.ID, t2p.ID), fleet.ModifyTeamPolicyRequest{
		Query: new("SELECT * from file WHERE path = 'foobar';"),
	}, http.StatusForbidden, &fleet.ModifyTeamPolicyResponse{})

	// Attempt to view a team policy, should allow.
	s.DoJSON(
		"GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", t1.ID, t1p.ID), fleet.GetTeamPolicyByIDRequest{}, http.StatusOK,
		&fleet.GetTeamPolicyByIDResponse{},
	)

	// Attempt to view another team's policy, should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/%d", t2.ID, t2p.ID), fleet.GetTeamPolicyByIDRequest{}, http.StatusForbidden, &fleet.GetTeamPolicyByIDResponse{})

	// Attempt to delete a team policy, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/policies/delete", t1.ID), fleet.DeleteTeamPoliciesRequest{
		IDs: []uint{t1p.ID},
	}, http.StatusForbidden, &fleet.DeleteTeamPoliciesResponse{})

	// Attempt to view own team, should allow, but enroll secrets should be masked.
	teamRes = teamResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/fleets/%d", t1.ID), getTeamRequest{}, http.StatusOK, &teamRes)
	require.Len(t, teamRes.Team.Secrets, 1)
	require.Equal(t, fleet.MaskedPassword, teamRes.Team.Secrets[0].Secret)

	// Attempt to list teams, should allow, but all enroll secrets should be masked.
	ltr = listTeamsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/fleets", listTeamsRequest{}, http.StatusOK, &ltr)
	require.Len(t, ltr.Teams, 2)
	for _, team := range ltr.Teams {
		require.Len(t, team.Secrets, 1)
		require.Equal(t, fleet.MaskedPassword, team.Secrets[0].Secret)
	}

	// Attempt to edit own team, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", t1.ID), modifyTeamRequest{
		Name: new("foo123456"),
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to edit another team, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d", t2.ID), modifyTeamRequest{
		Name: new("foo123456"),
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to edit own team's agent options, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/agent_options", t1.ID), json.RawMessage(`{
			"config": {
				"options": {
					"aws_debug": true
				}
			}
		}`), http.StatusForbidden, &teamResponse{})

	// Attempt to edit another team's agent options, should fail.
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/fleets/%d/agent_options", t2.ID), json.RawMessage(`{
			"config": {
				"options": {
					"aws_debug": true
				}
			}
		}`), http.StatusForbidden, &teamResponse{})

	// Attempt to add users from team it owns to another team it owns, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team it owns, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/fleets/%d/users", t3.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to add users to another team it doesn't own, should fail.
	s.DoJSON("PATCH", fmt.Sprintf("/api/latest/fleet/fleets/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u3,
				Role: "maintainer",
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to delete users from team it doesn't own, should fail.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/fleets/%d/users", t2.ID), modifyTeamUsersRequest{
		Users: []fleet.TeamUser{
			{
				User: *u2,
			},
		},
	}, http.StatusForbidden, &teamResponse{})

	// Attempt to search hosts, should allow.
	s.DoJSON("POST", "/api/latest/fleet/targets", fleet.SearchTargetsRequest{
		MatchQuery: "foo",
		QueryID:    &q1.ID,
	}, http.StatusOK, &fleet.SearchTargetsResponse{})

	// Attempt to count target hosts, should allow.
	s.DoJSON("POST", "/api/latest/fleet/targets/count", fleet.CountTargetsRequest{
		Selected: fleet.HostTargets{
			HostIDs:  []uint{h1.ID},
			LabelIDs: []uint{clr.Label.ID},
			TeamIDs:  []uint{t1.ID},
		},
		QueryID: &q1.ID,
	}, http.StatusOK, &fleet.CountTargetsResponse{})

	// Attempt to download installer from t1, should allow.
	tokenResp = getSoftwareInstallerTokenResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", softwareTitleID), nil, http.StatusOK,
		&tokenResp, "fleet_id", fmt.Sprint(t1.ID))
	require.NotEmpty(t, tokenResp.Token)
	r = s.DoRawNoAuth("GET", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token/%s", softwareTitleID, tokenResp.Token), nil,
		http.StatusOK)
	checkDownloadResponse(t, r, payload.Filename)

	// Attempt to download installer from t2, should fail.
	tokenResp = getSoftwareInstallerTokenResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/software/titles/%d/package/token?alt=media", softwareTitleID2), nil, http.StatusForbidden,
		&tokenResp, "fleet_id", fmt.Sprint(t2.ID))

	// Attempt to install software on a host in t1, should allow.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", team1Host.ID, softwareTitleID), &installSoftwareRequest{}, http.StatusAccepted)

	// Record successful installation.
	listUpcomingAct = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", team1Host.ID), nil, http.StatusOK, &listUpcomingAct)
	require.Len(t, listUpcomingAct.Activities, 1)
	details = fleet.ActivityTypeInstalledSoftware{}
	err = json.Unmarshal([]byte(*listUpcomingAct.Activities[0].Details), &details)
	require.NoError(t, err)
	installID = details.InstallUUID
	s.Do("POST", "/api/fleet/orbit/software_install/result",
		json.RawMessage(fmt.Sprintf(`{
			"orbit_node_key": %q,
			"install_uuid": %q,
			"pre_install_condition_output": "1",
			"install_script_exit_code": 0,
			"install_script_output": "ok"
		}`, *team1Host.OrbitNodeKey, installID)),
		http.StatusNoContent)

	// Attempt to uninstall software on a host in t1, should allow.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", team1Host.ID, softwareTitleID), &uninstallSoftwareRequest{}, http.StatusAccepted)

	// Attempt to install software on a global host, should fail.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/install", globalHost.ID, softwareTitleIDNoTeam), &installSoftwareRequest{}, http.StatusForbidden)

	// Attempt to uninstall software on a global host, should fail.
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/software/%d/uninstall", globalHost.ID, softwareTitleIDNoTeam), &uninstallSoftwareRequest{}, http.StatusForbidden)

	// Attempt to read all profiles on t1, should allow.
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusOK, &listMDMConfigProfilesResponse{}, "fleet_id", fmt.Sprint(t1.ID))
	// Attempt to read the summary of profiles on t1, should allow.
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles/summary", getMDMProfilesSummaryRequest{}, http.StatusOK, &getMDMProfilesSummaryResponse{}, "fleet_id", fmt.Sprint(t1.ID))
	// Attempt to read a profile on t1, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", mcUUID), getMDMConfigProfileRequest{}, http.StatusOK, &getMDMConfigProfileResponse{})
	// Attempt to read all profiles on "No team", should fail.
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", listMDMConfigProfilesRequest{}, http.StatusForbidden, &listMDMConfigProfilesResponse{}, "fleet_id", "0")
	// Attempt to read a profile on "No team", should fail.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", mcUUID2), getMDMConfigProfileRequest{}, http.StatusForbidden, &getMDMConfigProfileResponse{})

	// Attempt to create a profile on t1, should fail.
	profileBody, profileHeaders = generateNewProfileMultipartRequest(
		t,
		"test_prof",
		mobileconfigForTest("test_prof2", "test_prof2"),
		s.token,
		map[string][]string{"fleet_id": {fmt.Sprint(t1.ID)}},
	)
	s.DoRawWithHeaders("POST", "/api/latest/fleet/mdm/apple/profiles", profileBody.Bytes(), http.StatusForbidden, profileHeaders)

	// Attempt to resend a profile on t1, should allow.
	// First set the profile to failed, so that we can resend.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `UPDATE host_mdm_apple_profiles SET status = ? WHERE profile_uuid = ? AND host_uuid = ?`
		_, err := q.ExecContext(t.Context(), stmt, fleet.MDMDeliveryFailed, mcUUID, team1MacOSHost.UUID)
		return err
	})
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", team1MacOSHost.ID, mcUUID), nil, http.StatusAccepted)

	// Attempt to resend a profile on "No team" host, should fail.
	_ = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/configuration_profiles/%s/resend", globalHost.ID, mcUUID2), nil, http.StatusForbidden)

	// Attempt to list scripts on t1, should allow.
	s.DoJSON("GET", "/api/latest/fleet/scripts", fleet.ListScriptsRequest{}, http.StatusOK, &fleet.ListScriptsResponse{}, "fleet_id", fmt.Sprint(t1.ID))
	// Attempt to list scripts on "No team", should fail.
	s.DoJSON("GET", "/api/latest/fleet/scripts", fleet.ListScriptsRequest{}, http.StatusForbidden, &fleet.ListScriptsResponse{})
	// Attempt to get a script, should allow.
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/scripts/%d", scriptT1.ID), fleet.GetScriptRequest{}, http.StatusOK, &fleet.GetScriptResponse{})
	// Attempt to run script on a host, should allow.
	rsr = fleet.RunScriptResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{
		HostID:     team1Host.ID,
		ScriptName: "exit_0.sh",
		TeamID:     t1.ID,
	}, http.StatusAccepted, &rsr)
	// Attempt to run an ad hoc script on a host, should fail.
	rsr = fleet.RunScriptResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{
		HostID:         team1Host.ID,
		ScriptContents: "echo hello",
	}, http.StatusForbidden, &rsr)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/sync", fleet.HostScriptRequestPayload{
		HostID:         team1Host.ID,
		ScriptContents: "echo hello",
	}, http.StatusForbidden, &rsr)
	// Attempt to run script on a host in "No team", should fail.
	rsr = fleet.RunScriptResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run", fleet.HostScriptRequestPayload{
		HostID:     globalHost.ID,
		ScriptName: "no_team_exit_0.sh",
	}, http.StatusForbidden, &rsr)

	// Attempt to run script on batch of hosts, should fail.
	batchRes = fleet.BatchScriptRunResponse{}
	s.DoJSON("POST", "/api/latest/fleet/scripts/run/batch", fleet.BatchScriptRunRequest{
		ScriptID: scriptT1.ID,
		HostIDs:  []uint{team1MacOSHost.ID},
	}, http.StatusForbidden, &batchRes)

	// Attempt to transfer a host between teams the technician manages (t1 -> t3),
	// should allow.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t3.ID,
		HostIDs: []uint{team1Host.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})
	// And back from t3 -> t1, should allow. Also leaves team1Host on t1 so the
	// rest of the test setup remains intact.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t1.ID,
		HostIDs: []uint{team1Host.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})

	// Attempt to transfer a host out of a team the technician manages to a team
	// they do not manage (t1 -> t2), should fail.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t2.ID,
		HostIDs: []uint{team1Host.ID},
	}, http.StatusForbidden, &addHostsToTeamResponse{})

	// Attempt to transfer a host from a team the technician does not manage to
	// a team they do manage (no team -> t1), should fail. The technician can't
	// read the "No team" host, so it's reported as not found rather than
	// forbidden to avoid confirming its existence.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &t1.ID,
		HostIDs: []uint{globalHost.ID},
	}, http.StatusNotFound, &addHostsToTeamResponse{})

	// Attempt to transfer a host out of a team the technician manages to
	// "No team" (a team they do not manage), should fail.
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  nil,
		HostIDs: []uint{team1Host.ID},
	}, http.StatusForbidden, &addHostsToTeamResponse{})
}
