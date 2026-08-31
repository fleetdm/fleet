package netskope

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// healthyNsdiag is the parsed state of a connected client, as runNsdiag returns it.
func healthyNsdiag() map[string]string {
	return map[string]string{
		"orgname":          "CompanyName",
		"tenant_url":       "companyname.goskope.com",
		"addonhost":        "addon-companyname.goskope.com",
		"addoncheckerhost": "achecker-companyname.goskope.com",
		"gateway":          "gateway-xyz.goskope.com",
		"gateway_ip":       "203.0.113.10",
		"config":           "Pop Pinning Client Configuration",
		"steering_config":  "Default tenant config",
		"email":            "alice@example.com",
		"peruser_config":   "FALSE",
		"tunnel_status":    "NSTUNNEL_CONNECTED",
		"client_status":    "enable",
		"dynamic_steering": "FALSE",
		"onpremdetection":  "Not Configured",
		"explicit_proxy":   "FALSE",
		"tunnel_protocol":  "TLS",
		"sni_enable":       "FALSE",
		"traffic_mode":     "All Traffic",
		"client_version":   "117.1.0.1234",
	}
}

// mockNsdiag replaces the install-path lookup and nsdiag invocation for the
// duration of the test.
// It must not be used from a parallel test: the swapped values are
// package-level.
func mockNsdiag(t *testing.T, installPath string, parsed map[string]string, err error) {
	t.Helper()

	origInstall, origRun := detectInstallPath, runNsdiag
	t.Cleanup(func() {
		detectInstallPath, runNsdiag = origInstall, origRun
	})

	detectInstallPath = func() string { return installPath }
	runNsdiag = func(context.Context, string) (map[string]string, error) { return parsed, err }
}

func TestColumns(t *testing.T) {
	t.Parallel()

	cols := Columns()
	require.Len(t, cols, len(columnOrder))
	for i, col := range cols {
		assert.Equal(t, columnOrder[i], col.Name)
		want := table.ColumnTypeText
		if _, isInt := integerColumns[col.Name]; isInt {
			want = table.ColumnTypeInteger
		}
		assert.Equal(t, want, col.Type, "column %q has the wrong type", col.Name)
	}
}

// TestNsdiagKeyToColumnMatchesColumnOrder guards against the column list and the
// nsdiag key map drifting apart: a key mapped to an undeclared column would be
// silently dropped from every row.
func TestNsdiagKeyToColumnMatchesColumnOrder(t *testing.T) {
	t.Parallel()

	declared := make(map[string]struct{}, len(columnOrder))
	for _, col := range columnOrder {
		_, duplicate := declared[col]
		assert.False(t, duplicate, "column %q declared twice", col)
		declared[col] = struct{}{}
	}

	for key, col := range nsdiagKeyToColumn {
		assert.Contains(t, declared, col, "nsdiag key %q maps to undeclared column %q", key, col)
		assert.Equal(t, key, strings.ToLower(key), "nsdiag key %q must be lowercased to match the parser", key)
	}
	for col := range integerColumns {
		assert.Contains(t, declared, col, "integer column %q is not declared", col)
	}
}

func TestGenerateAlwaysReturnsEveryColumn(t *testing.T) {
	mockNsdiag(t, "/Library/Application Support/Netskope/STAgent", map[string]string{"client_status": "enable"}, nil)

	rows, err := Generate(t.Context(), table.QueryContext{})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	for _, col := range columnOrder {
		assert.Contains(t, rows[0], col, "row is missing column %q", col)
	}
}

func TestGenerate(t *testing.T) {
	const darwinPath = "/Library/Application Support/Netskope/STAgent"

	for _, tc := range []struct {
		name        string
		installPath string
		parsed      map[string]string
		runErr      error

		wantInstalled string
		wantError     string
		wantColumns   map[string]string
	}{
		{
			name:          "connected client",
			installPath:   darwinPath,
			parsed:        healthyNsdiag(),
			wantInstalled: "1",
			wantColumns: map[string]string{
				"install_path":    darwinPath,
				"client_status":   "enable",
				"tunnel_status":   "NSTUNNEL_CONNECTED",
				"client_version":  "117.1.0.1234",
				"orgname":         "CompanyName",
				"tenant_url":      "companyname.goskope.com",
				"steering_config": "Default tenant config",
				"traffic_mode":    "All Traffic",
			},
		},
		{
			// The degraded state this table exists to catch: the client is
			// installed and nsdiag answers, but the tunnel is down.
			name:        "installed but disconnected",
			installPath: darwinPath,
			parsed: map[string]string{
				"client_status":  "disable",
				"tunnel_status":  "NSTUNNEL_DISCONNECTED",
				"client_version": "117.1.0.1234",
			},
			wantInstalled: "1",
			wantColumns: map[string]string{
				"client_status": "disable",
				"tunnel_status": "NSTUNNEL_DISCONNECTED",
			},
		},
		{
			name:          "not installed",
			installPath:   "",
			wantInstalled: "0",
			wantError:     notInstalledMsg,
			wantColumns:   map[string]string{"install_path": "", "client_status": ""},
		},
		{
			name:          "nsdiag fails",
			installPath:   "/opt/netskope/stagent",
			runErr:        errors.New("permission denied"),
			wantInstalled: "1",
			wantError:     "nsdiag failed: permission denied",
			wantColumns:   map[string]string{"install_path": "/opt/netskope/stagent", "client_status": ""},
		},
		{
			// A Netskope release that renames nsdiag's fields must not look like a
			// healthy client with empty columns.
			name:          "nsdiag reports nothing recognizable",
			installPath:   darwinPath,
			parsed:        map[string]string{},
			wantInstalled: "1",
			wantError:     "nsdiag reported no recognized fields",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mockNsdiag(t, tc.installPath, tc.parsed, tc.runErr)

			rows, err := Generate(t.Context(), table.QueryContext{})
			require.NoError(t, err, "Generate must never error, so the table stays queryable")
			require.Len(t, rows, 1, "the table always reports exactly one row")

			row := rows[0]
			assert.Equal(t, tc.wantInstalled, row["client_installed"])
			assert.Equal(t, tc.wantError, row["error"])
			for col, want := range tc.wantColumns {
				assert.Equal(t, want, row[col], "column %q", col)
			}
		})
	}
}
