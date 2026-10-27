package service

// Puppet integration tests for the MDM suite.
//
// Belongs here: the preassign/match profile endpoints used by the Fleet Puppet module, and full
// Puppet runs.
//
// Does not belong here: profiles uploaded through the regular or batch endpoints
// (integration_mdm_profiles_crud_test.go, integration_mdm_profiles_batch_test.go).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestPuppetMatchPreassignProfiles() {
	ctx := context.Background()
	t := s.T()

	// before we switch to a gitops token, ensure ABM is setup
	s.enableABM(t.Name())

	// Use a gitops user for all Puppet actions
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops-TestPuppetMatchPreassignProfiles@example.com",
		GlobalRole: new(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	s.setTokenForTest(t, "gitops-TestPuppetMatchPreassignProfiles@example.com", test.GoodPassword)

	runWithAdminToken := func(cb func()) {
		s.token = s.getTestAdminToken()
		cb()
		s.token = s.getCachedUserToken("gitops-TestPuppetMatchPreassignProfiles@example.com", test.GoodPassword)
	}

	// create a host enrolled in fleet
	mdmHost, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// create a host that's not enrolled into MDM
	nonMDMHost, err := s.ds.NewHost(context.Background(), &fleet.Host{
		OsqueryHostID: new("not-mdm-enrolled"),
		NodeKey:       new("not-mdm-enrolled"),
		UUID:          uuid.New().String(),
		Hostname:      fmt.Sprintf("%sfoo.local.not.enrolled", t.Name()),
		Platform:      "darwin",
	})
	require.NoError(t, err)

	// create a setup assistant for no team, for this we need to:
	// 1. mock the ABM API, as it gets called to set the profile
	// 2. run the DEP schedule, as this registers the default profile
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"auth_session_token": "xyz"}`))
	}))
	s.runDEPSchedule()
	noTeamProf := `{"x": 1}`
	var globalAsstResp createMDMAppleSetupAssistantResponse
	s.DoJSON("POST", "/api/latest/fleet/enrollment_profiles/automatic", createMDMAppleSetupAssistantRequest{
		TeamID:            nil,
		Name:              "no-team",
		EnrollmentProfile: json.RawMessage(noTeamProf),
	}, http.StatusOK, &globalAsstResp)

	// set the global Enable Release Device manually setting to true,
	// will be inherited by teams created via preassign/match.
	s.Do("PATCH", "/api/latest/fleet/setup_experience",
		json.RawMessage(jsonMustMarshal(t, map[string]any{"enable_release_device_manually": true})),
		http.StatusNoContent)

	s.awaitRunAppleMDMWorkerSchedule()

	// preassign an empty profile, fails
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "empty", HostUUID: nonMDMHost.UUID, Profile: nil}, http.StatusUnprocessableEntity)

	// preassign a valid profile to the MDM host
	prof1 := mobileconfigForTest("n1", "i1")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof1}, http.StatusNoContent)

	// preassign another valid profile to the MDM host
	prof2 := mobileconfigForTest("n2", "i2")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "mdm1", HostUUID: mdmHost.UUID, Profile: prof2, Group: "g1"}, http.StatusNoContent)

	// preassign a valid profile to the non-MDM host, still works as the host is not validated in this call
	prof3 := mobileconfigForTest("n3", "i3")
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "non-mdm", HostUUID: nonMDMHost.UUID, Profile: prof3, Group: "g2"}, http.StatusNoContent)

	// match with an invalid external host id, succeeds as it is the same as if
	// there was no matching to do (no preassignment was done)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "no-such-id"}, http.StatusNoContent)

	// match with the non-mdm host fails
	res := s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "non-mdm"}, http.StatusBadRequest)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "host is not enrolled in Fleet MDM")

	// match with the mdm host succeeds and creates a team based on the group labels
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "mdm1"}, http.StatusNoContent)

	// the host is now part of that team
	h, err := s.ds.Host(ctx, mdmHost.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	tm1, err := s.ds.TeamLite(ctx, *h.TeamID)
	require.NoError(t, err)
	require.Equal(t, "g1", tm1.Name)
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)
	require.True(t, tm1.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

	runWithAdminToken(func() {
		// it create activities for the new team, the profiles assigned to it,
		// the host moved to it, and setup assistant
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeCreatedTeam{}.ActivityName(),
			fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm1.ID, tm1.Name, tm1.ID, tm1.Name),
			0)
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeEditedMacosProfile{}.ActivityName(),
			fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "fleet_id": %d, "fleet_name": %q}`, tm1.ID, tm1.Name, tm1.ID, tm1.Name),
			0)
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeTransferredHostsToTeam{}.ActivityName(),
			fmt.Sprintf(`{"fleet_id": %d, "fleet_name": %q, "fleet_id": %d, "fleet_name": %q, "host_ids": [%d], "host_display_names": [%q]}`,
				tm1.ID, tm1.Name, tm1.ID, tm1.Name, h.ID, h.DisplayName()),
			0)
		s.lastActivityOfTypeMatches(
			fleet.ActivityTypeChangedMacosSetupAssistant{}.ActivityName(),
			fmt.Sprintf(`{"fleet_id": %d, "name": %q, "fleet_name": %q, "fleet_id": %d, "fleet_name": %q}`,
				tm1.ID, globalAsstResp.Name, tm1.Name, tm1.ID, tm1.Name),
			0)
	})

	// and the team has the expected profiles (prof1 and prof2)
	profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	// order is guaranteed by profile name
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	// setup assistant settings are copyied from "no team"
	teamAsst, err := s.ds.GetMDMAppleSetupAssistant(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Equal(t, globalAsstResp.Name, teamAsst.Name)
	require.JSONEq(t, string(globalAsstResp.Profile), string(teamAsst.Profile))

	// trigger the schedule so profiles are set in their state
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	// the mdm host has the same profiles (i1, i2, plus fleetd config and disk encryption)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		mdmHost: {
			{Identifier: "i1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "i2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetFileVaultPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// create a team and set profiles to it (note that it doesn't have disk encryption enabled)
	tm2, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "g1 - g4",
		Secrets: []*fleet.EnrollSecret{{Secret: "tm2secret"}},
	})
	require.NoError(t, err)
	prof4 := mobileconfigForTest("n4", "i4")
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof4,
	}}, http.StatusNoContent, "fleet_id", fmt.Sprint(tm2.ID))
	// tm2 has disk encryption and release device manually disabled
	require.False(t, tm2.Config.MDM.EnableDiskEncryption)
	require.False(t, tm2.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

	// create another team with a superset of profiles
	tm3, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "team3_" + t.Name(),
		Secrets: []*fleet.EnrollSecret{{Secret: "tm3secret"}},
	})
	require.NoError(t, err)
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof2, prof4,
	}}, http.StatusNoContent, "fleet_id", fmt.Sprint(tm3.ID))

	// and yet another team with the same profiles as tm3
	tm4, err := s.ds.NewTeam(context.Background(), &fleet.Team{
		Name:    "team4_" + t.Name(),
		Secrets: []*fleet.EnrollSecret{{Secret: "tm4secret"}},
	})
	require.NoError(t, err)
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{
		prof1, prof2, prof4,
	}}, http.StatusNoContent, "fleet_id", fmt.Sprint(tm4.ID))

	// preassign the MDM host to prof1 and prof4, should match existing team tm2
	//
	// additionally, use external host identifiers with different
	// suffixes to simulate real world distributed scenarios where more
	// than one puppet server might be running at the time.
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler06.test.example.com", HostUUID: mdmHost.UUID, Profile: prof1, Group: "g1"}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler01.test.example.com", HostUUID: mdmHost.UUID, Profile: prof4, Group: "g4"}, http.StatusNoContent)

	// match with the mdm host succeeds and assigns it to tm2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "6f36ab2c-1a40-429b-9c9d-07c9029f4aa8-puppetcompiler03.test.example.com"}, http.StatusNoContent)

	// the host is now part of that team
	h, err = s.ds.Host(ctx, mdmHost.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	require.Equal(t, tm2.ID, *h.TeamID)

	// tmLite2 still has disk encryption and release device manually disabled
	tmLite2, err := s.ds.TeamLite(ctx, *h.TeamID)
	require.NoError(t, err)
	require.False(t, tmLite2.Config.MDM.EnableDiskEncryption)
	require.False(t, tmLite2.Config.MDM.MacOSSetup.EnableReleaseDeviceManually.Value)

	// the host's profiles are:
	// - the same as the team's and are pending (prof1 + prof4)
	// - prof2 + old filevault are pending removal
	// - fleetd config being reinstalled (for new enroll secret)
	s.awaitTriggerProfileSchedule(t)

	// useful for debugging
	// mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
	// 	mysqltest.DumpTable(t, q, "host_mdm_apple_profiles")
	// 	return nil
	// })
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		mdmHost: {
			{Identifier: "i1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "i4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			// Profiles from previous team being deleted
			{Identifier: "i2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{
				Identifier: mobileconfig.FleetFileVaultPayloadIdentifier, OperationType: fleet.MDMOperationTypeRemove,
				Status: &fleet.MDMDeliveryPending,
			},
		},
	})

	// create a new mdm host enrolled in fleet
	mdmHost2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// make it part of team 2
	s.Do("POST", "/api/v1/fleet/hosts/transfer",
		addHostsToTeamRequest{TeamID: &tmLite2.ID, HostIDs: []uint{mdmHost2.ID}}, http.StatusOK)

	s.awaitTriggerProfileSchedule(t)

	// simulate having its profiles installed
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		res, err := q.ExecContext(ctx, `UPDATE host_mdm_apple_profiles SET status = ? WHERE host_uuid = ?`, fleet.OSSettingsVerifying, mdmHost2.UUID)
		n, _ := res.RowsAffected()
		require.Equal(t, 4, int(n))
		return err
	})

	// preassign the MDM host using "g1" and "g4", should match existing
	// team tmLite2, and nothing be done since the host is already in tmLite2
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "mdm2", HostUUID: mdmHost2.UUID, Profile: prof1, Group: "g1"}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/preassign", preassignMDMAppleProfileRequest{ExternalHostIdentifier: "mdm2", HostUUID: mdmHost2.UUID, Profile: prof4, Group: "g4"}, http.StatusNoContent)
	s.Do("POST", "/api/latest/fleet/mdm/apple/profiles/match", matchMDMApplePreassignmentRequest{ExternalHostIdentifier: "mdm2"}, http.StatusNoContent)

	// the host is still part of tmLite2
	h, err = s.ds.Host(ctx, mdmHost2.ID)
	require.NoError(t, err)
	require.NotNil(t, h.TeamID)
	require.Equal(t, tmLite2.ID, *h.TeamID)

	// and its profiles have been left untouched
	s.awaitTriggerProfileSchedule(t)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		mdmHost2: {
			{Identifier: "i1", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: "i4", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryVerifying},
		},
	})
}

