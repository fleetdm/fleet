package settings

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

type storeFixture struct {
	dir      string
	store    *Store
	events   atomic.Int32
	reserved map[string]bool
	resErr   error
}

func newFixture(t *testing.T) *storeFixture {
	t.Helper()
	f := &storeFixture{dir: t.TempDir(), reserved: map[string]bool{"s2": true}}
	s := Default()
	a := defaultServer(0)
	b := defaultServer(1)
	b.WorktreePath = ptr("/wt/task")
	b.Branch = ptr("task")
	b.FleetServe.ConfigPath = ptr("/wt/task/fleet.yml")
	s.Servers = []ServerProfile{a, b}
	if err := Save(f.dir, s); err != nil {
		t.Fatal(err)
	}
	st, err := NewStore(f.dir, func(string) { f.events.Add(1) }, func() (map[string]bool, error) {
		return f.reserved, f.resErr
	})
	if err != nil {
		t.Fatal(err)
	}
	f.store = st
	return f
}

func ptr(s string) *string { return &s }

func server(s Settings, id string) *ServerProfile {
	for i := range s.Servers {
		if s.Servers[i].ID == id {
			return &s.Servers[i]
		}
	}
	return nil
}

func TestSaveModeAndNoTempLeft(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, Default()); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, fileName))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", fi.Mode().Perm())
	}
	ents, _ := os.ReadDir(dir)
	if len(ents) != 1 {
		t.Errorf("dir has %d entries, want only settings.json", len(ents))
	}
}

func TestSaveFromGUIKeepsReservedIdentity(t *testing.T) {
	f := newFixture(t)
	in := f.store.Snapshot()
	p := server(in, "s2")
	p.WorktreePath = ptr("/wt/other")
	p.Branch = ptr("other")
	p.FleetServe.ConfigPath = ptr("/wt/other/fleet.yml")
	p.Ports.Server = 9999
	p.Enabled = false
	p.FleetServe.Debug = !p.FleetServe.Debug
	in.Theme = ThemeDark

	out, err := f.store.SaveFromGUI(in)
	if err != nil {
		t.Fatal(err)
	}
	got := server(out, "s2")
	if *got.WorktreePath != "/wt/task" || *got.Branch != "task" || *got.FleetServe.ConfigPath != "/wt/task/fleet.yml" {
		t.Errorf("identity not kept: %+v", got)
	}
	if got.Ports != defaultServer(1).Ports || !got.Enabled {
		t.Errorf("ports/enabled not kept: %+v", got)
	}
	if got.FleetServe.Debug != p.FleetServe.Debug || out.Theme != ThemeDark {
		t.Error("non-identity fields should come from input")
	}
	if f.events.Load() != 1 {
		t.Errorf("events = %d, want 1", f.events.Load())
	}
	disk, _ := Load(f.dir)
	if *server(disk, "s2").WorktreePath != "/wt/task" {
		t.Error("disk has the reverted worktree")
	}
}

