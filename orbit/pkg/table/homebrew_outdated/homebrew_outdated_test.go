package homebrew_outdated

import (
	_ "embed"
	"errors"
	"os"
	"path/filepath"
	"testing"

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

func TestOutdatedPackagesPushdown(t *testing.T) {
	// Filtered call returns valid JSON -> used directly, no fallback.
	var calls [][]string
	run := func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return outdatedData, nil
	}
	pkgs, err := outdatedPackages(run, []string{"blake3"})
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
	require.Equal(t, [][]string{{"outdated", "--json=v2", "--", "blake3"}}, calls)
}

func TestOutdatedPackagesFallbackOnUnknownName(t *testing.T) {
	// First (pushed-down) call aborts with empty stdout because a name is unknown;
	// we fall back to a full scan and let osquery filter the rows.
	var calls [][]string
	run := func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		if len(calls) == 1 {
			return []byte(""), errors.New("exit status 1") // brew aborted on bad name
		}
		return outdatedData, nil // full scan succeeds
	}
	pkgs, err := outdatedPackages(run, []string{"blake3", "zzzbogus"})
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
	require.Equal(t, [][]string{
		{"outdated", "--json=v2", "--", "blake3", "zzzbogus"}, // attempted pushdown
		{"outdated", "--json=v2"},                             // fallback full scan, no names
	}, calls)
}

func TestOutdatedPackagesExitOneWithValidJSON(t *testing.T) {
	// `brew outdated <name>` exits non-zero when the package is outdated but still
	// prints valid JSON. That output must be used, not treated as a failure.
	var calls int
	run := func(args ...string) ([]byte, error) {
		calls++
		return outdatedData, errors.New("exit status 1")
	}
	pkgs, err := outdatedPackages(run, []string{"blake3"})
	require.NoError(t, err)
	require.NotEmpty(t, pkgs)
	require.Equal(t, 1, calls) // no fallback: the output parsed fine
}

func TestOutdatedPackagesEmptyResultNoFallback(t *testing.T) {
	// A valid name that isn't installed yields empty-but-parseable JSON. Parsing
	// succeeds, so there must be no fallback and no error.
	var calls int
	run := func(args ...string) ([]byte, error) {
		calls++
		return []byte(`{"formulae":[],"casks":[]}`), nil
	}
	pkgs, err := outdatedPackages(run, []string{"valid-not-installed"})
	require.NoError(t, err)
	require.Empty(t, pkgs)
	require.Equal(t, 1, calls)
}

func TestOutdatedPackagesNoFallbackWithoutNames(t *testing.T) {
	// Without a name filter the first call is already a full scan; a failure must
	// not trigger a pointless second scan.
	var calls int
	run := func(args ...string) ([]byte, error) {
		calls++
		return []byte(""), errors.New("boom")
	}
	_, err := outdatedPackages(run, nil)
	require.Error(t, err)
	require.Equal(t, 1, calls)
}

func TestOutdatedPackagesFallbackAlsoFails(t *testing.T) {
	// A genuine brew failure (not just a bad name) surfaces after the fallback.
	var calls int
	run := func(args ...string) ([]byte, error) {
		calls++
		return []byte(""), errors.New("boom")
	}
	_, err := outdatedPackages(run, []string{"x"})
	require.Error(t, err)
	require.Equal(t, 2, calls)
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
