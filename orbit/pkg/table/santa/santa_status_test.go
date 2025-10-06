//go:build darwin

package santa

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

func TestGenerateStatus_HappyPath(t *testing.T) {
	t.Cleanup(func() { execCommandContext = exec.CommandContext })
	execCommandContext = fakeExecCommandContext(t, sampleStatusJSON())

	rows, err := GenerateStatus(context.Background(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]

	// spot check a few fields and types
	require.Equal(t, "2025-09-01T12:34:56Z", row["last_successful_rule"])
	require.Equal(t, "apns", row["push_notifications"])
	require.Equal(t, "1", row["bundle_scanning"]) // bool → "1"
	require.Equal(t, "0", row["clean_required"])
	require.Equal(t, "monitor", row["mode"])
	require.Equal(t, "3", row["watchdog_cpu_events"])
	require.Equal(t, "42", row["root_cache_count"])
	require.Equal(t, "1", row["watch_items_enabled"])
	// float formatting should be plain (no trailing zeros or scientific unless big)
	require.True(t, strings.Contains(row["watchdog_ram_peak"], "1024"))
}

func TestGenerateStatus_CommandErrorReturnsEmptyNoError(t *testing.T) {
	t.Cleanup(func() { execCommandContext = exec.CommandContext })
	execCommandContext = fakeExecCommandContext(t, "ERROR: nope", withExitCode(1))

	rows, err := GenerateStatus(context.Background(), table.QueryContext{})
	require.NoError(t, err)
	require.Empty(t, rows)
}

func TestGenerateStatus_BadJSONReturnsError(t *testing.T) {
	t.Cleanup(func() { execCommandContext = exec.CommandContext })
	execCommandContext = fakeExecCommandContext(t, "{not-json}")

	rows, err := GenerateStatus(context.Background(), table.QueryContext{})
	require.Error(t, err)
	require.Nil(t, rows)
}

func TestGenerateStatus_ContextCancelBehavesLikeCmdError(t *testing.T) {
	t.Cleanup(func() { execCommandContext = exec.CommandContext })
	// Simulate a slow command; we'll cancel the context before it returns
	execCommandContext = fakeExecCommandContext(t, sampleStatusJSON(), withSleep(200*time.Millisecond))

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	rows, err := GenerateStatus(ctx, table.QueryContext{})
	require.NoError(t, err) // your code treats cmd failure gracefully
	require.Empty(t, rows)
}

func TestStatusColumns_Contract(t *testing.T) {
	cols := StatusColumns()
	require.Greater(t, len(cols), 10)
	names := make(map[string]struct{}, len(cols))
	for _, c := range cols {
		names[c.Name] = struct{}{}
	}
	// a few key columns to lock contract
	for _, name := range []string{
		"last_successful_rule", "push_notifications", "bundle_scanning",
		"file_logging", "mode", "watchdog_cpu_events", "watch_items_enabled",
	} {
		if _, ok := names[name]; !ok {
			t.Fatalf("missing column %q", name)
		}
	}
}

func TestHelpers(t *testing.T) {
	require.Equal(t, "1", boolToIntString(true))
	require.Equal(t, "0", boolToIntString(false))

	// float formatting: expect compact representation
	got := floatToString(1.25)
	require.Equal(t, "1.25", got)

	got = floatToString(1024.0)
	require.Equal(t, "1024", got)
}

// ---- test helpers ----

type fakeOpt func(*fakeCfg)

type fakeCfg struct {
	exitCode int
	sleep    time.Duration
}

func withExitCode(code int) fakeOpt {
	return func(c *fakeCfg) { c.exitCode = code }
}

func withSleep(d time.Duration) fakeOpt {
	return func(c *fakeCfg) { c.sleep = d }
}

func fakeExecCommandContext(t *testing.T, payload string, opts ...fakeOpt) func(ctx context.Context, name string, args ...string) *exec.Cmd {
	t.Helper()
	cfg := &fakeCfg{}
	for _, o := range opts {
		o(cfg)
	}
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess") //nolint: gosec
		cmd.Env = append(os.Environ(),
			"GO_WANT_HELPER_PROCESS=1",
			"FAKE_PAYLOAD="+payload,
			"FAKE_EXIT_CODE="+strconv.Itoa(cfg.exitCode),
			"FAKE_SLEEP_MS="+strconv.Itoa(int(cfg.sleep.Milliseconds())),
		)
		return cmd
	}
}

// This is invoked as a subprocess by fakeExecCommandContext
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	sleepMS := os.Getenv("FAKE_SLEEP_MS")
	if sleepMS != "" && sleepMS != "0" {
		if d, _ := time.ParseDuration(sleepMS + "ms"); d > 0 {
			time.Sleep(d)
		}
	}
	exit := 0
	if v := os.Getenv("FAKE_EXIT_CODE"); v != "" && v != "0" {
		exit = 1
	}
	_, _ = os.Stdout.WriteString(os.Getenv("FAKE_PAYLOAD"))
	if exit != 0 {
		os.Exit(exit)
	}
	os.Exit(0)
}

func sampleStatusJSON() string {
	return `{
	  "watch_items": { "enabled": true },
	  "daemon": {
	    "file_logging": true,
	    "watchdog_ram_events": 5,
	    "driver_connected": true,
	    "log_type": "file",
	    "watchdog_cpu_events": 3,
	    "mode": "monitor",
	    "watchdog_cpu_peak": 1.25,
	    "watchdog_ram_peak": 1024,
	    "transitive_rules": true,
	    "remount_usb_mode": "ro",
	    "block_usb": false,
	    "on_start_usb_options": "block"
	  },
	  "cache": { "root_cache_count": 42, "non_root_cache_count": 7 },
	  "static_rules": { "rule_count": 9 },
	  "database": {
	    "certificate_rules": 1,
	    "cdhash_rules": 2,
	    "transitive_rules": 3,
	    "teamid_rules": 4,
	    "signingid_rules": 5,
	    "compiler_rules": 6,
	    "binary_rules": 7,
	    "events_pending_upload": 8
	  },
	  "sync": {
	    "last_successful_rule": "2025-09-01T12:34:56Z",
	    "push_notifications": "apns",
	    "bundle_scanning": true,
	    "clean_required": false,
	    "server": "https://example.test",
	    "last_successful_full": "2025-09-01T12:34:56Z"
	  }
	}`
}
