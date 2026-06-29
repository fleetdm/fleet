// Package mysql provides the MySQL datastore implementation for the chart bounded context.
package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/api"
	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

// onlineIntervalBufferSeconds mirrors fleet.OnlineIntervalBuffer (the grace
// period added on top of a host's own check-in interval before it's
// considered offline). Duplicated rather than imported because the chart
// bounded context must not depend on server/fleet — arch test enforced.
const onlineIntervalBufferSeconds = 60

// mobileOnlineWindowSeconds is the window within which a mobile
// (iOS/iPadOS/Android) host's most recent MDM activity signal must fall for it
// to count as online. Mobile MDM devices have no osquery check-in interval
// (distributed_interval/config_tls_refresh are 0), so instead of a per-host
// interval we anchor the window to the iOS/iPadOS refetch cadence (1 hour; see
// ListIOSAndIPadOSToRefetch) plus the same grace buffer used for osquery hosts.
const mobileOnlineWindowSeconds = 3600 + onlineIntervalBufferSeconds

// neverTimestamp mirrors server.NeverTimestamp, the sentinel written to
// detail_updated_at before a host's first full detail refetch. Duplicated
// rather than imported because the chart bounded context must not depend on the
// server package — arch test enforced.
const neverTimestamp = "2000-01-01 00:00:00"

// Datastore is the MySQL implementation of the chart datastore.
type Datastore struct {
	primary *sqlx.DB
	replica *sqlx.DB
	logger  *slog.Logger
}

// NewDatastore creates a new MySQL datastore for the chart bounded context.
func NewDatastore(conns *platform_mysql.DBConnections, logger *slog.Logger) *Datastore {
	return &Datastore{primary: conns.Primary, replica: conns.Replica, logger: logger}
}

// Ensure Datastore implements types.Datastore at compile time.
var _ types.Datastore = (*Datastore)(nil)

func (ds *Datastore) reader(ctx context.Context) sqlx.QueryerContext {
	if ctxdb.IsPrimaryRequired(ctx) {
		return ds.primary
	}
	return ds.replica
}

func (ds *Datastore) writer(_ context.Context) *sqlx.DB {
	return ds.primary
}

// rebind rewrites a query from ? placeholders to the driver-specific format.
func (ds *Datastore) rebind(query string) string {
	return ds.primary.Rebind(query)
}

func (ds *Datastore) GetHostIDsForFilter(ctx context.Context, hostFilter *types.HostFilter) ([]uint, error) {
	subquery, args := buildHostFilterClauses(hostFilter)

	query := fmt.Sprintf(`SELECT h.id FROM hosts h WHERE 1=1 %s`, subquery)

	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand host IDs filter query args")
	}
	query = ds.rebind(query)

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, query, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get host IDs for filter")
	}
	return ids, nil
}

// FindOnlineHostIDs returns host IDs that are "online" at `now`, using a
// platform-specific predicate:
//
//   - Non-mobile (osquery-capable) hosts use the same predicate as the hosts
//     list status=online filter (filterHostsByStatus in
//     server/datastore/mysql/hosts.go): a host_seen_times row whose seen_time
//     falls within the host's own check-in interval (LEAST of
//     distributed_interval and config_tls_refresh) plus the OnlineIntervalBuffer
//     grace period.
//   - Mobile hosts (iOS, iPadOS, Android) have no osquery check-in interval, so
//     they use their MDM activity signal — the most recent of
//     nano_enrollments.last_seen_at (bumped on every MDM check-in, and only
//     considered for enabled enrollments since last_seen_at is also bumped when
//     an enrollment is disabled on checkout) and
//     host_seen_times.seen_time, falling back to detail_updated_at (the
//     neverTimestamp sentinel treated as null) — within mobileOnlineWindowSeconds
//     of `now`. There is deliberately no created_at fallback: a freshly enrolled
//     device that never checked in is not "online".
func (ds *Datastore) FindOnlineHostIDs(ctx context.Context, now time.Time, disabledFleetIDs []uint) ([]uint, error) {
	query := fmt.Sprintf(`
		SELECT h.id
		FROM hosts h
			LEFT JOIN host_seen_times hst ON h.id = hst.host_id
			LEFT JOIN nano_enrollments ne ON ne.id = h.uuid
				AND ne.enabled = 1
				AND ne.type IN ('Device', 'User Enrollment (Device)')
		WHERE (
			(
				h.platform NOT IN ('ios', 'ipados', 'android')
				AND hst.seen_time IS NOT NULL
				AND DATE_ADD(hst.seen_time,
					INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND
				) > ?
			)
			OR
			(
				h.platform IN ('ios', 'ipados', 'android')
				AND DATE_ADD(
					COALESCE(
						GREATEST(
							COALESCE(hst.seen_time, ne.last_seen_at),
							COALESCE(ne.last_seen_at, hst.seen_time)
						),
						NULLIF(h.detail_updated_at, ?)
					),
					INTERVAL %d SECOND
				) > ?
			)
		)`, onlineIntervalBufferSeconds, mobileOnlineWindowSeconds)
	args := []any{now.UTC(), neverTimestamp, now.UTC()}

	if len(disabledFleetIDs) > 0 {
		query += ` AND (h.team_id IS NULL OR h.team_id NOT IN (?))`
		args = append(args, disabledFleetIDs)
	}

	expanded, expandedArgs, err := sqlx.In(query, args...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand online host args")
	}
	expanded = ds.rebind(expanded)

	var ids []uint
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &ids, expanded, expandedArgs...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "find online host IDs")
	}
	return ids, nil
}

