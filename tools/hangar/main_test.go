package main

import (
	"testing"

	"github.com/fleetdm/fleet/tools/hangar/internal/applog"
	"github.com/fleetdm/fleet/tools/hangar/internal/paths"
)

func TestBootstrapLocksBeforeLogSetupAndReap(t *testing.T) {
	dataDir, logDir := t.TempDir(), t.TempDir()
	var steps []string
	setupLog := func(string) *applog.Session { steps = append(steps, "log"); return nil }
	reap := func(string) { steps = append(steps, "reap") }

	release, err := paths.LockDataDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := bootstrap(logDir, dataDir, setupLog, reap); err == nil {
		t.Fatal("bootstrap should fail while another holder has the data dir")
	}
	if len(steps) != 0 {
		t.Fatalf("ran %v while the data dir was held", steps)
	}
	release()

	_, unlock, err := bootstrap(logDir, dataDir, setupLog, reap)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if len(steps) != 2 || steps[0] != "log" || steps[1] != "reap" {
		t.Fatalf("steps = %v, want [log reap]", steps)
	}
}
