package service

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/fleetdm/fleet/v4/pkg/mdm/mdmtest"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/fleetdm/fleet/v4/server/mdm/nanodep/godep"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	mdmtesting "github.com/fleetdm/fleet/v4/server/mdm/testing_utils"
	"github.com/fleetdm/fleet/v4/server/ptr"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file contains MDM integrations tests that cover device command communication.

func (s *integrationMDMTestSuite) TestLockUnlockWipeMacOS() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	host, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)

	// get the host's information
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to unlock the host (which is already its status)
	var unlockResp fleet.UnlockHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusConflict, &unlockResp)

	// lock the host
	var lockResp fleet.LockHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusOK, &lockResp, "view_pin", "true")
	assert.Len(t, lockResp.UnlockPIN, 6)
	require.Equal(t, fleet.PendingActionLock, lockResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusUnlocked, lockResp.DeviceStatus)

	// refresh the host's status, it is now pending lock
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	// we should go straight to the lock action, since we don't get host location data
	// during this flow for macOS hosts.
	require.Equal(t, "lock", *getHostResp.Host.MDM.PendingAction)

	// try locking the host while it is pending lock returns error
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity, &lockResp, "view_pin", "true")

	// simulate a successful MDM result for the lock command
	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "DeviceLock", cmd.Command.RequestType)
	_, err = mdmClient.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// refresh the host's status, it is now locked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to lock the host again
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusConflict)
	// try to wipe a locked host
	res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg := extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host cannot be wiped until it is unlocked.")

	// unlock the host
	unlockResp = fleet.UnlockHostResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusOK, &unlockResp)
	require.NotNil(t, unlockResp.HostID)
	require.Equal(t, fleet.PendingActionUnlock, unlockResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusLocked, unlockResp.DeviceStatus)
	require.Equal(t, host.ID, *unlockResp.HostID)
	require.Len(t, unlockResp.UnlockPIN, 6)
	unlockPIN := unlockResp.UnlockPIN
	unlockActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeUnlockedHost{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "host_platform": %q}`, host.ID, host.DisplayName(), host.FleetPlatform()), 0)

	// refresh the host's status, it is still locked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "locked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	assert.Empty(t, *getHostResp.Host.MDM.PendingAction)

	// try unlocking the host again simply returns the PIN again
	unlockResp = fleet.UnlockHostResponse{}
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusOK, &unlockResp)
	require.Equal(t, unlockPIN, unlockResp.UnlockPIN)
	require.Equal(t, fleet.PendingActionUnlock, unlockResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusLocked, unlockResp.DeviceStatus)
	// a new unlock host activity is created every time the unlock PIN is viewed
	newUnlockActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeUnlockedHost{}.ActivityName(),
		fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "host_platform": %q}`, host.ID, host.DisplayName(), host.FleetPlatform()), 0)
	require.NotEqual(t, unlockActID, newUnlockActID)

	// simulate passage of time: backdate unlock_ref so that CleanAppleMDMLock's
	// 5-minute guard doesn't block the upcoming Idle from clearing the lock state.
	mysqltest.ExecAdhocSQL(t, s.ds, func(q sqlx.ExtContext) error {
		_, err := q.ExecContext(context.Background(),
			fmt.Sprintf(`UPDATE host_mdm_actions hma JOIN hosts h ON hma.host_id = h.id
			SET hma.unlock_ref = DATE_FORMAT(UTC_TIMESTAMP() - INTERVAL %d MINUTE, '%%Y-%%m-%%d %%H:%%i:%%s')
			WHERE h.uuid = ?`, mysql.MDMLockCleanupMinutes+1), host.UUID)
		return err
	})

	// as soon as the host sends an Idle MDM request, it is marked as unlocked
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.Nil(t, cmd)

	// refresh the host's status, it is unlocked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// wipe the host
	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)
	wipeActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

	// try to wipe the host again, already have it pending
	res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Host has pending wipe request.")
	// no activity created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

	// refresh the host's status, it is unlocked, pending wipe
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "wipe", *getHostResp.Host.MDM.PendingAction)

	// simulate a successful MDM result for the wipe command
	cmd, err = mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "EraseDevice", cmd.Command.RequestType)
	_, err = mdmClient.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// refresh the host's status, it is wiped
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "wiped", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// try to lock/unlock the host fails
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Cannot process lock requests once host is wiped.")
	res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusUnprocessableEntity)
	errMsg = extractServerErrorText(res.Body)
	require.Contains(t, errMsg, "Cannot process unlock requests once host is wiped.")

	// try to wipe the host again, conflict (already wiped)
	s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusConflict)
	// no activity created
	s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

	// re-enroll the host, simulating that another user received the wiped host
	err = mdmClient.Enroll()
	require.NoError(t, err)

	// refresh the host's status, it is back to unlocked
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

	// lock the host without requesting the PIN
	lockResp = fleet.LockHostResponse{} // to zero out leftover fields from existing lock response
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusOK, &lockResp)
	require.Equal(t, fleet.PendingActionLock, lockResp.PendingAction)
	require.Empty(t, lockResp.UnlockPIN)
}

