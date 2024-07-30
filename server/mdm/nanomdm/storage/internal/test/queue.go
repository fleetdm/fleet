package test

import (
	"context"
	"testing"

	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/mdm"
	"github.com/fleetdm/fleet/v4/server/mdm/nanomdm/storage"
	"github.com/groob/plist"
)

// QueueInterfaces are the storage interfaces needed for testing queue operations.
type QueueInterfaces interface {
	storage.CommandEnqueuer
	storage.CommandAndReportResultsStore
}

// newCommand assembles a fake command including the plist raw value
func newCommand(cmd string) (*mdm.Command, error) {
	// assemble a fake struct just for marshalling to plist
	fCmd := &struct {
		CommandUUID string
		Command     struct {
			RequestType string
		}
	}{
		CommandUUID: cmd,
		Command:     struct{ RequestType string }{cmd},
	}
	// marshal it to plist
	rawBytes, err := plist.Marshal(fCmd)
	if err != nil {
		return nil, err
	}
	// return a real *mdm.Command which includes the marshalled JSON
	return &mdm.Command{
		CommandUUID: fCmd.CommandUUID,
		Command:     fCmd.Command,
		Raw:         rawBytes,
	}, nil
}

// enqueue queues a new command
func enqueue(t *testing.T, q QueueInterfaces, ctx context.Context, id, cmdStr string) {
	cmd, err := newCommand(cmdStr)
	if err != nil {
		t.Fatal(err)
	}
	res, err := q.EnqueueCommand(ctx, []string{id}, cmd)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range res {
		t.Fatalf("enqueuing to ID %s: %v", k, v)
	}
}

// compareCommand compares makes sure cmd looks similar to newCommand(cmdStr)
func compareCommand(t *testing.T, cmdStr string, cmd *mdm.Command) {
	if cmdStr != "" && cmd == nil {
		t.Errorf("expected next command, but got empty response. wanted: %q", cmdStr)
		return
	}
	if cmdStr == "" && cmd != nil {
		t.Errorf("expected empty next command, but got: %q", cmd.CommandUUID)
	}
	if cmd == nil {
		return
	}
	if cmd.CommandUUID != cmdStr {
		t.Errorf("mismatched command UUID. want: %q, have: %q", cmdStr, cmd.CommandUUID)
	}
	if cmd.Command.RequestType != cmdStr {
		t.Errorf("mismatched command RequestType. want: %q, have: %q", cmdStr, cmd.Command.RequestType)
	}
}

// retrieve retrieves the next command from the backend
func retrieve(t *testing.T, q QueueInterfaces, r *mdm.Request, cmdStr string, skipNotNow bool) {
	retCmd, err := q.RetrieveNextCommand(r, skipNotNow)
	if err != nil {
		t.Fatal(err)
	}
	compareCommand(t, cmdStr, retCmd)
}

// report fakes a command result and reports it to the backend
func report(t *testing.T, q QueueInterfaces, r *mdm.Request, cmdStr, status string) {
	fReport := &struct {
		CommandUUID string `plist:",omitempty"`
		Status      string
		RequestType string `plist:",omitempty"`
	}{CommandUUID: cmdStr, Status: status, RequestType: cmdStr}
	rawBytes, err := plist.Marshal(fReport)
	if err != nil {
		t.Fatal(err)
	}
	results := &mdm.CommandResults{
		CommandUUID: fReport.CommandUUID,
		Status:      fReport.Status,
		RequestType: fReport.RequestType,
		Raw:         rawBytes,
	}
	err = q.StoreCommandReport(r, results)
	if err != nil {
		t.Error(err)
	}
}

// reportRetrieve behaves similarly to an MDM client: it first reports
// the results and then retrieves the next command.
func reportRetrieve(t *testing.T, q QueueInterfaces, r *mdm.Request, reportCmd, reportStatus, expectedCmd string) {
	report(t, q, r, reportCmd, reportStatus)
	skipNotNow := false
	if reportStatus == "NotNow" {
		skipNotNow = true
	}
	retrieve(t, q, r, expectedCmd, skipNotNow)
}

// TestQueue performs basic testing of the storage queue
func TestQueue(t *testing.T, id string, q QueueInterfaces) {
	ctx := context.Background()

	// build a fake MDM request object
	r := &mdm.Request{
		EnrollID: &mdm.EnrollID{
			Type:     mdm.Device,
			ID:       id,
			ParentID: "",
		},
		Context: ctx,
	}

	t.Run("basic", func(t *testing.T) {
		reportRetrieve(t, q, r, "", "Idle", "")
		enqueue(t, q, ctx, id, "CMD1")
		enqueue(t, q, ctx, id, "CMD2")
		reportRetrieve(t, q, r, "", "Idle", "CMD1")
		reportRetrieve(t, q, r, "CMD1", "Acknowledged", "CMD2")
		reportRetrieve(t, q, r, "CMD2", "Acknowledged", "")
		reportRetrieve(t, q, r, "", "Idle", "")
	})

	t.Run("notnow", func(t *testing.T) {
		reportRetrieve(t, q, r, "", "Idle", "")
		enqueue(t, q, ctx, id, "CMD3")
		reportRetrieve(t, q, r, "", "Idle", "CMD3")
		reportRetrieve(t, q, r, "CMD3", "NotNow", "")
		reportRetrieve(t, q, r, "", "Idle", "CMD3")
		reportRetrieve(t, q, r, "CMD3", "Acknowledged", "")
		reportRetrieve(t, q, r, "", "Idle", "")
	})
}
