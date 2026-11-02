package service

// Profile reconciler edge case tests for the MDM suite.
//
// Belongs here: how the reconciler cancels, replaces, or removes profile commands — deletes and
// edits cancelling undelivered installs, NotNow responses, user-scoped pending installs, failed
// verifications, profile errors and removing failed profiles, resend race conditions, fleet
// transfers with identical profiles, and resends triggered by policy automations.
//
// Does not belong here: the normal delivery and status flows (integration_mdm_profiles_apple_test.go,
// integration_mdm_profiles_windows_delivery_test.go) and the batch-resend endpoint
// (integration_mdm_profiles_batch_test.go).

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	servermdm "github.com/fleetdm/fleet/v4/server/mdm"
	apple_mdm "github.com/fleetdm/fleet/v4/server/mdm/apple"
	"github.com/fleetdm/fleet/v4/server/mdm/apple/mobileconfig"
	"github.com/fleetdm/fleet/v4/server/mdm/microsoft/syncml"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	micromdm "github.com/micromdm/micromdm/mdm/mdm"
	"github.com/micromdm/plist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *integrationMDMTestSuite) TestDeleteMDMProfileCancelsInstalls() {
	t := s.T()
	s.setSkipWorkerJobs(t)

	// create some Apple, Windows and declaration profiles
	profiles := []fleet.MDMProfileBatchPayload{
		{
			Name:     "A1",
			Contents: mobileconfigForTest("A1", "A1"),
		},
		{
			Name:     "A2",
			Contents: mobileconfigForTest("A2", "A2"),
		},
		{
			Name:     "D1",
			Contents: declarationForTest("D1"),
		},
		{
			Name:     "D2",
			Contents: declarationForTest("D2"),
		},
		{
			Name:     "W1",
			Contents: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "W1", Data: "W1"}}),
		},
		{
			Name:     "W2",
			Contents: syncml.ForTestWithData([]syncml.TestCommand{{Verb: "Replace", LocURI: "W2", Data: "W2"}}),
		},
	}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: profiles}, http.StatusNoContent)

	// list the profiles to get the UUIDs
	var listResp listMDMConfigProfilesResponse
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp)
	profNameToPayload := make(map[string]*fleet.MDMConfigProfilePayload)
	for _, prof := range listResp.Profiles {
		if len(prof.Checksum) == 0 {
			// not important, but must not be empty or it causes issues when forcing a status
			prof.Checksum = []byte("checksum")
		}
		profNameToPayload[prof.Name] = prof
		t.Logf("profile %s: %s", prof.Name, prof.ProfileUUID)
	}

	// deleting without any affected host is fine
	var deleteResp deleteMDMConfigProfileResponse
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profNameToPayload["A1"].ProfileUUID), nil, http.StatusOK, &deleteResp) // nolint:nilaway // every name looked up here was inserted from the list response above
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profNameToPayload["D1"].ProfileUUID), nil, http.StatusOK, &deleteResp)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profNameToPayload["W1"].ProfileUUID), nil, http.StatusOK, &deleteResp)

	// create some Apple and Windows hosts
	host1, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host2, _ := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	host3, mdmDevice3 := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	host4, mdmDevice4 := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	for i, h := range []*fleet.Host{host1, host2, host3, host4} {
		t.Logf("host %d: %s", i+1, h.UUID)
	}
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		host3: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host4: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// for the declaration, set host1 as NULL and host2 as verified
	forceSetAppleHostDeclarationStatus(t, s.ds, host1.UUID, test.ToMDMAppleDecl(profNameToPayload["D2"]), fleet.MDMOperationTypeInstall, "")
	forceSetAppleHostDeclarationStatus(t, s.ds, host2.UUID, test.ToMDMAppleDecl(profNameToPayload["D2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryVerified)

	// delete the declaration, will have removed it for host1 and set to remove pending for host2
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profNameToPayload["D2"].ProfileUUID), nil, http.StatusOK, &deleteResp)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// for the Windows profile, set host4 as failed
	forceSetWindowsHostProfileStatus(t, s.ds, host4.UUID, test.ToMDMWindowsConfigProfile(profNameToPayload["W2"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)

	// delete the Windows profile. Removal is async: the delete retains the profile content, and the profile-manager cron flips both
	// hosts' surviving rows (pending/failed) to remove+pending and enqueues the <Delete>.
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profNameToPayload["W2"].ProfileUUID), nil, http.StatusOK, &deleteResp)
	s.awaitTriggerProfileSchedule(t)

	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		host3: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
		},
		host4: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
		},
	})

	// for the Apple profile, set host1 as NULL (pending not queued yet), and leave host2 as actually pending (queued)
	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["A2"]), fleet.MDMOperationTypeInstall, "")

	assertIsCommandActiveForHostAndProfile := func(hostUUID, profileUUID string, wantActive bool) {
		var active bool
		ctx := t.Context()
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &active, `SELECT neq.active
			FROM
				nano_enrollment_queue neq
				JOIN host_mdm_apple_profiles hmap
					ON hmap.command_uuid = neq.command_uuid AND hmap.host_uuid = neq.id
			WHERE
				hmap.host_uuid = ? AND
				hmap.profile_uuid = ?`, hostUUID, profileUUID)
		})
		if wantActive {
			require.True(t, active)
		} else {
			require.False(t, active)
		}
	}

	assertIsCommandActiveForHostAndProfile(host1.UUID, profNameToPayload["A2"].ProfileUUID, true)
	assertIsCommandActiveForHostAndProfile(host2.UUID, profNameToPayload["A2"].ProfileUUID, true)

	// delete the profile, will remove the row for host1 and set host2 to pending remove (and will deactivate the associated nano command)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profNameToPayload["A2"].ProfileUUID), nil, http.StatusOK, &deleteResp)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: "D2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	assertIsCommandActiveForHostAndProfile(host2.UUID, profNameToPayload["A2"].ProfileUUID, false)

	// set the remove operations to verifying and reconcile profiles
	forceSetAppleHostProfileStatus(t, s.ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["A2"]), fleet.MDMOperationTypeRemove, fleet.MDMDeliveryVerifying)
	forceSetAppleHostDeclarationStatus(t, s.ds, host2.UUID, test.ToMDMAppleDecl(profNameToPayload["D2"]), fleet.MDMOperationTypeRemove, fleet.MDMDeliveryVerifying)
	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
	// Both Windows hosts still have the W2 remove+pending row from the
	// earlier deletion -- no simulated device check-in has processed the
	// <Delete> command yet.
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		host3: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
		},
		host4: {
			{Name: "W2", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
		},
	})

	// Simulate device check-ins to process the <Delete> commands. After
	// the device responds with OK, the remove+verified rows are cleaned up.
	for _, device := range []*mdmtest.TestWindowsMDMClient{mdmDevice3, mdmDevice4} {
		cmds, err := device.StartManagementSession()
		require.NoError(t, err)
		msgID, err := device.GetCurrentMsgID()
		require.NoError(t, err)
		for _, c := range cmds {
			status := syncml.CmdStatusOK
			device.AppendResponse(fleet.SyncMLCmd{
				XMLName: xml.Name{Local: fleet.CmdStatus},
				MsgRef:  &msgID,
				CmdRef:  &c.Cmd.CmdID.Value,
				Cmd:     new(c.Verb),
				Data:    &status,
				CmdID:   fleet.CmdID{Value: uuid.NewString()},
			})
		}
		_, err = device.SendResponse()
		require.NoError(t, err)
	}

	// After the devices confirmed the <Delete>, the remove rows should be
	// cleaned up (verified removes are deleted from host_mdm_windows_profiles).
	s.assertHostWindowsConfigProfiles(map[*fleet.Host][]fleet.HostMDMWindowsProfile{
		host3: {},
		host4: {},
	})

	// add new profile A3 and re-add A2, behaves as if a new profile because it has a new uuid
	oldA2Contents := profiles[1].Contents
	profiles = []fleet.MDMProfileBatchPayload{
		{
			Name:     "A2",
			Contents: oldA2Contents,
		},
		{
			Name:     "A3",
			Contents: mobileconfigForTest("A3", "A3"),
		},
	}
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: profiles}, http.StatusNoContent)

	// list the profiles to get the UUIDs
	s.DoJSON("GET", "/api/latest/fleet/configuration_profiles", nil, http.StatusOK, &listResp)
	profNameToPayload = make(map[string]*fleet.MDMConfigProfilePayload)
	for _, prof := range listResp.Profiles {
		if len(prof.Checksum) == 0 {
			// not important, but must not be empty or it causes issues when forcing a status
			prof.Checksum = []byte("checksum")
		}
		profNameToPayload[prof.Name] = prof
		t.Logf("new profile %s: %s", prof.Name, prof.ProfileUUID)
	}
	s.awaitTriggerProfileSchedule(t)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "A3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "A3", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	// set A3 as failed on host1, and removed on host2
	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["A3"]), fleet.MDMOperationTypeInstall, fleet.MDMDeliveryFailed)
	forceSetAppleHostProfileStatus(t, s.ds, host2.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["A3"]), fleet.MDMOperationTypeRemove, fleet.MDMDeliveryFailed)

	// delete the profile, will mark host1 as pending remove and will not touch host2 (not installed)
	s.DoJSON("DELETE", fmt.Sprintf("/api/latest/fleet/configuration_profiles/%s", profNameToPayload["A3"].ProfileUUID), nil, http.StatusOK, &deleteResp)

	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "A3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "A3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryFailed},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})

	forceSetAppleHostProfileStatus(t, s.ds, host1.UUID, test.ToMDMAppleConfigProfile(profNameToPayload["A3"]), fleet.MDMOperationTypeRemove, fleet.MDMDeliveryVerifying)
	s.awaitTriggerProfileSchedule(t)
	s.assertHostAppleConfigProfiles(map[*fleet.Host][]fleet.HostMDMAppleProfile{
		host1: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
		host2: {
			{Identifier: "A2", OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: "A3", OperationType: fleet.MDMOperationTypeRemove, Status: &fleet.MDMDeliveryFailed},
			{Identifier: mobileconfig.FleetdConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
			{Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, OperationType: fleet.MDMOperationTypeInstall, Status: &fleet.MDMDeliveryPending},
		},
	})
}

