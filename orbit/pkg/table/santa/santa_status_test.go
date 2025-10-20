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
	require.Equal(t, "Monitor", row["mode"])
	require.Equal(t, "0", row["watchdog_cpu_events"])
	require.Equal(t, "3", row["watchdog_ram_events"])
	require.Equal(t, "175", row["root_cache_count"])
	require.Equal(t, "0", row["watch_items_enabled"])
	require.Equal(t, "1", row["file_logging"])
	require.Equal(t, "file", row["log_type"])
	require.Equal(t, "6", row["static_rule_count"])
	require.Equal(t, "rdonly", row["remount_usb_mode"])
	require.Equal(t, "0", row["sync_enabled"])
	require.Equal(t, "0", row["metrics_enabled"])
	require.Equal(t, "0", row["events_pending_upload"])
	// float formatting should be plain (no trailing zeros or scientific unless big)
	require.True(t, strings.Contains(row["watchdog_ram_peak"], "252.453125"))
	require.True(t, strings.Contains(row["watchdog_cpu_peak"], "4.759"))
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
		"file_logging", "mode", "watchdog_cpu_events", "watch_items_enabled",
		"sync_enabled", "metrics_enabled", "events_pending_upload",
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
	"daemon" : {
		"watchdog_ram_events" : 3,
		"block_usb" : false,
		"log_type" : "file",
		"mode" : "Monitor",
		"watchdog_cpu_events" : 0,
		"static_rules" : 6,
		"watchdog_ram_peak" : 252.453125,
		"watchdog_cpu_peak" : 4.7590483333333333,
		"file_logging" : true,
		"remount_usb_mode" : "rdonly",
		"on_start_usb_options" : "None"
	},
	"sync" : {
		"enabled" : false
	},
	"rule_types" : {
		"cdhash_rules" : 0,
		"teamid_rules" : 4,
		"certificate_rules" : 0,
		"signingid_rules" : 1,
		"binary_rules" : 1
	},
	"cache" : {
		"root_cache_count" : 175,
		"non_root_cache_count" : 5
	},
	"watch_items" : {
		"enabled" : false
	},
	"metrics" : {
		"enabled" : false
	},
	"transitive_allowlisting" : {
		"enabled" : false,
		"compiler_rules" : 0,
		"transitive_rules" : 0
	}
}`
}