func (s *integrationMDMTestSuite) TestWipeMacOSCancelsUpcomingActivities() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	host, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	// orbit enrollment is required so that /scripts/run is accepted
	setOrbitEnrollment(t, host, s.ds)

	// enqueue two upcoming script-run activities
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo one"},
		http.StatusAccepted, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo two"},
		http.StatusAccepted, &runResp)

	// confirm both are in the upcoming activities list
	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 2)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[0].Type)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[1].Type)

	// wipe the host
	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)

	// host is now unlocked, pending wipe
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionWipe), *getHostResp.Host.MDM.PendingAction)

	// simulate a successful MDM result for the wipe command
	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "EraseDevice", cmd.Command.RequestType)
	_, err = mdmClient.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// host is now in the terminal "wiped" state
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusWiped), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

	// upcoming activities for this host should now be empty: a wiped device
	// will never execute them
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.Activities)
}

func (s *integrationMDMTestSuite) TestWipeMacOSUserChannelErrorKeepsUpcomingActivities() {
	t := s.T()
	s.setSkipWorkerJobs(t)
	host, mdmClient := createHostThenEnrollMDM(s.ds, s.server.URL, t)
	setOrbitEnrollment(t, host, s.ds)

	// add a user-channel enrollment so the device can respond on the user channel
	require.NoError(t, mdmClient.UserEnroll())

	// enqueue two upcoming script-run activities
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo one"},
		http.StatusAccepted, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo two"},
		http.StatusAccepted, &runResp)

	// confirm both are in the upcoming activities list
	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 2)

	// wipe the host
	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)

	// pull the queued EraseDevice command
	cmd, err := mdmClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, "EraseDevice", cmd.Command.RequestType)

	// device reports an Error for EraseDevice on the user channel — this
	// matches what a user-enrolled device does on iOS/iPadOS 18+, where the
	// command is rejected instead of wiping. The host's upcoming activities
	// must NOT be cleared because the device wasn't actually wiped.
	_, err = mdmClient.UserChannelErr(cmd.CommandUUID, []mdm.ErrorChain{{ErrorCode: 1234}})
	require.NoError(t, err)

	// upcoming activities should still be present
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 2)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[0].Type)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[1].Type)
}

func (s *integrationMDMTestSuite) TestWipeWindowsCancelsUpcomingActivities() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	host, winMDMClient := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	err := s.ds.SetOrUpdateMDMData(ctx, host.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", false)
	require.NoError(t, err)

	// enqueue two upcoming script-run activities
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo one"},
		http.StatusAccepted, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo two"},
		http.StatusAccepted, &runResp)

	// confirm both are in the upcoming activities list
	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 2)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[0].Type)
	require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), listResp.Activities[1].Type)

	// wipe the host
	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)

	// host is unlocked, pending wipe
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionWipe), *getHostResp.Host.MDM.PendingAction)

	// simulate a successful wipe from the Windows device's MDM response
	status, err := s.ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	cmds, err := winMDMClient.StartManagementSession()
	require.NoError(t, err)
	// two status + the wipe command we enqueued
	require.Len(t, cmds, 3)
	wipeCmd := cmds[status.WipeMDMCommand.CommandUUID]
	require.NotNil(t, wipeCmd)
	require.Equal(t, fleet.CmdExec, wipeCmd.Verb)
	require.Len(t, wipeCmd.Cmd.Items, 1)
	require.Equal(t, "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected", *wipeCmd.Cmd.Items[0].Target)

	msgID, err := winMDMClient.GetCurrentMsgID()
	require.NoError(t, err)
	winMDMClient.AppendResponse(fleet.SyncMLCmd{
		XMLName: xml.Name{Local: fleet.CmdStatus},
		MsgRef:  &msgID,
		CmdRef:  &status.WipeMDMCommand.CommandUUID,
		Cmd:     ptr.String("Exec"),
		Data:    ptr.String("200"),
		Items:   nil,
		CmdID:   fleet.CmdID{Value: uuid.NewString()},
	})
	cmds, err = winMDMClient.SendResponse()
	require.NoError(t, err)
	// the ack of the message should be the only returned command
	require.Len(t, cmds, 1)

	// host is now in the terminal "wiped" state
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusWiped), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

	// upcoming activities for this host should now be empty: a wiped device
	// will never execute them
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.Activities)
}

