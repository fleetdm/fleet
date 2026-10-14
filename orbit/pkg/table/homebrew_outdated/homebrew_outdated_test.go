package homebrew_outdated

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/osquery/osquery-go/plugin/table"
	"github.com/stretchr/testify/require"
)

//go:embed test_data_outdated.json
var outdatedData []byte

//go:embed test_data_info.json
var infoData []byte

func TestParseOutdated(t *testing.T) {
	pkgs, err := parseOutdated(outdatedData)
	require.NoError(t, err)
	// blake3 (1) + git (1) + openssl@3 (2 installed versions) + wget (1) + mitmproxy (1) + google-chrome (1)
	require.Len(t, pkgs, 7)

	require.Equal(t, outdatedPackage{
		name:             "blake3",
		installedVersion: "1.8.3",
		currentVersion:   "1.8.5",
		pkgType:          typeFormula,
	}, pkgs[0])

	// A package with multiple installed versions yields one row per version.
	openssl := filterByName(pkgs, "openssl@3")
	require.Len(t, openssl, 2)
	require.Equal(t, "3.3.1", openssl[0].installedVersion)
	require.Equal(t, "3.3.2", openssl[1].installedVersion)
	require.Equal(t, "3.4.0", openssl[0].currentVersion)
	require.Equal(t, "3.4.0", openssl[1].currentVersion)

	// A pinned package carries its pinned_version; unpinned packages leave it empty.
	wget := filterByName(pkgs, "wget")
	require.Len(t, wget, 1)
	require.Equal(t, "1.21.3", wget[0].pinnedVersion)
	require.Empty(t, openssl[0].pinnedVersion)
}

func filterByName(pkgs []outdatedPackage, name string) []outdatedPackage {
	var out []outdatedPackage
	for _, p := range pkgs {
		if p.name == name {
			out = append(out, p)
		}
	}
	return out
}

func TestNameConstraints(t *testing.T) {
	qc := table.QueryContext{Constraints: map[string]table.ConstraintList{
		"name": {Constraints: []table.Constraint{
			{Operator: table.OperatorEquals, Expression: "ffmpeg"},
			{Operator: table.OperatorEquals, Expression: "ffmpeg"}, // duplicate, deduped
			{Operator: table.OperatorLike, Expression: "wg%"},      // non-equality, ignored
			{Operator: table.OperatorEquals, Expression: ""},       // empty, ignored
			{Operator: table.OperatorEquals, Expression: "wget"},
		}},
	}}
	require.Equal(t, []string{"ffmpeg", "wget"}, nameConstraints(qc))

	// No name constraint -> nil (Generate then runs a full scan).
	require.Nil(t, nameConstraints(table.QueryContext{Constraints: map[string]table.ConstraintList{}}))
}

func TestOutdatedPackagesFullScan(t *testing.T) {
	var calls [][]string
	run := func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return outdatedData, nil
	}
	pkgs, err := outdatedPackages(run)
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
	require.Equal(t, [][]string{{"outdated", "--json=v2"}}, calls)
}

func TestOutdatedPackagesExitOneWithValidJSON(t *testing.T) {
	// brew can exit non-zero yet print valid JSON; that output is used.
	run := func(args ...string) ([]byte, error) {
		return outdatedData, errors.New("exit status 1")
	}
	pkgs, err := outdatedPackages(run)
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
}

func TestOutdatedPackagesEmptyResult(t *testing.T) {
	run := func(args ...string) ([]byte, error) {
		return []byte(`{"formulae":[],"casks":[]}`), nil
	}
	pkgs, err := outdatedPackages(run)
	require.NoError(t, err)
	require.Empty(t, pkgs)
}

func TestOutdatedPackagesFailure(t *testing.T) {
	// A brew failure with no parseable output is reported with brew's error.
	var calls int
	run := func(args ...string) ([]byte, error) {
		calls++
		return []byte(""), errors.New("boom")
	}
	_, err := outdatedPackages(run)
	require.ErrorContains(t, err, "running brew outdated: boom")
	require.Equal(t, 1, calls)
}

func TestFilterRows(t *testing.T) {
	rows := []map[string]string{{"name": "git"}, {"name": "wget"}, {"name": "git"}}

	// No names: everything, untouched.
	require.Equal(t, rows, filterRows(rows, nil))

	// Names: only matching rows, in order, including every row of a name.
	require.Equal(t, []map[string]string{{"name": "git"}, {"name": "git"}}, filterRows(rows, []string{"git"}))
	require.Equal(t, []map[string]string{{"name": "git"}, {"name": "wget"}, {"name": "git"}}, filterRows(rows, []string{"wget", "git"}))

	// Unknown name: empty, not nil, so the table yields no rows rather than an error.
	require.Equal(t, []map[string]string{}, filterRows(rows, []string{"nope"}))
}

