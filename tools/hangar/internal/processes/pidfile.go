package processes

// ----- crash-survival PID tracking -----
//
// We don't kill children when the Manager goes away (a hard parent death —
// dev reload, force-quit, OS panic — skips any cleanup anyway). Instead we
// persist every running spawn to <data-dir>/running.json and, on next
// startup, look up each pid: if it's still alive AND its `ps` command line
// still matches what we recorded, we SIGTERM (then SIGKILL) it. The
// command-line match is what keeps us from killing a recycled pid that now
// belongs to something innocent.

import (
	"encoding/json"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// PidRecord is one persisted running spawn.
type PidRecord struct {
	ID      string   `json:"id"`
	PID     int      `json:"pid"`
	Program string   `json:"program"`
	Args    []string `json:"args"`
}

func pidFilePath(dataDir string) string {
	return filepath.Join(dataDir, "running.json")
}

// writePidFile persists records, or removes the file when empty (a missing
// file on next startup is the "nothing to clean" fast path).
func writePidFile(dataDir string, records []PidRecord) {
	path := pidFilePath(dataDir)
	if len(records) == 0 {
		_ = os.Remove(path)
		return
	}
	b, err := json.Marshal(records)
	if err != nil {
		slog.Warn("marshal running.json failed", "err", err)
		return
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		slog.Warn("write running.json failed", "path", path, "err", err)
	}
}

func readPidRecords(dataDir string) []PidRecord {
	b, err := os.ReadFile(pidFilePath(dataDir))
	if err != nil {
		return nil
	}
	var recs []PidRecord
	if json.Unmarshal(b, &recs) != nil {
		return nil
	}
	return recs
}

// pidIsAlive reports whether pid exists (signal 0: nil=alive, ESRCH=gone,
// EPERM=alive-but-unsignalable).
func pidIsAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return err != syscall.ESRCH
}

// pidMatchesRecord confirms pid still belongs to our spawn by reading its
// command line (`ps`) and requiring both the program basename and the first
// arg to appear — guards against pid recycling.
func pidMatchesRecord(pid int, program string, args []string) bool {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil {
		return false
	}
	line := strings.TrimSpace(string(out))
	if line == "" {
		return false
	}
	if !strings.Contains(line, filepath.Base(program)) {
		return false
	}
	if len(args) > 0 && !strings.Contains(line, args[0]) {
		return false
	}
	return true
}

// signalGroup signals the process group (we spawn with Setpgid so pgid==pid),
// falling back to the bare pid if the group send fails.
func signalGroup(pid int, sig syscall.Signal) {
	if err := syscall.Kill(-pid, sig); err != nil {
		_ = syscall.Kill(pid, sig)
	}
}

// CleanOrphansFromPriorRun SIGTERM/SIGKILLs anything from a prior session
// still alive whose command line matches what we recorded. Call once at
// startup before the tray/command pipeline come up.
func CleanOrphansFromPriorRun(dataDir string) {
	path := pidFilePath(dataDir)
	if _, err := os.Stat(path); err != nil {
		return
	}
	records := readPidRecords(dataDir)
	// Wipe immediately — stale bookkeeping either way, and we don't want it
	// tripping the next startup.
	_ = os.Remove(path)

	cleaned := 0
	for _, r := range records {
		if !pidIsAlive(r.PID) || !pidMatchesRecord(r.PID, r.Program, r.Args) {
			continue
		}
		slog.Info("reaping orphan from prior run", "id", r.ID, "pid", r.PID, "program", r.Program)
		signalGroup(r.PID, syscall.SIGTERM)
		// Brief grace before escalating (plenty for python/ngrok; fleet
		// serve usually dies cleanly on SIGTERM too).
		time.Sleep(500 * time.Millisecond)
		if pidIsAlive(r.PID) {
			signalGroup(r.PID, syscall.SIGKILL)
		}
		cleaned++
	}
	if cleaned > 0 {
		slog.Info("orphan cleanup complete", "reaped", cleaned)
	}
}