func (s *integrationMDMTestSuite) TestWipeLinuxCancelsUpcomingActivities() {
	t := s.T()
	ctx := context.Background()
	s.setSkipWorkerJobs(t)

	host := createOrbitEnrolledHost(t, "linux", "wipe_cancels_upcoming_linux", s.ds)

	// wipe the host first: on Linux the wipe is itself a script in the
	// unified queue, so we want it activated (no pending activities ahead
	// of it) and ready to accept its result
	var wipeResp fleet.WipeHostResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
	require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
	require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)

	// host is unlocked, pending wipe
	var getHostResp getHostResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionWipe), *getHostResp.Host.MDM.PendingAction)

	// now enqueue two more upcoming script-run activities behind the
	// in-flight wipe
	var runResp fleet.RunScriptResponse
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo one"},
		http.StatusAccepted, &runResp)
	s.DoJSON("POST", "/api/latest/fleet/scripts/run",
		fleet.HostScriptRequestPayload{HostID: host.ID, ScriptContents: "echo two"},
		http.StatusAccepted, &runResp)

	// upcoming list contains 3 script activities: the in-flight wipe and
	// the two pending user scripts queued behind it
	var listResp listHostUpcomingActivitiesResponse
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Len(t, listResp.Activities, 3)
	for _, a := range listResp.Activities {
		require.Equal(t, fleet.ActivityTypeRanScript{}.ActivityName(), a.Type)
	}

	// simulate a successful wipe via the orbit script result
	status, err := s.ds.GetHostLockWipeStatus(ctx, host)
	require.NoError(t, err)
	var orbitScriptResp fleet.OrbitPostScriptResultResponse
	s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
		json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, status.WipeScript.ExecutionID)),
		http.StatusOK, &orbitScriptResp)

	// host is now in the terminal "wiped" state
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
	require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
	require.Equal(t, string(fleet.DeviceStatusWiped), *getHostResp.Host.MDM.DeviceStatus)
	require.NotNil(t, getHostResp.Host.MDM.PendingAction)
	require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

	// upcoming activities for this host should now be empty: a wiped device
	// will never execute them
	listResp = listHostUpcomingActivitiesResponse{}
	s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d/activities/upcoming", host.ID),
		nil, http.StatusOK, &listResp)
	require.Empty(t, listResp.Activities)
}

