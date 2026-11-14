package processes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/shellpath"
)

// fakeDocker makes shellpath resolve `docker` to a script running body, via a
// fake login shell whose PATH puts the script's dir first. Returns that dir.
func fakeDocker(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("docker", "#!/bin/sh\n"+body)
	write("login-shell", "#!/bin/sh\nprintf %s '"+dir+":/usr/bin:/bin:/usr/sbin:/sbin'\n")
	t.Cleanup(func() { shellpath.Refresh() }) // runs after SHELL is restored
	t.Setenv("SHELL", filepath.Join(dir, "login-shell"))
	shellpath.Refresh()
	return dir
}

func waitForFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("%s never appeared", path)
}

func sleeper(owner string) StartArgs {
	return StartArgs{Program: "sleep", Args: []string{"30"}, Owner: owner}
}

// perfArgs runs a sleep whose argv carries --server_url like osquery-perf's.
func perfArgs(serverURLArgs ...string) StartArgs {
	return StartArgs{Program: "sh", Args: append([]string{"-c", "sleep 30", "perf"}, serverURLArgs...)}
}

func (r *recorder) count(name string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e.name == name {
			n++
		}
	}
	return n
}

func TestHoldAdmission(t *testing.T) {
	rec := &recorder{}
	m := New(t.TempDir(), t.TempDir(), rec)
	if err := m.Start("s2:fleet-serve", sleeper("")); err != nil {
		t.Fatal(err)
	}
	defer m.Stop("s2:fleet-serve")

	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1"}); err == nil {
		t.Fatal("Hold should refuse while a foreign s2 process runs")
	}
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", Replace: []string{"s2:fleet-serve"}}); err != nil {
		t.Fatalf("Hold with the running id in Replace: %v", err)
	}
	if owner, held := m.HeldBy("s2"); !held || owner != "job-1" {
		t.Errorf("HeldBy = %q, %v; want job-1, true", owner, held)
	}
	if got := m.Holds(); len(got) != 1 || got["s2"] != "job-1" {
		t.Errorf("Holds = %v", got)
	}
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-2", Allow: []string{"s2:fleet-serve"}}); err == nil {
		t.Error("second Hold on a held slot should fail")
	}
	m.Release("s2", "job-2")
	if _, held := m.HeldBy("s2"); !held {
		t.Error("Release by a non-holder dropped the hold")
	}
	m.Release("s2", "job-1")
	if _, held := m.HeldBy("s2"); held {
		t.Error("Release by the holder kept the hold")
	}
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-3", Allow: []string{"s2:fleet-serve"}}); err != nil {
		t.Fatalf("Hold with the running id in Allow: %v", err)
	}
	m.Release("s2", "job-3")
	if n := rec.count("agent:slots"); n != 4 {
		t.Errorf("agent:slots emitted %d times, want 4 (2 holds, 2 releases)", n)
	}
}

func TestHoldRefusedByPerfOnSlotPort(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Start("perf-a", perfArgs("--server_url", "https://localhost:8380")); err != nil {
		t.Fatal(err)
	}
	defer m.Stop("perf-a")

	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", ServerPort: 8380}); err == nil {
		t.Error("Hold should refuse while a perf run targets the slot's server port")
	}
	if err := m.Hold(HoldSpec{Slot: "s3", Owner: "job-1", ServerPort: 8480}); err != nil {
		t.Errorf("Hold on another port: %v", err)
	}
}

func TestHeldSlotRefusesPerfStartOnItsPort(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", ServerPort: 8380}); err != nil {
		t.Fatal(err)
	}
	if err := m.Start("perf-a", perfArgs("--server_url=https://localhost:8380")); err == nil {
		m.Stop("perf-a")
		t.Error("perf start aimed at a held slot's port should be refused")
	}
	if err := m.Start("perf-b", perfArgs("--server_url", "https://localhost:8381")); err != nil {
		t.Errorf("perf start on another port: %v", err)
	}
	m.Stop("perf-b")
}

