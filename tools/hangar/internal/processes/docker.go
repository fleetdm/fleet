package processes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// dockerCmd builds a `docker ...` command with the login-shell PATH applied
// (docker lives in /usr/local/bin or /opt/homebrew/bin, neither on a
// Finder-launched app's bare PATH).
func dockerCmd(args ...string) *exec.Cmd {
	return shellpath.Command("docker", args...)
}

// composeTimeout bounds a synchronous compose call so a hung Docker can't
// block a stop, a job, or quit forever. It's generous because a slow Docker
// VM can take minutes to stop a stack.
var composeTimeout = 5 * time.Minute

// runCompose runs `docker compose [-p project] rest...` in cwd until it exits,
// ctx ends, or composeTimeout passes. Either way it then kills the CLI's whole
// process group (the compose plugin runs as a child of docker) and returns
// once the group is gone. If the CLI's exit can't be watched, it fails closed:
// the group is killed at once and the run returns an error.
func (m *Manager) runCompose(ctx context.Context, cwd, project string, rest ...string) (stdout, stderr string, exitCode *int, err error) {
	ctx, cancel := context.WithTimeout(ctx, composeTimeout)
	defer cancel()
	cmd := dockerCmd(composeArgs(project, rest...)...)
	cmd.Dir = cwd
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.WaitDelay = time.Second // pipes held by something outside the group
	if err := cmd.Start(); err != nil {
		return "", "", nil, err
	}
	pgid := cmd.Process.Pid
	exited := make(chan bool, 1)
	go func() { exited <- m.waitExit(pgid) }()
	watched := true
	select {
	case watched = <-exited:
	case <-ctx.Done():
	}
	// The CLI is unreaped until cmd.Wait, so its pgid is still ours.
	m.killGroup(pgid)
	err = cmd.Wait()
	waitGroupGone(pgid)
	if ps := cmd.ProcessState; ps != nil && ps.ExitCode() >= 0 {
		c := ps.ExitCode()
		exitCode = &c
	}
	switch {
	case !watched:
		err = fmt.Errorf("docker compose %s: could not watch the process; stopped it", strings.Join(rest, " "))
	case err != nil && ctx.Err() != nil:
		err = fmt.Errorf("docker compose %s: %w", strings.Join(rest, " "), ctx.Err())
	}
	return out.String(), errOut.String(), exitCode, err
}

// killGroup SIGKILLs pgid's group, falling back to the bare pid.
func (m *Manager) killGroup(pgid int) {
	if err := m.kill(-pgid, syscall.SIGKILL); err != nil {
		_ = m.kill(pgid, syscall.SIGKILL)
	}
}

// composeErr reports a compose run that exited non-zero by its stderr.
func composeErr(err error, stderr string) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return errors.New(stderr)
	}
	return err
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

// DockerComposeRestartAs runs `docker compose [-p project] restart` in cwd,
// tracked as an in-flight op on project. A held project refuses anyone but
// its holder.
func (m *Manager) DockerComposeRestartAs(cwd, project, owner string) (string, error) {
	end, err := m.beginDockerOp(project, owner)
	if err != nil {
		return "", err
	}
	defer end()

	stdout, stderr, _, err := m.runCompose(context.Background(), cwd, project, "restart")
	if err != nil {
		return "", composeErr(err, stderr)
	}
	return stdout, nil
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
	return m.DockerComposeDownAs(id, cwd, project, "")
}

// DockerComposeDownAs is DockerComposeDown on behalf of owner, tracked as an
// in-flight op on project. A held project refuses anyone but its holder.
func (m *Manager) DockerComposeDownAs(id, cwd, project, owner string) (string, error) {
	return m.DockerComposeDownContext(context.Background(), id, cwd, project, owner)
}

// DockerComposeDownContext is DockerComposeDownAs that also ends when ctx
// does, killing the compose CLI.
func (m *Manager) DockerComposeDownContext(ctx context.Context, id, cwd, project, owner string) (string, error) {
	if id == "" {
		id = dockerComposeProcID
	}
	end, err := m.beginDockerOp(project, owner)
	if err != nil {
		return "", err
	}
	defer end()

	m.setComposeState(id, "stopping", true, false)
	m.emitState(id, "stopping", nil, nil)

	stdout, stderr, _, runErr := m.runCompose(ctx, cwd, project, "down")

	m.setComposeState(id, "done", false, true)
	m.emitState(id, "done", nil, nil)

	if runErr != nil {
		return "", composeErr(runErr, stderr)
	}
	return stdout, nil
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

// dockerComposeDownFor runs compose down for a managed docker row. signalStop
// has already checked the hold, registered the op, and waited out a running up.
func (m *Manager) dockerComposeDownFor(id, cwd, project string) error {
	stdout, stderr, exitCode, runErr := m.runCompose(context.Background(), cwd, project, "down")

	final := "failed"
	if runErr == nil {
		final = "done"
	}
	m.stateMu.Lock()
	if _, respawned := m.live[id]; respawned {
		m.stateMu.Unlock()
		return nil
	}
	if info := m.procs[id]; info != nil {
		for _, line := range append(splitLines(stdout), splitLines(stderr)...) {
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
	m.stateMu.Unlock()
	m.emitState(id, final, exitCode, nil)
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