func TestScanCacheReusesWithinTTL(t *testing.T) {
	var scans int
	scan := func(context.Context) ([]map[string]string, error) {
		scans++
		return []map[string]string{{"name": "git"}}, nil
	}
	var c scanCache
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rows, err := c.get(t.Context(), "k", now, scan)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	// Same key within the TTL: served from cache.
	rows, err = c.get(t.Context(), "k", now.Add(scanCacheTTL-time.Second), scan)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, 1, scans)

	// Past the TTL: a new scan.
	_, err = c.get(t.Context(), "k", now.Add(scanCacheTTL), scan)
	require.NoError(t, err)
	require.Equal(t, 2, scans)
}

func TestScanCacheKeyChangeRescans(t *testing.T) {
	// A different console user or Homebrew install must not see another's rows.
	var scans int
	scan := func(context.Context) ([]map[string]string, error) {
		scans++
		return []map[string]string{{"name": fmt.Sprint(scans)}}, nil
	}
	var c scanCache
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	rows, err := c.get(t.Context(), "brew:501", now, scan)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "1", rows[0]["name"])

	rows, err = c.get(t.Context(), "brew:502", now, scan)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, "2", rows[0]["name"])
	require.Equal(t, 2, scans)
}

func TestScanCacheCachesFailures(t *testing.T) {
	// A failure is served until the TTL passes, not retried on every query.
	var scans int
	scan := func(context.Context) ([]map[string]string, error) {
		scans++
		return nil, errors.New("exit status 1: Error: boom")
	}
	var c scanCache
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := c.get(t.Context(), "k", now, scan)
	require.ErrorContains(t, err, "boom")
	_, err = c.get(t.Context(), "k", now.Add(time.Second), scan)
	require.ErrorContains(t, err, "boom")
	require.Equal(t, 1, scans)

	_, err = c.get(t.Context(), "k", now.Add(scanCacheTTL), scan)
	require.ErrorContains(t, err, "boom")
	require.Equal(t, 2, scans)
}

func TestScanCacheCallerDeadlineDoesNotPoisonCache(t *testing.T) {
	// A query giving up early must neither fail the shared scan nor get its
	// deadline cached.
	var scans atomic.Int32
	release := make(chan struct{})
	scanErr := make(chan error, 1)
	scan := func(ctx context.Context) ([]map[string]string, error) {
		scans.Add(1)
		<-release
		scanErr <- ctx.Err()
		return []map[string]string{{"name": "git"}}, nil
	}
	var c scanCache
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	short, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	_, err := c.get(short, "k", now, scan)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	close(release)
	require.NoError(t, <-scanErr, "the scan must not inherit the caller's deadline")
	rows, err := c.get(t.Context(), "k", now, scan)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	require.Equal(t, int32(1), scans.Load())
}

func TestScanCacheWaiterHonorsItsContext(t *testing.T) {
	// A query arriving mid-scan can give up without waiting for the scan.
	started := make(chan struct{})
	release := make(chan struct{})
	defer close(release)
	scan := func(context.Context) ([]map[string]string, error) {
		close(started)
		<-release
		return nil, nil
	}
	var c scanCache
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	go func() { _, _ = c.get(t.Context(), "k", now, scan) }()
	<-started
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := c.get(cancelled, "k", now, scan)
	require.ErrorIs(t, err, context.Canceled)
}

func TestScanCacheSerializesConcurrentScans(t *testing.T) {
	// Concurrent queries share one scan.
	var scans atomic.Int32
	started := make(chan struct{})
	release := make(chan struct{})
	scan := func(context.Context) ([]map[string]string, error) {
		if scans.Add(1) == 1 {
			close(started)
			<-release
		}
		return []map[string]string{{"name": "git"}}, nil
	}
	var c scanCache
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() {
			rows, err := c.get(t.Context(), "k", now, scan)
			require.NoError(t, err)
			require.Len(t, rows, 1)
		})
	}
	<-started
	close(release)
	wg.Wait()
	require.Equal(t, int32(1), scans.Load())
}

func TestFirstExistingFile(t *testing.T) {
	dir := t.TempDir()
	brew := filepath.Join(dir, "brew")
	require.NoError(t, os.WriteFile(brew, []byte("x"), 0o600))

	// Returns the first path that exists as a regular file.
	require.Equal(t, brew, firstExistingFile([]string{filepath.Join(dir, "missing"), brew}))
	// None exist -> "".
	require.Empty(t, firstExistingFile([]string{filepath.Join(dir, "missing")}))
	// A directory is not a match.
	require.Empty(t, firstExistingFile([]string{dir}))
}