// This test verifies that there is no longer a race condition in apple profile resending
func (s *integrationMDMTestSuite) TestAppleProfileResendRaceCondition() {
	t := s.T()
	ctx := context.Background()

	// Create a host and enroll it in MDM
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)

	// Delete the key to let the reconciler pick up the profiles
	err := s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	scimUserID, err := s.ds.CreateScimUser(ctx, &fleet.ScimUser{UserName: "user@example.com"})
	require.NoError(t, err)

	// Assign scim user to host
	hostIdStr := fmt.Sprint(host.ID)
	s.Do("PUT", "/api/latest/fleet/hosts/"+hostIdStr+"/device_mapping", putHostDeviceMappingRequest{
		Email:  "user@example.com",
		Source: "idp",
	}, http.StatusOK)

	// Create a profile that uses IDP variables
	profileWithIDPVar := mobileconfigForTestWithContent("TestProfile", "com.test.profile", "com.test.profile.content", "com.test.profile", "Test IDP Variable Profile")

	// Replace the profile content to include an IDP variable
	profileContent := string(profileWithIDPVar)
	profileContent = strings.Replace(profileContent, "<key>ShowRecoveryKey</key>", "<key>TestVariable</key>", 1)
	profileContent = strings.Replace(profileContent, "<false/>", "<string>$FLEET_VAR_HOST_END_USER_IDP_USERNAME</string>", 1)
	profileWithIDPVar = []byte(profileContent)

	// Upload the profile using the new endpoint that supports variables
	s.Do("POST", "/api/latest/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "TestProfile", Contents: profileWithIDPVar},
	}}, http.StatusNoContent) // Setup SCIM user data for the host

	// No profiles until reconciler
	hostProfiles, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Empty(t, hostProfiles, "Host should not have any profiles before sync")

	// Trigger initial profile sync - profile should be set to pending/installing
	s.awaitTriggerProfileSchedule(t)

	// Check that install command was sent, but do not acknowledge it yet
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)

	seenProfile := false
	profileCmdID := ""
	for cmd != nil {
		if cmd.Command.RequestType != "InstallProfile" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}
		var fullCmd micromdm.CommandPayload
		err = plist.Unmarshal(cmd.Raw, &fullCmd)
		require.NoError(t, err)
		if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), "TestProfile") {
			seenProfile = true
			profileCmdID = cmd.CommandUUID
			break
		}

		// Acknowledge other commands
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	require.True(t, seenProfile, "Expected install command for TestProfile not found")

	// Verify profile is in pending status
	hostProfiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)

	var testProfile *fleet.HostMDMAppleProfile
	for _, p := range hostProfiles {
		if p.Identifier == "com.test.profile" {
			testProfile = &p
			break
		}
	}
	require.NotNil(t, testProfile)
	require.Equal(t, fleet.MDMDeliveryPending, *testProfile.Status)

	// Now simulate the race condition:
	// we trigger a resend before the acknowledgement comes back

	// 1. Trigger an IDP variable change by updating SCIM user
	_, err = s.ds.ReplaceScimUser(ctx, &fleet.ScimUser{ID: scimUserID, UserName: "newuser@example.com"})
	require.NoError(t, err)

	// 2. At this point, the profile should be marked for resend (status = NULL)
	hostProfiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)

	for _, p := range hostProfiles {
		if p.Identifier == "com.test.profile" {
			testProfile = &p
			break
		}
	}
	require.NotNil(t, testProfile)
	require.Equal(t, fleet.MDMDeliveryPending, *testProfile.Status) // Should be NULL (pending for the user)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status *fleet.MDMDeliveryStatus
		err := sqlx.GetContext(t.Context(), q, &status, `SELECT status FROM host_mdm_apple_profiles WHERE profile_identifier = ?`, testProfile.Identifier)
		require.Nil(t, status)
		return err
	})

	// Acknowledge the original install command now, simulating the device response
	cmd, err = mdmDevice.Acknowledge(profileCmdID)
	require.NoError(t, err)

	// Now check if we see any new TestProfile cmds (we need to ack them here, since we might have skipped some above.)
	seenProfile = false
	for cmd != nil {
		if cmd.Command.RequestType != "InstallProfile" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}
		var fullCmd micromdm.CommandPayload
		err = plist.Unmarshal(cmd.Raw, &fullCmd)
		require.NoError(t, err)
		if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), "TestProfile") {
			seenProfile = true
			// Acknowledge the resend command
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		} else {
			// Acknowledge other commands
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}
	require.Nil(t, cmd, "No further commands should be pending after acknowledging install")
	require.False(t, seenProfile, "No resend install command for TestProfile should be sent due to race condition")

	// Verify the profile is still in pending status (null in the DB)
	// aka. no race condition.
	hostProfiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)

	for _, p := range hostProfiles {
		if p.Identifier == "com.test.profile" {
			testProfile = &p
			break
		}
	}
	require.NotNil(t, testProfile)
	require.Equal(t, fleet.MDMDeliveryPending, *testProfile.Status)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status *fleet.MDMDeliveryStatus
		err := sqlx.GetContext(t.Context(), q, &status, `SELECT status FROM host_mdm_apple_profiles WHERE profile_identifier = ?`, testProfile.Identifier)
		require.Nil(t, status)
		return err
	})

	// run reconciler to resend any pending profiles
	s.awaitTriggerProfileSchedule(t)
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	// Now we should see the resend command
	seenProfile = false
	for cmd != nil {
		if cmd.Command.RequestType != "InstallProfile" {
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
			continue
		}
		var fullCmd micromdm.CommandPayload
		err = plist.Unmarshal(cmd.Raw, &fullCmd)
		require.NoError(t, err)
		if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), "TestProfile") {
			seenProfile = true
			// Acknowledge the resend command
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		} else {
			// Acknowledge other commands
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}
	require.True(t, seenProfile, "Resend install command for TestProfile should be sent after reconciler runs")

	// And we should now also see the profile being marked as verifying
	hostProfiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	for _, p := range hostProfiles {
		if p.Identifier == "com.test.profile" {
			testProfile = &p
			break
		}
	}
	require.NotNil(t, testProfile)
	require.Equal(t, fleet.MDMDeliveryVerifying, *testProfile.Status)

	// Verify deleting the device mapping also triggers a resend
	s.Do("PUT", "/api/latest/fleet/hosts/"+hostIdStr+"/device_mapping", putHostDeviceMappingRequest{
		Email:  "",
		Source: "idp",
	}, http.StatusOK)
	hostProfiles, err = s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	testProfile = nil
	for _, p := range hostProfiles {
		if p.Identifier == "com.test.profile" {
			testProfile = &p
			break
		}
	}
	require.NotNil(t, testProfile)
	require.Equal(t, fleet.MDMDeliveryPending, *testProfile.Status) // Should be pending again due to device mapping deletion

	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		var status *fleet.MDMDeliveryStatus
		err := sqlx.GetContext(t.Context(), q, &status, `SELECT status FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_identifier = ?`, host.UUID, testProfile.Identifier)
		require.Nil(t, status)
		return err
	})

	// returns the TestProfile payload the device received, acknowledging every command
	installedTestProfile := func() string {
		s.awaitTriggerProfileSchedule(t)
		var payload string
		cmd, err := mdmDevice.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == "InstallProfile" {
				var fullCmd micromdm.CommandPayload
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))
				if p := string(fullCmd.Command.InstallProfile.Payload); strings.Contains(p, "TestProfile") {
					payload = p
				}
			}
			cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
		return payload
	}
	testProfileStatus := func() fleet.MDMDeliveryStatus {
		hostProfiles, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
		require.NoError(t, err)
		for _, p := range hostProfiles {
			if p.Identifier == "com.test.profile" {
				require.NotNil(t, p.Status)
				return *p.Status
			}
		}
		require.FailNow(t, "TestProfile not found on host")
		return ""
	}

	// the cleared username can't be resolved, so the profile fails
	require.Empty(t, installedTestProfile())
	require.Equal(t, fleet.MDMDeliveryFailed, testProfileStatus())

	// without a SCIM user, every new IdP username resends the profile with that value
	for _, email := range []string{"manual.one@example.com", "manual.two@example.com"} {
		s.Do("PUT", "/api/latest/fleet/hosts/"+hostIdStr+"/device_mapping", putHostDeviceMappingRequest{
			Email:  email,
			Source: "idp",
		}, http.StatusOK)
		require.Equal(t, fleet.MDMDeliveryPending, testProfileStatus())
		require.Contains(t, installedTestProfile(), email)
		require.Equal(t, fleet.MDMDeliveryVerifying, testProfileStatus())
	}
}