// The matcher list exists as a performance optimization that bounds which CVEs
// the chart collects.
// TODO: implement bitmap compression so we can collect more CVE data.

// cveSoftwareMatcher filters `software` rows by a MySQL LIKE pattern and an
// optional source allowlist. Empty Sources means any source. Category groups
// the matcher under one of the api.CVECategory* keys so the read-time filter
// can include/exclude whole categories.
type cveSoftwareMatcher struct {
	Category    string
	NamePattern string
	Sources     []string
}

// trackedCVESoftwareMatchers is the hard-coded curated list of software whose
// CVEs contribute to the CVE chart. Patterns are deliberately broad (trailing
// `%`) so packaging variants (Chrome Beta/Canary, Firefox ESR/Nightly, kernel
// metapackages) are absorbed without maintenance. The kernel matchers belong
// to the OS category alongside operating-system vulnerabilities.
var trackedCVESoftwareMatchers = []cveSoftwareMatcher{
	// Browsers.
	{api.CVECategoryBrowsers, "Google Chrome%", nil},
	{api.CVECategoryBrowsers, "Firefox%", nil},
	{api.CVECategoryBrowsers, "Mozilla Firefox%", nil},
	{api.CVECategoryBrowsers, "Brave Browser%", nil},
	{api.CVECategoryBrowsers, "Safari%", []string{"apps"}},
	{api.CVECategoryBrowsers, "Opera%", nil},

	// Microsoft Office.
	{api.CVECategoryOffice, "Microsoft Word%", nil},
	{api.CVECategoryOffice, "Microsoft Excel%", nil},
	{api.CVECategoryOffice, "Microsoft PowerPoint%", nil},
	{api.CVECategoryOffice, "Microsoft Outlook%", nil},
	{api.CVECategoryOffice, "Microsoft Office%", nil},

	// Adobe.
	{api.CVECategoryAdobe, "Adobe Flash%", nil},
	{api.CVECategoryAdobe, "Shockwave Flash%", nil},
	{api.CVECategoryAdobe, "Adobe Acrobat%", nil},

	// Linux kernel (OS category). Debian/Ubuntu metapackages are linux-image-*
	// and linux-signed-image-*; RHEL/Fedora/Amazon Linux are kernel-*
	// (confirmed via server/vulnerabilities/osv/analyzer.go rhelKernelPackages).
	{api.CVECategoryOS, "linux-image-%", []string{"deb_packages"}},
	{api.CVECategoryOS, "linux-signed-image-%", []string{"deb_packages"}},
	{api.CVECategoryOS, "kernel-%", []string{"rpm_packages"}},
}

