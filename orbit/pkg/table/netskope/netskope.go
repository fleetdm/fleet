// Package netskope exposes the local Netskope client's state, as reported
// by `nsdiag -f`, as the `netskope` osquery table.
//
// Netskope can enter a silently degraded state where the STAgent process and
// system extensions still look healthy but traffic is no longer steered. Process
// and system-extension checks miss that; the state nsdiag reports does not.
//
// Ported from the standalone extension in kc9wwh/playground#1.
package netskope

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/rs/zerolog/log"
)

// execTimeout is the maximum time to wait for a single nsdiag invocation.
const execTimeout = 10 * time.Second

// notInstalledMsg is reported in the error column when no Netskope install is
// found. Kept verbatim from the standalone extension so queries and policies
// written against it keep working.
const notInstalledMsg = "netskope client not installed"

// runNsdiag and detectInstallPath are variables so tests can inject fixture
// output without a Netskope install.
var (
	runNsdiag         = defaultRunNsdiag
	detectInstallPath = defaultDetectInstallPath
)

// Columns returns the schema of the netskope table.
func Columns() []table.ColumnDefinition {
	cols := make([]table.ColumnDefinition, 0, len(columnOrder))
	for _, name := range columnOrder {
		if _, ok := integerColumns[name]; ok {
			cols = append(cols, table.IntegerColumn(name))
			continue
		}
		cols = append(cols, table.TextColumn(name))
	}
	return cols
}

// Generate returns exactly one row describing the local Netskope client. It
// never returns an error, so `SELECT * FROM netskope` succeeds on hosts
// without Netskope; those report client_installed = 0 and an error column
// explaining why the remaining columns are empty.
//
// A row is returned even when the client is missing or nsdiag fails so that
// "not installed", "installed but degraded" and "installed but state could not
// be read" stay distinguishable. Collapsing them into zero rows would make a
// health policy fail identically in all three cases.
func Generate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	row := newRow()

	installPath := detectInstallPath()
	if installPath == "" {
		log.Debug().Msg("no netskope install found; reporting netskope row as not installed")
		row["error"] = notInstalledMsg
		return []map[string]string{row}, nil
	}
	row["client_installed"] = "1"
	row["install_path"] = installPath

	parsed, err := runNsdiag(ctx, installPath)
	if err != nil {
		log.Debug().Err(err).Msg("nsdiag -f failed; reporting netskope row without state")
		row["error"] = "nsdiag failed: " + err.Error()
		return []map[string]string{row}, nil
	}
	if len(parsed) == 0 {
		// nsdiag ran but reported nothing recognizable. Its flags and field names
		// drift between Netskope releases, so surface this instead of reporting a
		// row of empty columns that reads like a healthy-but-unconfigured client.
		log.Debug().Msg("nsdiag -f reported no recognized fields")
		row["error"] = "nsdiag reported no recognized fields"
		return []map[string]string{row}, nil
	}

	maps.Copy(row, parsed)

	return []map[string]string{row}, nil
}

// newRow returns a row with every column present, so osquery always sees the
// full schema regardless of which fields nsdiag reported.
func newRow() map[string]string {
	row := make(map[string]string, len(columnOrder))
	for _, col := range columnOrder {
		row[col] = ""
	}
	row["client_installed"] = "0"
	return row
}

func defaultDetectInstallPath() string {
	return findInstallPath(installPathCandidates(), os.Stat)
}

// defaultRunNsdiag runs `nsdiag -f` from the install directory and parses its
// output.
//
// Only the absolute path under the detected install directory is executed.
// osquery runs this as root, so resolving nsdiag through $PATH would let anyone
// who can write to a directory on it run code as root.
//
// This still trusts the installer's directory: anyone who can write there gets
// root execution when the table is queried. All the candidate paths are
// root-owned (Program Files is Administrators-only), so writing to them already
// requires the privileges it would grant. Ownership is not re-verified here — a
// release that ships different ownership would silently disable the table
// rather than report a problem.
func defaultRunNsdiag(ctx context.Context, installPath string) (map[string]string, error) {
	bin := filepath.Join(installPath, nsdiagBinaryName())
	if _, err := os.Stat(bin); err != nil {
		return nil, fmt.Errorf("nsdiag not found at %s: %w", bin, err)
	}

	ctx, cancel := context.WithTimeout(ctx, execTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, bin, "-f").Output()
	if err != nil {
		return nil, nsdiagError(err)
	}
	return parseNsdiagText(out), nil
}

// nsdiagError prefers the message nsdiag wrote to stderr, which explains a
// permission or licensing failure far better than the bare exit status.
func nsdiagError(err error) error {
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if msg := strings.TrimSpace(string(exitErr.Stderr)); msg != "" {
			return fmt.Errorf("%w: %s", err, msg)
		}
	}
	return err
}