// Moving a host between fleets that hold a byte-identical profile must keep the
// pending install the host already has queued, whether the device hasn't
// fetched it yet or answered NotNow.
func (s *integrationMDMTestSuite) TestAppleProfileTransferKeepsPendingInstallForIdenticalProfile() {
	t := s.T()
	ctx := t.Context()

	const identifier = "com.example.transfer.shared"
	shared := mobileconfigForTest("TransferShared", identifier)

	teamA, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_A"})
	require.NoError(t, err)
	teamB, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name() + "_B"})
	require.NoError(t, err)
	for _, tm := range []*fleet.Team{teamA, teamB} {
		s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: [][]byte{shared}},
			http.StatusNoContent, "team_id", fmt.Sprint(tm.ID))
	}

	profileUUIDForTeam := func(teamID uint) string {
		var profUUID string
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &profUUID,
				`SELECT profile_uuid FROM mdm_apple_configuration_profiles WHERE team_id = ? AND identifier = ?`, teamID, identifier)
		})
		return profUUID
	}

	type hostProfileRow struct {
		ProfileUUID string                   `db:"profile_uuid"`
		Status      *fleet.MDMDeliveryStatus `db:"status"`
		CommandUUID string                   `db:"command_uuid"`
	}
	getRow := func(hostUUID string) hostProfileRow {
		var row hostProfileRow
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &row,
				`SELECT profile_uuid, status, command_uuid FROM host_mdm_apple_profiles WHERE host_uuid = ? AND profile_identifier = ?`,
				hostUUID, identifier)
		})
		return row
	}
	isQueued := func(hostUUID, cmdUUID string) bool {
		var n int
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			return sqlx.GetContext(ctx, q, &n,
				`SELECT COUNT(*) FROM nano_enrollment_queue WHERE id = ? AND command_uuid = ? AND active = 1`, hostUUID, cmdUUID)
		})
		return n > 0
	}
	transfer := func(h *fleet.Host, teamID uint) {
		s.Do("POST", "/api/v1/fleet/hosts/transfer",
			addHostsToTeamRequest{TeamID: &teamID, HostIDs: []uint{h.ID}}, http.StatusOK)
	}

	// drainCommands answers every queued command, NotNow for notNowCmd and
	// Acknowledged for the rest, and reports whether wantCmd was delivered.
	drainCommands := func(device *mdmtest.TestAppleMDMClient, wantCmd, notNowCmd string) bool {
		var seen bool
		cmd, err := device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.CommandUUID == wantCmd {
				seen = true
			}
			if cmd.CommandUUID == notNowCmd {
				cmd, err = device.NotNow(cmd.CommandUUID)
			} else {
				cmd, err = device.Acknowledge(cmd.CommandUUID)
			}
			require.NoError(t, err)
		}
		return seen
	}

	offlineHost, offlineDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	notNowHost, notNowDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	for _, h := range []*fleet.Host{offlineHost, notNowHost} {
		transfer(h, teamA.ID)
		require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+h.UUID))
	}
	s.awaitTriggerProfileSchedule(t)

	profA, profB := profileUUIDForTeam(teamA.ID), profileUUIDForTeam(teamB.ID)
	require.NotEqual(t, profA, profB)

	queuedCmds := make(map[string]string, 2)
	for _, h := range []*fleet.Host{offlineHost, notNowHost} {
		row := getRow(h.UUID)
		require.Equal(t, profA, row.ProfileUUID)
		require.NotNil(t, row.Status)
		require.Equal(t, fleet.MDMDeliveryPending, *row.Status)
		require.NotEmpty(t, row.CommandUUID)
		queuedCmds[h.UUID] = row.CommandUUID
	}

	// the offline host never checks in; the other fetches the install and defers it
	require.True(t, drainCommands(notNowDevice, queuedCmds[notNowHost.UUID], queuedCmds[notNowHost.UUID]))

	for _, h := range []*fleet.Host{offlineHost, notNowHost} {
		transfer(h, teamB.ID)
		require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+h.UUID))
	}
	s.awaitTriggerProfileSchedule(t)

	for _, tc := range []struct {
		host   *fleet.Host
		device *mdmtest.TestAppleMDMClient
	}{
		{offlineHost, offlineDevice},
		{notNowHost, notNowDevice},
	} {
		cmdUUID := queuedCmds[tc.host.UUID]
		row := getRow(tc.host.UUID)
		require.Equal(t, profB, row.ProfileUUID, "host %s", tc.host.UUID)
		require.NotNil(t, row.Status)
		require.Equal(t, fleet.MDMDeliveryPending, *row.Status)
		require.Equal(t, cmdUUID, row.CommandUUID, "the new fleet's row takes over the queued install")
		require.True(t, isQueued(tc.host.UUID, cmdUUID), "the taken-over install must still be queued for the device")

		require.True(t, drainCommands(tc.device, cmdUUID, ""), "device must receive the queued install")
		row = getRow(tc.host.UUID)
		require.NotNil(t, row.Status)
		require.Equal(t, fleet.MDMDeliveryVerifying, *row.Status)
	}
}

