// Package homebrew_outdated implements the fleetd `homebrew_outdated` osquery
// table, which returns one row per installed version of each outdated Homebrew
// package (formula or cask) on macOS.
package homebrew_outdated

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
// constraints in the query. Non-equality operators (LIKE, etc.) are ignored;
// osquery still applies them to the returned rows.
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

// outdatedPackages runs a full `brew outdated` scan. Parseable output is used
// even when brew exits non-zero.
func outdatedPackages(run brewRunner) ([]outdatedPackage, error) {
	out, err := run("outdated", "--json=v2")
	pkgs, perr := parseOutdated(out)
	if perr != nil {
		if err != nil {
			return nil, fmt.Errorf("running brew outdated: %w", err)
		}
		return nil, perr
	}
	return pkgs, nil
}

// filterRows returns the rows whose name is in names, or all rows when names is
// empty. Filtering here instead of pushing names down to brew lets every query
// share the cached scan.
func filterRows(rows []map[string]string, names []string) []map[string]string {
	if len(names) == 0 {
		return rows
	}
	want := make(map[string]struct{}, len(names))
	for _, n := range names {
		want[n] = struct{}{}
	}
	out := make([]map[string]string, 0, len(rows))
	for _, r := range rows {
		if _, ok := want[r["name"]]; ok {
			out = append(out, r)
		}
	}
	return out
}

// scanCacheTTL is how long a scan is served to later queries, so a batch of
// policies costs one brew run instead of one per policy.
const scanCacheTTL = 5 * time.Minute

// scanCache holds the last scan. Failures are cached too so a slow or broken
// brew is hit once per window, not once per query. Returned rows are shared and
// must not be modified.
type scanCache struct {
	mu      sync.Mutex
	key     string
	expires time.Time
	rows    []map[string]string
	err     error
	flight  *scanFlight // the scan in progress, if any
}

// scanFlight is one scan shared by every caller that arrives while it runs.
type scanFlight struct {
	key  string
	done chan struct{} // closed once rows and err are set
	rows []map[string]string
	err  error
}

// get returns the rows cached under key (console user + Homebrew install),
// running scan when they are missing or expired. Concurrent callers share one
// scan. The scan is detached from the callers so one giving up early neither
// cuts it short nor leaves its own deadline cached; each caller still returns
// as soon as its ctx ends.
func (c *scanCache) get(ctx context.Context, key string, now time.Time, scan func(context.Context) ([]map[string]string, error)) ([]map[string]string, error) {
	c.mu.Lock()
	if c.key == key && now.Before(c.expires) {
		rows, err := c.rows, c.err
		c.mu.Unlock()
		return rows, err
	}
	f := c.flight
	if f == nil || f.key != key {
		f = &scanFlight{key: key, done: make(chan struct{})}
		c.flight = f
		go func() {
			rows, err := scan(context.WithoutCancel(ctx))
			c.mu.Lock()
			f.rows, f.err = rows, err
			if c.flight == f {
				c.flight = nil
				c.key, c.expires, c.rows, c.err = key, now.Add(scanCacheTTL), rows, err
			}
			c.mu.Unlock()
			close(f.done)
		}()
	}
	c.mu.Unlock()

	select {
	case <-f.done:
		return f.rows, f.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// maxBrewStderr caps how much of brew's stderr is included in a returned error.
const maxBrewStderr = 1000

// describeBrewError appends brew's stderr to an exit error so the table reports
// why brew failed rather than a bare "exit status 1", from the fatal "Error:"
// line onward when there is one. Other errors pass through.
func describeBrewError(err error) error {
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	if !ok {
		return err
	}
	stderr := strings.TrimSpace(string(exitErr.Stderr))
	if stderr == "" {
		return err
	}
	if msg := brewErrorMessage(stderr); msg != "" {
		stderr = msg
	}
	if len(stderr) > maxBrewStderr {
		stderr = strings.ToValidUTF8(stderr[:maxBrewStderr], "") + "..."
	}
	return fmt.Errorf("%w: %s", err, stderr)
}

// brewErrorMessage returns stderr from the first line starting with "Error:",
// brew's fatal-message prefix, or "" when there is none.
func brewErrorMessage(stderr string) string {
	for off := 0; off < len(stderr); {
		if strings.HasPrefix(stderr[off:], "Error:") {
			return stderr[off:]
		}
		nl := strings.IndexByte(stderr[off:], '\n')
		if nl < 0 {
			break
		}
		off += nl + 1
	}
	return ""
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
