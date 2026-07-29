// Package homebrew_outdated implements the fleetd `homebrew_outdated` osquery
// table, which returns one row per installed version of each outdated Homebrew
// package (formula or cask) on macOS.
package homebrew_outdated

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/osquery/osquery-go/plugin/table"
)

const TableName = "homebrew_outdated"

func Columns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("app_name"),
		table.IntegerColumn("auto_updates"),
		table.TextColumn("name"),
		table.TextColumn("install_path"),
		table.TextColumn("type"),
		table.TextColumn("installed_version"),
		table.TextColumn("current_version"),
		table.TextColumn("pinned_version"),
	}
}

const (
	typeFormula = "formula"
	typeCask    = "cask"
)

// outdatedPackage is the internal, normalized model for a single outdated
// package occurrence: one package name paired with one installed version. A
// brewOutdatedEntry with multiple installed versions expands into several of
// these (see mapEntry). buildRows turns each one into an osquery result row.
type outdatedPackage struct {
	name             string
	installedVersion string
	currentVersion   string
	pinnedVersion    string
	pkgType          string // typeFormula or typeCask
}

// brewOutdatedEntry is one formula or cask in `brew outdated --json=v2` output.
// A single entry can report more than one installed version (e.g. a
// versioned/keg-only formula), hence InstalledVersions is a slice.
type brewOutdatedEntry struct {
	Name              string   `json:"name"`
	InstalledVersions []string `json:"installed_versions"`
	CurrentVersion    string   `json:"current_version"`
	PinnedVersion     *string  `json:"pinned_version"` // null when the package is not pinned
}

// brewInfoCask is one cask in `brew info --json=v2` output. Only casks are read
// from brew info: app_name and auto_updates are cask-only concepts used to enrich
// the rows that `brew outdated` alone can't fully describe.
type brewInfoCask struct {
	Token       string   `json:"token"`        // matches the cask "name" in brew outdated
	Name        []string `json:"name"`         // display name(s); first entry is the app name
	AutoUpdates *bool    `json:"auto_updates"` // null/false -> cask does not auto-update
}

// nameConstraints returns the deduplicated values of any `name = <x>` equality
// constraints in the query. Non-equality operators (LIKE, etc.)
// are ignored — osquery still applies them to the returned rows.
func nameConstraints(queryContext table.QueryContext) []string {
	q, ok := queryContext.Constraints["name"]
	if !ok {
		return nil
	}
	var names []string
	seen := make(map[string]struct{})
	for _, c := range q.Constraints {
		if c.Operator != table.OperatorEquals || c.Expression == "" {
			continue
		}
		if _, dup := seen[c.Expression]; dup {
			continue
		}
		seen[c.Expression] = struct{}{}
		names = append(names, c.Expression)
	}
	return names
}

// brewRunner runs brew with the given args and returns stdout (plus any exec
// error). It exists so outdatedPackages can be unit-tested without invoking brew.
type brewRunner func(args ...string) ([]byte, error)

// outdatedPackages runs `brew outdated` with the given name filter and
// parses the result. When a name filter is supplied but the call produces no
// parseable output, it falls back to a full scan this is because
// brew aborts entirely (empty stdout) if any pushed-down name is not a known
// formula/cask.
func outdatedPackages(run brewRunner, names []string) ([]outdatedPackage, error) {
	args := []string{"outdated", "--json=v2"}
	if len(names) > 0 {
		// "--" terminates option parsing so a pushed-down name beginning with "-"
		// is treated as a package name rather than a brew option.
		args = append(args, "--")
		args = append(args, names...)
	}
	out, err := run(args...)
	pkgs, perr := parseOutdated(out)

	if perr != nil && len(names) > 0 {
		out, err = run("outdated", "--json=v2")
		pkgs, perr = parseOutdated(out)
	}

	if perr != nil {
		if err != nil {
			return nil, fmt.Errorf("running brew outdated: %w", err)
		}
		return nil, perr
	}
	return pkgs, nil
}