// AffectedHostIDsByCVE returns host IDs grouped by CVE, scoped to the given
// cves set. It streams two joins (software-level and OS-level vulnerabilities)
// and merges the results into a single map. Duplicates across sources are
// harmless — the downstream HostIDsToBlob setBit is idempotent.
//
// nil or empty cves returns an empty map without running any query.
// TODO: support `nil` meaning "all CVEs" once bitmap compression is implemented.
func (ds *Datastore) AffectedHostIDsByCVE(ctx context.Context, disabledFleetIDs []uint, cves []string) (map[string][]uint, error) {
	result := make(map[string][]uint)
	if len(cves) == 0 {
		return result, nil
	}

	// Both subqueries gain a hosts JOIN + WHERE only when there are fleets to
	// exclude. Skipping the JOIN entirely when the slice is empty keeps the
	// existing query plan unchanged for the common case (no fleets disabled).
	swQuery := `
		SELECT sc.cve, hs.host_id
		FROM software_cve sc
		JOIN host_software hs ON hs.software_id = sc.software_id`
	osQuery := `
		SELECT osv.cve, hos.host_id
		FROM operating_system_vulnerabilities osv
		JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id`

	swWhere := []string{"sc.cve IN (?)"}
	osWhere := []string{"osv.cve IN (?)"}
	swArgs := []any{cves}
	osArgs := []any{cves}

	if len(disabledFleetIDs) > 0 {
		swQuery += `
			JOIN hosts h ON h.id = hs.host_id`
		swWhere = append(swWhere, "(h.team_id IS NULL OR h.team_id NOT IN (?))")
		swArgs = append(swArgs, disabledFleetIDs)

		osQuery += `
			JOIN hosts h ON h.id = hos.host_id`
		osWhere = append(osWhere, "(h.team_id IS NULL OR h.team_id NOT IN (?))")
		osArgs = append(osArgs, disabledFleetIDs)
	}

	swQuery += " WHERE " + strings.Join(swWhere, " AND ")
	osQuery += " WHERE " + strings.Join(osWhere, " AND ")

	if err := ds.streamCVEHostPairs(ctx, swQuery, swArgs, result); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "stream software CVE host pairs")
	}
	if err := ds.streamCVEHostPairs(ctx, osQuery, osArgs, result); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "stream OS CVE host pairs")
	}

	return result, nil
}

// CollectibleCVEs returns the deduplicated set of CVE IDs, at all severities,
// that are (a) linked to any `software` row matching trackedCVESoftwareMatchers,
// OR (b) present in `operating_system_vulnerabilities`. This is the wide set the
// CVE collector records; display-time severity/category/EPSS narrowing happens
// at read time via ResolveCVEChartEntities.
//
// Returns a non-nil empty slice when no CVEs match.
func (ds *Datastore) CollectibleCVEs(ctx context.Context) ([]string, error) {
	set := make(map[string]struct{})

	// Software-side: every tracked-matcher CVE, no cve_meta join or severity
	// filter — we collect all severities and narrow only at read time.
	if swClause, swArgs, ok := softwareMatcherClause(nil); ok {
		swQuery := `
			SELECT DISTINCT sc.cve
			FROM software_cve sc
			JOIN software s ON s.id = sc.software_id
			WHERE ` + swClause
		expanded, expandedArgs, err := sqlx.In(swQuery, swArgs...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "expand collectible-CVE software args")
		}
		expanded = ds.rebind(expanded)
		if err := streamCVEStrings(ctx, ds.reader(ctx), expanded, expandedArgs, set); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "stream collectible-CVE software results")
		}
	}

	// OS-side: all OS vulnerabilities. Fleet's OS vuln coverage is already
	// scoped to desktop OSes.
	const osQuery = `SELECT DISTINCT osv.cve FROM operating_system_vulnerabilities osv`
	if err := streamCVEStrings(ctx, ds.reader(ctx), osQuery, nil, set); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "stream collectible-CVE OS results")
	}

	return setToSlice(set), nil
}