// mdmActiveCmdCount returns the number of active (undelivered) nano_enrollment_queue
// rows for the given command UUID.
func mdmActiveCmdCount(t *testing.T, ds *mysql.Datastore, cmdUUID string) int {
	var n int
	mysqltest.ExecAdhocSQL(t, ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(context.Background(), q, &n,
			`SELECT COUNT(*) FROM nano_enrollment_queue WHERE command_uuid = ? AND active = 1`, cmdUUID)
	})
	return n
}

// ackUntilThenNotNow acknowledges queued commands until it reaches cmdUUID, which it
// answers with NotNow. Fails the test if the queue drains without delivering cmdUUID.
func ackUntilThenNotNow(t *testing.T, device *mdmtest.TestAppleMDMClient, cmdUUID string) {
	cmd, err := device.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.CommandUUID == cmdUUID {
			_, err = device.NotNow(cmdUUID)
			require.NoError(t, err)
			return
		}
		cmd, err = device.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	t.Fatalf("command %s was never delivered", cmdUUID)
}

// labelGateHost creates a label, makes the host a member, and marks the host's labels
// as reported so include-any gating evaluates it as a match.
func (s *integrationMDMTestSuite) labelGateHost(t *testing.T, host *fleet.Host, name string) *fleet.Label {
	ctx := t.Context()
	label, err := s.ds.NewLabel(ctx, &fleet.Label{Name: name, Query: "select 1;"})
	require.NoError(t, err)
	host.LabelUpdatedAt = time.Now()
	require.NoError(t, s.ds.UpdateHost(ctx, host))
	require.NoError(t, s.ds.AsyncBatchInsertLabelMembership(ctx, [][2]uint{{label.ID, host.ID}}))
	return label
}

