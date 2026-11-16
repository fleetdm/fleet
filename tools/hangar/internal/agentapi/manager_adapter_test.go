package agentapi

import (
	"context"
	"testing"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
)

func TestManagerAdapter(t *testing.T) {
	m := NewManager(processes.New(t.TempDir(), t.TempDir(), nil))
	if err := m.Hold(processes.HoldSpec{Slot: "s2", Owner: "job-1", Project: "p2", ServerPort: 8380}); err != nil {
		t.Fatal(err)
	}
	defer m.Release("s2", "job-1")

	if _, err := m.Start("s2:make-deps", processes.StartArgs{Program: "sh", Args: []string{"-c", "exit 0"}}); err == nil {
		t.Fatal("start without the holder's owner should be refused")
	}
	inv, err := m.Start("s2:make-deps", processes.StartArgs{Program: "sh", Args: []string{"-c", "exit 3"}, Owner: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if code, err := m.Wait(ctx, inv); err != nil || code != 3 {
		t.Fatalf("wait = %d, %v", code, err)
	}

	inv, err = m.Start("s2:fleet-serve", processes.StartArgs{Program: "sleep", Args: []string{"30"}, Owner: "job-1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Stop("s2:fleet-serve", "job-1"); err != nil {
		t.Fatal(err)
	}
	if code, err := m.Wait(ctx, inv); err != nil || code != -1 {
		t.Fatalf("stopped wait = %d, %v", code, err)
	}
}