func TestHoldRefusedDuringDockerOp(t *testing.T) {
	dir := fakeDocker(t, `d=$(dirname "$0")
touch "$d/started"
while [ ! -e "$d/release" ]; do sleep 0.05; done
`)
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	ops := map[string]func() error{
		"down": func() error {
			_, err := m.DockerComposeDown("s2:docker-compose-up", dir, "p2")
			return err
		},
		"restart": func() error {
			_, err := m.DockerComposeRestartAs(dir, "p2", "")
			return err
		},
	}
	for name, op := range ops {
		_ = os.Remove(filepath.Join(dir, "started"))
		_ = os.Remove(filepath.Join(dir, "release"))
		errc := make(chan error, 1)
		go func() { errc <- op() }()
		waitForFile(t, filepath.Join(dir, "started"))

		if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", Project: "p2"}); err == nil {
			t.Errorf("%s: Hold should refuse while compose %s is in flight", name, name)
			m.Release("s2", "job-1")
		}
		if err := os.WriteFile(filepath.Join(dir, "release"), nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := <-errc; err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", Project: "p2"}); err != nil {
			t.Errorf("%s: Hold after the op finished: %v", name, err)
		}
		m.Release("s2", "job-1")
	}
}

func TestHeldSlotRefusesForeignCallers(t *testing.T) {
	dir := fakeDocker(t, `touch "$(dirname "$0")/ran"`+"\n")
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Start("s2:fleet-serve", sleeper("")); err != nil {
		t.Fatal(err)
	}
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", Project: "p2", Allow: []string{"s2:fleet-serve"}}); err != nil {
		t.Fatal(err)
	}

	if err := m.Start("s2:build", sleeper("")); err == nil {
		t.Error("foreign Start on a held slot should be refused")
	}
	if err := m.Start("s2:build", sleeper("job-2")); err == nil {
		t.Error("Start by another owner on a held slot should be refused")
	}
	if err := m.Start("s2:build", sleeper("job-1")); err != nil {
		t.Errorf("holder's Start: %v", err)
	}
	if err := m.Stop("s2:fleet-serve"); err == nil {
		t.Error("foreign Stop on a held slot should be refused")
	}
	if err := m.Restart("s2:fleet-serve"); err == nil {
		t.Error("foreign Restart on a held slot should be refused")
	}
	waitForState(t, m, "s2:fleet-serve", "running", time.Second)
	if _, err := m.DockerComposeDown("s2:docker-compose-up", dir, "p2"); err == nil {
		t.Error("foreign compose down on a held project should be refused")
	}
	if _, err := m.DockerComposeRestartAs(dir, "p2", ""); err == nil {
		t.Error("foreign compose restart on a held project should be refused")
	}
	if _, err := os.Stat(filepath.Join(dir, "ran")); err == nil {
		t.Error("a refused compose op still ran docker")
	}
	if err := m.Start("s3:fleet-serve", sleeper("")); err != nil {
		t.Errorf("Start on an unheld slot: %v", err)
	}

	for _, id := range []string{"s2:fleet-serve", "s2:build"} {
		if err := m.StopAs(id, "job-1"); err != nil {
			t.Errorf("holder's StopAs(%s): %v", id, err)
		}
	}
	if _, err := m.DockerComposeDownAs("s2:docker-compose-up", dir, "p2", "job-1"); err != nil {
		t.Errorf("holder's compose down: %v", err)
	}
	m.Stop("s3:fleet-serve")
}