func TestUniqueCaskNames(t *testing.T) {
	pkgs := []outdatedPackage{
		{name: "git", pkgType: typeFormula}, // formula: excluded
		{name: "mitmproxy", pkgType: typeCask},
		{name: "mitmproxy", pkgType: typeCask}, // duplicate: deduped
		{name: "ngrok", pkgType: typeCask},
	}
	require.Equal(t, []string{"mitmproxy", "ngrok"}, uniqueCaskNames(pkgs))
	require.Empty(t, uniqueCaskNames(nil))
	// No casks -> empty, which lets Generate skip the brew info call entirely.
	require.Empty(t, uniqueCaskNames([]outdatedPackage{{name: "git", pkgType: typeFormula}}))
}

func TestExpandEntryNoInstalledVersions(t *testing.T) {
	// A package with no reported installed version still yields exactly one row.
	rows := mapEntry(brewOutdatedEntry{Name: "x", CurrentVersion: "2.0"}, typeFormula)
	require.Len(t, rows, 1)
	require.Equal(t, "x", rows[0].name)
	require.Empty(t, rows[0].installedVersion)
	require.Equal(t, "2.0", rows[0].currentVersion)
}

func TestBuildRowsCaskWithoutEnrichment(t *testing.T) {
	// When brew info is unavailable, cask rows still build with empty app_name /
	// auto_updates but a valid Caskroom install_path.
	pkgs := []outdatedPackage{
		{name: "ngrok", installedVersion: "3.37.2,abc", currentVersion: "3.39.9,def", pkgType: typeCask},
	}
	rows := buildRows(pkgs, nil, "/opt/homebrew")
	require.Len(t, rows, 1)
	require.Empty(t, rows[0]["app_name"])
	require.Empty(t, rows[0]["auto_updates"])
	require.Equal(t, "/opt/homebrew/Caskroom/ngrok", rows[0]["install_path"])
	require.Equal(t, "3.37.2,abc", rows[0]["installed_version"])
}

func TestParseOutdatedEmpty(t *testing.T) {
	pkgs, err := parseOutdated([]byte(`{"formulae":[],"casks":[]}`))
	require.NoError(t, err)
	require.Empty(t, pkgs)
}

func TestParseCaskInfo(t *testing.T) {
	casks, err := parseCaskInfo(infoData)
	require.NoError(t, err)
	require.Len(t, casks, 2)

	// auto_updates: null -> "0"
	require.Equal(t, caskDetail{appName: "mitmproxy", autoUpdates: "0"}, casks["mitmproxy"])
	// auto_updates: true -> "1", app name taken from the first entry of the name array
	require.Equal(t, caskDetail{appName: "Google Chrome", autoUpdates: "1"}, casks["google-chrome"])
}

func TestBuildRows(t *testing.T) {
	pkgs, err := parseOutdated(outdatedData)
	require.NoError(t, err)
	casks, err := parseCaskInfo(infoData)
	require.NoError(t, err)

	rows := buildRows(pkgs, casks, "/opt/homebrew")
	require.Len(t, rows, 7)

	byName := make(map[string]map[string]string, len(rows))
	for _, r := range rows {
		byName[r["name"]] = r
	}

	// The multi-version formula produces two rows, one per installed version.
	var opensslVersions []string
	for _, r := range rows {
		if r["name"] == "openssl@3" {
			opensslVersions = append(opensslVersions, r["installed_version"])
		}
	}
	require.ElementsMatch(t, []string{"3.3.1", "3.3.2"}, opensslVersions)

	// Formula: no app_name/auto_updates, opt-based install path, unpinned.
	require.Equal(t, map[string]string{
		"name":              "git",
		"type":              "formula",
		"installed_version": "2.53.0",
		"current_version":   "2.55.0",
		"pinned_version":    "",
		"app_name":          "",
		"auto_updates":      "",
		"install_path":      "/opt/homebrew/opt/git",
	}, byName["git"])

	// Cask: enriched with app_name/auto_updates, Caskroom-based install path.
	require.Equal(t, map[string]string{
		"name":              "google-chrome",
		"type":              "cask",
		"installed_version": "120.0",
		"current_version":   "121.0",
		"pinned_version":    "",
		"app_name":          "Google Chrome",
		"auto_updates":      "1",
		"install_path":      "/opt/homebrew/Caskroom/google-chrome",
	}, byName["google-chrome"])

	require.Equal(t, "0", byName["mitmproxy"]["auto_updates"])

	// A pinned package exposes its pinned_version.
	require.Equal(t, "1.21.3", byName["wget"]["pinned_version"])
}

