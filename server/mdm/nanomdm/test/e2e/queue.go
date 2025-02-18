package e2e

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
)

type queueDevice interface {
	CMDDoReportAndFetch(ctx context.Context, cmd *mdm.CommandResults) (*mdm.Command, error)
	NewCommandReport(uuid, status string, errors []mdm.ErrorChain) *mdm.CommandResults
	IDer
}

// enqueue enqueues cmd to id using a.
func enqueue(t *testing.T, ctx context.Context, a NanoMDMAPI, id string, cmd *mdm.Command) {
	err := a.RawCommandEnqueue(ctx, []string{id}, cmd, true)
	if err != nil {
		t.Fatal(err)
	}
}

// simpleCmd makes a command with a CommandUUID and RequestType the same string.
func simpleCmd(cmdID string) *mdm.Command {
	return newCommand(cmdID, cmdID)
}

// sendReportExpectCommandReply send a command report and expect a certain command reply.
func sendReportExpectCommandReply(t *testing.T, ctx context.Context, d queueDevice, reportCmd, reportStatus, expectedCmd string) {
	cr := d.NewCommandReport(reportCmd, reportStatus, nil)
	cmd, err := d.CMDDoReportAndFetch(ctx, cr)
	if err != nil {
		t.Fatal(fmt.Errorf("reporting cmd=%s status=%s: %w", reportCmd, reportStatus, err))
	}

	// make sure the command we expect was received
	if have, want := cmd, simpleCmd(expectedCmd); !reflect.DeepEqual(have, want) {
		t.Errorf("command: have: %v, want: %v", have, want)
	}
}

// enqueueSimple enqueues cmd to a for d.
func enqueueSimple(t *testing.T, ctx context.Context, d queueDevice, a NanoMDMAPI, cmd string) {
	// we're assuming the UDID is all we need here.
	enqueue(t, ctx, a, d.ID(), simpleCmd(cmd))
}

func queue(t *testing.T, ctx context.Context, d queueDevice, a NanoMDMAPI) {
	t.Run("basic", func(t *testing.T) {
		// report Idle.
		// expect no command (empty queue for this id).
		sendReportExpectCommandReply(t, ctx, d, "", "Idle", "")
		// enqueue a couple commands.
		enqueueSimple(t, ctx, d, a, "CMD1")
		enqueueSimple(t, ctx, d, a, "CMD2")
		// report Idle.
		// but now expect the CMD1 result (first on the queue).
		sendReportExpectCommandReply(t, ctx, d, "", "Idle", "CMD1")
		// ack CMD1.
		// expect CMD2.
		sendReportExpectCommandReply(t, ctx, d, "CMD1", "Acknowledged", "CMD2")
		// ack CMD2 (effectively clearning the queue).
		// expect no command (only two commands queued).
		sendReportExpectCommandReply(t, ctx, d, "CMD2", "Acknowledged", "")
		// report Idle.
		// expect no command (empty queue).
		sendReportExpectCommandReply(t, ctx, d, "", "Idle", "")
	})
	t.Run("notnow", func(t *testing.T) {
		// report Idle.
		// expect no command (empty queue).
		sendReportExpectCommandReply(t, ctx, d, "", "Idle", "")
		// enqueue CMD3.
		enqueueSimple(t, ctx, d, a, "CMD3")
		// report Idle.
		// expect CMD3.
		sendReportExpectCommandReply(t, ctx, d, "", "Idle", "CMD3")
		// report NotNow for CMD3.
		// expect no command (only NotNow commands in queue).
		sendReportExpectCommandReply(t, ctx, d, "CMD3", "NotNow", "")
		// report Idle.
		// this could be considered as "resetting" NotNow for CMD3.
		// expect CMD3 (the NotNow'd command).
		sendReportExpectCommandReply(t, ctx, d, "", "Idle", "CMD3")
		// ack CMD3.
		// expect no command (empty queue).
		sendReportExpectCommandReply(t, ctx, d, "CMD3", "Acknowledged", "")
		// report Idle.
		// expect no command (empty queue).
		sendReportExpectCommandReply(t, ctx, d, "", "Idle", "")
	})

}