func TestRestartDoesNotInheritOwner(t *testing.T) {
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1"}); err != nil {
		t.Fatal(err)
	}
	inv, err := m.StartInvocation("s2:x", sleeper("job-1"))
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Restart("s2:x"); err == nil {
		t.Fatal("GUI Restart picked up the holder's owner from remembered args")
	}
	m.stateMu.Lock()
	same := m.live["s2:x"] == inv.inv
	m.stateMu.Unlock()
	if !same {
		t.Fatal("refused Restart replaced the running invocation")
	}
	if err := m.RestartAs("s2:x", "job-1"); err != nil {
		t.Fatalf("holder's RestartAs: %v", err)
	}

	m.Release("s2", "job-1")
	if err := m.Restart("s2:x"); err != nil {
		t.Fatalf("Restart after release: %v", err)
	}
	m.stateMu.Lock()
	owner := m.lastArgs["s2:x"].Owner
	m.stateMu.Unlock()
	if owner != "" {
		t.Errorf("GUI Restart respawned with owner %q, want none", owner)
	}
	m.Stop("s2:x")
}

func TestServerURLPort(t *testing.T) {
	cases := []struct {
		args []string
		want uint16
	}{
		{[]string{"run", ".", "--server_url", "https://localhost:8380"}, 8380},
		{[]string{"--server_url=https://localhost:8480", "--x"}, 8480},
		{[]string{"-server_url", "https://127.0.0.1:9000/"}, 9000},
		{[]string{"--server_url", "https://localhost"}, 0},
		{[]string{"--server_url"}, 0},
		{[]string{"server_url", "https://localhost:8380"}, 0},
		{[]string{"--enroll_secret", "x"}, 0},
	}
	for _, c := range cases {
		if got := serverURLPort(c.args); got != c.want {
			t.Errorf("serverURLPort(%q) = %d, want %d", c.args, got, c.want)
		}
	}
}

// gateEmitter blocks the first final proc:state for id until gate closes,
// pausing a finalizer after it has made the slot look idle.
type gateEmitter struct {
	recorder
	id      string
	once    sync.Once
	reached chan struct{}
	gate    chan struct{}
}

func (g *gateEmitter) Emit(name string, data any) {
	g.recorder.Emit(name, data)
	if ev, ok := data.(ProcEvent); ok && ev.ProcID == g.id && ev.State != "running" && ev.State != "stopping" {
		g.once.Do(func() {
			close(g.reached)
			<-g.gate
		})
	}
}

func TestStopTracksComposeDownFromAdmission(t *testing.T) {
	dir := fakeDocker(t, `d=$(dirname "$0")
case " $* " in
*" up "*) exec sleep 30 ;;
*" down "*) touch "$d/down-ran" ;;
esac
`)
	const id = "s2:docker-compose-up"
	em := &gateEmitter{id: id, reached: make(chan struct{}), gate: make(chan struct{})}
	m := New(t.TempDir(), t.TempDir(), em)
	if err := m.Start(id, StartArgs{Label: "docker compose up", Cwd: dir, Program: "docker", Args: []string{"compose", "-p", "p2", "up"}}); err != nil {
		t.Fatal(err)
	}
	stopped := make(chan error, 1)
	go func() { stopped <- m.Stop(id) }()
	select {
	case <-em.reached:
	case <-time.After(5 * time.Second):
		t.Fatal("compose up never finalized")
	}

	// The up is gone and its row is final, but the stop's down hasn't run.
	err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", Project: "p2"})
	close(em.gate)
	if err == nil {
		m.Release("s2", "job-1")
		t.Error("Hold admitted on p2 before a GUI stop's compose down ran")
	}
	if err := <-stopped; err != nil {
		t.Fatal(err)
	}
	waitForFile(t, filepath.Join(dir, "down-ran"))
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", Project: "p2"}); err != nil {
		t.Errorf("Hold after the stop finished: %v", err)
	}
}