func exitErrWithStderr(stderr string) error {
	return &exec.ExitError{Stderr: []byte(stderr)}
}

func TestDescribeBrewErrorUsesErrorLines(t *testing.T) {
	// Only the "Error:" line onward belongs in the error, not the chatter before.
	stderr := "==> Auto-updating Homebrew...\n==> New Formulae\nfoo: does things\nError: No available formula with the name \"zzz\".\nDid you mean zz?\n"
	err := describeBrewError(exitErrWithStderr(stderr))
	require.Error(t, err)
	require.True(t, strings.HasSuffix(err.Error(), ": Error: No available formula with the name \"zzz\".\nDid you mean zz?"), err.Error())
	require.NotContains(t, err.Error(), "Auto-updating")

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
}

func TestDescribeBrewErrorSkipsMidLineErrorPrefix(t *testing.T) {
	// Only "Error:" at the start of a line counts.
	stderr := "Warning: see Error: docs\n==> Auto-updating Homebrew...\nError: real failure\n"
	err := describeBrewError(exitErrWithStderr(stderr))
	require.Error(t, err)
	require.True(t, strings.HasSuffix(err.Error(), ": Error: real failure"), err.Error())
	require.NotContains(t, err.Error(), "Auto-updating")
}

func TestBrewErrorMessage(t *testing.T) {
	require.Equal(t, "Error: boom\nmore", brewErrorMessage("==> chatter\nError: boom\nmore"))
	require.Equal(t, "Error: boom", brewErrorMessage("Error: boom"))
	require.Equal(t, "Error: second", brewErrorMessage("first Error: mid-line\nError: second"))
	require.Empty(t, brewErrorMessage("first Error: mid-line only"))
	require.Empty(t, brewErrorMessage("==> all good\n"))
	require.Empty(t, brewErrorMessage(""))
}

func TestDescribeBrewErrorWholeStderrWithoutErrorLine(t *testing.T) {
	err := describeBrewError(exitErrWithStderr("  something went wrong\n"))
	require.Error(t, err)
	require.True(t, strings.HasSuffix(err.Error(), ": something went wrong"), err.Error())
}

func TestDescribeBrewErrorTruncatesLongStderr(t *testing.T) {
	long := strings.Repeat("x", maxBrewStderr+100)
	err := describeBrewError(exitErrWithStderr(long))
	require.Error(t, err)
	require.True(t, strings.HasSuffix(err.Error(), ": "+strings.Repeat("x", maxBrewStderr)+"..."), err.Error())
}

func TestDescribeBrewErrorPassthrough(t *testing.T) {
	require.NoError(t, describeBrewError(nil))

	// Not an ExitError (e.g. context deadline exceeded): unchanged.
	plain := errors.New("context deadline exceeded")
	require.Equal(t, plain, describeBrewError(plain))

	// ExitError with no stderr: unchanged.
	empty := exitErrWithStderr("  \n")
	require.Equal(t, empty, describeBrewError(empty))
}

func TestDescribeBrewErrorKeepsDeadlineSignal(t *testing.T) {
	// A deadline kill reports the deadline and stderr, and still matches
	// context.DeadlineExceeded.
	killed := fmt.Errorf("%w: %w", context.DeadlineExceeded, exitErrWithStderr("==> Auto-updating Homebrew...\n"))
	err := describeBrewError(killed)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.True(t, strings.HasPrefix(err.Error(), "context deadline exceeded: "), err.Error())
	require.True(t, strings.HasSuffix(err.Error(), ": ==> Auto-updating Homebrew..."), err.Error())
}

func TestDescribeBrewErrorTruncationIsUTF8Safe(t *testing.T) {
	// The cut must not split a multi-byte rune.
	stderr := strings.Repeat("x", maxBrewStderr-1) + "é"
	err := describeBrewError(exitErrWithStderr(stderr))
	require.True(t, utf8.ValidString(err.Error()))
	require.True(t, strings.HasSuffix(err.Error(), "x..."))
}

func TestOutdatedPackagesKeepsDeadlineError(t *testing.T) {
	// A scan killed at the deadline reports both the deadline and brew's stderr.
	run := func(args ...string) ([]byte, error) {
		// Mirrors what runBrew returns for a call killed at the deadline.
		return nil, describeBrewError(fmt.Errorf("%w: %w", context.DeadlineExceeded, exitErrWithStderr("==> Downloading Homebrew API data")))
	}
	_, err := outdatedPackages(run)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Contains(t, err.Error(), "Downloading Homebrew API data")
}
