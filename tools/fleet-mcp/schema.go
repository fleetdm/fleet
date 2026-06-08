package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

// SchemaTable is the source-of-truth representation of an osquery/Fleet table.
// Field tags match the canonical schema JSON shape published in the Fleet
// monorepo at /schema/osquery_fleet_schema.json (also rendered at
// https://fleetdm.com/tables). MCP clients receive these structs as-is in the
// `get_osquery_schema` and `prepare_live_query` responses.
type SchemaTable struct {
	Name        string   `json:"name"`
	Platforms   []string `json:"platforms"`
	Description string   `json:"description"`
	Columns     []Column `json:"columns"`
	Examples    string   `json:"examples,omitempty"`
	Notes       string   `json:"notes,omitempty"`
	URL         string   `json:"url,omitempty"`
}

// Column is a single column on a SchemaTable. Type is lowercased per the
// canonical JSON ("text", "integer", "bigint", "double"). The validator and
// callers compare types case-insensitively.
type Column struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Notes       string `json:"notes,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Hidden      bool   `json:"hidden,omitempty"`
}

// schemaSource records where the live schema came from on the most recent
// successful load — surfaced in the get_osquery_schema response so callers can
// tell whether the data is fresh from fleetdm.com or the embedded fallback.
//
// FetchedAt is only set on a live HTTP fetch. The embedded snapshot leaves it
// nil so a vendored copy doesn't masquerade as freshly fetched data.
type schemaSource struct {
	Origin    string     `json:"origin"`               // "live" or "embed"
	URL       string     `json:"url"`                  // canonical URL fetched (or empty for embed)
	FetchedAt *time.Time `json:"fetched_at,omitempty"` // set only when Origin == "live"
}

const (
	// canonicalSchemaURL is the live source the schema is refreshed from.
	// It points at the same JSON file the fleetdm.com/tables page is generated
	// from, so the in-memory schema always reflects the documented surface.
	canonicalSchemaURL = "https://raw.githubusercontent.com/fleetdm/fleet/main/schema/osquery_fleet_schema.json"

	defaultRefreshInterval = 24 * time.Hour
	fetchTimeout           = 15 * time.Second
)

// The vendored copy next to this file is what //go:embed pulls into the
// binary as the offline fallback. Refresh from the canonical Fleet monorepo
// via `go generate ./tools/fleet-mcp/...` whenever Fleet upstream rebuilds
// the schema.

//go:generate cp ../../schema/osquery_fleet_schema.json ./osquery_fleet_schema.json

//go:embed osquery_fleet_schema.json
var embeddedSchemaJSON []byte

var (
	schemaMu     sync.RWMutex
	schemaTables []SchemaTable
	schemaIndex  map[string]SchemaTable // lowercase table name → table

	// schemaInfo is set atomically so concurrent readers don't need to take
	// schemaMu for the cheap "where did this come from?" lookup.
	schemaInfo atomic.Pointer[schemaSource]

	refreshOnce sync.Once
)

// defaultCuratedTables is the curated list of "most useful for security ops"
// table names returned by `get_osquery_schema(platform=...)` when the caller
// does not request specific tables. The list intentionally stays small — full
// canonical coverage (~360 tables) is available via the `tables` parameter on
// the same tool, and via GetOsquerySchemaForTables below.
var defaultCuratedTables = []string{
	// Universal
	"os_version", "system_info", "uptime", "users", "logged_in_users",
	"processes", "listening_ports", "interface_addresses", "interface_details",
	"kernel_info", "osquery_info", "file", "hash", "certificates",
	// macOS
	"apps", "browser_plugins", "managed_policies", "system_extensions",
	"munki_info", "battery", "disk_encryption",
	// Windows
	"programs", "windows_update_history", "windows_security_center",
	"patches", "services", "scheduled_tasks",
	"bitlocker_info", "windows_optional_features",
	// Linux
	"deb_packages", "rpm_packages", "apt_sources",
	"kernel_modules", "systemd_units",
}

func init() {
	// Always load the embedded snapshot synchronously so the rest of the
	// binary can serve schema lookups immediately, even if the live fetch
	// later fails or the process runs offline.
	tables, err := parseSchemaJSON(embeddedSchemaJSON)
	if err != nil {
		panic(fmt.Sprintf("fleet-mcp: failed to parse embedded osquery schema: %v", err))
	}
	storeSchema(tables, schemaSource{Origin: "embed"})
}

// StartSchemaRefresh kicks off a background goroutine that fetches the
// canonical schema from canonicalSchemaURL once on entry and then on
// `interval` ticks. Safe to call exactly once; subsequent calls are no-ops.
// If interval is zero, defaultRefreshInterval is used. If the env var
// FLEET_MCP_SCHEMA_REFRESH_DISABLE is set, the function returns without
// spawning anything (useful for offline / air-gapped deployments).
func StartSchemaRefresh(interval time.Duration) {
	if os.Getenv("FLEET_MCP_SCHEMA_REFRESH_DISABLE") != "" {
		logrus.Info("Schema refresh disabled via FLEET_MCP_SCHEMA_REFRESH_DISABLE; using embedded snapshot only.")
		return
	}
	if interval <= 0 {
		interval = defaultRefreshInterval
		if v := os.Getenv("FLEET_MCP_SCHEMA_REFRESH_INTERVAL"); v != "" {
			if d, err := time.ParseDuration(v); err == nil && d > 0 {
				interval = d
			}
		}
	}
	refreshOnce.Do(func() {
		go func() {
			// Initial fetch happens shortly after startup so it doesn't block
			// the MCP server from becoming responsive on launch.
			time.Sleep(2 * time.Second)
			if err := RefreshSchemaNow(); err != nil {
				logrus.Warnf("Initial schema refresh failed (using embedded fallback): %v", err)
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for range ticker.C {
				if err := RefreshSchemaNow(); err != nil {
					logrus.Warnf("Periodic schema refresh failed (keeping previous data): %v", err)
				}
			}
		}()
	})
}

// RefreshSchemaNow performs a one-shot fetch from canonicalSchemaURL and
// replaces the in-memory schema on success. The previous schema is retained on
// failure — never returns the binary into a no-schema state. Exposed both for
// the background loop and for the MCP `refresh_osquery_schema` tool.
func RefreshSchemaNow() error {
	body, err := fetchCanonicalSchema(canonicalSchemaURL)
	if err != nil {
		return err
	}
	tables, err := parseSchemaJSON(body)
	if err != nil {
		return fmt.Errorf("parse fetched schema: %w", err)
	}
	now := time.Now()
	storeSchema(tables, schemaSource{
		Origin:    "live",
		URL:       canonicalSchemaURL,
		FetchedAt: &now,
	})
	logrus.Infof("Refreshed osquery schema from %s (%d tables)", canonicalSchemaURL, len(tables))
	return nil
}

func fetchCanonicalSchema(url string) ([]byte, error) {
	client := &http.Client{Timeout: fetchTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "fleet-mcp/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: HTTP %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024*1024)) // hard cap @ 16 MiB
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return body, nil
}

func parseSchemaJSON(data []byte) ([]SchemaTable, error) {
	var tables []SchemaTable
	if err := json.Unmarshal(data, &tables); err != nil {
		return nil, fmt.Errorf("unmarshal schema: %w", err)
	}
	if len(tables) == 0 {
		return nil, errors.New("schema payload contained zero tables")
	}
	return tables, nil
}

func storeSchema(tables []SchemaTable, src schemaSource) {
	idx := make(map[string]SchemaTable, len(tables))
	for _, t := range tables {
		idx[strings.ToLower(t.Name)] = t
	}
	schemaMu.Lock()
	schemaTables = tables
	schemaIndex = idx
	schemaMu.Unlock()
	srcCopy := src
	schemaInfo.Store(&srcCopy)
}

// SchemaSource returns metadata describing where the in-memory schema was last
// loaded from (live HTTP vs embedded snapshot) and when. Safe for concurrent
// callers; cheap (atomic load).
func SchemaSource() schemaSource {
	if p := schemaInfo.Load(); p != nil {
		return *p
	}
	return schemaSource{}
}

// GetOsquerySchema returns a curated subset of common tables filtered by
// platform. Use GetOsquerySchemaForTables for full coverage of specific tables.
// Platform values: "darwin"/"macos", "windows", "linux", "chrome"/"chromeos",
// "all" (returns the curated set unfiltered).
func GetOsquerySchema(platform string) ([]SchemaTable, error) {
	p := normalizePlatform(platform)

	schemaMu.RLock()
	defer schemaMu.RUnlock()

	if len(schemaIndex) == 0 {
		return nil, errors.New("schema not loaded")
	}

	out := make([]SchemaTable, 0, len(defaultCuratedTables))
	missing := []string{}
	for _, name := range defaultCuratedTables {
		t, ok := schemaIndex[strings.ToLower(name)]
		if !ok {
			missing = append(missing, name)
			continue
		}
		if p == "" || p == "all" || tablePlatformMatches(t, p) {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no curated tables found for platform %q (supported: darwin/macos, windows, linux, chrome/chromeos, all)", platform)
	}
	if len(missing) > 0 {
		// Not fatal — the canonical schema may have renamed/dropped a table the
		// curated list still references. Log so we notice.
		logrus.Debugf("Curated tables missing from canonical schema: %v", missing)
	}
	return out, nil
}

// GetOsquerySchemaForTables returns the canonical schema for a specific set of
// table names (case-insensitive). Unknown names are reported as a single error
// alongside the known matches so the caller can still use what was found.
func GetOsquerySchemaForTables(names []string) ([]SchemaTable, error) {
	schemaMu.RLock()
	defer schemaMu.RUnlock()

	if len(schemaIndex) == 0 {
		return nil, errors.New("schema not loaded")
	}

	out := make([]SchemaTable, 0, len(names))
	unknown := []string{}
	for _, raw := range names {
		n := strings.ToLower(strings.TrimSpace(raw))
		if n == "" {
			continue
		}
		if t, ok := schemaIndex[n]; ok {
			out = append(out, t)
		} else {
			unknown = append(unknown, raw)
		}
	}
	if len(out) == 0 && len(unknown) > 0 {
		return nil, fmt.Errorf("none of the requested tables exist in canonical schema: %v", unknown)
	}
	if len(unknown) > 0 {
		return out, fmt.Errorf("unknown tables (omitted from result): %v", unknown)
	}
	return out, nil
}

func tablePlatformMatches(t SchemaTable, normalizedPlatform string) bool {
	if normalizedPlatform == "" || normalizedPlatform == "all" {
		return true
	}
	for _, tp := range t.Platforms {
		if normalizePlatform(tp) == normalizedPlatform {
			return true
		}
	}
	return false
}

// IsTableSupportedOnPlatform returns true if the table either supports the
// platform or is unknown to the canonical schema (we never block on unknowns —
// Fleet itself decides). Returns false only for known-incompatible pairs.
func IsTableSupportedOnPlatform(tableName, platform string) bool {
	p := normalizePlatform(platform)
	schemaMu.RLock()
	defer schemaMu.RUnlock()

	t, ok := schemaIndex[strings.ToLower(tableName)]
	if !ok {
		return true
	}
	for _, tp := range t.Platforms {
		if normalizePlatform(tp) == p {
			return true
		}
	}
	return false
}

// columnTypeRe captures `<column> <op> <number>` pairs so the validator can
// flag the common bug class of comparing a TEXT column against a bare integer
// literal (e.g. `result_code = 2`). Quoted strings, NULL, and parameter
// placeholders are intentionally not captured.
var columnTypeRe = regexp.MustCompile(`(?i)\b([a-z_][a-z0-9_]*)\s*(?:=|!=|<>|>=|<=|>|<)\s*([0-9]+)\b`)

// fromJoinTableRe captures table tokens following FROM / JOIN keywords. Simpler
// and a little more robust than the previous tokenizer since it survives
// embedded newlines and parens.
var fromJoinTableRe = regexp.MustCompile(`(?i)\b(?:FROM|JOIN)\s+(\(?\s*)([a-z_][a-z0-9_]*)`)

// ValidateSQLForPlatforms is a best-effort pre-flight check for live queries.
// It enforces two contracts:
//  1. Every referenced table must support every supplied platform.
//  2. Bare integer literals in WHERE clauses must not be compared against TEXT
//     columns (the #1 cause of silent zero-row queries — see the
//     windows_update_history.result_code regression).
//
// Either failure produces an actionable error pointing the caller back at
// get_osquery_schema. Quoted-string-vs-integer comparisons are NOT flagged
// because osquery / SQLite type-coerces them and they are not a real bug
// source in our query corpus.
func ValidateSQLForPlatforms(sql string, platforms []string) error {
	if sql == "" {
		return nil
	}

	// Resolve the tables referenced in the SQL. Used by both checks.
	usedTables := extractTables(sql)

	// (1) Platform compatibility.
	if len(platforms) > 0 {
		normalized := make([]string, 0, len(platforms))
		for _, p := range platforms {
			if np := normalizePlatform(p); np != "" {
				normalized = append(normalized, np)
			}
		}
		for _, table := range usedTables {
			for _, platform := range normalized {
				if !IsTableSupportedOnPlatform(table, platform) {
					return fmt.Errorf(
						"SQL validation error: table %q is not supported on platform %q. "+
							"Call get_osquery_schema(platform=%q) to see supported tables, "+
							"or get_osquery_schema(tables=%q) to verify the table name is correct. "+
							"Common cause: assuming a table exists on this platform without checking the schema.",
						table, platform, platform, table,
					)
				}
			}
		}
	}

	// (2) Defense-in-depth: TEXT column compared to bare integer literal.
	// Only fires when the column belongs to a known table and every matching
	// table agrees the column is TEXT — avoids false positives from same-named
	// columns in JOINed tables with diverging types.
	if len(usedTables) == 0 {
		return nil
	}
	schemaMu.RLock()
	defer schemaMu.RUnlock()

	for _, m := range columnTypeRe.FindAllStringSubmatch(sql, -1) {
		colName := m[1]
		lit := m[2]
		if isSQLKeyword(colName) {
			continue
		}
		// Skip unambiguous integer-domain identifiers like LIMIT/OFFSET trailing
		// numbers — those don't take a column on the LHS but the regex requires
		// a leading word, so SQL keywords are filtered above.
		var (
			matchedKnown bool
			allText      = true
		)
		for _, table := range usedTables {
			canon, ok := schemaIndex[strings.ToLower(table)]
			if !ok {
				continue
			}
			for _, c := range canon.Columns {
				if !strings.EqualFold(c.Name, colName) {
					continue
				}
				matchedKnown = true
				if !strings.EqualFold(c.Type, "text") {
					allText = false
				}
			}
		}
		if matchedKnown && allText {
			// Don't flag literal 0 / 1 against TEXT (some queries use them as
			// boolean flags via SQLite coercion). The original bug used 2/4 etc.
			if v, err := strconv.Atoi(lit); err == nil && (v == 0 || v == 1) {
				continue
			}
			return fmt.Errorf(
				"SQL validation error: column %q is TEXT but compared against bare integer literal %s. "+
					"TEXT columns require quoted string literals (e.g. %s = 'Succeeded'). "+
					"Call get_osquery_schema(tables=%q) to confirm column type and valid enum values. "+
					"This is the most common cause of silent zero-row queries on Fleet.",
				colName, lit, colName, strings.Join(usedTables, ","),
			)
		}
	}

	return nil
}

func extractTables(sql string) []string {
	out := []string{}
	seen := map[string]struct{}{}
	for _, m := range fromJoinTableRe.FindAllStringSubmatch(sql, -1) {
		name := strings.TrimSpace(m[2])
		key := strings.ToLower(name)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, name)
	}
	return out
}

// sqlKeywords lists tokens the table/column extractor must not treat as
// identifiers when they appear right of FROM/JOIN or left of `=`.
var sqlKeywords = map[string]struct{}{
	"select": {}, "from": {}, "where": {}, "and": {}, "or": {}, "not": {},
	"in": {}, "is": {}, "null": {}, "like": {}, "between": {},
	"join": {}, "inner": {}, "outer": {}, "left": {}, "right": {},
	"on": {}, "as": {}, "by": {}, "group": {}, "order": {},
	"limit": {}, "offset": {}, "asc": {}, "desc": {},
	"case": {}, "when": {}, "then": {}, "else": {}, "end": {},
	"distinct": {}, "all": {}, "union": {}, "exists": {},
	"true": {}, "false": {},
}

func isSQLKeyword(tok string) bool {
	_, ok := sqlKeywords[strings.ToLower(tok)]
	return ok
}
