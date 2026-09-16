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

// composeArgs prefixes a `compose [-p project] <rest...>` argv. An empty
// project falls back to docker's default (the cwd's basename), preserving
// single-server behavior.
func composeArgs(project string, rest ...string) []string {
	args := []string{"compose"}
	if project != "" {
		args = append(args, "-p", project)
	}
	return append(args, rest...)
}

// composeProjectFromArgs extracts the `-p` / `--project-name` value from a
// stored compose argv, so a teardown can target the same project the spawn
// used. Returns "" when none is present.
func composeProjectFromArgs(args []string) string {
	for i, a := range args {
		if a == "-p" || a == "--project-name" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
		if v, ok := strings.CutPrefix(a, "--project-name="); ok {
			return v
		}
	}
	return ""
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

// DockerComposeStatus runs `docker compose [-p project] ps --format json` in
// cwd. A ran-but-failed command (e.g. no compose file) yields "not running";
// only a spawn failure (docker missing) is an error. project scopes the query
// to one server's stack; "" uses the default (cwd-basename) project.
func DockerComposeStatus(cwd, project string) (DockerStatus, error) {
	cmd := dockerCmd(composeArgs(project, "ps", "--format", "json")...)
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

// DockerComposeRestart runs `docker compose [-p project] restart` in cwd.
func DockerComposeRestart(cwd, project string) (string, error) {
	cmd := dockerCmd(composeArgs(project, "restart")...)
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

// dockerComposeProcID is the legacy single-server compose row id, kept as the
// default when a caller doesn't pass an explicit (per-server) id.
const dockerComposeProcID = "docker-compose-up"

// DockerComposeDown runs `docker compose [-p project] down` in cwd, flipping
// the given compose row (id) to stopping for the duration so the UI doesn't
// flash "not running" while containers tear down. id is the per-server
// `<serverID>:docker-compose-up` process id; an empty id targets the legacy
// default row.
func (m *Manager) DockerComposeDown(id, cwd, project string) (string, error) {
	if id == "" {
		id = dockerComposeProcID
	}
	m.setComposeState(id, "stopping", true, false)
	m.emitState(id, "stopping", nil, nil)

	cmd := dockerCmd(composeArgs(project, "down")...)
	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()

	m.setComposeState(id, "done", false, true)
	m.emitState(id, "done", nil, nil)

	if runErr != nil {
		var ee *exec.ExitError
		if errors.As(runErr, &ee) {
			return "", errors.New(stderr.String())
		}
		return "", runErr
	}
	return stdout.String(), nil
}

// setComposeState mutates the given compose row if present. When stamp is
// true it also sets ended_at.
func (m *Manager) setComposeState(id, state string, userStopped, stamp bool) {
	m.stateMu.Lock()
	defer m.stateMu.Unlock()
	info := m.procs[id]
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
	var cwd, project string
	if info := m.procs[id]; info != nil {
		cwd = info.Cwd
	}
	if a, ok := m.lastArgs[id]; ok {
		project = composeProjectFromArgs(a.Args)
	}
	m.stateMu.Unlock()
	if cwd == "" {
		return nil
	}

	cmd := dockerCmd(composeArgs(project, "down")...)
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
