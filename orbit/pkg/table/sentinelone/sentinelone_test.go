package sentinelone

import (
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestColumns(t *testing.T) {
	t.Parallel()

	cols := Columns()
	require.Len(t, cols, len(columnOrder))
	for i, col := range cols {
		assert.Equal(t, columnOrder[i], col.Name)
		assert.Equal(t, table.ColumnTypeText, col.Type)
	}
}

// TestColumnPathMapMatchesColumnOrder guards against the two column lists
// drifting apart: a path mapped to a column that isn't declared would be
// silently dropped from every row.
func TestColumnPathMapMatchesColumnOrder(t *testing.T) {
	t.Parallel()

	declared := make(map[string]struct{}, len(columnOrder))
	for _, col := range columnOrder {
		_, duplicate := declared[col]
		assert.False(t, duplicate, "column %q declared twice", col)
		declared[col] = struct{}{}
	}

	mapped := make(map[string]struct{}, len(columnPathMap))
	for path, col := range columnPathMap {
		assert.Contains(t, declared, col, "path %q maps to undeclared column %q", path, col)
		mapped[col] = struct{}{}
	}
	for _, col := range columnOrder {
		assert.Contains(t, mapped, col, "column %q is never populated by any path", col)
	}
	for _, col := range epochColumns {
		assert.Contains(t, declared, col, "epoch column %q is not declared", col)
	}
}

func TestGenerateDarwinStatus(t *testing.T) {
	mockStatusOutput(t, darwinStatusFixture)

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]

	// Timestamps are converted in the host's local zone, so derive the
	// expectation the same way instead of hardcoding an epoch.
	wantValues := map[string]string{
		"agent_version":                    "25.3.4.8365",
		"agent_id":                         "d7f94b33-7f99-4c43-9c6b-1c6e0d01aabb",
		"install_date":                     localEpoch(t, "6/1/26, 10:09:51 AM"),
		"es_framework":                     "started",
		"operational_state":                "enabled",
		"remote_profiler":                  "not running",
		"network_monitoring":               "started",
		"network_extension":                "running",
		"network_extension_content_filter": "active",
		"ready":                            "yes",
		"protection":                       "enabled",
		"infected":                         "no",
		"network_quarantine":               "no",
		"compatible_os":                    "compatible",
		"command_authentication":           "enabled",
		"service_agent_helper":             "ready",
		"service_agent_ui":                 "ready",
		"service_cleaner":                  "ready",
		"service_control_service":          "ready",
		"service_framework":                "ready",
		"service_guard":                    "ready",
		"service_helper_service":           "ready",
		"service_lib_hooks_service":        "not ready",
		"service_lib_logs_service":         "not ready",
		"service_shell":                    "ready",
		"integrity_sentineld":              "ok",
		"integrity_sentineld_guard":        "ok",
		"integrity_sentineld_helper":       "ok",
		"integrity_sentineld_shell":        "not running",
		"launchd_agent_helper":             "valid",
		"launchd_agent_ui":                 "valid",
		"launchd_sentinel_extensions":      "valid",
		"launchd_sentineld":                "valid",
		"launchd_sentineld_guard":          "valid",
		"launchd_sentineld_helper":         "valid",
		"launchd_sentineld_shell":          "valid",
		"management_server":                "https://euce1-109.sentinelone.net",
		"management_site_key":              "site-key-abc-123",
		"management_last_seen":             localEpoch(t, "6/1/26, 1:14:28 PM"),
		"management_connected":             "yes",
		// Windows-only columns stay empty on macOS.
		"tamper_protection": "",
		"agent_run_mode":    "",
		"mitigation_policy": "",
	}
	for col, want := range wantValues {
		assert.Equal(t, want, row[col], "column %q", col)
	}

	// Every declared column is present, so osquery never sees a partial row.
	assert.Len(t, row, len(columnOrder))
	for _, col := range columnOrder {
		assert.Contains(t, row, col)
	}
}

func TestGenerateWindowsStatus(t *testing.T) {
	mockStatusOutput(t, windowsStatusFixture)

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]

	wantValues := map[string]string{
		"agent_version":      "25.1.4.434",
		"operational_state":  "enabled",
		"tamper_protection":  "enabled",
		"network_monitoring": "started",
		"es_framework":       "started",
		"ready":              "yes",
		"agent_run_mode":     "running as PPL",
		"mitigation_policy":  "none",
		// Windows sentinelctl reports no threat protection state, and no
		// management or daemon detail.
		"protection":           "",
		"management_server":    "",
		"management_connected": "",
		"service_agent_helper": "",
		"install_date":         "",
	}
	for col, want := range wantValues {
		assert.Equal(t, want, row[col], "column %q", col)
	}
	assert.Len(t, row, len(columnOrder))
}

func TestGenerateNoRows(t *testing.T) {
	tests := []struct {
		name string
		out  string
		err  error
	}{
		{
			name: "sentinelctl not installed",
			err:  errors.New(`exec: "sentinelctl": executable file not found in $PATH`),
		},
		{
			name: "no recognized fields",
			out:  "banana slug !@#$%\n",
		},
		{
			name: "empty output",
			out:  "",
		},
		{
			name: "only unmapped fields",
			out:  "Agent\n   Some Future Field: 1\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSentinelctl(t, func([]string) ([]byte, error) {
				return []byte(tt.out), tt.err
			})

			rows, err := Generate(t.Context(), table.QueryContext{})
			require.NoError(t, err, "a missing or broken agent must not error the query")
			assert.Empty(t, rows)
		})
	}
}

func TestGeneratePartialStatus(t *testing.T) {
	mockStatusOutput(t, "Management\n   Server:    https://example.net\n   Connected: yes\n")

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, "https://example.net", rows[0]["management_server"])
	assert.Equal(t, "yes", rows[0]["management_connected"])
	assert.Empty(t, rows[0]["agent_version"])
}

// TestGenerateClearsUnparseableTimestamp covers a status output whose dates
// can't be read: the epoch columns must come back empty rather than carrying
// text that a numeric comparison in SQL would coerce to 0.
func TestGenerateClearsUnparseableTimestamp(t *testing.T) {
	mockStatusOutput(t,
		"Agent\n"+
			"   Version:      25.3.4.8365\n"+
			"   Install Date: not-a-date\n"+
			"Management\n"+
			"   Last Seen:    also-garbage\n"+
			"   Connected:    yes\n",
	)

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	row := rows[0]

	assert.Empty(t, row["install_date"])
	assert.Empty(t, row["management_last_seen"])
	assert.Equal(t, "25.3.4.8365", row["agent_version"])
	assert.Equal(t, "yes", row["management_connected"])
}

// localEpoch returns value, parsed in the host's zone, as Unix epoch seconds.
func localEpoch(t *testing.T, value string) string {
	t.Helper()
	parsed, err := time.ParseInLocation("1/2/06, 3:04:05 PM", value, time.Local)
	require.NoError(t, err)
	return strconv.FormatInt(parsed.Unix(), 10)
}