// TestProfileReconcilerCancelsInstallationWhenNotNowResponseRecorded is a regression test that
// ensures a pending device-scoped InstallProfile command is cancelled when the host leaves scope after a NotNow response.
func (s *integrationMDMTestSuite) TestProfileReconcilerCancelsInstallationWhenNotNowResponseRecorded() {
	t := s.T()
	ctx := t.Context()
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	host, mdmDevice := s.enrollHostDrainInitialProfiles(t)
	label := s.labelGateHost(t, host, t.Name()+"-lbl")

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "P1", Contents: mobileconfigForTest("P1", "P1"), LabelsIncludeAny: []string{label.Name}},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// take P1's install command from its hmap row (Idle may deliver other
	// re-enqueued profiles first), ack up to it and answer it with NotNow
	ok, i := hostHasAppleProfileOp(t, s.ds, host.UUID, "P1", fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	require.NotEmpty(t, i)
	ackUntilThenNotNow(t, mdmDevice, i)

	// remove the host from the label (scope change, not a deletion), then reconcile
	require.NoError(t, s.ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{label.ID, host.ID}}))
	s.awaitTriggerProfileSchedule(t)

	// desired: the install command is cancelled; bug: the NotNow result defeats the DELETE
	require.Zero(t, mdmActiveCmdCount(t, s.ds, i), "install command still active after profile left scope")
}

// TestProfileUserScopedPendingInstallCancelled is a regression test where when label descoping a user-scoped profile, that is pending installation
// ensures the install command is cancelled. To avoid NotNow'ing producing an incorrect order and could leave profiles that Fleet is no longer tracking on the device.
func (s *integrationMDMTestSuite) TestProfileUserScopedPendingInstallCancelled() {
	t := s.T()
	ctx := t.Context()
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	host, mdmDevice := s.enrollHostDrainInitialProfiles(t)
	require.NoError(t, mdmDevice.UserEnroll())
	userEnr, err := s.ds.GetNanoMDMUserEnrollment(ctx, host.UUID)
	require.NoError(t, err)
	require.NotNil(t, userEnr)
	label := s.labelGateHost(t, host, t.Name()+"-lbl")

	scope := fleet.PayloadScopeUser
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "P2", Contents: scopedMobileconfigForTest("P2", "P2.user", &scope), LabelsIncludeAny: []string{label.Name}},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// take P2's install command from its hmap row (no device interaction needed;
	// this is a pure keying bug)
	ok, i := hostHasAppleProfileOp(t, s.ds, host.UUID, "P2.user", fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	require.NotEmpty(t, i)

	// sanity: the queue row is keyed by the user enrollment ID, not the host UUID
	var qid string
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		return sqlx.GetContext(ctx, q, &qid, `SELECT id FROM nano_enrollment_queue WHERE command_uuid = ?`, i)
	})
	require.Equal(t, userEnr.ID, qid)

	require.NoError(t, s.ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{label.ID, host.ID}}))
	s.awaitTriggerProfileSchedule(t)

	// desired: the user-channel install is cancelled; bug: cleanup keyed by host UUID misses it
	require.Zero(t, mdmActiveCmdCount(t, s.ds, i), "user-scoped install still active after profile left scope")
}

// TestProfileFailedVerificationGetsRemove is a regression test for the case where osquery profile verification fails on a previous ACK'ed InstallProfile command.
// This test ensures a RemoveProfile command gets sent to ensure no lingering profiles is left.
func (s *integrationMDMTestSuite) TestProfileFailedVerificationGetsRemove() {
	t := s.T()
	ctx := t.Context()
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	host, mdmDevice := s.enrollHostDrainInitialProfiles(t)
	label := s.labelGateHost(t, host, t.Name()+"-lbl")

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "P3", Contents: mobileconfigForTest("P3", "P3"), LabelsIncludeAny: []string{label.Name}},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// ack all queued installs; P3 is now genuinely on the device
	checkNextPayloads(t, mdmDevice, false)

	// exactly what setMDMProfilesFailedDB produces when the verifier can't see the profile
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(ctx,
			`UPDATE host_mdm_apple_profiles SET status = 'failed', detail = 'Failed, was verifying'
			 WHERE host_uuid = ? AND profile_identifier = 'P3' AND operation_type = 'install'`, host.UUID)
		return err
	})

	require.NoError(t, s.ds.AsyncBatchDeleteLabelMembership(ctx, [][2]uint{{label.ID, host.ID}}))
	s.awaitTriggerProfileSchedule(t)

	// desired: a RemoveProfile is enqueued to pull the profile off the device
	ok, _ := hostHasAppleProfileOp(t, s.ds, host.UUID, "P3", fleet.MDMOperationTypeRemove)
	require.True(t, ok, "failed install that left scope got no RemoveProfile; profile stranded on device")
}

// TestProfileEditLeaksOldInstallCommand is a regression test that ensures editing a profile via gitops or edit, cancels the previous in-flight command
// to avoid NotNow responses producing out of order commands.
func (s *integrationMDMTestSuite) TestProfileEditLeaksOldInstallCommand() {
	t := s.T()
	ctx := t.Context()
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	host, mdmDevice := s.enrollHostDrainInitialProfiles(t)

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "P5", Contents: mobileconfigForTest("P5", "P5")},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// take P5's install command from its hmap row (Idle may deliver other
	// re-enqueued profiles first), ack up to it and answer it with NotNow
	ok, iOld := hostHasAppleProfileOp(t, s.ds, host.UUID, "P5", fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	require.NotEmpty(t, iOld)
	ackUntilThenNotNow(t, mdmDevice, iOld)

	// edit P5's content (new random PayloadUUID -> new checksum), same name/identifier
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "P5", Contents: mobileconfigForTest("P5", "P5")},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	ok, iNew := hostHasAppleProfileOp(t, s.ds, host.UUID, "P5", fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	require.NotEmpty(t, iNew)

	// desired: the old install command is cancelled; bug: it stays active and can apply v1 over v2
	require.Zero(t, mdmActiveCmdCount(t, s.ds, iOld), "old install command still active after profile edit")
	require.Equal(t, 1, mdmActiveCmdCount(t, s.ds, iNew), "new install command not active after profile edit")
}

