// Package mysql provides the MySQL datastore implementation for the chart bounded context.
package mysql

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/fleetdm/fleet/v4/server/chart/internal/types"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxdb"
	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	platform_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

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

// FindOnlineHostIDs returns host IDs that are "online" at `now` per the same
// per-host predicate used by the hosts list status=online filter
// (filterHostsByStatus in server/datastore/mysql/hosts.go): the host has a
// host_seen_times row whose seen_time falls within the host's own check-in
// interval (LEAST of distributed_interval and config_tls_refresh) plus the
// OnlineIntervalBuffer grace period.
//
// Because host_seen_times is updated only by osquery check-ins, MDM-only
// mobile devices (iOS, iPadOS, Android) are currently excluded by design.
func (ds *Datastore) FindOnlineHostIDs(ctx context.Context, now time.Time, disabledFleetIDs []uint) ([]uint, error) {
	query := fmt.Sprintf(`
		SELECT h.id
		FROM hosts h
		JOIN host_seen_times hst ON h.id = hst.host_id
		WHERE DATE_ADD(hst.seen_time,
			INTERVAL LEAST(h.distributed_interval, h.config_tls_refresh) + %d SECOND
		) > ?`, fleet.OnlineIntervalBuffer)
	args := []any{now.UTC()}

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

// The matcher list and TrackedCriticalCVEs exist as performance optimizations.
// TODO: implement bitmap compression so we can collect more CVE data.
// TODO: implement more filtering options for users.

// cveSoftwareMatcher filters `software` rows by a MySQL LIKE pattern and an
// optional source allowlist. Empty Sources means any source.
type cveSoftwareMatcher struct {
	NamePattern string
	Sources     []string
}

// trackedCVESoftwareMatchers is the hard-coded curated list of software
// whose critical CVEs contribute to the CVE chart. Patterns are deliberately
// broad (trailing `%`) so packaging variants (Chrome Beta/Canary, Firefox
// ESR/Nightly, kernel metapackages) are absorbed without maintenance.
var trackedCVESoftwareMatchers = []cveSoftwareMatcher{
	// Browsers.
	{"Google Chrome%", nil},
	{"Firefox%", nil},
	{"Mozilla Firefox%", nil},
	{"Brave Browser%", nil},
	{"Safari%", []string{"apps"}},
	{"Opera%", nil},

	// Microsoft Office.
	{"Microsoft Word%", nil},
	{"Microsoft Excel%", nil},
	{"Microsoft PowerPoint%", nil},
	{"Microsoft Outlook%", nil},
	{"Microsoft Office%", nil},

	// Adobe.
	{"Adobe Flash%", nil},
	{"Shockwave Flash%", nil},
	{"Adobe Acrobat%", nil},

	// Linux kernel. Debian/Ubuntu metapackages are linux-image-* and
	// linux-signed-image-*; RHEL/Fedora/Amazon Linux are kernel-* (confirmed
	// via server/vulnerabilities/osv/analyzer.go rhelKernelPackages).
	{"linux-image-%", []string{"deb_packages"}},
	{"linux-signed-image-%", []string{"deb_packages"}},
	{"kernel-%", []string{"rpm_packages"}},
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

// TrackedCriticalCVEs returns the deduplicated set of CVE IDs that are
// (a) linked to any `software` row matching trackedCVESoftwareMatchers with
// `cve_meta.cvss_score >= 9.0`, OR (b) present in
// `operating_system_vulnerabilities` with `cve_meta.cvss_score >= 9.0`.
//
// Returns a non-nil empty slice when no CVEs match, so callers can
// distinguish "filter resolved to empty" from "no filter requested" (nil).
// See GetSCDData for how empty vs nil is interpreted at the query layer.
//
// TODO: replace with user-configurable filtering. See the
// matcher-list comment above.
func (ds *Datastore) TrackedCriticalCVEs(ctx context.Context) ([]string, error) {
	const criticalCVSS = 9.0
	set := make(map[string]struct{})

	// Software-side: build an OR-chained matcher clause. Each matcher adds one
	// `(name LIKE ? AND source IN (?))` or `name LIKE ?` subclause.
	softwareArgs := []any{criticalCVSS}
	matcherClauses := make([]string, 0, len(trackedCVESoftwareMatchers))
	for _, m := range trackedCVESoftwareMatchers {
		if len(m.Sources) == 0 {
			matcherClauses = append(matcherClauses, "s.name LIKE ?")
			softwareArgs = append(softwareArgs, m.NamePattern)
		} else {
			matcherClauses = append(matcherClauses, "(s.name LIKE ? AND s.source IN (?))")
			softwareArgs = append(softwareArgs, m.NamePattern, m.Sources)
		}
	}
	softwareQuery := `
		SELECT DISTINCT sc.cve
		FROM software_cve sc
		JOIN software s  ON s.id = sc.software_id
		JOIN cve_meta cm ON cm.cve = sc.cve
		WHERE cm.cvss_score >= ?
		  AND (` + strings.Join(matcherClauses, " OR ") + `)`

	expanded, expandedArgs, err := sqlx.In(softwareQuery, softwareArgs...)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "expand tracked-CVE software args")
	}
	expanded = ds.rebind(expanded)
	if err := streamCVEStrings(ctx, ds.reader(ctx), expanded, expandedArgs, set); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "stream tracked-CVE software results")
	}

	// OS-side: all OS vulnerabilities at or above the critical threshold.
	// Fleet's OS vuln coverage is already scoped to desktop OSes.
	const osQuery = `
		SELECT DISTINCT osv.cve
		FROM operating_system_vulnerabilities osv
		JOIN cve_meta cm ON cm.cve = osv.cve
		WHERE cm.cvss_score >= ?`
	if err := streamCVEStrings(ctx, ds.reader(ctx), osQuery, []any{criticalCVSS}, set); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "stream tracked-CVE OS results")
	}

	out := make([]string, 0, len(set))
	for cve := range set {
		out = append(out, cve)
	}
	return out, nil
}

// streamCVEStrings runs a single-column SELECT of CVE IDs and inserts each
// into the provided set. Helper for TrackedCriticalCVEs. Streams rather than
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
