package slots

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fleetdm/fleet/tools/hangar/internal/processes"
	"github.com/fleetdm/fleet/tools/hangar/internal/settings"
)

func TestMain(m *testing.M) {
	pollInterval, portFreeWait, stopWait = 5*time.Millisecond, 50*time.Millisecond, 2*time.Second
	os.Exit(m.Run())
}

type invocation struct {
	done chan struct{}
	code int
}

// fakeRunner records calls. Steps exit with codes[id] (default 0) unless
// listed in block, which run until stopped or exit(id, code).
type fakeRunner struct {
	mu      sync.Mutex
	calls   []string
	args    map[string]processes.StartArgs
	codes   map[string]int
	block   map[string]bool
	invs    map[uint64]*invocation
	live    map[string]uint64
	seq     uint64
	onStart func(id string, a processes.StartArgs)
}

func newRunner() *fakeRunner {
	return &fakeRunner{args: map[string]processes.StartArgs{}, codes: map[string]int{}, block: map[string]bool{},
		invs: map[uint64]*invocation{}, live: map[string]uint64{}}
}

func (r *fakeRunner) record(c string) {
	r.mu.Lock()
	r.calls = append(r.calls, c)
	r.mu.Unlock()
}

func (r *fakeRunner) Calls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.calls)
}

