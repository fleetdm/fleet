package troubleshoot

import (
	"os/exec"
	"testing"
	"time"
)

func TestParsePIDs(t *testing.T) {
	raw := "1234\n5678\n\n  91011  \nnot-a-pid\n12\n"
	got := parsePIDs(raw)
	want := []uint32{1234, 5678, 91011, 12}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("pid[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

// allAlive is an alive-stub for tests that don't exercise the liveness filter.
func allAlive(uint32) bool { return true }

func TestDetectedDedup(t *testing.T) {
	cmd := func(pid uint32) string { return "cmd-" + string(rune('0'+pid)) }
	// Port-style: dedup on, no exclusion.
	got := detected([]uint32{2, 2, 3, 2}, true, 0, allAlive, cmd)
	if len(got) != 2 || got[0].PID != 2 || got[1].PID != 3 {
		t.Errorf("dedup failed: %+v", got)
	}
}

func TestDetectedExcludeSelf(t *testing.T) {
	cmd := func(pid uint32) string { return "x" }
	// Pattern-style: no dedup, exclude self (pid 5). Duplicates are kept.
	got := detected([]uint32{4, 5, 6, 6}, false, 5, allAlive, cmd)
	if len(got) != 3 {
		t.Fatalf("got %d, want 3 (self excluded, dups kept): %+v", len(got), got)
	}
	for _, d := range got {
		if d.PID == 5 {
			t.Error("self pid 5 should be excluded")
		}
	}
}

func TestDetectedSkipsDeadOrExited(t *testing.T) {
	cmd := func(pid uint32) string {
		if pid == 3 {
			return "(process exited)" // raced: died between liveness check and ps
		}
		return "live"
	}
	// pid 2 is dead (alive=false), pid 3 raced to exit, pid 4 is a real match.
	alive := func(pid uint32) bool { return pid != 2 }
	got := detected([]uint32{2, 3, 4}, false, 0, alive, cmd)
	if len(got) != 1 || got[0].PID != 4 {
		t.Errorf("expected only live pid 4, got %+v", got)
	}
}

func TestPidAliveAndKill(t *testing.T) {
	// Spawn a real, well-behaved child and confirm SIGTERM stops it.
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	pid := uint32(cmd.Process.Pid)
	// Reap in the background so the OS doesn't leave a zombie that still
	// answers kill(pid, 0).
	go func() { _ = cmd.Wait() }()

	if !pidAlive(pid) {
		t.Fatal("freshly spawned process should be alive")
	}

	out := KillPID(pid)
	if !out.Gone {
		t.Errorf("process should be gone after kill: %+v", out)
	}
	if out.UsedKill {
		t.Error("a plain sleep should die on SIGTERM, not need SIGKILL")
	}
	if out.Error != nil {
		t.Errorf("unexpected error: %v", *out.Error)
	}

	// Give the reaper a moment, then confirm it's really gone.
	time.Sleep(50 * time.Millisecond)
	if pidAlive(pid) {
		t.Error("process still alive after KillPID reported gone")
	}
}
