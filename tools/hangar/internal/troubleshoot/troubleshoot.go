// Package troubleshoot scans for processes by listening port or command
// pattern and can terminate them. Ported from src-tauri/src/troubleshoot.rs.
// macOS-only (lsof/pgrep/ps + POSIX signals).
package troubleshoot

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// DetectedProcess is a process found by a scan.
type DetectedProcess struct {
	PID     uint32 `json:"pid"`
	Command string `json:"command"`
}

// KillOutcome reports the result of terminating a pid.
type KillOutcome struct {
	PID      uint32  `json:"pid"`
	Gone     bool    `json:"gone"`
	UsedKill bool    `json:"used_kill"`
	Error    *string `json:"error"`
}

// runCapture runs program and returns stdout. lsof/pgrep exit 1 with empty
// output when nothing matches — that's "no results", not an error.
func runCapture(program string, args ...string) (string, error) {
	out, err := exec.Command(program, args...).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			// Empty stdout on a non-zero exit = no matches.
			if len(out) == 0 && len(ee.Stderr) == 0 {
				return "", nil
			}
			if len(out) == 0 {
				return "", nil
			}
		} else {
			return "", fmt.Errorf("%s: %w", program, err)
		}
	}
	return string(out), nil
}

// parsePIDs extracts numeric PIDs from one-per-line output, skipping blanks
// and non-numeric lines.
func parsePIDs(raw string) []uint32 {
	var out []uint32
	for _, line := range strings.Split(raw, "\n") {
		n, err := strconv.ParseUint(strings.TrimSpace(line), 10, 32)
		if err != nil {
			continue
		}
		out = append(out, uint32(n))
	}
	return out
}

// detected turns PIDs into DetectedProcess entries. dedup drops repeats
// (keeping first); exclude (nonzero) skips a pid (e.g. our own). cmd
// resolves each pid's command line.
//
// PIDs that have already exited are skipped: pgrep/lsof can match a process
// that's mid-teardown (exactly the state right after stopping a server), and a
// dead process is neither killable nor a real orphan. This keeps the scan
// self-consistent — it reports only live matches, matching what a re-scan a
// moment later would show.
func detected(pids []uint32, dedup bool, exclude uint32, alive func(uint32) bool, cmd func(uint32) string) []DetectedProcess {
	var out []DetectedProcess
	seen := map[uint32]bool{}
	for _, pid := range pids {
		if exclude != 0 && pid == exclude {
			continue
		}
		if dedup {
			if seen[pid] {
				continue
			}
			seen[pid] = true
		}
		if !alive(pid) {
			continue
		}
		c := cmd(pid)
		if c == "(process exited)" {
			// Raced: exited between the liveness check and reading its command.
			continue
		}
		out = append(out, DetectedProcess{PID: pid, Command: c})
	}
	return out
}

// pidCommand returns the command line for pid via `ps`, or a placeholder if
// the process has exited.
func pidCommand(pid uint32) string {
	out, err := exec.Command("ps", "-p", strconv.FormatUint(uint64(pid), 10), "-o", "command=").Output()
	if err != nil {
		return "(process exited)"
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "(process exited)"
	}
	return s
}

// ScanPort lists processes listening on the given TCP port (deduped).
func ScanPort(port uint16) ([]DetectedProcess, error) {
	raw, err := runCapture("lsof", "-nP", fmt.Sprintf("-iTCP:%d", port), "-sTCP:LISTEN", "-t")
	if err != nil {
		return nil, err
	}
	return detected(parsePIDs(raw), true, 0, pidAlive, pidCommand), nil
}

// ScanPattern lists processes whose full command line matches pattern,
// excluding our own process (so searching "fleet" doesn't list Hangar).
func ScanPattern(pattern string) ([]DetectedProcess, error) {
	raw, err := runCapture("pgrep", "-f", pattern)
	if err != nil {
		return nil, err
	}
	return detected(parsePIDs(raw), false, uint32(os.Getpid()), pidAlive, pidCommand), nil
}

func signalPID(pid uint32, sig syscall.Signal) error {
	err := syscall.Kill(int(pid), sig)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("kill(%d, %d): %w", pid, sig, err)
	}
	return nil
}

// pidAlive reports whether pid exists. Signal 0 checks existence/permission
// without delivering anything: nil = exists, ESRCH = gone, EPERM = exists.
func pidAlive(pid uint32) bool {
	err := syscall.Kill(int(pid), 0)
	if err == nil {
		return true
	}
	return !errors.Is(err, syscall.ESRCH)
}

// KillPID sends SIGTERM, waits up to 2s for graceful exit, then escalates to
// SIGKILL.
func KillPID(pid uint32) KillOutcome {
	strp := func(s string) *string { return &s }

	if err := signalPID(pid, syscall.SIGTERM); err != nil {
		return KillOutcome{PID: pid, Gone: !pidAlive(pid), UsedKill: false, Error: strp(err.Error())}
	}
	for range 20 {
		time.Sleep(100 * time.Millisecond)
		if !pidAlive(pid) {
			return KillOutcome{PID: pid, Gone: true}
		}
	}
	if err := signalPID(pid, syscall.SIGKILL); err != nil {
		return KillOutcome{PID: pid, Gone: !pidAlive(pid), UsedKill: true, Error: strp(err.Error())}
	}
	for range 5 {
		time.Sleep(50 * time.Millisecond)
		if !pidAlive(pid) {
			return KillOutcome{PID: pid, Gone: true, UsedKill: true}
		}
	}
	return KillOutcome{PID: pid, Gone: false, UsedKill: true, Error: strp("process still alive after SIGKILL")}
}
