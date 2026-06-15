package processes

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestWritePidFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	recs := []PidRecord{{ID: "fleet-serve", PID: 123, Program: "/x/fleet", Args: []string{"serve"}}}
	writePidFile(dir, recs)

	got := readPidRecords(dir)
	if len(got) != 1 || got[0].ID != "fleet-serve" || got[0].PID != 123 {
		t.Fatalf("round-trip mismatch: %+v", got)
	}

	// Empty records removes the file.
	writePidFile(dir, nil)
	if _, err := os.Stat(pidFilePath(dir)); !os.IsNotExist(err) {
		t.Error("empty records should remove running.json")
	}
}

func TestPidIsAlive(t *testing.T) {
	if !pidIsAlive(os.Getpid()) {
		t.Error("our own pid should be alive")
	}

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	pid := cmd.Process.Pid
	if !pidIsAlive(pid) {
		t.Error("spawned sleep should be alive")
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait() // reap so it's not a zombie answering kill(pid,0)
	if pidIsAlive(pid) {
		t.Error("killed+reaped pid should be dead")
	}
}

func TestPidMatchesRecord(t *testing.T) {
	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	if !pidMatchesRecord(pid, "sleep", []string{"30"}) {
		t.Error("should match the real command line")
	}
	if pidMatchesRecord(pid, "not-sleep", []string{"30"}) {
		t.Error("wrong program should not match")
	}
	if pidMatchesRecord(pid, "sleep", []string{"99"}) {
		t.Error("wrong first arg should not match")
	}
}

func TestCleanOrphansFromPriorRun(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	pid := cmd.Process.Pid
	go func() { _ = cmd.Wait() }() // reap whenever it dies

	writePidFile(dir, []PidRecord{{ID: "x", PID: pid, Program: "sleep", Args: []string{"30"}}})

	CleanOrphansFromPriorRun(dir)

	// running.json wiped.
	if _, err := os.Stat(pidFilePath(dir)); !os.IsNotExist(err) {
		t.Error("running.json should be removed after cleanup")
	}
	// Process gone (give the reaper a beat).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !pidIsAlive(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Error("orphan should have been killed")
}

func TestCleanOrphansSkipsMismatch(t *testing.T) {
	dir := t.TempDir()

	cmd := exec.Command("sleep", "30")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	pid := cmd.Process.Pid
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	// Record a mismatching command line — cleanup must NOT kill it.
	writePidFile(dir, []PidRecord{{ID: "x", PID: pid, Program: "totally-different-binary", Args: []string{"xyz"}}})
	CleanOrphansFromPriorRun(dir)

	time.Sleep(700 * time.Millisecond) // longer than the SIGTERM grace
	if !pidIsAlive(pid) {
		t.Error("process with non-matching command line should be left alone")
	}
}