// TestProfileEditCancelsUndeliveredInstallCommand is the companion to
// TestProfileEditLeaksOldInstallCommand: the device never picks up the first install
// (offline host), so the superseded command has no result row at all. This isolates the
// toInstall cancellation wiring (root cause E) from the NotNow-tolerant DELETE (fix 1) —
// if only this test goes red, the wiring broke; if only the NotNow variant goes red, the
// DELETE's NotNow guard broke.
func (s *integrationMDMTestSuite) TestProfileEditCancelsUndeliveredInstallCommand() {
	t := s.T()
	ctx := t.Context()
	require.NoError(t, s.ds.ApplyEnrollSecrets(ctx, nil, []*fleet.EnrollSecret{{Secret: t.Name()}}))
	host, _ := s.enrollHostDrainInitialProfiles(t)

	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "P6", Contents: mobileconfigForTest("P6", "P6")},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// the install is queued but the device never checks in
	ok, iOld := hostHasAppleProfileOp(t, s.ds, host.UUID, "P6", fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	require.NotEmpty(t, iOld)
	require.Equal(t, 1, mdmActiveCmdCount(t, s.ds, iOld))

	// edit P6's content (new random PayloadUUID -> new checksum), same name/identifier
	s.Do("POST", "/api/v1/fleet/mdm/profiles/batch", batchSetMDMProfilesRequest{Profiles: []fleet.MDMProfileBatchPayload{
		{Name: "P6", Contents: mobileconfigForTest("P6", "P6")},
	}}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	ok, iNew := hostHasAppleProfileOp(t, s.ds, host.UUID, "P6", fleet.MDMOperationTypeInstall)
	require.True(t, ok)
	require.NotEmpty(t, iNew)
	require.NotEqual(t, iOld, iNew)

	// the undelivered v1 install must be cancelled so the host doesn't run v1 then v2
	require.Zero(t, mdmActiveCmdCount(t, s.ds, iOld), "undelivered install command still active after profile edit")
	require.Equal(t, 1, mdmActiveCmdCount(t, s.ds, iNew), "new install command not active after profile edit")
}

// TestPolicyAutomationResendConfigurationProfile covers the policy automation that resends a
// configuration profile when its policy starts failing on a host: the profile is queued for
// redelivery, actually reaches the device, and the resend is recorded as a Fleet-initiated
// activity naming the policy that triggered it. The activity is the only record of an automated
// resend, so its payload is asserted in full.
func (s *integrationMDMTestSuite) TestPolicyAutomationResendConfigurationProfile() {
	t := s.T()
	ctx := context.Background()

	tm, err := s.ds.NewTeam(ctx, &fleet.Team{Name: t.Name()})
	require.NoError(t, err)

	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setupPusher(s, t, mdmDevice)
	// Process the enrollment and drain the profiles it queues, so later payload checks see only
	// what this test triggers.
	s.awaitRunAppleMDMWorkerSchedule()
	checkNextPayloads(t, mdmDevice, false)

	// Add a profile to the team and move the host into it. The profile schedule throttles
	// per-host reprocessing, so the key is cleared before each trigger.
	mcUUID := "a" + uuid.NewString()
	profName := "name-" + mcUUID
	prof := mcBytesForTest(profName, "identifier-"+mcUUID, mcUUID)
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		stmt := `INSERT INTO mdm_apple_configuration_profiles (profile_uuid, team_id, name, identifier, mobileconfig, checksum, uploaded_at) VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP);`
		_, err := q.ExecContext(ctx, stmt, mcUUID, tm.ID, profName, "identifier-"+mcUUID, prof, test.MakeTestBytes())
		return err
	})
	require.NoError(t, s.ds.AddHostsToTeam(ctx, fleet.NewAddHostsToTeamParams(&tm.ID, []uint{host.ID})))

	syncProfiles := func() {
		require.NoError(t, s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID))
		s.awaitTriggerProfileSchedule(t)
	}
	// Deliver the team's profiles, including this one. The payloads also cover the fleetd config
	// and CA profiles swapped out by the team move, so they aren't asserted here.
	syncProfiles()
	checkNextPayloads(t, mdmDevice, false)

	profileStatus := func() *fleet.MDMDeliveryStatus {
		hostProfs, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
		require.NoError(t, err)
		for _, p := range hostProfs {
			if p.ProfileUUID == mcUUID {
				return p.Status
			}
		}
		t.Fatalf("profile %s not found on host %s", mcUUID, host.UUID)
		return nil
	}

	// Mark it verified: a host that has drifted from a verified profile is what the automation
	// exists for. (A pending or verifying profile is deliberately left alone.)
	setProfileStatus := func(status fleet.MDMDeliveryStatus) {
		mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
			stmt := `UPDATE host_mdm_apple_profiles SET status = ? WHERE profile_uuid = ? AND host_uuid = ?`
			_, err := q.ExecContext(ctx, stmt, status, mcUUID, host.UUID)
			return err
		})
	}
	setProfileStatus(fleet.MDMDeliveryVerified)
	require.Equal(t, &fleet.MDMDeliveryVerified, profileStatus())

	// A policy that resends the profile when it fails, plus one with no profile attached, to
	// confirm only the former triggers a resend.
	newPolicy := func(name string, profileUUID *string) *fleet.Policy {
		var resp fleet.TeamPolicyResponse
		s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/teams/%d/policies", tm.ID), fleet.TeamPolicyRequest{
			Name:        name,
			Query:       "SELECT 1;",
			Platform:    "darwin",
			ProfileUUID: profileUUID,
		}, http.StatusOK, &resp)
		require.NotNil(t, resp.Policy)
		return resp.Policy
	}
	resendPolicy := newPolicy("resend profile policy", &mcUUID)
	require.NotNil(t, resendPolicy.ResendConfigurationProfile)
	require.Equal(t, mcUUID, resendPolicy.ResendConfigurationProfile.UUID)
	plainPolicy := newPolicy("plain policy", nil)

	reportPolicies := func(results map[uint]*bool) {
		distributedResp := submitDistributedQueryResultsResponse{}
		s.DoJSON("POST", "/api/osquery/distributed/write",
			genDistributedReqWithPolicyResults(host, results), http.StatusOK, &distributedResp)
	}

	// Both policies passing: nothing is resent, the profile stays verified.
	reportPolicies(map[uint]*bool{resendPolicy.ID: new(true), plainPolicy.ID: new(true)})
	require.Equal(t, &fleet.MDMDeliveryVerified, profileStatus())

	// The policy with no profile failing: still nothing to resend.
	reportPolicies(map[uint]*bool{resendPolicy.ID: new(true), plainPolicy.ID: new(false)})
	require.Equal(t, &fleet.MDMDeliveryVerified, profileStatus())

	// The resend policy fails: the profile is queued for redelivery. Its status is cleared to NULL
	// in the DB, which the API reports as pending, and the resend is recorded.
	reportPolicies(map[uint]*bool{resendPolicy.ID: new(false), plainPolicy.ID: new(true)})
	require.Equal(t, &fleet.MDMDeliveryPending, profileStatus())

	// The activity names the host, the profile and the policy that triggered the resend, and is
	// attributed to Fleet rather than to a user.
	s.lastActivityMatchesExtended(
		fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "profile_name": %q, "profile_uuid": %q, "policy_id": %d, "policy_name": %q}`,
			host.ID, host.DisplayName(), profName, mcUUID, resendPolicy.ID, resendPolicy.Name),
		0,
		new(true),
	)

	// The policy keeps failing on the next run, but it did not pass in between, so nothing is
	// resent and no further activity is recorded.
	lastActivityID := s.lastActivityMatches("", "", 0)
	reportPolicies(map[uint]*bool{resendPolicy.ID: new(false)})
	require.Equal(t, lastActivityID, s.lastActivityMatches("", "", 0),
		"a policy that was already failing should not resend again")

	// The profile really is redelivered to the device on the next profile schedule run.
	syncProfiles()
	installs, removes := checkNextPayloads(t, mdmDevice, false)
	require.Len(t, installs, 1)
	s.signedProfilesMatch([][]byte{prof}, installs)
	require.Empty(t, removes)
	require.Equal(t, &fleet.MDMDeliveryVerifying, profileStatus())

	// Even with the profile back to verified — so nothing else would block a resend — a policy
	// that was already failing still doesn't trigger one.
	setProfileStatus(fleet.MDMDeliveryVerified)
	reportPolicies(map[uint]*bool{resendPolicy.ID: new(false)})
	require.Equal(t, lastActivityID, s.lastActivityMatches("", "", 0),
		"a verified profile should not be resent for a policy that was already failing")
	require.Equal(t, &fleet.MDMDeliveryVerified, profileStatus())

	// Once the policy passes and fails again, the transition triggers another resend.
	reportPolicies(map[uint]*bool{resendPolicy.ID: new(true)})
	reportPolicies(map[uint]*bool{resendPolicy.ID: new(false)})
	require.Equal(t, &fleet.MDMDeliveryPending, profileStatus())
	s.lastActivityMatchesExtended(
		fleet.ActivityTypeResentConfigurationProfile{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "profile_name": %q, "profile_uuid": %q, "policy_id": %d, "policy_name": %q}`,
			host.ID, host.DisplayName(), profName, mcUUID, resendPolicy.ID, resendPolicy.Name),
		0,
		new(true),
	)

	// A Windows profile on a policy failing on this macOS host resends nothing and records no
	// activity: the host has no row for that profile, so the resend is rejected rather than
	// recorded as something that happened.
	winProf, err := s.ds.NewMDMWindowsConfigProfile(ctx, fleet.MDMWindowsConfigProfile{
		Name:   "win-" + uuid.NewString(),
		SyncML: []byte("<Replace></Replace>"),
		TeamID: &tm.ID,
	}, nil)
	require.NoError(t, err)
	winResendPolicy := newPolicy("windows resend profile policy", &winProf.ProfileUUID)

	lastActivityID = s.lastActivityMatches("", "", 0)
	reportPolicies(map[uint]*bool{winResendPolicy.ID: new(false)})
	require.Equal(t, lastActivityID, s.lastActivityMatches("", "", 0),
		"no activity should be recorded for a profile the host can't receive")
}