// while s.TestPuppetMatchPreassignProfiles focuses on many edge cases/extra
// checks around profile assignment, this test is mainly focused on
// simulating a few puppet runs in scenarios we want to support, and ensuring that:
//
// - different hosts end up in the right teams
// - teams get edited as expected
// - commands to add/remove profiles are issued adequately
func (s *integrationMDMTestSuite) TestPuppetRun() {
	t := s.T()
	ctx := context.Background()

	// define a few profiles
	prof1, prof2, prof3, prof4 := mobileconfigForTest("n1", "i1"),
		mobileconfigForTest("n2", "i2"),
		mobileconfigForTest("n3", "i3"),
		mobileconfigForTest("n4", "i4")

	// create three hosts
	host1, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host3, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	// Set up a mock Apple DEP API
	s.enableABM(t.Name())
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			_, _ = w.Write([]byte(`{"auth_session_token": "session123"}`))
		case "/account":
			_, _ = w.Write([]byte(fmt.Sprintf(`{"admin_id": "admin123", "org_name": "%s"}`, "foo")))
		case "/profile":
			w.WriteHeader(http.StatusOK)
			assert.NoError(t, encoder.Encode(godep.ProfileResponse{ProfileUUID: "profile123"}))
		}
	}))

	// Use a gitops user for all Puppet actions
	u := &fleet.User{
		Name:       "GitOps",
		Email:      "gitops-TestPuppetRun@example.com",
		GlobalRole: new(fleet.RoleGitOps),
	}
	require.NoError(t, u.SetPassword(test.GoodPassword, 10, 10))
	_, err := s.ds.NewUser(context.Background(), u)
	require.NoError(t, err)
	s.setTokenForTest(t, "gitops-TestPuppetRun@example.com", test.GoodPassword)

	// preassignAndMatch simulates the puppet module doing all the
	// preassign/match calls for a given set of profiles.
	preassignAndMatch := func(profs []fleet.MDMApplePreassignProfilePayload) {
		require.NotEmpty(t, profs)
		for _, prof := range profs {
			s.Do(
				"POST",
				"/api/latest/fleet/mdm/apple/profiles/preassign",
				preassignMDMAppleProfileRequest{MDMApplePreassignProfilePayload: prof},
				http.StatusNoContent,
			)
		}
		s.Do(
			"POST",
			"/api/latest/fleet/mdm/apple/profiles/match",
			matchMDMApplePreassignmentRequest{ExternalHostIdentifier: profs[0].ExternalHostIdentifier},
			http.StatusNoContent,
		)
	}

	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   fleetdm::profile { 'n3':
	//     template => template('n3.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_2_uuid' {
	//       fleetdm::profile { 'n4':
	//         template => template('fleetdm/n4.mobileconfig.erb'),
	//         group    => 'kiosks',
	//       }
	//   }
	puppetRun := func(host *fleet.Host) {
		payload := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "workstations",
			},
		}

		if host.UUID == host2.UUID {
			payload = append(payload, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof4,
				Group:                  "kiosks",
			})
		}

		preassignAndMatch(payload)
	}

	// host1 checks in
	puppetRun(host1)

	// the host now belongs to a team
	h1, err := s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.NotNil(t, h1.TeamID)

	// the team has the right name
	tm1, err := s.ds.TeamWithExtras(ctx, *h1.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - workstations", tm1.Name)
	// and the right profiles
	profs, err := s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in
	puppetRun(host2)
	// a new team is created
	h2, err := s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.NotNil(t, h2.TeamID)

	// the team has the right name
	tm2, err := s.ds.TeamWithExtras(ctx, *h2.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - kiosks - workstations", tm2.Name)
	// and the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 4)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[3].Mobileconfig))
	require.True(t, tm2.Config.MDM.EnableDiskEncryption)

	// host3 checks in
	puppetRun(host3)
	// it belongs to the same team as host1
	h3, err := s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.Equal(t, h1.TeamID, h3.TeamID)

	// prof2 is edited
	oldProf2 := prof2
	prof2 = mobileconfigForTest("n2", "i2-v2")
	// host3 checks in again
	puppetRun(host3)
	// still belongs to the same team
	h3, err = s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h3.TeamID)

	// but the team has prof2 updated
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.NotEqual(t, oldProf2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in, still belongs to the same team
	puppetRun(host2)
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm2.ID, *h2.TeamID)

	// but the team has prof2 updated as well
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 4)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[3].Mobileconfig))
	require.NotEqual(t, oldProf2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// the puppet manifest is changed, and prof3 is removed
	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_2_uuid' {
	//       fleetdm::profile { 'n4':
	//         template => template('fleetdm/n4.mobileconfig.erb'),
	//         group    => 'kiosks',
	//       }
	//   }
	puppetRun = func(host *fleet.Host) {
		payload := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
		}

		if host.UUID == host2.UUID {
			payload = append(payload, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof4,
				Group:                  "kiosks",
			})
		}

		preassignAndMatch(payload)
	}

	// host1 checks in again
	puppetRun(host1)
	// still belongs to the same team
	h1, err = s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h1.TeamID)

	// but the team doesn't have prof3 anymore
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// same for host2
	puppetRun(host2)
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm2.ID, *h2.TeamID)
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm2.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof4, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// The puppet manifest is drastically updated, this time to use exclusions on host3:
	//
	// node default {
	//   fleetdm::profile { 'n1':
	//     template => template('n1.mobileconfig.erb'),
	//     group    => 'base',
	//   }
	//
	//   fleetdm::profile { 'n2':
	//     template => template('n2.mobileconfig.erb'),
	//     group    => 'workstations',
	//   }
	//
	//   if $facts['system_profiler']['hardware_uuid'] == 'host_3_uuid' {
	//       fleetdm::profile { 'n3':
	//         template => template('fleetdm/n3.mobileconfig.erb'),
	//         group    => 'no-nudge',
	//       }
	//   } else {
	//       fleetdm::profile { 'n3':
	//         ensure => absent,
	//         template => template('fleetdm/n3.mobileconfig.erb'),
	//         group    => 'workstations',
	//       }
	//   }
	// }
	puppetRun = func(host *fleet.Host) {
		manifest := []fleet.MDMApplePreassignProfilePayload{
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof1,
				Group:                  "base",
			},
			{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof2,
				Group:                  "workstations",
			},
		}

		if host.UUID == host3.UUID {
			manifest = append(manifest, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "no-nudge",
				Exclude:                true,
			})
		} else {
			manifest = append(manifest, fleet.MDMApplePreassignProfilePayload{
				ExternalHostIdentifier: host.Hostname,
				HostUUID:               host.UUID,
				Profile:                prof3,
				Group:                  "workstations",
			})
		}

		preassignAndMatch(manifest)
	}

	// host1 checks in
	puppetRun(host1)

	// the host belongs to the same team
	h1, err = s.ds.Host(ctx, host1.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h1.TeamID)

	// the team has the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm1.ID)
	require.NoError(t, err)
	require.Len(t, profs, 3)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.Equal(t, prof3, []byte(profs[2].Mobileconfig))
	require.True(t, tm1.Config.MDM.EnableDiskEncryption)

	// host2 checks in
	puppetRun(host2)
	// it is assigned to tm1
	h2, err = s.ds.Host(ctx, host2.ID)
	require.NoError(t, err)
	require.Equal(t, tm1.ID, *h2.TeamID)

	// host3 checks in
	puppetRun(host3)

	// it is assigned to a new team
	h3, err = s.ds.Host(ctx, host3.ID)
	require.NoError(t, err)
	require.NotNil(t, h3.TeamID)
	require.NotEqual(t, tm1.ID, *h3.TeamID)
	require.NotEqual(t, tm2.ID, *h3.TeamID)

	// a new team is created
	tm3, err := s.ds.TeamWithExtras(ctx, *h3.TeamID)
	require.NoError(t, err)
	require.Equal(t, "base - no-nudge - workstations", tm3.Name)
	// and the right profiles
	profs, err = s.ds.ListMDMAppleConfigProfiles(ctx, &tm3.ID)
	require.NoError(t, err)
	require.Len(t, profs, 2)
	require.Equal(t, prof1, []byte(profs[0].Mobileconfig))
	require.Equal(t, prof2, []byte(profs[1].Mobileconfig))
	require.True(t, tm3.Config.MDM.EnableDiskEncryption)
}