func TestComposeDownEndsWithContextAndTimeout(t *testing.T) {
	// down hangs with a child in its group holding the output pipe.
	dir := fakeDocker(t, `d=$(dirname "$0")
sleep 30 &
echo $! > "$d/down-child.pid"
wait
`)
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	childGone := func() {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(dir, "down-child.pid"))
		if err != nil {
			t.Fatal(err)
		}
		pid, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })
		deadline := time.Now().Add(2 * time.Second)
		for syscall.Kill(pid, 0) == nil && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		if syscall.Kill(pid, 0) == nil {
			t.Errorf("compose child %d survived the cancelled down", pid)
		}
	}

	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(300*time.Millisecond, cancel)
	start := time.Now()
	if _, err := m.DockerComposeDownContext(ctx, "s2:docker-compose-up", dir, "p2", ""); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled down = %v, want context.Canceled", err)
	}
	if d := time.Since(start); d > 3*time.Second {
		t.Errorf("cancelled down took %v", d)
	}
	childGone()

	prev := composeTimeout
	composeTimeout = 300 * time.Millisecond
	t.Cleanup(func() { composeTimeout = prev })
	_ = os.Remove(filepath.Join(dir, "down-child.pid"))
	if _, err := m.DockerComposeDown("s2:docker-compose-up", dir, "p2"); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("hung down = %v, want context.DeadlineExceeded", err)
	}
	childGone()
	if err := m.Hold(HoldSpec{Slot: "s2", Owner: "job-1", Project: "p2"}); err != nil {
		t.Errorf("ended downs still count as in flight: %v", err)
	}
}

func TestComposeReturnsOnlyOnceItsGroupIsGone(t *testing.T) {
	// down exits 0 but leaves a TERM-ignoring child holding its output pipe.
	dir := fakeDocker(t, `(trap '' TERM; exec sleep 30) &
echo $! > "$(dirname "$0")/child.pid"
`)
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	_, err := m.DockerComposeDown("s2:docker-compose-up", dir, "p2")
	b, rerr := os.ReadFile(filepath.Join(dir, "child.pid"))
	if rerr != nil {
		t.Fatal(rerr)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	t.Cleanup(func() { _ = syscall.Kill(child, syscall.SIGKILL) })
	if err != nil {
		t.Errorf("down = %v, want success once its leftover child is killed", err)
	}
	if syscall.Kill(child, 0) == nil {
		t.Errorf("compose child %d still alive after DockerComposeDown returned", child)
	}
}

func TestComposeFailsClosedWithoutExitWait(t *testing.T) {
	dir := fakeDocker(t, `sleep 30 &
echo $! > "$(dirname "$0")/child.pid"
wait
`)
	childFile := filepath.Join(dir, "child.pid")
	m := New(t.TempDir(), t.TempDir(), &recorder{})
	// As when kqueue can't be set up, once the CLI has started a child.
	m.waitExit = func(int) bool {
		waitForFile(t, childFile)
		return false
	}
	var mu sync.Mutex
	var kills, afterReap []string
	m.kill = func(pid int, sig syscall.Signal) error {
		if pid < 0 && sig != 0 {
			mu.Lock()
			kills = append(kills, fmt.Sprintf("%v to group %d", sig, -pid))
			if !listed(-pid) {
				afterReap = append(afterReap, kills[len(kills)-1])
			}
			mu.Unlock()
		}
		return syscall.Kill(pid, sig)
	}

	_, err := m.DockerComposeRestartAs(dir, "p2", "")
	if err == nil || !strings.Contains(err.Error(), "could not watch the process") {
		t.Errorf("restart = %v, want the could-not-watch error", err)
	}
	mu.Lock()
	if len(kills) == 0 || len(afterReap) > 0 {
		t.Errorf("group kills = %v, after reap = %v; want one before the CLI is reaped", kills, afterReap)
	}
	mu.Unlock()
	b, err := os.ReadFile(childFile)
	if err != nil {
		t.Fatal(err)
	}
	child, _ := strconv.Atoi(strings.TrimSpace(string(b)))
	t.Cleanup(func() { _ = syscall.Kill(child, syscall.SIGKILL) })
	if syscall.Kill(child, 0) == nil {
		t.Errorf("compose child %d outlived runCompose", child)
	}
}