func (s *integrationMDMTestSuite) TestLockUnlockWipeIOSIpadOS() {
	t := s.T()

	devices := []godep.Device{
		{SerialNumber: mdmtest.RandSerialNumber(), Model: "iPhone 16 Pro", OS: "ios", DeviceFamily: "iPhone", OpType: "added"},
		{SerialNumber: mdmtest.RandSerialNumber(), Model: "iPad", OS: "ipados", OpType: "added"},
	}

	s.enableABM(t.Name())
	abmTok, err := s.ds.GetABMTokenByOrgName(t.Context(), t.Name())
	require.NoError(t, err)
	s.mockDEPResponse(t.Name(), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		encoder := json.NewEncoder(w)
		switch r.URL.Path {
		case "/session":
			err := encoder.Encode(map[string]string{"auth_session_token": "xyz"})
			require.NoError(t, err)
		case "/profile":
			err := encoder.Encode(godep.ProfileResponse{ProfileUUID: uuid.New().String()})
			require.NoError(t, err)
		case "/server/devices":
			// This endpoint  is used to get an initial list of
			// devices, return a single device
			err := encoder.Encode(godep.DeviceResponse{Devices: devices})
			require.NoError(t, err)
		case "/devices/sync":
			// This endpoint is polled over time to sync devices from
			// ABM, send a repeated serial and a new one
			err := encoder.Encode(godep.DeviceResponse{Devices: devices, Cursor: "foo"})
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
	s.setSkipWorkerJobs(t)

	// ensure fleet profiles
	s.awaitTriggerProfileSchedule(t)

	iosHost, iosMDMClient := s.createAppleMobileHostThenDEPEnrollMDM("ios", devices[0].SerialNumber)
	iPadOSHost, iPadOSMDMClient := s.createAppleMobileHostThenDEPEnrollMDM("ipados", devices[1].SerialNumber)

	s.awaitRunAppleMDMWorkerSchedule()

	// empty the command queue for both hosts
	cmd, err := iosMDMClient.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = iosMDMClient.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}
	cmd, err = iPadOSMDMClient.Idle()
	require.NoError(t, err)
	for cmd != nil {
		cmd, err = iPadOSMDMClient.Acknowledge(cmd.CommandUUID)
		require.NoError(t, err)
	}

	// We fake set installed_from_dep to emulate the devices was enrolled with DEP.
	require.NoError(t, s.ds.SetOrUpdateMDMData(t.Context(), iosHost.ID, false, true, s.server.URL, true, t.Name(), "", false))
	require.NoError(t, s.ds.SetOrUpdateMDMData(t.Context(), iPadOSHost.ID, false, true, s.server.URL, true, t.Name(), "", false))
	s.Require().NoError(s.ds.UpsertMDMAppleHostDEPAssignments(t.Context(), []fleet.Host{*iosHost, *iPadOSHost}, abmTok.ID, nil))

	for _, tc := range []struct {
		name      string
		host      *fleet.Host
		mdmClient *mdmtest.TestAppleMDMClient
	}{
		{"iOS", iosHost, iosMDMClient},
		{"iPadOS", iPadOSHost, iPadOSMDMClient},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// get the host's information
			var getHostResp getHostResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// try to unlock the host (which is already its status)
			var unlockResp fleet.UnlockHostResponse
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", tc.host.ID), nil, http.StatusConflict, &unlockResp)

			// lock the host
			var lockResp fleet.LockHostResponse
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", tc.host.ID), nil, http.StatusOK, &lockResp)
			assert.Empty(t, lockResp.UnlockPIN)
			require.Equal(t, fleet.PendingActionLock, lockResp.PendingAction)
			require.Equal(t, fleet.DeviceStatusUnlocked, lockResp.DeviceStatus)

			// refresh the host's status, it is now pending lock
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionLocation), *getHostResp.Host.MDM.PendingAction)

			// try locking the host while it is pending lock returns error
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", tc.host.ID), nil, http.StatusUnprocessableEntity, &lockResp)

			// simulate a successful MDM result for the lock command
			cmd, err := tc.mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd)
			require.Equal(t, "EnableLostMode", cmd.Command.RequestType)
			_, err = tc.mdmClient.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)

			cmd, err = tc.mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd)
			require.Equal(t, "DeviceLocation", cmd.Command.RequestType)
			expectedLat, expectedLong := 42.42, 26.26
			_, err = tc.mdmClient.AcknowledgeDeviceLocation(getHostResp.Host.UUID, cmd.CommandUUID, expectedLat, expectedLong)
			require.NoError(t, err)

			// Run device location handler
			s.awaitRunAppleMDMWorkerSchedule()

			// refresh the host's status, it is now locked
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusLocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)
			// Fleet should have the device's location data now
			s.Assert().NotNil(getHostResp.Host.Geolocation)
			s.Assert().Equal([]float64{expectedLat, expectedLong}, getHostResp.Host.Geolocation.Geometry.Coordinates)

			// try to lock the host again
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", tc.host.ID), nil, http.StatusConflict)
			// try to wipe a locked host
			res := s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", tc.host.ID), nil, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host cannot be wiped until it is unlocked.")

			// Refetch the host, should update the location data
			_ = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/refetch", tc.host.ID), nil, http.StatusOK)

			testCerts := []*x509.Certificate{mdmtesting.NewTestMDMAppleCertTemplate()}
			var hostResp getHostResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &hostResp)
			assert.Equal(t, tc.host.ID, hostResp.Host.ID)
			assert.True(t, hostResp.Host.RefetchRequested)

			// Check the MDM commands and send response
			cmd, err = tc.mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd)

			expectedSoftware := []fleet.HostSoftwareEntry{
				{
					Software: fleet.Software{
						BundleIdentifier: "com.evernote.iPhone.Evernote",
						Name:             "Evernote",
						Version:          "10.98.0",
						Source:           "ios_apps",
					},
				},
			}
			require.Equal(t, "InstalledApplicationList", cmd.Command.RequestType)
			cmd, err = tc.mdmClient.AcknowledgeInstalledApplicationList(tc.mdmClient.UUID, cmd.CommandUUID,
				[]fleet.Software{expectedSoftware[0].Software})
			require.NoError(t, err)
			require.Equal(t, "CertificateList", cmd.Command.RequestType)
			cmd, err = tc.mdmClient.AcknowledgeCertificateList(tc.mdmClient.UUID, cmd.CommandUUID, testCerts)
			require.NoError(t, err)
			require.Equal(t, "DeviceInformation", cmd.Command.RequestType)
			_, err = tc.mdmClient.AcknowledgeDeviceInformation(tc.mdmClient.UUID, cmd.CommandUUID, tc.host.DisplayName(), "", "America/Los_Angeles")
			require.NoError(t, err)

			cmd, err = tc.mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd)
			require.Equal(t, "DeviceLocation", cmd.Command.RequestType)
			expectedLat, expectedLong = 10.10, 45.45
			_, err = tc.mdmClient.AcknowledgeDeviceLocation(getHostResp.Host.UUID, cmd.CommandUUID, expectedLat, expectedLong)
			require.NoError(t, err)

			// Run device location handler
			s.awaitRunAppleMDMWorkerSchedule()

			// Get host data
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusLocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)
			// Fleet should have the updated location data now
			s.Assert().NotNil(getHostResp.Host.Geolocation)
			s.Assert().Equal([]float64{expectedLat, expectedLong}, getHostResp.Host.Geolocation.Geometry.Coordinates)

			// unlock the host
			unlockResp = fleet.UnlockHostResponse{}
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", tc.host.ID), nil, http.StatusOK, &unlockResp)
			require.NotNil(t, unlockResp.HostID)
			require.Equal(t, fleet.PendingActionUnlock, unlockResp.PendingAction)
			require.Equal(t, fleet.DeviceStatusLocked, unlockResp.DeviceStatus)
			require.Equal(t, tc.host.ID, *unlockResp.HostID)
			require.Empty(t, unlockResp.UnlockPIN)
			s.lastActivityOfTypeMatches(fleet.ActivityTypeUnlockedHost{}.ActivityName(),
				fmt.Sprintf(`{"host_id": %d, "host_display_name": %q, "host_platform": %q}`, tc.host.ID, tc.host.DisplayName(), tc.host.FleetPlatform()), 0)

			// refresh the host's status, it is still locked and pending unlock
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusLocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionUnlock), *getHostResp.Host.MDM.PendingAction)

			// try unlocking the host again errors
			unlockResp = fleet.UnlockHostResponse{}
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", tc.host.ID), nil, http.StatusUnprocessableEntity, &unlockResp)

			// send idle to simulate the host checking in, and see DisableLostMode is sent.
			cmd, err = tc.mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd)
			require.Equal(t, "DisableLostMode", cmd.Command.RequestType)
			_, err = tc.mdmClient.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)

			// refresh the host's status, it is now unlocked
			getHostResp = getHostResponse{}
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// Host location data should have been deleted
			s.Assert().Nil(getHostResp.Host.Geolocation)

			// wipe the host
			var wipeResp fleet.WipeHostResponse
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", tc.host.ID), nil, http.StatusOK, &wipeResp)
			require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
			require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)
			wipeActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, tc.host.ID, tc.host.DisplayName()), 0)

			// try to wipe the host again, already have it pending
			res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", tc.host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host has pending wipe request.")
			// no activity created
			s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, tc.host.ID, tc.host.DisplayName()), wipeActID)

			// refresh the host's status, it is unlocked, pending wipe
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "wipe", *getHostResp.Host.MDM.PendingAction)

			// simulate a successful MDM result for the wipe command
			cmd, err = tc.mdmClient.Idle()
			require.NoError(t, err)
			require.NotNil(t, cmd)
			require.Equal(t, "EraseDevice", cmd.Command.RequestType)
			_, err = tc.mdmClient.Acknowledge(cmd.CommandUUID)
			require.NoError(t, err)

			// refresh the host's status, it is wiped
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "wiped", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// try to lock/unlock the host fails
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", tc.host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Cannot process lock requests once host is wiped.")
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", tc.host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Cannot process unlock requests once host is wiped.")

			// try to wipe the host again, conflict (already wiped)
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", tc.host.ID), nil, http.StatusConflict)
			// no activity created
			s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, tc.host.ID, tc.host.DisplayName()), wipeActID)

			// re-enroll the host, simulating that another user received the wiped host
			err = tc.mdmClient.Enroll()
			require.NoError(t, err)

			// refresh the host's status, it is back to unlocked
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// lock the host without requesting the PIN
			lockResp = fleet.LockHostResponse{} // to zero out leftover fields from existing lock response
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", tc.host.ID), nil, http.StatusOK, &lockResp)
			require.Equal(t, fleet.PendingActionLock, lockResp.PendingAction)
			require.Empty(t, lockResp.UnlockPIN)
		})
	}

	iosHost, iosMDMClient = s.createAppleMobileHostThenDEPEnrollMDM("ios", mdmtest.RandSerialNumber())
	iPadOSHost, iPadOSMDMClient = s.createAppleMobileHostThenDEPEnrollMDM("ipados", mdmtest.RandSerialNumber())

	for _, tc := range []struct {
		name      string
		host      *fleet.Host
		mdmClient *mdmtest.TestAppleMDMClient
	}{
		{"iOS can't lock manually enrolled host", iosHost, iosMDMClient},
		{"iPadOS can't lock manually enrolled host", iPadOSHost, iPadOSMDMClient},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// get the host's information
			var getHostResp getHostResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", tc.host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// lock the host
			res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", tc.host.ID), nil, http.StatusBadRequest)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Couldn't lock. This command isn't available for manually enrolled iOS/iPadOS hosts.")
		})
	}
}

