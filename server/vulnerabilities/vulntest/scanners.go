package vulntest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"github.com/fleetdm/fleet/v4/server/fleet"
	goval_dictionary "github.com/fleetdm/fleet/v4/server/vulnerabilities/goval_dictionary"
	"github.com/fleetdm/fleet/v4/server/vulnerabilities/oval"
	"github.com/stretchr/testify/require"
)

// OVALScanner returns a Scanner that uses the OVAL analyzer for RHEL package vulnerabilities.
// defDir is the directory containing OVAL definition fixtures, relative to testdata/
// (e.g. "rhel/2026").
func OVALScanner(defDir string) Scanner {
	return Scanner{
		Name:   "oval",
		Source: fleet.RHELOVALSource,
		Setup: func(t *testing.T, vulnPath string, ver fleet.OSVersion) {
			p := oval.NewPlatform(ver.Platform, ver.Name)
			src := filepath.Join(TestdataRoot, defDir, fmt.Sprintf("%s-oval_def.json.bz2", p))
			dst := filepath.Join(vulnPath, p.ToFilename(time.Now(), "json"))
			ExtractBzip2(src, dst, t)
		},
		Analyze: func(ctx context.Context, ds fleet.Datastore, ver fleet.OSVersion, vulnPath string) ([]fleet.SoftwareVulnerability, error) {
			return oval.Analyze(ctx, ds, ver, vulnPath, true)
		},
	}
}

// GovalDictionaryScanner returns a Scanner that uses the goval-dictionary analyzer
// for RHEL kernel vulnerabilities.
//
// Setup creates a minimal SQLite database inline rather than downloading the full
// goval-dictionary database (~90MB uncompressed for RHEL 9). The full database is
// published as an xz-compressed asset in the fleetdm/vulnerabilities GitHub releases
// (e.g. rhel_09.sqlite3.xz). Building a targeted SQLite with only the kernel entries
// needed for the test keeps fixtures small and self-contained, while exercising the
// same Analyze → Eval → Rpmvercmp code path that production uses.
//
// kernelEntries defines the test data: each entry maps a fixed version string
// (epoch:version-release format, e.g. "0:5.14.0-611.8.1.el9_7") to its CVE IDs.
// These values are sourced from the real rhel_09.sqlite3 goval-dictionary database.
func GovalDictionaryScanner(kernelEntries map[string][]string) Scanner {
	return Scanner{
		Name:   "goval-dictionary",
		Source: fleet.GovalDictionarySource,
		Setup: func(t *testing.T, vulnPath string, ver fleet.OSVersion) {
			platform := oval.NewPlatform(ver.Platform, ver.Name)
			dbPath := filepath.Join(vulnPath, platform.ToGovalDictionaryFilename())

			db, err := sql.Open("sqlite3", dbPath)
			require.NoError(t, err)
			defer db.Close()

			for _, q := range []string{
				"CREATE TABLE packages (name TEXT NOT NULL, arch TEXT NOT NULL, version TEXT NOT NULL, definition_id INTEGER NOT NULL)",
				"CREATE TABLE definitions (id INTEGER NOT NULL PRIMARY KEY)",
				"CREATE TABLE advisories (id INTEGER NOT NULL PRIMARY KEY, definition_id INTEGER NOT NULL)",
				"CREATE TABLE cves (cve_id TEXT NOT NULL, advisory_id INTEGER NOT NULL)",
			} {
				_, err := db.Exec(q)
				require.NoError(t, err)
			}

			defID := 1
			for fixedVersion, cveIDs := range kernelEntries {
				_, err := db.Exec("INSERT INTO definitions (id) VALUES (?)", defID)
				require.NoError(t, err)
				_, err = db.Exec("INSERT INTO advisories (id, definition_id) VALUES (?, ?)", defID, defID)
				require.NoError(t, err)
				_, err = db.Exec("INSERT INTO packages (name, arch, version, definition_id) VALUES ('kernel', 'x86_64', ?, ?)", fixedVersion, defID)
				require.NoError(t, err)
				for _, cve := range cveIDs {
					_, err = db.Exec("INSERT INTO cves (cve_id, advisory_id) VALUES (?, ?)", cve, defID)
					require.NoError(t, err)
				}
				defID++
			}
		},
		Analyze: func(ctx context.Context, ds fleet.Datastore, ver fleet.OSVersion, vulnPath string) ([]fleet.SoftwareVulnerability, error) {
			logger := slog.New(slog.DiscardHandler)
			return goval_dictionary.Analyze(ctx, ds, ver, vulnPath, true, logger)
		},
	}
}