func (s *integrationMDMTestSuite) TestDontIgnoreAnyProfileErrors() {
	t := s.T()
	ctx := context.Background()

	appConfig, err := s.ds.AppConfig(ctx)
	require.NoError(t, err)
	appConfig.MDM.MacOSSetup = fleet.MacOSSetup{
		EnableEndUserAuthentication: false,
	}
	err = s.ds.SaveAppConfig(ctx, appConfig)
	require.NoError(t, err)

	// Create a host and a couple of profiles
	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitTriggerProfileSchedule(t)
	s.awaitRunAppleMDMWorkerSchedule()

	err = s.keyValueStore.Delete(ctx, fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", "I1"),
		mobileconfigForTest("N2", "I2"),
	}

	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// The profiles should be associated with the host we made + the standard fleet configs
	profs, err := s.ds.GetHostMDMAppleProfiles(ctx, host.UUID)
	require.NoError(t, err)
	require.Len(t, profs, 4)

	// Acknowledge the profiles so we can mark them as verified
	cmd, err := mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I1": {Identifier: "I1", DisplayName: "I1", InstallDate: time.Now()},
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetdConfigPayloadIdentifier:      {Identifier: mobileconfig.FleetdConfigPayloadIdentifier, DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetCARootConfigPayloadIdentifier: {Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, DisplayName: "I2", InstallDate: time.Now()},
	}))

	// Check that the profile is marked as verified when fetching the host
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.EqualValues(t, fleet.MDMDeliveryVerified, *hm.Status)
	}

	// remove the profiles
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	// On the host side, return errors for the two profile removal actions
	cmd, err = mdmDevice.Idle()
	require.NoError(t, err)
	for cmd != nil {
		if cmd.Command.RequestType == "RemoveProfile" {
			var errChain []mdm.ErrorChain
			var fullCmd micromdm.CommandPayload
			require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

			if fullCmd.Command.RemoveProfile.Identifier == "I1" {
				errChain = append(errChain, mdm.ErrorChain{ErrorCode: 89, ErrorDomain: "MDMClientError", USEnglishDescription: "Profile with identifier 'I1' not found."})
			} else {
				errChain = append(errChain, mdm.ErrorChain{ErrorCode: 96, ErrorDomain: "MDMClientError", USEnglishDescription: "Cannot replace profile 'I2' because it was not installed by the MDM server."})
			}
			cmd, err = mdmDevice.Err(cmd.CommandUUID, errChain)
			require.NoError(t, err)
			continue
		}
		cmd, err = mdmDevice.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// get that host - it should report "failed" for the profiles and include the error message detail.
	expectedErrs := map[string]string{
		"N1": "Failed to remove: MDMClientError (89): Profile with identifier 'I1' not found.\n",
		"N2": "Failed to remove: MDMClientError (96): Cannot replace profile 'I2' because it was not installed by the MDM server.\n",
	}
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		if wantErr, ok := expectedErrs[hm.Name]; ok {
			require.EqualValues(t, fleet.MDMDeliveryFailed, *hm.Status)
			require.Equal(t, wantErr, hm.Detail)
			continue
		}
		require.EqualValues(t, fleet.MDMDeliveryVerified, *hm.Status)
	}
}