func TestSaveFromGUIStaleSaveAfterAgentAssign(t *testing.T) {
	f := newFixture(t)
	stale := f.store.Snapshot()
	if _, err := f.store.Update(func(s *Settings) error {
		server(*s, "s2").WorktreePath = ptr("/wt/assigned")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	stale.Theme = ThemeLight
	out, err := f.store.SaveFromGUI(stale)
	if err != nil {
		t.Fatal(err)
	}
	if *server(out, "s2").WorktreePath != "/wt/assigned" {
		t.Errorf("stale save replayed an old worktree: %v", *server(out, "s2").WorktreePath)
	}
	if out.Theme != ThemeLight {
		t.Error("unrelated field lost")
	}
}

func TestSaveFromGUIPutsBackRemovedReserved(t *testing.T) {
	f := newFixture(t)
	in := f.store.Snapshot()
	in.Servers = in.Servers[:1]
	in.ActiveServerID = "s1"
	out, err := f.store.SaveFromGUI(in)
	if err != nil {
		t.Fatal(err)
	}
	if server(out, "s2") == nil || len(out.Servers) != 2 || out.Servers[1].ID != "s2" {
		t.Errorf("reserved profile not restored in place: %+v", out.Servers)
	}
	if f.events.Load() != 1 {
		t.Errorf("events = %d, want 1", f.events.Load())
	}
}

func TestSaveFromGUIEnvEditOnReservedSlotSaves(t *testing.T) {
	f := newFixture(t)
	in := f.store.Snapshot()
	server(in, "s2").FleetServe.Env = []EnvVar{{Key: "A", Value: "1", Enabled: true}}
	out, err := f.store.SaveFromGUI(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(server(out, "s2").FleetServe.Env) != 1 {
		t.Error("env row not saved")
	}
	if f.events.Load() != 0 {
		t.Errorf("events = %d, want 0 when nothing was kept", f.events.Load())
	}
}

func TestSaveFromGUIUnreservedSlotChanges(t *testing.T) {
	f := newFixture(t)
	f.reserved = map[string]bool{}
	in := f.store.Snapshot()
	server(in, "s2").WorktreePath = ptr("/wt/new")
	out, err := f.store.SaveFromGUI(in)
	if err != nil {
		t.Fatal(err)
	}
	if *server(out, "s2").WorktreePath != "/wt/new" {
		t.Error("unreserved slot should take the GUI worktree")
	}
}

func TestSaveFromGUIHeldIdentityRefused(t *testing.T) {
	f := newFixture(t)
	f.store.SetHeldBy(func(id string) (string, bool) { return "job-7", id == "s2" })
	in := f.store.Snapshot()
	server(in, "s2").Ports.MySQL = 4000
	_, err := f.store.SaveFromGUI(in)
	if err == nil || !strings.Contains(err.Error(), "job-7") {
		t.Fatalf("err = %v, want one naming job-7", err)
	}
	// Removal while idle is put back; while held with unchanged fields it saves.
	in = f.store.Snapshot()
	in.Theme = ThemeDark
	if _, err := f.store.SaveFromGUI(in); err != nil {
		t.Errorf("unchanged held slot should save: %v", err)
	}
	if f.store.Snapshot().Servers[1].Ports.MySQL == 4000 {
		t.Error("refused save leaked into the store")
	}
}

func TestSaveFromGUIReservationErrorRefuses(t *testing.T) {
	f := newFixture(t)
	f.resErr = errors.New("bad json")
	in := f.store.Snapshot()
	in.Theme = ThemeDark
	if _, err := f.store.SaveFromGUI(in); err == nil {
		t.Fatal("want error")
	}
	if f.store.Snapshot().Theme == ThemeDark {
		t.Error("refused save changed the store")
	}
	disk, _ := Load(f.dir)
	if disk.Theme == ThemeDark {
		t.Error("refused save changed the file")
	}
}

func TestUpdateEmitsAndSerializes(t *testing.T) {
	f := newFixture(t)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := f.store.Update(func(s *Settings) error {
				s.FavoriteCrons = append(s.FavoriteCrons, "x")
				return nil
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if n := len(f.store.Snapshot().FavoriteCrons); n != 20 {
		t.Errorf("got %d appends, want 20", n)
	}
	if f.events.Load() != 20 {
		t.Errorf("events = %d, want 20", f.events.Load())
	}
	disk, _ := Load(f.dir)
	if len(disk.FavoriteCrons) != 20 {
		t.Error("disk lost updates")
	}
}

func TestUpdateErrorLeavesStateAlone(t *testing.T) {
	f := newFixture(t)
	if _, err := f.store.Update(func(s *Settings) error {
		s.Theme = ThemeDark
		return errors.New("no")
	}); err == nil {
		t.Fatal("want error")
	}
	if f.store.Snapshot().Theme == ThemeDark || f.events.Load() != 0 {
		t.Error("failed update leaked")
	}
}
