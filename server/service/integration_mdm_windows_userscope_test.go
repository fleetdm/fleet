package service

import (
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ackWindowsCmds answers every protocol command in a batch with SyncML 200 and returns the server's reply to that ack.
func ackWindowsCmds(t *testing.T, d *mdmtest.TestWindowsMDMClient, cmds map[string]fleet.ProtoCmdOperation) map[string]fleet.ProtoCmdOperation {
	msgID, err := d.GetCurrentMsgID()
	require.NoError(t, err)
	for _, c := range cmds {
		if c.Verb == fleet.CmdStatus {
			continue
		}
		status := syncml.CmdStatusOK
		d.AppendResponse(fleet.SyncMLCmd{
			XMLName: xml.Name{Local: fleet.CmdStatus},
			MsgRef:  &msgID, CmdRef: &c.Cmd.CmdID.Value,
			Cmd: &c.Verb, Data: &status,
			CmdID: fleet.CmdID{Value: uuid.NewString()},
		})
	}
	resp, err := d.SendResponse()
	require.NoError(t, err)
	return resp
}

// findWindowsCmdForURI returns the command in the batch that targets locURI with the given verb, whether it stands on
// its own or is nested inside an Atomic, and nil when the batch carries no such command. The verb matters: a pending
// removal that Fleet wrongly satisfied by re-sending the install would otherwise look identical to a Delete.
func findWindowsCmdForURI(cmds map[string]fleet.ProtoCmdOperation, verb, locURI string) *fleet.ProtoCmdOperation {
	nestedForVerb := func(cmd fleet.SyncMLCmd) []fleet.SyncMLCmd {
		switch verb {
		case fleet.CmdAdd:
			return cmd.AddCommands
		case fleet.CmdReplace:
			return cmd.ReplaceCommands
		case fleet.CmdDelete:
			return cmd.DeleteCommands
		case fleet.CmdExec:
			return cmd.ExecCommands
		default:
			return nil
		}
	}

	for _, c := range cmds {
		if c.Verb == verb && c.Cmd.GetTargetURI() == locURI {
			return &c
		}
		if c.Verb != fleet.CmdAtomic {
			continue
		}
		for _, nested := range nestedForVerb(c.Cmd) {
			if nested.GetTargetURI() == locURI {
				return &c
			}
		}
	}
	return nil
}

// findWindowsUserReleaseCmd returns the user-scope ServerHasFinishedProvisioning Replace that ends the Windows ESP,
// nil when the batch does not carry it.
func findWindowsUserReleaseCmd(cmds map[string]fleet.ProtoCmdOperation) *fleet.ProtoCmdOperation {
	for _, c := range cmds {
		uri := c.Cmd.GetTargetURI()
		if c.Verb == fleet.CmdReplace && strings.Contains(uri, "./User/") && strings.Contains(uri, "ServerHasFinishedProvisioning") {
			return &c
		}
	}
	return nil
}

// windowsHostProfilesByName returns the host's Windows profile rows keyed by profile name.
func windowsHostProfilesByName(t *testing.T, ds fleet.Datastore, hostUUID string) map[string]fleet.HostMDMWindowsProfile {
	profs, err := ds.GetHostMDMWindowsProfiles(t.Context(), hostUUID)
	require.NoError(t, err)
	byName := make(map[string]fleet.HostMDMWindowsProfile, len(profs))
	for _, p := range profs {
		byName[p.Name] = p
	}
	return byName
}

// linkWindowsHostToMDMEnrollment attaches an already-enrolled MDM device to a host record the way orbit does when it
// comes up: it points the enrollment at the host UUID and marks the host MDM-enrolled with Fleet. Like
// enrollWindowsHostInMDMViaOrbit it ignores whether the link changed a row, because a programmatic enrollment already
// carries the host UUID from the orbit node key it enrolled with.
func linkWindowsHostToMDMEnrollment(t *testing.T, ds fleet.Datastore, fleetServerURL string, host *fleet.Host, d *mdmtest.TestWindowsMDMClient) {
	_, err := ds.UpdateMDMWindowsEnrollmentsHostUUID(t.Context(), host.UUID, d.DeviceID)
	require.NoError(t, err)
	require.NoError(t, ds.SetOrUpdateMDMData(t.Context(), host.ID, false, true, fleetServerURL, false, fleet.WellKnownMDMFleet, "", false))
}

// windowsProfileRetries returns the host's retry count for one profile, which must stay at zero for a profile the
// user-scope gate is holding: a hold is Fleet declining to send, not a delivery that failed.
func windowsProfileRetries(t *testing.T, ds fleet.Datastore, host *fleet.Host, profileName string) uint {
	counts, err := ds.GetHostMDMProfilesRetryCounts(t.Context(), host)
	require.NoError(t, err)
	for _, c := range counts {
		if c.ProfileName == profileName {
			return c.Retries
		}
	}
	return 0
}

const (
	userScopeTestUserLocURI   = "./User/Vendor/MSFT/Policy/Config/Experience/AllowCortana"
	userScopeTestDeviceLocURI = "./Device/Vendor/MSFT/Policy/Config/Bluetooth/AllowDiscoverableMode"
)

// TestWindowsUserScopedProfileHold covers the gate on an Autopilot enrollment: a user-scoped profile is held while the
// device reports a login status other than "user", it does not block the ESP release while held, and it delivers on
// its own once the device reports "user".
func (s *integrationMDMTestSuite) TestWindowsUserScopedProfileHold() {
	t := s.T()
	ctx := t.Context()

	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	tenantID := uuid.New().String()
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`),
		http.StatusOK, &appConfigResponse{})

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	// One user-scoped profile and one device-scoped profile, so the same run shows that the gate is selective.
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "user-scoped", Contents: syncml.ForTestWithData([]syncml.TestCommand{
			{Verb: fleet.CmdReplace, LocURI: userScopeTestUserLocURI, Data: "0"},
		})},
		{Name: "device-scoped", Contents: syncml.ForTestWithData([]syncml.TestCommand{
			{Verb: fleet.CmdReplace, LocURI: userScopeTestDeviceLocURI, Data: "0"},
		})},
	}}, http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))

	awaiting := func(t *testing.T, d *mdmtest.TestWindowsMDMClient) fleet.WindowsMDMAwaitingConfiguration {
		enrolledDevice, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.DeviceID)
		require.NoError(t, err)
		return enrolledDevice.AwaitingConfiguration
	}

	// Autopilot enrollment, reporting the login status Windows reports during OOBE while the setup account holds the
	// session.
	d := s.enrollWindowsHostInMDMViaAutopilot(s.server.URL, "userscope@example.com", tenantID,
		mdmtest.TestWindowsMDMClientWithLoginStatus(string(fleet.WindowsMDMLoginStatusOthers)))

	cmds, err := d.StartManagementSession()
	require.NoError(t, err)
	ackWindowsCmds(t, d, cmds)

	// The login status the device reported has to be on the enrollment, otherwise nothing downstream can gate on it.
	enrolledDevice, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.DeviceID)
	require.NoError(t, err)
	require.NotNil(t, enrolledDevice.LastLoginStatus, "the LoginStatus alert must be recorded on the enrollment")
	require.Equal(t, fleet.WindowsMDMLoginStatusOthers, *enrolledDevice.LastLoginStatus)
	require.NotNil(t, enrolledDevice.LastLoginStatusAt)

	// fleetd comes up and links the host, which advances the ESP to Active.
	host := createOrbitEnrolledHost(t, "windows", "userscope-h1", s.ds)
	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
		TeamID:  &tm.ID,
		HostIDs: []uint{host.ID},
	}, http.StatusOK, &addHostsToTeamResponse{})
	require.NoError(t, s.ds.SetOrUpdateHostOrbitInfo(ctx, host.ID, "1.23", sql.NullString{}, sql.NullBool{}))
	linkWindowsHostToMDMEnrollment(t, s.ds, s.server.URL, host, d)

	cmds, err = d.StartManagementSession()
	require.NoError(t, err)
	require.Equal(t, fleet.WindowsMDMAwaitingConfigurationActive, awaiting(t, d))
	ackWindowsCmds(t, d, cmds)

	t.Run("user-scoped profile is held while the device reports others", func(t *testing.T) {
		s.awaitTriggerProfileSchedule(t)
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)

		assert.Nil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI),
			"the user-scoped profile must not be sent before the device reports a signed-in user")
		assert.NotNil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestDeviceLocURI),
			"the device-scoped profile must deliver during OOBE exactly as before")

		held := windowsHostProfilesByName(t, s.ds, host.UUID)["user-scoped"]
		require.NotNil(t, held.Status)
		assert.Equal(t, fleet.MDMDeliveryPending, *held.Status, "a held profile reports pending")
		assert.Equal(t, fleet.WindowsUserScopeHoldDetail, held.Detail)
		assert.Empty(t, held.CommandUUID, "a held profile must not point at a command")
		assert.Zero(t, windowsProfileRetries(t, s.ds, host, "user-scoped"), "holding a profile must not spend its retry budget")

		ackWindowsCmds(t, d, cmds)
	})

	t.Run("a held user-scoped profile does not block the ESP release", func(t *testing.T) {
		// The two waits would otherwise be circular: the profile waits for a signed-in user, the user cannot sign
		// in until OOBE moves past the ESP, and the ESP waits for every install profile to reach a terminal state.
		// A held profile is therefore exempt from the release gate and delivers after sign-in instead.
		s.awaitTriggerProfileSchedule(t)
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		require.NotNil(t, findWindowsUserReleaseCmd(cmds),
			"a profile held for a user context must not block the ESP release")
		assert.Nil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI),
			"the release does not deliver the held profile early")
		ackWindowsCmds(t, d, cmds)
		assert.Equal(t, fleet.WindowsMDMAwaitingConfigurationNone, awaiting(t, d),
			"the 200 ack of the user-scope release completes the ESP")
	})

	t.Run("the held profile delivers once the device reports a signed-in user", func(t *testing.T) {
		d.SetLoginStatus(string(fleet.WindowsMDMLoginStatusUser))

		// One session to report the new login status, then the reconciler releases the hold.
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		ackWindowsCmds(t, d, cmds)

		enrolledDevice, err := s.ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctx, d.DeviceID)
		require.NoError(t, err)
		require.NotNil(t, enrolledDevice.LastLoginStatus)
		require.Equal(t, fleet.WindowsMDMLoginStatusUser, *enrolledDevice.LastLoginStatus)

		s.awaitTriggerProfileSchedule(t)
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		require.NotNil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI),
			"the held profile must be sent once the device reports a signed-in user, with no operator action")
		ackWindowsCmds(t, d, cmds)

		delivered := windowsHostProfilesByName(t, s.ds, host.UUID)["user-scoped"]
		require.NotNil(t, delivered.Status)
		// A non-SCEP profile acked 200 is verified outright; only proxied SCEP installs wait in verifying for the
		// certificate to be observed.
		assert.Equal(t, fleet.MDMDeliveryVerified, *delivered.Status)
		assert.Empty(t, delivered.Detail, "the hold detail is cleared once the profile ships")
		assert.Zero(t, windowsProfileRetries(t, s.ds, host, "user-scoped"), "a held-then-delivered profile spends no retries")
	})
}

// TestWindowsUserScopedProfileRemovalAndScope covers the parts of the gate that the OOBE test does not: held
// removals, removals of a profile whose install was itself held, and enrollments with no bound user identity, which
// are not gated at all. The devices here enroll outside OOBE so the ESP does not participate.
func (s *integrationMDMTestSuite) TestWindowsUserScopedProfileRemovalAndScope() {
	t := s.T()
	ctx := t.Context()

	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	tenantID := uuid.New().String()
	s.DoJSON("PATCH", "/api/latest/fleet/config", json.RawMessage(`{ "mdm": { "windows_entra_tenant_ids": ["`+tenantID+`"] } }`),
		http.StatusOK, &appConfigResponse{})

	userProfile := syncml.ForTestWithData([]syncml.TestCommand{
		{Verb: fleet.CmdReplace, LocURI: userScopeTestUserLocURI, Data: "0"},
	})
	setTeamProfiles := func(teamID uint, profiles []fleet.MDMProfileBatchPayload) {
		s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: profiles},
			http.StatusNoContent, "team_id", fmt.Sprint(teamID))
	}
	// enrollEntraHost enrolls a UPN-bound (Entra) device outside OOBE, reporting the given login status, and links it
	// to a fresh host on the given team.
	enrollEntraHost := func(t *testing.T, teamID uint, email, hostName, loginStatus string) (*fleet.Host, *mdmtest.TestWindowsMDMClient) {
		d := s.enrollWindowsMDMViaSettingsApp(s.server.URL, email, tenantID,
			mdmtest.TestWindowsMDMClientWithLoginStatus(loginStatus))
		host := createOrbitEnrolledHost(t, "windows", hostName, s.ds)
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID: &teamID, HostIDs: []uint{host.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})
		linkWindowsHostToMDMEnrollment(t, s.ds, s.server.URL, host, d)
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		ackWindowsCmds(t, d, cmds)
		return host, d
	}
	// reportLoginStatus runs one session so the device reports a new login status to Fleet.
	reportLoginStatus := func(t *testing.T, d *mdmtest.TestWindowsMDMClient, status fleet.WindowsMDMLoginStatus) {
		d.SetLoginStatus(string(status))
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		ackWindowsCmds(t, d, cmds)
	}

	t.Run("a removal waits for the user just like an install", func(t *testing.T) {
		tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		setTeamProfiles(tm.ID, []fleet.MDMProfileBatchPayload{{Name: "user-scoped", Contents: userProfile}})

		// The user is signed in, so the profile delivers normally.
		host, d := enrollEntraHost(t, tm.ID, "removal@example.com", "userscope-removal", string(fleet.WindowsMDMLoginStatusUser))
		s.awaitTriggerProfileSchedule(t)
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		require.NotNil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI),
			"the profile must deliver while the user is signed in")
		ackWindowsCmds(t, d, cmds)

		// The user signs out, then the profile is unassigned.
		reportLoginStatus(t, d, fleet.WindowsMDMLoginStatusNone)
		setTeamProfiles(tm.ID, nil)

		s.awaitTriggerProfileSchedule(t)
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		assert.Nil(t, findWindowsCmdForURI(cmds, fleet.CmdDelete, userScopeTestUserLocURI),
			"no Delete may be sent while nobody is signed in")
		ackWindowsCmds(t, d, cmds)

		held := windowsHostProfilesByName(t, s.ds, host.UUID)["user-scoped"]
		require.NotNil(t, held.Status)
		assert.Equal(t, fleet.MDMDeliveryPending, *held.Status)
		assert.Equal(t, fleet.MDMOperationTypeRemove, held.OperationType)
		assert.Equal(t, fleet.WindowsUserScopeRemoveHoldDetail, held.Detail)
		assert.Empty(t, held.CommandUUID, "a held removal must not point at a command")

		// The user signs back in: the Delete goes out and the row clears.
		reportLoginStatus(t, d, fleet.WindowsMDMLoginStatusUser)
		s.awaitTriggerProfileSchedule(t)
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		require.NotNil(t, findWindowsCmdForURI(cmds, fleet.CmdDelete, userScopeTestUserLocURI),
			"the held removal must be sent as a Delete once the user signs back in")
		ackWindowsCmds(t, d, cmds)
		assert.NotContains(t, windowsHostProfilesByName(t, s.ds, host.UUID), "user-scoped",
			"the row clears once the removal is acked")
	})

	t.Run("removing a profile that was never delivered sends nothing", func(t *testing.T) {
		tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		setTeamProfiles(tm.ID, []fleet.MDMProfileBatchPayload{{Name: "user-scoped", Contents: userProfile}})

		host, d := enrollEntraHost(t, tm.ID, "neverdelivered@example.com", "userscope-never", string(fleet.WindowsMDMLoginStatusOthers))
		s.awaitTriggerProfileSchedule(t)
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		require.Nil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI))
		ackWindowsCmds(t, d, cmds)
		require.Contains(t, windowsHostProfilesByName(t, s.ds, host.UUID), "user-scoped")

		// Unassigning a held profile resolves without contacting the device: nothing was ever written.
		setTeamProfiles(tm.ID, nil)
		s.awaitTriggerProfileSchedule(t)
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		assert.Nil(t, findWindowsCmdForURI(cmds, fleet.CmdDelete, userScopeTestUserLocURI),
			"no Delete may be sent for a profile the device never received")
		ackWindowsCmds(t, d, cmds)
		assert.NotContains(t, windowsHostProfilesByName(t, s.ds, host.UUID), "user-scoped",
			"the row is dropped rather than held for removal")
	})

	t.Run("a device-bound enrollment with no observation is not gated", func(t *testing.T) {
		tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)
		setTeamProfiles(tm.ID, []fleet.MDMProfileBatchPayload{{Name: "user-scoped", Contents: userProfile}})

		// A programmatic (fleetd) enrollment stores an orbit node key rather than a UPN, and this one has never run a
		// management session, so no login status is on record. Fleet cannot tell whether the user channel is writable
		// (it resolves to whoever is signed in), so it does not gate: delivery proceeds exactly as it did before.
		host := createOrbitEnrolledHost(t, "windows", "userscope-fleetd", s.ds)
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID: &tm.ID, HostIDs: []uint{host.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})
		d := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey,
			mdmtest.TestWindowsMDMClientWithLoginStatus(string(fleet.WindowsMDMLoginStatusOthers)))
		require.NoError(t, d.Enroll())
		linkWindowsHostToMDMEnrollment(t, s.ds, s.server.URL, host, d)

		s.awaitTriggerProfileSchedule(t)
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		assert.NotNil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI),
			"a user-scoped profile on an enrollment with no user identity and no observation delivers exactly as before")
		ackWindowsCmds(t, d, cmds)
	})

	t.Run("a device-bound enrollment that reported nobody signed in holds until someone does", func(t *testing.T) {
		tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
		require.NoError(t, err)

		// The device reports "none" on its first session, BEFORE the profile is assigned, so the reconciler sees a
		// positively observed empty console rather than a never-observed enrollment.
		host := createOrbitEnrolledHost(t, "windows", "userscope-fleetd-hold", s.ds)
		s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{
			TeamID: &tm.ID, HostIDs: []uint{host.ID},
		}, http.StatusOK, &addHostsToTeamResponse{})
		d := mdmtest.NewTestMDMClientWindowsProgramatic(s.server.URL, *host.OrbitNodeKey,
			mdmtest.TestWindowsMDMClientWithLoginStatus(string(fleet.WindowsMDMLoginStatusNone)))
		require.NoError(t, d.Enroll())
		linkWindowsHostToMDMEnrollment(t, s.ds, s.server.URL, host, d)
		cmds, err := d.StartManagementSession()
		require.NoError(t, err)
		ackWindowsCmds(t, d, cmds)

		setTeamProfiles(tm.ID, []fleet.MDMProfileBatchPayload{{Name: "user-scoped", Contents: userProfile}})
		s.awaitTriggerProfileSchedule(t)
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		assert.Nil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI),
			"the profile must not be sent while the device says nobody is signed in")
		ackWindowsCmds(t, d, cmds)

		byName := windowsHostProfilesByName(t, s.ds, host.UUID)
		require.Contains(t, byName, "user-scoped")
		require.NotNil(t, byName["user-scoped"].Status)
		assert.Equal(t, fleet.MDMDeliveryPending, *byName["user-scoped"].Status)
		assert.Equal(t, fleet.WindowsUserScopeHoldDetailAnyUser, byName["user-scoped"].Detail,
			"a device-bound hold waits for any user, so its detail must not mention Entra")

		// Someone signs in: this enrollment is bound to no user, so any console user is a usable context.
		d.SetLoginStatus(string(fleet.WindowsMDMLoginStatusUser))
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		ackWindowsCmds(t, d, cmds)

		s.awaitTriggerProfileSchedule(t)
		cmds, err = d.StartManagementSession()
		require.NoError(t, err)
		require.NotNil(t, findWindowsCmdForURI(cmds, fleet.CmdReplace, userScopeTestUserLocURI),
			"the held profile must deliver once the device reports a signed-in user")
		ackWindowsCmds(t, d, cmds)

		byName = windowsHostProfilesByName(t, s.ds, host.UUID)
		require.Contains(t, byName, "user-scoped")
		require.NotNil(t, byName["user-scoped"].Status)
		assert.Contains(t, []fleet.MDMDeliveryStatus{fleet.MDMDeliveryVerifying, fleet.MDMDeliveryVerified},
			*byName["user-scoped"].Status)
		assert.Empty(t, byName["user-scoped"].Detail)
	})
}