// ResolveCVEChartEntities resolves the read-time CVE allow-set by intersecting
// the curated universe with the filter's predicates (software category, CVSS
// range, EPSS range, known-exploit) and subtracting any excluded CVEs.
//
// With the default filter (CVSS 9.0–10.0, all categories, no EPSS bound, no
// known-exploit, no exclusions) this reproduces the iteration-1 "tracked
// critical CVEs" set, so the chart's default display is unchanged.
//
// Returns a non-nil empty slice when the filter resolves to nothing, so callers
// never pass nil to GetSCDData (which would mean "all collected", leaking
// lower-severity CVEs into the chart).
func (ds *Datastore) ResolveCVEChartEntities(ctx context.Context, filter types.CVEChartFilter) ([]string, error) {
	set := make(map[string]struct{})
	cats := categorySet(filter.Categories) // nil == all categories
	metaClause, metaArgs := cveMetaPredicate(filter)

	// Software-side: skip entirely when no matcher falls in the selected
	// categories (e.g. only the OS category is selected).
	if swClause, swArgs, ok := softwareMatcherClause(cats); ok {
		args := slices.Concat(swArgs, metaArgs)
		swQuery := `
			SELECT DISTINCT sc.cve
			FROM software_cve sc
			JOIN software s  ON s.id = sc.software_id
			JOIN cve_meta cm ON cm.cve = sc.cve
			WHERE ` + swClause + metaClause
		expanded, expandedArgs, err := sqlx.In(swQuery, args...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "expand resolve-CVE software args")
		}
		expanded = ds.rebind(expanded)
		if err := streamCVEStrings(ctx, ds.reader(ctx), expanded, expandedArgs, set); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "stream resolve-CVE software results")
		}
	}

	// OS-side: only when the OS category is selected (or no category filter).
	if cats == nil || containsCategory(cats, api.CVECategoryOS) {
		osQuery := `
			SELECT DISTINCT osv.cve
			FROM operating_system_vulnerabilities osv
			JOIN cve_meta cm ON cm.cve = osv.cve
			WHERE 1=1` + metaClause
		expanded, expandedArgs, err := sqlx.In(osQuery, metaArgs...)
		if err != nil {
			return nil, ctxerr.Wrap(ctx, err, "expand resolve-CVE OS args")
		}
		expanded = ds.rebind(expanded)
		if err := streamCVEStrings(ctx, ds.reader(ctx), expanded, expandedArgs, set); err != nil {
			return nil, ctxerr.Wrap(ctx, err, "stream resolve-CVE OS results")
		}
	}

	// Subtract excluded CVEs. Excluding a CVE not in the set is a no-op, so a
	// user may freely exclude CVEs that were never collected.
	for _, cve := range filter.ExcludeCVEs {
		delete(set, cve)
	}

	return setToSlice(set), nil
}

// softwareMatcherClause builds the OR-chained software matcher subclause for the
// matchers in the selected categories, plus its args. A nil categories map
// means "all categories". Returns ok=false when no matcher falls in the
// selected categories, signaling the caller to skip the software-side query.
func softwareMatcherClause(categories map[string]struct{}) (clause string, args []any, ok bool) {
	subclauses := make([]string, 0, len(trackedCVESoftwareMatchers))
	for _, m := range trackedCVESoftwareMatchers {
		if categories != nil {
			if _, sel := categories[m.Category]; !sel {
				continue
			}
		}
		if len(m.Sources) == 0 {
			subclauses = append(subclauses, "s.name LIKE ?")
			args = append(args, m.NamePattern)
		} else {
			subclauses = append(subclauses, "(s.name LIKE ? AND s.source IN (?))")
			args = append(args, m.NamePattern, m.Sources)
		}
	}
	if len(subclauses) == 0 {
		return "", nil, false
	}
	return "(" + strings.Join(subclauses, " OR ") + ")", args, true
}

// cveMetaPredicate builds the cve_meta WHERE fragment (with a leading " AND ")
// shared by the software- and OS-side resolve queries, plus its args. The CVSS
// range is always applied; EPSS bounds and the known-exploit flag are optional.
func cveMetaPredicate(filter types.CVEChartFilter) (string, []any) {
	clauses := []string{"cm.cvss_score >= ?", "cm.cvss_score <= ?"}
	args := []any{filter.CVSSMin, filter.CVSSMax}
	if filter.EPSSMin != nil {
		clauses = append(clauses, "cm.epss_probability >= ?")
		args = append(args, *filter.EPSSMin)
	}
	if filter.EPSSMax != nil {
		clauses = append(clauses, "cm.epss_probability <= ?")
		args = append(args, *filter.EPSSMax)
	}
	if filter.KnownExploit {
		clauses = append(clauses, "cm.cisa_known_exploit = 1")
	}
	return " AND " + strings.Join(clauses, " AND "), args
}

func categorySet(categories []string) map[string]struct{} {
	if len(categories) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(categories))
	for _, c := range categories {
		set[c] = struct{}{}
	}
	return set
}

func containsCategory(set map[string]struct{}, c string) bool {
	_, ok := set[c]
	return ok
}

func setToSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for cve := range set {
		out = append(out, cve)
	}
	return out
}

// streamCVEStrings runs a single-column SELECT of CVE IDs and inserts each
// into the provided set. Helper for the CVE resolver queries. Streams rather than
// using SelectContext so we don't materialize the full result set.
func streamCVEStrings(ctx context.Context, q sqlx.QueryerContext, query string, args []any, out map[string]struct{}) error {
	// sqlclosecheck can't see through the QueryerContext interface to verify
	// Close() is reached. The defer below provides it.
	rows, err := q.QueryxContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return err
	}
	defer rows.Close()

	var cve string
	for rows.Next() {
		if err := rows.Scan(&cve); err != nil {
			return err
		}
		out[cve] = struct{}{}
	}
	return rows.Err()
}

// streamCVEHostPairs runs a query yielding (cve, host_id) pairs and appends
// host IDs into out under each CVE key. Streams rather than materializing the
// join result, since on a large fleet the (cve, host_id) row count can reach
// millions.
//
// args are expanded via sqlx.In for slice arguments (e.g. team IDs) and
// rebinds to the driver dialect.
func (ds *Datastore) streamCVEHostPairs(ctx context.Context, query string, args []any, out map[string][]uint) error {
	if len(args) > 0 {
		expanded, expandedArgs, err := sqlx.In(query, args...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "expand CVE-host-pair args")
		}
		query = ds.rebind(expanded)
		args = expandedArgs
	}

	// sqlclosecheck can't see through the QueryerContext interface to verify
	// Close() is reached. The defer below provides it.
	rows, err := ds.reader(ctx).QueryxContext(ctx, query, args...) //nolint:sqlclosecheck
	if err != nil {
		return err
	}
	defer rows.Close()

	var cve string
	var hostID uint
	for rows.Next() {
		if err := rows.Scan(&cve, &hostID); err != nil {
			return err
		}
		out[cve] = append(out[cve], hostID)
	}
	return rows.Err()
}

// buildHostFilterClauses translates a HostFilter into SQL WHERE clauses for
// the hosts table. Uses "h" as the table alias. Args may contain slices —
// caller must use sqlx.In to expand them.
func buildHostFilterClauses(filter *types.HostFilter) (string, []any) {
	if filter == nil {
		return "", nil
	}

	var clauses []string
	var args []any

	if filter.TeamIDs != nil {
		// Empty non-nil: caller is team-scoped with zero accessible teams;
		// emit a guaranteed-empty clause so we never run IN () and never return
		// hosts the caller can't see.
		if len(filter.TeamIDs) == 0 {
			clauses = append(clauses, "1=0")
		} else {
			// Split "no team" (id 0) from real team ids — the two map to
			// different SQL (IS NULL vs = ?), so they're OR-ed together when
			// both are present.
			var positive []uint
			includesNoTeam := false
			for _, tid := range filter.TeamIDs {
				if tid == 0 {
					includesNoTeam = true
				} else {
					positive = append(positive, tid)
				}
			}
			switch {
			case includesNoTeam && len(positive) > 0:
				clauses = append(clauses, "(h.team_id IS NULL OR h.team_id IN (?))")
				args = append(args, positive)
			case includesNoTeam:
				clauses = append(clauses, "h.team_id IS NULL")
			default:
				clauses = append(clauses, "h.team_id IN (?)")
				args = append(args, positive)
			}
		}
	}

	if len(filter.LabelIDs) > 0 {
		clauses = append(clauses, "h.id IN (SELECT DISTINCT host_id FROM label_membership WHERE label_id IN (?))")
		args = append(args, filter.LabelIDs)
	}

	if len(filter.Platforms) > 0 {
		clauses = append(clauses, "h.platform IN (?)")
		args = append(args, filter.Platforms)
	}

	if len(filter.IncludeHostIDs) > 0 {
		clauses = append(clauses, "h.id IN (?)")
		args = append(args, filter.IncludeHostIDs)
	}

	if len(filter.ExcludeHostIDs) > 0 {
		clauses = append(clauses, "h.id NOT IN (?)")
		args = append(args, filter.ExcludeHostIDs)
	}

	if len(clauses) == 0 {
		return "", nil
	}

	return " AND " + strings.Join(clauses, " AND "), args
}