func (r *fakeRunner) Start(id string, a processes.StartArgs) (processes.Invocation, error) {
	r.record("start " + id)
	if r.onStart != nil {
		r.onStart(id, a)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	inv := &invocation{done: make(chan struct{})}
	r.invs[r.seq] = inv
	r.args[id] = a
	if r.block[id] {
		r.live[id] = r.seq
	} else {
		inv.code = r.codes[id]
		close(inv.done)
	}
	return processes.Invocation{ID: id, Seq: r.seq}, nil
}

func (r *fakeRunner) exit(id string, code int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if seq, ok := r.live[id]; ok {
		delete(r.live, id)
		r.invs[seq].code = code
		close(r.invs[seq].done)
	}
}

func (r *fakeRunner) Wait(ctx context.Context, inv processes.Invocation) (int, error) {
	r.mu.Lock()
	i := r.invs[inv.Seq]
	r.mu.Unlock()
	select {
	case <-i.done:
		return i.code, nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

func (r *fakeRunner) Stop(id, owner string) error {
	r.record("stop " + id + " as " + owner)
	r.exit(id, -1)
	return nil
}

func (r *fakeRunner) ComposeDown(id, cwd, project, owner string) error {
	r.record(fmt.Sprintf("down %s %s %s as %s", id, cwd, project, owner))
	return nil
}

type fakeProbe struct {
	inspect    []byte
	inspectErr error
	tables     int
	tablesErr  error
	listening  atomic.Bool
}

func (p *fakeProbe) DockerInspect(context.Context, string) ([]byte, error) {
	return p.inspect, p.inspectErr
}
func (p *fakeProbe) FleetTableCount(context.Context, string) (int, error) {
	return p.tables, p.tablesErr
}
func (p *fakeProbe) ProjectRunning(context.Context, string) (bool, error) { return false, nil }
func (p *fakeProbe) Listening(uint16) bool                                { return p.listening.Load() }

// inspectJSON is docker inspect output for a mysql container.
func inspectJSON(datadirArg string, mounts ...string) []byte {
	var ms []string
	for _, m := range mounts {
		dest, name, _ := strings.Cut(m, "=")
		ms = append(ms, fmt.Sprintf(`{"Type":"volume","Name":%q,"Source":"/var/lib/docker/volumes/x","Destination":%q}`, name, dest))
	}
	args := `["mysqld","--enforce-gtid-consistency=ON"]`
	if datadirArg != "" {
		args = fmt.Sprintf(`["mysqld",%q,"--enforce-gtid-consistency=ON"]`, datadirArg)
	}
	return fmt.Appendf(nil, `[{"Path":"docker-entrypoint.sh","Args":%s,"Mounts":[%s]}]`, args, strings.Join(ms, ","))
}

// helperOverride is a provisioned override: per-worktree volume names and an
// optional data generation.
func helperOverride(id, generation string) string {
	suffix, x := "", ""
	if generation != "" {
		suffix = "-" + generation
		x = fmt.Sprintf(`, "x-fleet-workflow": {"worktree": "/w", "generation": %q}`, generation)
	}
	return fmt.Sprintf(`{"services": {"mysql": {"volumes": [{"type": "volume", "source": "fleet-workflow-mysql-data", "target": "/var/lib/mysql"}]}},
 "volumes": {"mysql-persistent-volume": {"name": "fleet-task-%[1]s-mysql"},
             "data-s3": {"name": "fleet-task-%[1]s-s3%[2]s"},
             "fleet-workflow-mysql-data": {"name": "fleet-task-%[1]s-mysql-data%[2]s"}}%[3]s}`, id, suffix, x)
}

func profile(worktree string) settings.ServerProfile {
	cfg := "fleet.yml"
	branch := "feature"
	return settings.ServerProfile{
		ID: "s2", WorktreePath: &worktree, Branch: &branch,
		Ports:          settings.ServerPorts{Server: 8090, MySQL: 3326, Redis: 6389, S3: 9020, S3Console: 9011},
		ComposeProject: "fleet-s2",
		FleetServe: settings.FleetServeConfig{ConfigPath: &cfg, Premium: true, Debug: true, Logging: false, Env: []settings.EnvVar{
			{Key: " FOO ", Value: "1", Enabled: true}, {Key: "OFF", Value: "x", Enabled: false}, {Key: " ", Value: "y", Enabled: true},
		}},
		Enabled: true,
	}
}

type harness struct {
	r   *fakeRunner
	p   *fakeProbe
	s   *Sequencer
	wt  string
	dir string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	wt := t.TempDir()
	if err := os.WriteFile(filepath.Join(wt, overrideFile), []byte(helperOverride("abc", "")), 0o600); err != nil {
		t.Fatal(err)
	}
	h := &harness{r: newRunner(), p: &fakeProbe{inspect: inspectJSON("--datadir=/tmp/mysqldata", "/tmp=fleet-task-abc-mysql")}, wt: wt, dir: t.TempDir()}
	running := func() []string {
		h.r.mu.Lock()
		defer h.r.mu.Unlock()
		var ids []string
		for id := range h.r.live {
			ids = append(ids, id)
		}
		return ids
	}
	h.s = &Sequencer{
		Runner: h.r, Probe: h.p, Slot: profile(wt), Owner: "job-1", BackupsDir: h.dir, Running: running,
		Now: func() time.Time { return time.Date(2026, 11, 12, 14, 30, 5, 0, time.FixedZone("x", 3600)) },
	}
	// Backup steps write whatever gzipBody holds to the redirect target.
	h.r.onStart = func(id string, a processes.StartArgs) {
		if id != "s2:db-backup" {
			return
		}
		cmd := a.Args[1]
		target := strings.Trim(cmd[strings.LastIndex(cmd, "> ")+2:], "'")
		if err := os.WriteFile(target, gzipped("-- dump"), 0o600); err != nil {
			t.Error(err)
		}
	}
	return h
}

func gzipped(s string) []byte {
	var b bytes.Buffer
	zw := gzip.NewWriter(&b)
	_, _ = zw.Write([]byte(s))
	_ = zw.Close()
	return b.Bytes()
}

func (h *harness) backups(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(h.dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestBuilders(t *testing.T) {
	s := profile("/w/task")
	eq := func(name string, got, want any) {
		t.Helper()
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s:\n got  %v\n want %v", name, got, want)
		}
	}
	eq("serve", ServeArgs(s), []string{"serve", "--dev", "--config", "fleet.yml", "--dev_license", "--debug",
		"--server_address", "localhost:8090", "--mysql_address", "localhost:3326", "--redis_address", "localhost:6389"})
	eq("serve env", ServeEnv(s), []processes.EnvPair{{Key: "FOO", Value: "1"},
		{Key: "FLEET_S3_SOFTWARE_INSTALLERS_ENDPOINT_URL", Value: "http://localhost:9020"},
		{Key: "FLEET_S3_CARVES_ENDPOINT_URL", Value: "http://localhost:9020"}})
	eq("prepare", PrepareDBArgs(s), []string{"prepare", "db", "--dev", "--mysql_address=localhost:3326",
		"--mysql_username=root", "--mysql_password=toor", "--mysql_database=fleet"})
	eq("up", DockerUpArgs(s), []string{"compose", "-p", "fleet-s2", "up", "-d", "mysql", "redis", "s3"})
	eq("docker env", DockerEnv(s), []processes.EnvPair{{Key: "FLEET_MYSQL_PORT", Value: "3326"},
		{Key: "FLEET_REDIS_PORT", Value: "6389"}, {Key: "FLEET_S3_PORT", Value: "9020"}, {Key: "FLEET_S3_CONSOLE_PORT", Value: "9011"}})
	eq("backup", DBBackupCommand(s, "/a/b c/x.sql.gz"), "docker run --rm --network 'fleet-s2_default' mysql:8.0.44 "+
		"bash -c 'set -o pipefail; mysqldump -hmysql -uroot -ptoor --default-character-set=utf8mb4 "+
		"--add-drop-database --databases fleet | gzip -' > '/a/b c/x.sql.gz'")
	eq("quote", shellQuote("it's"), `'it'\''s'`)
	eq("label", ServeLabel(s), "fleet serve --dev (premium)")
	var ids []string
	for _, st := range BuildChain(s) {
		ids = append(ids, st.ID+"="+st.Program+" "+strings.Join(st.Args, " "))
	}
	eq("build", ids, []string{"s2:make-deps=make deps", "s2:make-generate=make generate", "s2:make-build=make build"})
}

func TestBuildStopsOnFirstFailure(t *testing.T) {
	h := newHarness(t)
	h.r.codes["s2:make-generate"] = 2
	_, err := h.s.Run(t.Context(), ActionBuild, "")
	if err == nil || !strings.Contains(err.Error(), "code 2") {
		t.Fatalf("err = %v", err)
	}
	if got := h.r.Calls(); !slices.Equal(got, []string{"start s2:make-deps", "start s2:make-generate"}) {
		t.Fatalf("calls = %v", got)
	}
	a := h.r.args["s2:make-deps"]
	if a.Owner != "job-1" || a.LogChannel != "agent-s2" || a.Cwd != h.wt {
		t.Fatalf("args = %+v", a)
	}
}

func TestExternallyStoppedStepFails(t *testing.T) {
	h := newHarness(t)
	h.r.block["s2:make-deps"] = true
	errc := make(chan error, 1)
	go func() { _, err := h.s.Run(t.Context(), ActionBuild, ""); errc <- err }()
	waitFor(t, func() bool { return len(h.r.Calls()) == 1 })
	h.r.exit("s2:make-deps", -1) // the GUI stopped it; the Manager would call that "done"
	if err := <-errc; err == nil || !strings.Contains(err.Error(), "code -1") {
		t.Fatalf("err = %v", err)
	}
}

func TestBeforeErrorAbortsStep(t *testing.T) {
	h := newHarness(t)
	n := 0
	h.s.Before = func() error {
		if n++; n == 2 {
			return errors.New("reservation changed")
		}
		return nil
	}
	if _, err := h.s.Run(t.Context(), ActionBuild, ""); err == nil || !strings.Contains(err.Error(), "reservation changed") {
		t.Fatalf("err = %v", err)
	}
	if got := h.r.Calls(); len(got) != 1 {
		t.Fatalf("calls = %v", got)
	}
}

func TestCancelStopsCurrentStep(t *testing.T) {
	h := newHarness(t)
	h.r.block["s2:make-deps"] = true
	ctx, cancel := context.WithCancel(t.Context())
	errc := make(chan error, 1)
	go func() { _, err := h.s.Run(ctx, ActionBuild, ""); errc <- err }()
	waitFor(t, func() bool { return len(h.r.Calls()) == 1 })
	cancel()
	if err := <-errc; !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if got := h.r.Calls(); !slices.Equal(got, []string{"start s2:make-deps", "stop s2:make-deps as job-1"}) {
		t.Fatalf("calls = %v", got)
	}
}

func TestCancelledContextStartsNothing(t *testing.T) {
	h := newHarness(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := h.s.Run(ctx, ActionBuild, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	if got := h.r.Calls(); len(got) != 0 {
		t.Fatalf("calls = %v", got)
	}
}

func TestDockerUpDownsFirst(t *testing.T) {
	h := newHarness(t)
	if _, err := h.s.Run(t.Context(), ActionDockerUp, ""); err != nil {
		t.Fatal(err)
	}
	want := []string{"down s2:docker-compose-up " + h.wt + " fleet-s2 as job-1", "start s2:docker-compose-up"}
	if got := h.r.Calls(); !slices.Equal(got, want) {
		t.Fatalf("calls = %v", got)
	}
	if a := h.r.args["s2:docker-compose-up"]; a.Program != "docker" || len(a.Env) != 4 {
		t.Fatalf("args = %+v", a)
	}
}

func TestPrepareDBCheckpoint(t *testing.T) {
	t.Run("non-empty schema", func(t *testing.T) {
		h := newHarness(t)
		h.p.tables = 12
		res, err := h.s.Run(t.Context(), ActionPrepareDB, "")
		if err != nil {
			t.Fatal(err)
		}
		if got := h.r.Calls(); !slices.Equal(got, []string{"start s2:db-backup", "start s2:fleet-prepare-db"}) {
			t.Fatalf("calls = %v", got)
		}
		want := filepath.Join(h.dir, "checkpoint-s2-20261112T133005Z.sql.gz")
		if res.BackupPath != want {
			t.Fatalf("backup = %q", res.BackupPath)
		}
		if got := h.backups(t); !slices.Equal(got, []string{"checkpoint-s2-20261112T133005Z.sql.gz", "checkpoint-s2-20261112T133005Z.sql.gz.json"}) {
			t.Fatalf("dir = %v", got)
		}
	})
	t.Run("no override still checkpoints", func(t *testing.T) {
		h := newHarness(t)
		if err := os.Remove(filepath.Join(h.wt, overrideFile)); err != nil {
			t.Fatal(err)
		}
		h.p.inspect = inspectJSON("--datadir=/tmp/mysqldata", "/tmp=fleet-s2_mysql-persistent-volume")
		h.p.tables = 5
		if _, err := h.s.Run(t.Context(), ActionPrepareDB, ""); err != nil {
			t.Fatal(err)
		}
		if got := h.r.Calls(); !slices.Equal(got, []string{"start s2:db-backup", "start s2:fleet-prepare-db"}) {
			t.Fatalf("calls = %v", got)
		}
	})
	t.Run("no override refuses another task's data", func(t *testing.T) {
		h := newHarness(t)
		if err := os.Remove(filepath.Join(h.wt, overrideFile)); err != nil {
			t.Fatal(err)
		}
		h.p.tables = 5 // harness inspect mounts fleet-task-abc-mysql, not the project's volume
		if _, err := h.s.Run(t.Context(), ActionPrepareDB, ""); err == nil {
			t.Fatal("prepare-db ran against another task's volume")
		}
		if got := h.r.Calls(); len(got) != 0 {
			t.Fatalf("calls = %v", got)
		}
	})
	t.Run("empty schema", func(t *testing.T) {
		h := newHarness(t)
		if _, err := h.s.Run(t.Context(), ActionPrepareDB, ""); err != nil {
			t.Fatal(err)
		}
		if got := h.r.Calls(); !slices.Equal(got, []string{"start s2:fleet-prepare-db"}) {
			t.Fatalf("calls = %v", got)
		}
	})
	t.Run("checkpoint failure aborts", func(t *testing.T) {
		h := newHarness(t)
		h.p.tables = 1
		h.r.codes["s2:db-backup"] = 1
		if _, err := h.s.Run(t.Context(), ActionPrepareDB, ""); err == nil || !strings.Contains(err.Error(), "checkpoint") {
			t.Fatalf("err = %v", err)
		}
		if got := h.r.Calls(); !slices.Equal(got, []string{"start s2:db-backup"}) {
			t.Fatalf("calls = %v", got)
		}
	})
	t.Run("table count error refuses", func(t *testing.T) {
		h := newHarness(t)
		h.p.tablesErr = errors.New("exec failed")
		if _, err := h.s.Run(t.Context(), ActionPrepareDB, ""); err == nil {
			t.Fatal("expected error")
		}
		if got := h.r.Calls(); len(got) != 0 {
			t.Fatalf("calls = %v", got)
		}
	})
}

func TestCheckMount(t *testing.T) {
	cases := []struct {
		name     string
		override string
		inspect  []byte
		probeErr error
		wantErr  string
	}{
		{name: "this branch /tmp datadir", override: helperOverride("abc", ""),
			inspect: inspectJSON("--datadir=/tmp/mysqldata", "/tmp=fleet-task-abc-mysql", "/var/lib/mysql=fleet-task-abc-mysql-data")},
		{name: "newer /var/lib/mysql datadir", override: helperOverride("abc", "0123456789ab"),
			inspect: inspectJSON("", "/tmp=fleet-task-abc-mysql", "/var/lib/mysql=fleet-task-abc-mysql-data-0123456789ab")},
		{name: "fresh generation but /tmp holds the old DB", override: helperOverride("abc", "0123456789ab"),
			inspect: inspectJSON("--datadir=/tmp/mysqldata", "/tmp=fleet-task-abc-mysql", "/var/lib/mysql=fleet-task-abc-mysql-data-0123456789ab"),
			wantErr: "generation"},
		{name: "volume not in override", override: helperOverride("abc", ""),
			inspect: inspectJSON("--datadir=/tmp/mysqldata", "/tmp=fleet_mysql-persistent-volume"), wantErr: "doesn't name"},
		{name: "no mount above datadir", override: helperOverride("abc", ""),
			inspect: inspectJSON("--datadir=/data/mysql", "/tmp=fleet-task-abc-mysql"), wantErr: "no mount"},
		{name: "probe error", override: helperOverride("abc", ""), probeErr: errors.New("daemon down"), wantErr: "daemon down"},
		{name: "no override, base project volume",
			inspect: inspectJSON("--datadir=/tmp/mysqldata", "/tmp=fleet-s2_mysql-persistent-volume")},
		{name: "no override, foreign volume", wantErr: "fleet-s2_mysql-persistent-volume",
			inspect: inspectJSON("--datadir=/tmp/mysqldata", "/tmp=fleet-task-abc-mysql")},
		{name: "no override, probe error", probeErr: errors.New("daemon down"), wantErr: "daemon down"},
		{name: "no override, no container yet", probeErr: fmt.Errorf("%w: fleet-s2-mysql-1", ErrNoContainer)},
		{name: "override, no container", override: helperOverride("abc", ""),
			probeErr: fmt.Errorf("%w: fleet-s2-mysql-1", ErrNoContainer), wantErr: "no such container"},
		{name: "unparseable override", override: "volumes: [", probeErr: errors.New("inspect must not run"), wantErr: "parsing"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			wt := t.TempDir()
			if c.override != "" {
				if err := os.WriteFile(filepath.Join(wt, overrideFile), []byte(c.override), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := CheckMount(t.Context(), &fakeProbe{inspect: c.inspect, inspectErr: c.probeErr}, wt, "fleet-s2")
			if c.wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if c.wantErr != "" && (err == nil || !strings.Contains(err.Error(), c.wantErr)) {
				t.Fatalf("err = %v, want %q", err, c.wantErr)
			}
		})
	}
	t.Run("unreadable override", func(t *testing.T) {
		wt := t.TempDir()
		if err := os.Mkdir(filepath.Join(wt, overrideFile), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := CheckMount(t.Context(), &fakeProbe{}, wt, "fleet-s2"); err == nil || !strings.Contains(err.Error(), "reading") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestServe(t *testing.T) {
	t.Run("ready while running", func(t *testing.T) {
		h := newHarness(t)
		h.r.block["s2:fleet-serve"] = true
		h.r.onStart = func(id string, _ processes.StartArgs) { h.p.listening.Store(true) }
		if _, err := h.s.Run(t.Context(), ActionServe, ""); err != nil {
			t.Fatal(err)
		}
		if got := h.r.Calls(); !slices.Equal(got, []string{"stop s2:fleet-serve as job-1", "start s2:fleet-serve"}) {
			t.Fatalf("calls = %v", got)
		}
		if a := h.r.args["s2:fleet-serve"]; a.LogChannel != "fleet-serve-s2" || a.Owner != "job-1" {
			t.Fatalf("args = %+v", a)
		}
	})
	t.Run("occupied port refused", func(t *testing.T) {
		h := newHarness(t)
		h.p.listening.Store(true)
		if _, err := h.s.Run(t.Context(), ActionServe, ""); err == nil || !strings.Contains(err.Error(), "listener") {
			t.Fatalf("err = %v", err)
		}
		if got := h.r.Calls(); slices.Contains(got, "start s2:fleet-serve") {
			t.Fatalf("calls = %v", got)
		}
	})
	t.Run("exits before listening", func(t *testing.T) {
		h := newHarness(t)
		h.r.codes["s2:fleet-serve"] = 1
		if _, err := h.s.Run(t.Context(), ActionServe, ""); err == nil || !strings.Contains(err.Error(), "before listening") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("mount check refuses", func(t *testing.T) {
		h := newHarness(t)
		h.p.inspectErr = errors.New("no such container")
		if _, err := h.s.Run(t.Context(), ActionServe, ""); err == nil {
			t.Fatal("expected error")
		}
		if got := h.r.Calls(); len(got) != 0 {
			t.Fatalf("calls = %v", got)
		}
	})
}

func TestBackup(t *testing.T) {
	t.Run("publishes and writes sidecar", func(t *testing.T) {
		h := newHarness(t)
		res, err := h.s.Run(t.Context(), ActionBackup, "mine.sql.gz")
		if err != nil {
			t.Fatal(err)
		}
		if res.BackupPath != filepath.Join(h.dir, "mine.sql.gz") {
			t.Fatalf("path = %q", res.BackupPath)
		}
		if got := h.backups(t); !slices.Equal(got, []string{"mine.sql.gz", "mine.sql.gz.json"}) {
			t.Fatalf("dir = %v", got)
		}
	})
	t.Run("dump failure publishes nothing", func(t *testing.T) {
		h := newHarness(t)
		h.r.codes["s2:db-backup"] = 1
		if _, err := h.s.Run(t.Context(), ActionBackup, "mine.sql.gz"); err == nil {
			t.Fatal("expected error")
		}
		if got := h.backups(t); len(got) != 0 {
			t.Fatalf("dir = %v", got)
		}
	})
	t.Run("bad gzip publishes nothing", func(t *testing.T) {
		h := newHarness(t)
		h.r.onStart = func(id string, a processes.StartArgs) {
			cmd := a.Args[1]
			full := gzipped("-- dump")
			_ = os.WriteFile(strings.Trim(cmd[strings.LastIndex(cmd, "> ")+2:], "'"), full[:len(full)-6], 0o600)
		}
		if _, err := h.s.Run(t.Context(), ActionBackup, "mine.sql.gz"); err == nil || !strings.Contains(err.Error(), "gzip") {
			t.Fatalf("err = %v", err)
		}
		if got := h.backups(t); len(got) != 0 {
			t.Fatalf("dir = %v", got)
		}
	})
	t.Run("existing name refused", func(t *testing.T) {
		h := newHarness(t)
		if err := os.WriteFile(filepath.Join(h.dir, "mine.sql.gz"), []byte("keep"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := h.s.Run(t.Context(), ActionBackup, "mine.sql.gz"); !errors.Is(err, ErrBackupExists) {
			t.Fatalf("err = %v", err)
		}
		if got := h.r.Calls(); len(got) != 0 {
			t.Fatalf("calls = %v", got)
		}
	})
	t.Run("name taken during dump isn't replaced", func(t *testing.T) {
		h := newHarness(t)
		prev := h.r.onStart
		h.r.onStart = func(id string, a processes.StartArgs) {
			prev(id, a)
			_ = os.WriteFile(filepath.Join(h.dir, "mine.sql.gz"), []byte("keep"), 0o600)
		}
		if _, err := h.s.Run(t.Context(), ActionBackup, "mine.sql.gz"); !errors.Is(err, ErrBackupExists) {
			t.Fatalf("err = %v", err)
		}
		if b, _ := os.ReadFile(filepath.Join(h.dir, "mine.sql.gz")); string(b) != "keep" {
			t.Fatal("existing file replaced")
		}
		if got := h.backups(t); !slices.Equal(got, []string{"mine.sql.gz"}) {
			t.Fatalf("dir = %v", got)
		}
	})
}

func TestStopOrder(t *testing.T) {
	h := newHarness(t)
	h.r.block["s2:fleet-serve"] = true
	h.r.block["s2:make-build"] = true
	_, _ = h.r.Start("s2:fleet-serve", processes.StartArgs{})
	_, _ = h.r.Start("s2:make-build", processes.StartArgs{})
	h.r.calls = nil
	if _, err := h.s.Run(t.Context(), ActionStop, ""); err != nil {
		t.Fatal(err)
	}
	got := h.r.Calls()
	if len(got) != 3 || !strings.HasPrefix(got[0], "stop ") || !strings.HasPrefix(got[1], "stop ") ||
		got[2] != "down s2:docker-compose-up "+h.wt+" fleet-s2 as job-1" {
		t.Fatalf("calls = %v", got)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); time.Sleep(time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatal("condition not met")
}