func parseOutdated(data []byte) ([]outdatedPackage, error) {
	// `brew outdated --json=v2` splits results into formulae and casks.
	var out struct {
		Formulae []brewOutdatedEntry `json:"formulae"`
		Casks    []brewOutdatedEntry `json:"casks"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parsing brew outdated output: %w", err)
	}
	pkgs := make([]outdatedPackage, 0, len(out.Formulae)+len(out.Casks))
	for _, f := range out.Formulae {
		pkgs = append(pkgs, mapEntry(f, typeFormula)...)
	}
	for _, c := range out.Casks {
		pkgs = append(pkgs, mapEntry(c, typeCask)...)
	}
	return pkgs, nil
}

// firstExistingFile returns the first path in paths that exists and is a regular
// file (not a directory), or "" if none match.
func firstExistingFile(paths []string) string {
	for _, p := range paths {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

// uniqueCaskNames returns the deduplicated names of the cask packages in pkgs,
// preserving order. Only casks are enriched via brew info (app_name and
// auto_updates are cask-only), so formulae are skipped; a package can also appear
// multiple times (one row per installed version), so names are deduplicated.
func uniqueCaskNames(pkgs []outdatedPackage) []string {
	var names []string
	seen := make(map[string]struct{})
	for _, p := range pkgs {
		if p.pkgType != typeCask {
			continue
		}
		if _, ok := seen[p.name]; ok {
			continue
		}
		seen[p.name] = struct{}{}
		names = append(names, p.name)
	}
	return names
}

func mapEntry(e brewOutdatedEntry, pkgType string) []outdatedPackage {
	versions := e.InstalledVersions
	if len(versions) == 0 {
		versions = []string{""}
	}
	var pinnedVersion string
	if e.PinnedVersion != nil {
		pinnedVersion = *e.PinnedVersion
	}
	pkgs := make([]outdatedPackage, 0, len(versions))
	for _, v := range versions {
		pkgs = append(pkgs, outdatedPackage{
			name:             e.Name,
			installedVersion: v,
			currentVersion:   e.CurrentVersion,
			pinnedVersion:    pinnedVersion,
			pkgType:          pkgType,
		})
	}
	return pkgs
}

// caskDetail is the cask-only enrichment (from brew info) applied to a row,
// keyed by cask token in the map returned by parseCaskInfo.
type caskDetail struct {
	appName     string
	autoUpdates string // "1", "0", or "" when unknown
}

func parseCaskInfo(data []byte) (map[string]caskDetail, error) {
	var info struct {
		Casks []brewInfoCask `json:"casks"`
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("parsing brew info output: %w", err)
	}

	details := make(map[string]caskDetail, len(info.Casks))
	for _, c := range info.Casks {
		var appName string
		if len(c.Name) > 0 {
			appName = c.Name[0]
		}
		autoUpdates := "0"
		if c.AutoUpdates != nil && *c.AutoUpdates {
			autoUpdates = "1"
		}
		details[c.Token] = caskDetail{appName: appName, autoUpdates: autoUpdates}
	}
	return details, nil
}

// buildRows merges outdated packages with cask enrichment details and derives the
// install path from the Homebrew prefix, producing the final table rows.
//
// install_path is derived from Homebrew's standard layout under the prefix
// (<prefix>/opt/<name> for formulae, <prefix>/Caskroom/<name> for casks). It
// reflects the default layout; a non-default HOMEBREW_CELLAR/HOMEBREW_CASKROOM
// relocation is not accounted for (querying brew per package to discover it would
// be far too expensive). It is the Homebrew-managed location, not, for a cask, the
// app's final /Applications path.
//
// app_name and auto_updates only apply to casks; they are empty for formulae.
func buildRows(pkgs []outdatedPackage, casks map[string]caskDetail, prefix string) []map[string]string {
	rows := make([]map[string]string, 0, len(pkgs))
	for _, p := range pkgs {
		row := map[string]string{
			"name":              p.name,
			"type":              p.pkgType,
			"installed_version": p.installedVersion,
			"current_version":   p.currentVersion,
			"pinned_version":    p.pinnedVersion,
			"app_name":          "",
			"auto_updates":      "",
		}
		switch p.pkgType {
		case typeFormula:
			// <prefix>/opt/<formula> — the version-independent "opt" symlink
			// (equivalent to `brew --prefix <formula>`).
			row["install_path"] = filepath.Join(prefix, "opt", p.name)
		case typeCask:
			// <prefix>/Caskroom/<cask> (equivalent to `brew --caskroom <cask>`).
			row["install_path"] = filepath.Join(prefix, "Caskroom", p.name)
			if d, ok := casks[p.name]; ok {
				row["app_name"] = d.appName
				row["auto_updates"] = d.autoUpdates
			}
		}
		rows = append(rows, row)
	}
	return rows
}
