package service

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/datastore/mysql/mysqltest"
	"github.com/fleetdm/fleet/v4/server/fleet"
	microsoft_mdm "github.com/fleetdm/fleet/v4/server/mdm/microsoft"
	"github.com/fleetdm/fleet/v4/server/platform/mysql/testing_utils"
	"github.com/fleetdm/fleet/v4/server/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

// The dummy replica copies these tables only when the test replicates, so in between it lags the primary.
var windowsCommandQueueTables = []string{
	"windows_mdm_commands", "windows_mdm_command_queue", "windows_mdm_command_results", "mdm_windows_enrollments",
}

// A message that changes the device's command queue must get a reply built from the primary; one that changes
// nothing keeps reading the replica.
func TestWindowsMDMReplyReplicaLag(t *testing.T) {
	opts := &testing_utils.DatastoreTestOptions{DummyReplica: true}
	ds := mysqltest.CreateMySQLDSWithOptions(t, opts)
	t.Cleanup(func() { ds.Close() })
	replicate := func() { opts.RunReplication(windowsCommandQueueTables...) }

	fleetSvc, _ := newTestService(t, ds, nil, nil)
	svc := fleetSvc.(validationMiddleware).Service.(*Service)

	enrol := func(name string) *fleet.MDMWindowsEnrolledDevice {
		host := test.NewHost(t, ds, name, "1.2.3.4", name+"-key", name+"-uuid", time.Now(), test.WithPlatform("windows"))
		d := &fleet.MDMWindowsEnrolledDevice{
			MDMDeviceID:           uuid.NewString(),
			MDMHardwareID:         uuid.NewString() + uuid.NewString(),
			MDMDeviceState:        microsoft_mdm.MDMDeviceStateEnrolled,
			MDMDeviceType:         "CIMClient_Windows",
			MDMEnrollType:         "ProgrammaticEnrollment",
			MDMEnrollProtoVersion: "5.0",
			HostUUID:              host.UUID,
		}
		require.NoError(t, ds.MDMWindowsInsertEnrolledDevice(t.Context(), d))
		d, err := ds.MDMWindowsGetEnrolledDeviceWithDeviceID(ctxdb.RequirePrimary(t.Context(), true), d.MDMDeviceID)
		require.NoError(t, err)
		return d
	}
	dev := enrol("replica-lag-host")
	other := enrol("replica-lag-other-host")
	opts.RunReplication(append([]string{"hosts"}, windowsCommandQueueTables...)...)

	msgID := 0
	deviceMessageFor := func(d *fleet.MDMWindowsEnrolledDevice, sessionStart bool, statuses ...string) *fleet.SyncML {
		msgID++
		var body strings.Builder
		if sessionStart {
			body.WriteString(`<Alert><CmdID>1</CmdID><Data>1201</Data></Alert>`)
		}
		for i, s := range statuses {
			body.WriteString(fmt.Sprintf(`<Status><CmdID>%d</CmdID><MsgRef>%d</MsgRef>%s</Status>`, i+2, msgID-1, s))
		}
		raw := fmt.Sprintf(`<SyncML xmlns="SYNCML:SYNCML1.2"><SyncHdr><VerDTD>1.2</VerDTD><VerProto>DM/1.2</VerProto>`+
			`<SessionID>1</SessionID><MsgID>%d</MsgID><Target><LocURI>https://fleet.example.com</LocURI></Target>`+
			`<Source><LocURI>%s</LocURI></Source></SyncHdr><SyncBody>%s<Final/></SyncBody></SyncML>`,
			msgID, d.MDMDeviceID, body.String())
		var msg *fleet.SyncML
		require.NoError(t, xml.Unmarshal([]byte(raw), &msg))
		msg.Raw = []byte(raw)
		return msg
	}
	deviceMessage := func(sessionStart bool, statuses ...string) *fleet.SyncML {
		return deviceMessageFor(dev, sessionStart, statuses...)
	}
	status := func(cmdUUID, verb, code string) string {
		return fmt.Sprintf(`<CmdRef>%s</CmdRef><Cmd>%s</Cmd><Data>%s</Data>`, cmdUUID, verb, code)
	}

	// reply returns the XML the device would receive for an authenticated message.
	replyTo := func(t *testing.T, d *fleet.MDMWindowsEnrolledDevice, msg *fleet.SyncML) string {
		t.Helper()
		res, err := svc.getManagementResponse(t.Context(), msg, d, RequestAuthStateTrusted)
		require.NoError(t, err)
		out, err := xml.Marshal(res)
		require.NoError(t, err)
		t.Logf("message %d, Fleet's reply:\n%s", msgID, out)
		return string(out)
	}
	reply := func(t *testing.T, msg *fleet.SyncML) string {
		t.Helper()
		return replyTo(t, dev, msg)
	}

	enqueueFor := func(t *testing.T, devices []*fleet.MDMWindowsEnrolledDevice, verb, locURI, data string) string {
		t.Helper()
		cmdUUID := uuid.NewString()
		raw := fmt.Sprintf(`<%[1]s><CmdID>%[2]s</CmdID><Item><Target><LocURI>%[3]s</LocURI></Target>%[4]s</Item></%[1]s>`,
			verb, cmdUUID, locURI, data)
		deviceIDs := make([]string, 0, len(devices))
		for _, d := range devices {
			deviceIDs = append(deviceIDs, d.MDMDeviceID)
		}
		require.NoError(t, ds.MDMWindowsInsertCommandForHosts(t.Context(), deviceIDs, &fleet.MDMWindowsCommand{
			CommandUUID: cmdUUID, RawCommand: []byte(raw), TargetLocURI: locURI,
		}))
		return cmdUUID
	}
	enqueue := func(t *testing.T, verb, locURI, data string) string {
		t.Helper()
		return enqueueFor(t, []*fleet.MDMWindowsEnrolledDevice{dev}, verb, locURI, data)
	}

	t.Run("a message that changes nothing still reads the replica", func(t *testing.T) {
		// The command exists only on the primary until replicate runs. With nothing pending on the replica, a message
		// with no answers stays off the primary and does not see the command yet.
		execUUID := enqueue(t, fleet.CmdExec, "./Device/Vendor/MSFT/Reboot/RebootNow", "")
		require.NotContains(t, reply(t, deviceMessage(true)), execUUID, "an idle check-in must keep reading the replica")

		replicate()
		require.Contains(t, reply(t, deviceMessage(true)), execUUID)
		reply(t, deviceMessage(false, status(execUUID, fleet.CmdExec, "200")))
		replicate()
	})

	t.Run("an acknowledged command is not sent again", func(t *testing.T) {
		execUUID := enqueue(t, fleet.CmdExec, "./Device/Vendor/MSFT/Reboot/RebootNow", "")
		replicate()

		// Message 1 opens the session: the Exec is delivered.
		require.Contains(t, reply(t, deviceMessage(true)), execUUID)

		// Message 2 says the Exec ran (200). Fleet records that on the primary; the replica has not caught up.
		out := reply(t, deviceMessage(false, status(execUUID, fleet.CmdExec, "200")))
		require.NotContains(t, out, "<Exec>", "the device acknowledged the Exec in this message; it must not be sent again")
		replicate()
	})

	t.Run("an acknowledged command is not sent in the next session", func(t *testing.T) {
		execUUID := enqueue(t, fleet.CmdExec, "./Device/Vendor/MSFT/Reboot/RebootNow", "")
		replicate()
		require.Contains(t, reply(t, deviceMessage(true)), execUUID)
		require.NotContains(t, reply(t, deviceMessage(false, status(execUUID, fleet.CmdExec, "200"))), "<Exec>")

		// Windows opens its next session within a second, before the replica shows the acknowledgement.
		require.NotContains(t, reply(t, deviceMessage(true)), "<Exec>",
			"the device acknowledged the Exec in its previous session; it must not be sent again")
		replicate()
	})

	t.Run("one host's acknowledgement does not affect another host", func(t *testing.T) {
		// The same command goes to both hosts; each has its own queue row.
		execUUID := enqueueFor(t, []*fleet.MDMWindowsEnrolledDevice{dev, other}, fleet.CmdExec,
			"./Device/Vendor/MSFT/Reboot/RebootNow", "")
		replicate()
		require.Contains(t, reply(t, deviceMessage(true)), execUUID)
		require.NotContains(t, reply(t, deviceMessage(false, status(execUUID, fleet.CmdExec, "200"))), "<Exec>")

		// Before the replica catches up: the host that acknowledged it doesn't get it again, the other host still does.
		require.NotContains(t, reply(t, deviceMessage(true)), "<Exec>")
		require.Contains(t, replyTo(t, other, deviceMessageFor(other, true)), execUUID,
			"the other host has not acknowledged the command, so it must still be sent")
		replyTo(t, other, deviceMessageFor(other, false, status(execUUID, fleet.CmdExec, "200")))
		replicate()
	})

	t.Run("an Add answered with 418 goes out once, as a Replace", func(t *testing.T) {
		addUUID := enqueue(t, fleet.CmdAdd, "./Device/Vendor/MSFT/Example/Node", "<Data>1</Data>")
		replicate()
		require.Contains(t, reply(t, deviceMessage(true)), addUUID)

		// The node already exists (418): Fleet deletes the Add's queue row and queues the same item as a Replace.
		out := reply(t, deviceMessage(false, status(addUUID, fleet.CmdAdd, "418")))
		require.NotContains(t, out, "<Add>", "the Add was answered and dropped from the queue; it must not be sent again")
		require.Contains(t, out, "<Replace>", "the Replace queued for the 418 must go out in this same reply")
		require.Contains(t, out, "./Device/Vendor/MSFT/Example/Node")
		replicate()
	})

	t.Run("a poll schedule change goes out in the same reply", func(t *testing.T) {
		// A device whose fleetd can be woken on demand gets its poll relaxed: the Replace is enqueued just before
		// the queue is read, so that it ships in this session.
		capable := *dev
		capable.FleetdSyncCapable = true
		res, err := svc.getManagementResponse(t.Context(), deviceMessage(true), &capable, RequestAuthStateTrusted)
		require.NoError(t, err)
		out, err := xml.Marshal(res)
		require.NoError(t, err)
		t.Logf("message %d, Fleet's reply:\n%s", msgID, out)
		require.Contains(t, string(out), "DMClient/Provider/", "the poll schedule Replace must ship in the session that enqueued it")
		replicate()
	})
}