func (s *integrationMDMTestSuite) TestLockUnlockWipeWindowsLinux() {
	t := s.T()
	ctx := context.Background()

	// create an MDM-enrolled Windows host
	winHost, winMDMClient := createWindowsHostThenEnrollMDM(s.ds, s.server.URL, t)
	// set its MDM data so it shows as MDM-enrolled in the backend
	err := s.ds.SetOrUpdateMDMData(ctx, winHost.ID, false, true, s.server.URL, false, fleet.WellKnownMDMFleet, "", false)
	require.NoError(t, err)
	linuxHost := createOrbitEnrolledHost(t, "linux", "lock_unlock_linux", s.ds)

	for _, host := range []*fleet.Host{winHost, linuxHost} {
		t.Run(host.FleetPlatform(), func(t *testing.T) {
			// get the host's information
			var getHostResp getHostResponse
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, "unlocked", *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, "", *getHostResp.Host.MDM.PendingAction)

			// try to unlock the host (which is already its status)
			var unlockResp fleet.UnlockHostResponse
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusConflict, &unlockResp)

			// lock the host
			var lockHostResp fleet.LockHostResponse
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusOK, &lockHostResp)
			require.Equal(t, fleet.PendingActionLock, lockHostResp.PendingAction)
			require.Equal(t, fleet.DeviceStatusUnlocked, lockHostResp.DeviceStatus)

			// refresh the host's status, it is now pending lock
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionLock), *getHostResp.Host.MDM.PendingAction)

			// try locking the host while it is pending lock fails for Windows/Linux
			res := s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg := extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host has pending lock request.")

			// simulate a successful script result for the lock command
			status, err := s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)

			var orbitScriptResp fleet.OrbitPostScriptResultResponse
			s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
				json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, status.LockScript.ExecutionID)),
				http.StatusOK, &orbitScriptResp)

			// refresh the host's status, it is now locked
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusLocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

			// try to lock the host again
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusConflict)
			// try to wipe a locked host
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host cannot be wiped until it is unlocked.")

			// unlock the host
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusOK, &unlockResp)
			require.Equal(t, fleet.PendingActionUnlock, unlockResp.PendingAction)
			require.Equal(t, fleet.DeviceStatusLocked, unlockResp.DeviceStatus)

			// refresh the host's status, it is locked pending unlock
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusLocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionUnlock), *getHostResp.Host.MDM.PendingAction)

			// try unlocking the host while it is pending unlock fails
			res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host has pending unlock request.")

			// simulate a failed script result for the unlock command
			status, err = s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)

			s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
				json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": -1, "output": "fail"}`, *host.OrbitNodeKey, status.UnlockScript.ExecutionID)),
				http.StatusOK, &orbitScriptResp)

			// refresh the host's status, it is still locked, no pending action
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusLocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

			// unlock the host, simulate success
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusOK)
			status, err = s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)
			s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
				json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, status.UnlockScript.ExecutionID)),
				http.StatusOK, &orbitScriptResp)

			// refresh the host's status, it is unlocked, no pending action
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

			// wipe the host
			var wipeResp fleet.WipeHostResponse
			s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusOK, &wipeResp)
			require.Equal(t, fleet.PendingActionWipe, wipeResp.PendingAction)
			require.Equal(t, fleet.DeviceStatusUnlocked, wipeResp.DeviceStatus)
			wipeActID := s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), 0)

			// try to wipe the host again, already have it pending
			res = s.DoRaw("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Host has pending wipe request.")
			// no activity created
			s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

			// refresh the host's status, it is unlocked, pending wipe
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionWipe), *getHostResp.Host.MDM.PendingAction)

			status, err = s.ds.GetHostLockWipeStatus(ctx, host)
			require.NoError(t, err)
			if host.FleetPlatform() == "linux" {
				// simulate a successful wipe for the Linux host's script response
				s.DoJSON("POST", "/api/fleet/orbit/scripts/result",
					json.RawMessage(fmt.Sprintf(`{"orbit_node_key": %q, "execution_id": %q, "exit_code": 0, "output": "ok"}`, *host.OrbitNodeKey, status.WipeScript.ExecutionID)),
					http.StatusOK, &orbitScriptResp)
			} else {
				// simulate a successful wipe from the Windows device's MDM response
				cmds, err := winMDMClient.StartManagementSession()
				require.NoError(t, err)

				// two status + the wipe command we enqueued
				require.Len(t, cmds, 3)
				wipeCmd := cmds[status.WipeMDMCommand.CommandUUID]
				require.NotNil(t, wipeCmd)
				require.Equal(t, wipeCmd.Verb, fleet.CmdExec)
				require.Len(t, wipeCmd.Cmd.Items, 1)
				require.EqualValues(t, "./Device/Vendor/MSFT/RemoteWipe/doWipeProtected", *wipeCmd.Cmd.Items[0].Target)

				msgID, err := winMDMClient.GetCurrentMsgID()
				require.NoError(t, err)

				winMDMClient.AppendResponse(fleet.SyncMLCmd{
					XMLName: xml.Name{Local: fleet.CmdStatus},
					MsgRef:  &msgID,
					CmdRef:  &status.WipeMDMCommand.CommandUUID,
					Cmd:     ptr.String("Exec"),
					Data:    ptr.String("200"),
					Items:   nil,
					CmdID:   fleet.CmdID{Value: uuid.NewString()},
				})
				cmds, err = winMDMClient.SendResponse()
				require.NoError(t, err)
				// the ack of the message should be the only returned command
				require.Len(t, cmds, 1)
			}

			// refresh the host's status, it is wiped
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusWiped), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)

			// try to lock/unlock the host fails
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/lock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Cannot process lock requests once host is wiped.")
			res = s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/unlock", host.ID), nil, http.StatusUnprocessableEntity)
			errMsg = extractServerErrorText(res.Body)
			require.Contains(t, errMsg, "Cannot process unlock requests once host is wiped.")

			// try to wipe the host again, conflict (already wiped)
			s.Do("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/wipe", host.ID), nil, http.StatusConflict)
			// no activity created
			s.lastActivityOfTypeMatches(fleet.ActivityTypeWipedHost{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, host.ID, host.DisplayName()), wipeActID)

			// re-enroll the host, simulating that another user received the wiped host
			newOrbitKey := uuid.New().String()
			newHost, err := s.ds.EnrollOrbit(ctx,
				fleet.WithEnrollOrbitMDMEnabled(true),
				fleet.WithEnrollOrbitHostInfo(fleet.OrbitHostInfo{
					HardwareUUID:   *host.OsqueryHostID,
					HardwareSerial: host.HardwareSerial,
				}),
				fleet.WithEnrollOrbitNodeKey(newOrbitKey),
			)
			require.NoError(t, err)
			// it re-enrolled using the same host record
			require.Equal(t, host.ID, newHost.ID)

			// refresh the host's status, it is back to unlocked
			s.DoJSON("GET", fmt.Sprintf("/api/latest/fleet/hosts/%d", host.ID), nil, http.StatusOK, &getHostResp)
			require.NotNil(t, getHostResp.Host.MDM.DeviceStatus)
			require.Equal(t, string(fleet.DeviceStatusUnlocked), *getHostResp.Host.MDM.DeviceStatus)
			require.NotNil(t, getHostResp.Host.MDM.PendingAction)
			require.Equal(t, string(fleet.PendingActionNone), *getHostResp.Host.MDM.PendingAction)
		})
	}
}

