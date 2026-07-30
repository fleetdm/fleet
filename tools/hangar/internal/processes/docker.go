package processes

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// dockerCmd builds a `docker ...` command with the login-shell PATH applied
// (docker lives in /usr/local/bin or /opt/homebrew/bin, neither on a
// Finder-launched app's bare PATH).
func dockerCmd(args ...string) *exec.Cmd {
	return shellpath.Command("docker", args...)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// DockerComposeStatus runs `docker compose ps --format json` in cwd. A
// ran-but-failed command (e.g. no compose file) yields "not running"; only a
// spawn failure (docker missing) is an error.
func DockerComposeStatus(cwd string) (DockerStatus, error) {
	cmd := dockerCmd("compose", "ps", "--format", "json")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return DockerStatus{Running: false, Containers: []ContainerState{}}, nil
		}
		return DockerStatus{}, err
	}

	containers := []ContainerState{}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var v map[string]any
		if json.Unmarshal([]byte(line), &v) != nil {
			continue
		}
		name := str(v["Service"])
		if name == "" {
			name = str(v["Name"])
		}
		if name != "" {
			containers = append(containers, ContainerState{Name: name, State: str(v["State"])})
		}
	}
	running := false
	for _, c := range containers {
		if c.State == "running" {
			running = true
			break
		}
	}
	return DockerStatus{Running: running, Containers: containers}, nil
}

// DockerComposeRestart runs `docker compose restart` in cwd.
func DockerComposeRestart(cwd string) (string, error) {
	cmd := dockerCmd("compose", "restart")
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return "", errors.New(stderr.String())
		}
		return "", err
	}
	return stdout.String(), nil
}

const dockerComposeProcID = "docker-compose-up"

// DockerComposeDown runs `docker compose down` in cwd, flipping the
// docker-compose-up row to stopping for the duration so the UI doesn't flash
// "not running" while containers tear down.
func (m *Manager) DockerComposeDown(cwd string) (string, error) {
	m.setComposeState("stopping", true, false)
	m.emitState(dockerComposeProcID, "stopping", nil, nil)

	cmd := dockerCmd("compose", "down")
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()

	m.setComposeState("done", false, true)
	m.emitState(dockerComposeProcID, "done", nil, nil)

	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			return "", errors.New(stderr.String())
		}
		return "", runErr
	}
	return stdout.String(), nil
}

// setComposeState mutates the docker-compose-up row if present. When stamp is
// true it also sets ended_at.
func (m *Manager) setComposeState(state string, userStopped, stamp bool) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	info := m.procs[dockerComposeProcID]
	if info == nil {
		return
	}
	info.State = state
	if userStopped {
		info.WasUserStopped = true
	}
	if stamp {
		ended := nowMS()
		info.EndedAtMS = &ended
	}
}

// dockerComposeDownFor tears down compose for a managed docker process id
// (called from signalStop). `up -d` usually exits before the user quits, so
// the row is often "done" already — running down here is idempotent.
func (m *Manager) dockerComposeDownFor(id string) error {
	m.stateMu.Lock()
	var cwd string
	if info := m.procs[id]; info != nil {
		cwd = info.Cwd
	}
	m.stateMu.Unlock()
	if cwd == "" {
		return nil
	}

	cmd := dockerCmd("compose", "down")
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	success := runErr == nil
	var exitCode *int
	if cmd.ProcessState != nil {
		c := cmd.ProcessState.ExitCode()
		if c >= 0 {
			exitCode = &c
		}
	}

	final := "failed"
	if success {
		final = "done"
	}
	m.stateMu.Lock()
	if info := m.procs[id]; info != nil {
		for _, line := range append(splitLines(stdout.String()), splitLines(stderr.String())...) {
			info.RecentLog = append(info.RecentLog, line)
		}
		if len(info.RecentLog) > logTailCap {
			info.RecentLog = info.RecentLog[len(info.RecentLog)-logTailCap:]
		}
		info.State = final
		info.ExitCode = exitCode
		ended := nowMS()
		info.EndedAtMS = &ended
		info.WasUserStopped = true
	}
	delete(m.pids, id)
	m.stateMu.Unlock()
	m.emitState(id, final, exitCode, nil)
	m.finishLifecycle(id)
	return nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}