func (s *integrationMDMTestSuite) TestRemoveFailedProfiles() {
	t := s.T()

	teamName := t.Name()
	team := &fleet.Team{
		Name:        teamName,
		Description: "desc " + teamName,
	}

	var createTeamResp teamResponse
	s.DoJSON("POST", "/api/latest/fleet/teams", team, http.StatusOK, &createTeamResp)
	require.NotZero(t, createTeamResp.Team.ID)
	team.ID = createTeamResp.Team.ID

	// Ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)

	host, mdmDevice := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	s.awaitRunAppleMDMWorkerSchedule()
	ident := uuid.NewString()

	// simulate TTL expiration
	err := s.keyValueStore.Delete(context.Background(), fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	mdmDeviceRespond := func(device *mdmtest.TestAppleMDMClient) {
		cmd, err := device.Idle()
		require.NoError(t, err)
		for cmd != nil {
			if cmd.Command.RequestType == "InstallProfile" {
				var fullCmd micromdm.CommandPayload
				require.NoError(t, plist.Unmarshal(cmd.Raw, &fullCmd))

				if strings.Contains(string(fullCmd.Command.InstallProfile.Payload), ident) {
					var errChain []mdm.ErrorChain
					errChain = append(errChain, mdm.ErrorChain{ErrorCode: -102, ErrorDomain: "CPProfile", USEnglishDescription: "The profile is either missing some required information, or contains information in an invalid format."})
					cmd, err = device.Err(cmd.CommandUUID, errChain)
					require.NoError(t, err)
					continue
				}
			}
			cmd, err = device.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)
		}
	}

	globalProfiles := [][]byte{
		mobileconfigForTest("N1", ident),
		mobileconfigForTest("N2", "I2"),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)
	mdmDeviceRespond(mdmDevice)
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		"I1": {Identifier: "I1", DisplayName: "I1", InstallDate: time.Now()},
	}))

	// Do additional trigger + command fetching cycles until max retries are exhausted.
	// Each cycle: cron re-enqueues the profile, device responds with error, HandleHostMDMProfileInstallResult increments retries.
	for range servermdm.MaxAppleProfileRetries - 1 {
		s.awaitTriggerProfileSchedule(t)
		mdmDeviceRespond(mdmDevice)
	}

	// One more cycle: cron re-enqueues, device responds with error. Since retries == MaxAppleProfileRetries,
	// HandleHostMDMProfileInstallResult marks the profile as failed.
	s.awaitTriggerProfileSchedule(t)
	mdmDeviceRespond(mdmDevice)

	// Verify the non-failing profiles via osquery
	require.NoError(t, apple_mdm.VerifyHostMDMProfiles(context.Background(), s.ds, host, map[string]*fleet.HostMacOSProfile{
		"I2": {Identifier: "I2", DisplayName: "I2", InstallDate: time.Now()},
		mobileconfig.FleetdConfigPayloadIdentifier:      {Identifier: mobileconfig.FleetdConfigPayloadIdentifier, DisplayName: "dn1", InstallDate: time.Now()},
		mobileconfig.FleetCARootConfigPayloadIdentifier: {Identifier: mobileconfig.FleetCARootConfigPayloadIdentifier, DisplayName: "dn2", InstallDate: time.Now()},
	}))

	// Check that the profile is marked as failed when fetching the host
	getHostResp := getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 4)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		if hm.Name == "N1" {
			require.EqualValues(t, fleet.MDMDeliveryFailed, *hm.Status)
			continue
		}

		require.EqualValues(t, fleet.MDMDeliveryVerified, *hm.Status)
	}

	// transfer host to a team without the failed profile

	s.DoJSON("POST", "/api/latest/fleet/hosts/transfer", addHostsToTeamRequest{TeamID: &team.ID, HostIDs: []uint{host.ID}}, http.StatusOK, &addHostsToTeamResponse{})
	s.awaitTriggerProfileSchedule(t)

	// confirm that we remove the failed profile
	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3) // This would be 4 if we hadn't deleted the profile that failed to install.
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.NotEqual(t, "N1", hm.Name)
	}

	// Test case where the profile never makes it to the host at all
	host, _ = createHostThenEnrollMDM(s.ds, s.server.URL, t)
	ident = uuid.NewString()
	err = s.keyValueStore.Delete(context.Background(), fleet.MDMProfileProcessingKeyPrefix+":"+host.UUID)
	require.NoError(t, err)

	globalProfiles = [][]byte{
		mobileconfigForTest("N3", ident),
	}
	s.Do("POST", "/api/v1/fleet/mdm/apple/profiles/batch", batchSetMDMAppleProfilesRequest{Profiles: globalProfiles}, http.StatusNoContent)
	s.awaitTriggerProfileSchedule(t)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3)
	var profUUID string
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.EqualValues(t, fleet.MDMDeliveryPending, *hm.Status)
		if hm.Name == "N3" {
			profUUID = hm.ProfileUUID
		}
	}

	// delete the custom profile
	s.Do("DELETE", fmt.Sprintf("/api/latest/fleet/mdm/profiles/%s", profUUID), &deleteMDMAppleConfigProfileRequest{}, http.StatusOK)
	s.awaitTriggerProfileSchedule(t)

	getHostResp = getHostResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.Profiles)
	// Since Fleet doesn't know for sure whether profile was installed or not, it sends a remove command just in case.
	require.Len(t, *getHostResp.Host.MDM.Profiles, 3)
	for _, hm := range *getHostResp.Host.MDM.Profiles {
		require.EqualValues(t, fleet.MDMDeliveryPending, *hm.Status)
		if hm.Name == "N3" {
			assert.Equal(t, fleet.MDMOperationTypeRemove, hm.OperationType)
		}
	}
}