func (s *integrationMDMTestSuite) TestClearPasscodeCommand() {
	t := s.T()

	s.enableABM(t.Name())

	// Create iOS host and enroll in MDM
	iosHost, iosMDMClient := s.createAppleMobileHostThenDEPEnrollMDM("ios", mdmtest.RandSerialNumber())

	// Trigger ClearPasscode endpoint
	var clearPasscodeResp clearPasscodeResponse
	s.DoJSON("POST", fmt.Sprintf("/api/latest/fleet/hosts/%d/clear_passcode", iosHost.ID), nil, http.StatusOK, &clearPasscodeResp)
	require.Equal(t, fleet.AppleMDMCommandTypeClearPasscode, clearPasscodeResp.RequestType)
	require.Equal(t, "ios", clearPasscodeResp.Platform)

	// Check host and global activity
	s.lastHostActivityMatches(iosHost.ID, fleet.ActivityTypeClearedPasscode{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, iosHost.ID, iosHost.DisplayName()), 0)
	s.lastActivityMatches(fleet.ActivityTypeClearedPasscode{}.ActivityName(), fmt.Sprintf(`{"host_id": %d, "host_display_name": %q}`, iosHost.ID, iosHost.DisplayName()), 0)

	// Check in with the iOS device to receive the ClearPasscode command
	cmd, err := iosMDMClient.Idle()
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Equal(t, fleet.AppleMDMCommandTypeClearPasscode, cmd.Command.RequestType)
	b64Encoded := base64.StdEncoding.EncodeToString([]byte("unlocktoken" + iosMDMClient.SerialNumber))
	require.Contains(t, string(cmd.Raw), b64Encoded)

	// Acknowledge the ClearPasscode command
	_, err = iosMDMClient.Acknowledge(cmd.CommandUUID)
	require.NoError(t, err)

	// Fetch the command result and check the response is acknowledged (+ Payload has the expected unlock token value)
	commandResultResp := &getMDMCommandResultsResponse{}
	s.DoJSON("GET", "/api/latest/fleet/commands/results", &getMDMCommandResultsRequest{
		CommandUUID: clearPasscodeResp.CommandUUID,
	}, http.StatusOK, commandResultResp)
	require.Len(t, commandResultResp.Results, 1)
	require.Equal(t, fleet.AppleMDMCommandTypeClearPasscode, commandResultResp.Results[0].RequestType)
	require.NotNil(t, commandResultResp.Results[0].Result)
}
