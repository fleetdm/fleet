package mysql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	common_mysql "github.com/fleetdm/fleet/v4/server/platform/mysql"
	"github.com/jmoiron/sqlx"
)

var vulnerabilitiesAllowedOrderKeys = common_mysql.OrderKeyAllowlist{
	"cve":                    "cve",
	"cvss_score":             "cvss_score",
	"epss_probability":       "epss_probability",
	"cisa_known_exploit":     "cisa_known_exploit",
	"cve_published":          "cve_published",
	"created_at":             "created_at",
	"host_count":             "hosts_count",
	"hosts_count":            "hosts_count",
	"host_count_updated_at":  "hosts_count_updated_at",
	"hosts_count_updated_at": "hosts_count_updated_at",
}

func (ds *Datastore) Vulnerability(ctx context.Context, cve string, teamID *uint, includeCVEScores bool) (*fleet.VulnerabilityWithMetadata, error) {
	var vuln fleet.VulnerabilityWithMetadata

	eeSelectStmt := `
		SELECT DISTINCT
			cm.cve,
			LEAST(COALESCE(osv.created_at, NOW()), COALESCE(sc.created_at, NOW())) AS created_at,
			COALESCE(osv.source, sc.source, 0) AS source,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published as cve_published,
			cm.description,
			COALESCE(vhc.host_count, 0) as hosts_count,
			COALESCE(vhc.updated_at, NOW()) as hosts_count_updated_at
		FROM cve_meta cm
		JOIN (
			SELECT cve
			FROM software_cve
			WHERE cve = ?

			UNION

			SELECT cve
			FROM operating_system_vulnerabilities
			WHERE cve = ?
		) AS cve_table ON cm.cve = cve_table.cve
		LEFT JOIN operating_system_vulnerabilities osv ON osv.cve = cm.cve
		LEFT JOIN software_cve sc ON sc.cve = cm.cve
		LEFT JOIN vulnerability_host_counts vhc ON cm.cve = vhc.cve
`

	freeSelectStmt := `
		SELECT DISTINCT
			union_cve.cve,
			LEAST(COALESCE(osv.created_at, NOW()), COALESCE(sc.created_at, NOW())) AS created_at,
			COALESCE(osv.source, sc.source, 0) AS source,
			COALESCE(vhc.host_count, 0) as hosts_count,
			COALESCE(vhc.updated_at, NOW()) as hosts_count_updated_at
		FROM (
			SELECT cve, created_at, source
			FROM operating_system_vulnerabilities
			WHERE cve = ?

			UNION

			SELECT cve, created_at, source
			FROM software_cve
			WHERE cve = ?
		) AS union_cve
		LEFT JOIN operating_system_vulnerabilities osv ON osv.cve = union_cve.cve
		LEFT JOIN software_cve sc ON sc.cve = union_cve.cve
		LEFT JOIN vulnerability_host_counts vhc ON vhc.cve = union_cve.cve
	`

	var args []interface{}
	args = append(args, cve, cve)

	if teamID != nil {
		eeSelectStmt += " AND vhc.team_id = ? AND vhc.global_stats = 0"
		freeSelectStmt += " AND vhc.team_id = ? AND vhc.global_stats = 0"
		args = append(args, *teamID)
	} else {
		eeSelectStmt += " AND vhc.team_id = 0 AND vhc.global_stats = 1"
		freeSelectStmt += " AND vhc.team_id = 0 AND vhc.global_stats = 1"
	}

	var selectStmt string
	if includeCVEScores {
		selectStmt = eeSelectStmt
	} else {
		selectStmt = freeSelectStmt
	}

	err := sqlx.GetContext(ctx, ds.reader(ctx), &vuln, selectStmt, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ctxerr.Wrap(ctx, notFound("Vulnerability").WithName(cve))
		}
		return nil, ctxerr.Wrap(ctx, err, "fetching vulnerability")
	}

	if vuln.HostsCount == 0 {
		var msg string
		if teamID == nil {
			msg = "global"
		} else {
			msg = fmt.Sprintf("team %d", *teamID)
		}
		return nil, ctxerr.Wrap(ctx, notFound(fmt.Sprintf("Vulnerability for %s", msg)).WithName(cve))
	}

	return &vuln, nil
}

func (ds *Datastore) OSVersionsByCVE(ctx context.Context, cve string, teamID *uint) (vos []*fleet.VulnerableOS, updatedAt time.Time, err error) {
	var teamFilter *fleet.TeamFilter
	if teamID != nil {
		teamFilter = &fleet.TeamFilter{TeamID: teamID}
	}
	osvs, err := ds.OSVersions(ctx, teamFilter, nil, nil, nil)
	if err != nil && !fleet.IsNotFound(err) {
		return nil, updatedAt, ctxerr.Wrap(ctx, err, "fetching team OS versions")
	}

	if osvs == nil {
		return nil, updatedAt, nil
	}

	type osVersionWithResolvedType struct {
		OSVersionID     uint    `db:"os_version_id"`
		ResolvedVersion *string `db:"resolved_in_version"`
	}
	var osVersionWithResolved []osVersionWithResolvedType

	selectStmt := `
		SELECT os.os_version_id, osv.resolved_in_version
		FROM operating_system_vulnerabilities osv
		JOIN operating_systems os ON os.id = osv.operating_system_id
		WHERE osv.cve = ?
	`
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &osVersionWithResolved, selectStmt, cve)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, updatedAt, ctxerr.Wrap(ctx, notFound("Vulnerability").WithName(cve))
		}
		return vos, updatedAt, ctxerr.Wrap(ctx, err, "fetching OS version and resolved version by CVE")
	}

	// Remove duplicates, which may occur since the same OS can be installed on multiple architectures (amd64, arm64, etc.)
	type osVersionKey struct {
		OSVersionID     uint
		ResolvedVersion string
	}
	seen := make(map[osVersionKey]struct{}, len(osVersionWithResolved))
	verResolvedDedup := make([]osVersionWithResolvedType, 0, len(osVersionWithResolved))
	for _, id := range osVersionWithResolved {
		var resolved string
		if id.ResolvedVersion != nil {
			resolved = *id.ResolvedVersion
		}
		key := osVersionKey{OSVersionID: id.OSVersionID, ResolvedVersion: resolved}
		if _, ok := seen[key]; !ok {
			verResolvedDedup = append(verResolvedDedup, id)
			seen[key] = struct{}{}
		}
	}

	for _, osv := range osvs.OSVersions {
		for _, id := range verResolvedDedup {
			if osv.OSVersionID == id.OSVersionID {
				vos = append(vos, &fleet.VulnerableOS{
					OSVersion:         osv,
					ResolvedInVersion: id.ResolvedVersion,
				})
			}
		}
	}

	return
}

func (ds *Datastore) SoftwareByCVE(ctx context.Context, cve string, teamID *uint) (vs []*fleet.VulnerableSoftware, updatedAt time.Time, err error) {
	var args []interface{}
	selectStmt := `
		SELECT
			s.id,
			s.name,
			s.version,
			s.source,
			s.extension_for,
			COALESCE(scpe.cpe, '') as generated_cpe,
			COALESCE(shc.hosts_count, 0) as hosts_count,
			COALESCE(sc.resolved_in_version, '') as resolved_in_version
		FROM software s
		JOIN software_cve sc ON sc.software_id = s.id
		LEFT JOIN software_cpe scpe ON scpe.software_id = s.id
		LEFT JOIN software_host_counts shc ON shc.software_id = s.id
		WHERE sc.cve = ?
		`
	args = append(args, cve)

	switch {
	case teamID != nil && *teamID > 0:
		selectStmt += " AND shc.team_id = ? AND shc.global_stats = 0"
		args = append(args, *teamID)
	case teamID != nil && *teamID == 0:
		selectStmt += " AND shc.team_id = 0 AND shc.global_stats = 0"
	case teamID == nil:
		selectStmt += " AND shc.team_id = 0 AND shc.global_stats = 1"
	}

	err = sqlx.SelectContext(ctx, ds.reader(ctx), &vs, selectStmt, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, updatedAt, ctxerr.Wrap(ctx, notFound("Vulnerability").WithName(cve))
		}
		return vs, updatedAt, ctxerr.Wrap(ctx, err, "fetching software by CVE")
	}

	return
}

// vulnerabilitiesCMOrderKeys are the columns sourced from cve_meta that the API
// allows clients to sort by. When the OrderKey is one of these, the inner query
// must LEFT JOIN cve_meta so the ORDER BY in the paginated subquery can
// reference the column.
var vulnerabilitiesCMOrderKeys = map[string]struct{}{
	"cvss_score":         {},
	"epss_probability":   {},
	"cisa_known_exploit": {},
	"cve_published":      {},
}

func (ds *Datastore) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	opt.ListOptions.IncludeMetadata = !(opt.ListOptions.UsesCursorPagination())

	selectStmt, args, err := buildListVulnerabilitiesSQL(&opt)
	if err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list vulnerabilities")
	}

	var vulns []fleet.VulnerabilityWithMetadata
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &vulns, selectStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list vulnerabilities")
	}

	var metaData *fleet.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.ListOptions.Page > 0}
		if len(vulns) > int(opt.ListOptions.PerPage) { //nolint:gosec // dismiss G115
			metaData.HasNextResults = true
			vulns = vulns[:len(vulns)-1]
		}
	}

	return vulns, metaData, nil
}

// buildListVulnerabilitiesSQL constructs the SQL for ListVulnerabilities.
//
// The query is split into two stages so the expensive correlated scalar
// subqueries that compute `created_at` (MIN across software_cve and
// operating_system_vulnerabilities) and `source` only run on the paginated
// page, not on every matching row in vulnerability_host_counts.
//
// Inner query: filter, sort, and paginate vulnerability_host_counts (with
// an optional LEFT JOIN to cve_meta when filtering or sorting by a cve_meta
// column). The new idx_vhc_scope_cve makes the scope filter
// (global_stats, team_id, host_count > 0) an index range scan instead of
// the full-table scan previously observed.
//
// Outer query: enrich the paginated page with the cve_meta metadata
// columns (EE only) and the heavy created_at / source scalar subqueries.
//
// Special case: when OrderKey == "created_at" the value to sort by is the
// scalar subquery output itself, so the inner query has to include it.
// That falls back to the legacy single-statement form (preserved verbatim
// below) — performance is unchanged for that specific sort, but every
// other sort key benefits from the two-stage refactor.
func buildListVulnerabilitiesSQL(opt *fleet.VulnListOptions) (string, []any, error) {
	if opt.ListOptions.OrderKey == "created_at" {
		return buildListVulnerabilitiesLegacySQL(opt)
	}

	_, cmOrderKey := vulnerabilitiesCMOrderKeys[opt.ListOptions.OrderKey]
	needCMInInner := cmOrderKey || opt.KnownExploit

	var inner strings.Builder
	inner.WriteString(`
		SELECT
			vhc.cve,
			vhc.host_count AS hosts_count,
			vhc.updated_at AS hosts_count_updated_at`)
	if cmOrderKey {
		inner.WriteString(`,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published AS cve_published`)
	}
	inner.WriteString(`
		FROM vulnerability_host_counts vhc`)
	if needCMInInner {
		inner.WriteString(`
		LEFT JOIN cve_meta cm ON cm.cve = vhc.cve`)
	}
	inner.WriteString(`
		WHERE vhc.host_count > 0
		AND (
			EXISTS (SELECT 1 FROM software_cve WHERE cve = vhc.cve)
			OR EXISTS (SELECT 1 FROM operating_system_vulnerabilities WHERE cve = vhc.cve)
		)`)

	var args []any
	if opt.TeamID == nil {
		inner.WriteString(" AND vhc.global_stats = 1")
	} else {
		inner.WriteString(" AND vhc.global_stats = 0 AND vhc.team_id = ?")
		args = append(args, *opt.TeamID)
	}
	if opt.KnownExploit {
		inner.WriteString(" AND cm.cisa_known_exploit = 1")
	}

	innerSQL := inner.String()
	if match := opt.ListOptions.MatchQuery; match != "" {
		innerSQL, args = searchLike(innerSQL, args, match, "vhc.cve")
	}

	innerSQL, args, err := appendListOptionsWithCursorToSQLSecure(innerSQL, args, &opt.ListOptions, vulnerabilitiesAllowedOrderKeys)
	if err != nil {
		return "", nil, err
	}

	var outer strings.Builder
	outer.WriteString(`
		SELECT
			p.cve,
			(SELECT MIN(created_at) FROM (
				SELECT created_at FROM software_cve WHERE cve = p.cve
				UNION ALL
				SELECT created_at FROM operating_system_vulnerabilities WHERE cve = p.cve
			) AS combined_dates) AS created_at,
			COALESCE(
				(SELECT source FROM software_cve WHERE cve = p.cve LIMIT 1),
				(SELECT source FROM operating_system_vulnerabilities WHERE cve = p.cve LIMIT 1)
			) AS source,`)
	if opt.IsEE {
		outer.WriteString(`
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published AS cve_published,
			cm.description,`)
	}
	outer.WriteString(`
			p.hosts_count,
			p.hosts_count_updated_at
		FROM (`)
	outer.WriteString(innerSQL)
	outer.WriteString(`) AS p`)
	if opt.IsEE {
		outer.WriteString(`
		LEFT JOIN cve_meta cm ON cm.cve = p.cve`)
	}

	// The optimizer may not preserve the inner ORDER BY when wrapped in an
	// outer SELECT, so restate the sort. The inner has already limited rows
	// to the page, so this re-sort is bounded to perPage rows.
	if orderCol, ok := vulnerabilitiesAllowedOrderKeys[opt.ListOptions.OrderKey]; ok && orderCol != "" {
		direction := "ASC"
		if opt.ListOptions.OrderDirection == fleet.OrderDescending {
			direction = "DESC"
		}
		outer.WriteString(fmt.Sprintf(" ORDER BY %s %s", orderCol, direction))
	}

	return outer.String(), args, nil
}

// buildListVulnerabilitiesLegacySQL preserves the original single-statement
// query used when OrderKey == "created_at" (the only sort key that has to
// reference the cross-table scalar subquery result).
func buildListVulnerabilitiesLegacySQL(opt *fleet.VulnListOptions) (string, []any, error) {
	eeSelectStmt := `
		SELECT
			vhc.cve as cve,
			(SELECT MIN(created_at) FROM (
				SELECT created_at FROM software_cve WHERE cve = vhc.cve
				UNION ALL
				SELECT created_at FROM operating_system_vulnerabilities WHERE cve = vhc.cve
			) AS combined_dates) as created_at,
			COALESCE(
				(SELECT source FROM software_cve WHERE cve = vhc.cve LIMIT 1),
				(SELECT source FROM operating_system_vulnerabilities WHERE cve = vhc.cve LIMIT 1)
			) as source,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published as cve_published,
			cm.description,
			vhc.host_count as hosts_count,
			vhc.updated_at as hosts_count_updated_at
		FROM vulnerability_host_counts vhc
		LEFT JOIN cve_meta cm ON cm.cve = vhc.cve
		WHERE vhc.host_count > 0
		AND (
			EXISTS (SELECT 1 FROM software_cve WHERE cve = vhc.cve)
			OR EXISTS (SELECT 1 FROM operating_system_vulnerabilities WHERE cve = vhc.cve)
		)
		`
	freeSelectStmt := `
		SELECT
			vhc.cve as cve,
			(SELECT MIN(created_at) FROM (
				SELECT created_at FROM software_cve WHERE cve = vhc.cve
				UNION ALL
				SELECT created_at FROM operating_system_vulnerabilities WHERE cve = vhc.cve
			) AS combined_dates) as created_at,
			COALESCE(
				(SELECT source FROM software_cve WHERE cve = vhc.cve LIMIT 1),
				(SELECT source FROM operating_system_vulnerabilities WHERE cve = vhc.cve LIMIT 1)
			) as source,
			vhc.host_count as hosts_count,
			vhc.updated_at as hosts_count_updated_at
		FROM vulnerability_host_counts vhc
		WHERE vhc.host_count > 0
		AND (
			EXISTS (SELECT 1 FROM software_cve WHERE cve = vhc.cve)
			OR EXISTS (SELECT 1 FROM operating_system_vulnerabilities WHERE cve = vhc.cve)
		)
		`

	selectStmt := eeSelectStmt
	if !opt.IsEE {
		selectStmt = freeSelectStmt
	}

	var args []any
	if opt.TeamID == nil {
		selectStmt += " AND vhc.global_stats = 1"
	} else {
		selectStmt += " AND vhc.global_stats = 0 AND vhc.team_id = ?"
		args = append(args, *opt.TeamID)
	}
	if opt.KnownExploit {
		selectStmt += " AND cm.cisa_known_exploit = 1"
	}
	if match := opt.ListOptions.MatchQuery; match != "" {
		selectStmt, args = searchLike(selectStmt, args, match, "vhc.cve")
	}

	return appendListOptionsWithCursorToSQLSecure(selectStmt, args, &opt.ListOptions, vulnerabilitiesAllowedOrderKeys)
}

func (ds *Datastore) CountVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) (uint, error) {
	// vhc.cve is already unique within a (global_stats, team_id) scope due to
	// the existing UNIQUE KEY (cve, team_id, global_stats), so COUNT(*) gives
	// the same result as COUNT(DISTINCT vhc.cve) but lets the optimizer pick
	// idx_vhc_scope_cve without a dedup step.
	selectStmt := `
		SELECT COUNT(*)
		FROM vulnerability_host_counts vhc
		`
	if opt.KnownExploit {
		selectStmt += `LEFT JOIN cve_meta cm ON cm.cve = vhc.cve
		`
	}
	selectStmt += `WHERE vhc.host_count > 0
		AND (
			EXISTS (SELECT 1 FROM software_cve WHERE cve = vhc.cve)
			OR EXISTS (SELECT 1 FROM operating_system_vulnerabilities WHERE cve = vhc.cve)
		)
	`
	var args []any
	if opt.TeamID == nil {
		selectStmt += " AND vhc.global_stats = 1"
	} else {
		selectStmt += " AND vhc.global_stats = 0 AND vhc.team_id = ?"
		args = append(args, *opt.TeamID)
	}
	if opt.KnownExploit {
		selectStmt += " AND cm.cisa_known_exploit = 1"
	}
	if match := opt.ListOptions.MatchQuery; match != "" {
		selectStmt, args = searchLike(selectStmt, args, match, "vhc.cve")
	}

	var count uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, selectStmt, args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count vulnerabilities")
	}

	return count, nil
}

func (ds *Datastore) distinctCVEs(ctx context.Context) ([]string, error) {
	uniqueCVEQuery := `
        SELECT DISTINCT cve FROM (
            SELECT cve FROM software_cve
            UNION
            SELECT cve FROM operating_system_vulnerabilities
        ) AS combined_cves;
    `

	var cves []string
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &cves, uniqueCVEQuery)
	if err != nil {
		return nil, err
	}
	return cves, nil
}

type CountScope int

const (
	GlobalCount CountScope = iota
	NoTeamCount
	TeamCount
)

func (ds *Datastore) batchFetchVulnerabilityCounts(
	ctx context.Context,
	scope CountScope,
	maxRoutines int,
) ([]hostCount, error) {
	const (
		batchSize = 10
	)

	// Fetch distinct CVEs
	allCVEs, err := ds.distinctCVEs(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetching distinct CVEs: %w", err)
	}

	query := getVulnHostCountQuery(scope)
	if query == "" {
		return nil, ctxerr.Errorf(ctx, "invalid scope: %d", scope)
	}

	var (
		hostCounts []hostCount
		mu         sync.Mutex
		wg         sync.WaitGroup
		sem        = make(chan struct{}, maxRoutines)
		errChan    = make(chan error, len(allCVEs)/batchSize+1)
	)

	// Process CVEs in batches concurrently
	for i := 0; i < len(allCVEs); i += batchSize {
		end := i + batchSize
		if end > len(allCVEs) {
			end = len(allCVEs)
		}

		batchCVEs := allCVEs[i:end]
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(cves []string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			counts, err := ds.fetchBatchCounts(ctx, cves, query)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			hostCounts = append(hostCounts, counts...)
			mu.Unlock()
		}(batchCVEs)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	return hostCounts, nil
}

// fetchBatchCounts executes the query for a batch of CVEs.
func (ds *Datastore) fetchBatchCounts(
	ctx context.Context,
	batchCVEs []string,
	scopeConfig string,
) ([]hostCount, error) {
	query, args, err := sqlx.In(scopeConfig, batchCVEs, batchCVEs)
	if err != nil {
		return nil, err
	}

	var counts []hostCount
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &counts, query, args...)
	if err != nil {
		return nil, err
	}

	return counts, nil
}

// getScopeConfig returns the query configuration for the given scope.
func getVulnHostCountQuery(scope CountScope) string {
	switch scope {
	case GlobalCount:
		return `
				SELECT 0 as team_id, 1 as global_stats, combined_results.cve, COUNT(*) AS host_count
				FROM (
					SELECT sc.cve, hs.host_id
					FROM software_cve sc
					INNER JOIN host_software hs ON sc.software_id = hs.software_id
					WHERE sc.cve IN (?)

					UNION

					SELECT osv.cve, hos.host_id
					FROM operating_system_vulnerabilities osv
					INNER JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id
					WHERE osv.cve IN (?)
				) AS combined_results
				GROUP BY cve
			`
	case NoTeamCount:
		return `
				SELECT 0 as team_id, 0 as global_stats, combined_results.cve, COUNT(*) AS host_count
				FROM (
					SELECT sc.cve, hs.host_id
					FROM software_cve sc
					INNER JOIN host_software hs ON sc.software_id = hs.software_id
					WHERE sc.cve IN (?)

					UNION

					SELECT osv.cve, hos.host_id
					FROM operating_system_vulnerabilities osv
					INNER JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id
					WHERE osv.cve IN (?)
				) AS combined_results
				INNER JOIN hosts h ON combined_results.host_id = h.id
				WHERE h.team_id IS NULL
				GROUP BY cve
			`
	case TeamCount:
		return `
				SELECT h.team_id as team_id, 0 as global_stats, combined_results.cve, COUNT(*) AS host_count
				FROM (
					SELECT sc.cve, hs.host_id
					FROM software_cve sc
					INNER JOIN host_software hs ON sc.software_id = hs.software_id
					WHERE sc.cve IN (?)

					UNION

					SELECT osv.cve, hos.host_id
					FROM operating_system_vulnerabilities osv
					INNER JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id
					WHERE osv.cve IN (?)
				) AS combined_results
				INNER JOIN hosts h ON combined_results.host_id = h.id
				WHERE h.team_id IS NOT NULL
				GROUP BY h.team_id, combined_results.cve
			`
	default:
		return ""
	}
}

func (ds *Datastore) UpdateVulnerabilityHostCounts(ctx context.Context, maxRoutines int) error {
	globalHostCounts, err := ds.batchFetchVulnerabilityCounts(ctx, GlobalCount, maxRoutines)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching global vulnerability host counts")
	}

	teamHostCounts, err := ds.batchFetchVulnerabilityCounts(ctx, TeamCount, maxRoutines)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching team vulnerability host counts")
	}

	noTeamHostCounts, err := ds.batchFetchVulnerabilityCounts(ctx, NoTeamCount, maxRoutines)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching no team vulnerability host counts")
	}

	counts := vulnerabilityCounts{
		Global: globalHostCounts,
		Team:   teamHostCounts,
		NoTeam: noTeamHostCounts,
	}

	return ds.atomicTableSwapVulnerabilityCounts(ctx, counts)
}

type hostCount struct {
	TeamID      uint   `db:"team_id"`
	CVE         string `db:"cve"`
	HostCount   uint   `db:"host_count"`
	GlobalStats bool   `db:"global_stats"`
}

type vulnerabilityCounts struct {
	Global []hostCount
	Team   []hostCount
	NoTeam []hostCount
}

const (
	vulnerabilityHostCountsSwapTable       = "vulnerability_host_counts_swap"
	vulnerabilityHostCountsSwapTableSchema = `CREATE TABLE IF NOT EXISTS ` + vulnerabilityHostCountsSwapTable + ` LIKE vulnerability_host_counts`
)

// atomicTableSwapVulnerabilityCounts implements atomic table swap pattern
// 1. Populate swap table with new data
// 2. Atomically rename tables to swap them
// 3. Clean up old table
func (ds *Datastore) atomicTableSwapVulnerabilityCounts(ctx context.Context, counts vulnerabilityCounts) error {
	err := ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		// Create/recreate the swap table fresh
		_, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS "+vulnerabilityHostCountsSwapTable)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "dropping existing swap table")
		}

		_, err = tx.ExecContext(ctx, vulnerabilityHostCountsSwapTableSchema)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "creating swap table")
		}

		// Insert each group of counts separately
		if len(counts.Global) > 0 {
			err = ds.insertHostCountsIntoTable(ctx, tx, counts.Global, vulnerabilityHostCountsSwapTable)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "populating swap table with global counts")
			}
		}

		if len(counts.Team) > 0 {
			err = ds.insertHostCountsIntoTable(ctx, tx, counts.Team, vulnerabilityHostCountsSwapTable)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "populating swap table with team counts")
			}
		}

		if len(counts.NoTeam) > 0 {
			err = ds.insertHostCountsIntoTable(ctx, tx, counts.NoTeam, vulnerabilityHostCountsSwapTable)
			if err != nil {
				return ctxerr.Wrap(ctx, err, "populating swap table with no-team counts")
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	// Atomic table swap using RENAME TABLE
	return ds.withRetryTxx(ctx, func(tx sqlx.ExtContext) error {
		_, err := tx.ExecContext(ctx, fmt.Sprintf(`
			RENAME TABLE
				vulnerability_host_counts TO vulnerability_host_counts_old,
				%s TO vulnerability_host_counts
		`, vulnerabilityHostCountsSwapTable))
		if err != nil {
			return ctxerr.Wrap(ctx, err, "atomic table swap")
		}

		// Clean up old table (drop it)
		_, err = tx.ExecContext(ctx, "DROP TABLE vulnerability_host_counts_old")
		if err != nil {
			return ctxerr.Wrap(ctx, err, "dropping old table")
		}

		return nil
	})
}

// insertHostCountsIntoTable inserts counts into specified table
func (ds *Datastore) insertHostCountsIntoTable(ctx context.Context, tx sqlx.ExtContext, counts []hostCount, tableName string) error {
	if len(counts) == 0 {
		return nil
	}

	insertStmt := fmt.Sprintf("INSERT INTO %s (team_id, cve, host_count, global_stats) VALUES ", tableName)

	// Use smaller chunks to avoid parameter limits
	chunkSize := 500
	for i := 0; i < len(counts); i += chunkSize {
		end := min(i+chunkSize, len(counts))

		valueStrings := make([]string, 0, end-i)
		chunkArgs := make([]interface{}, 0, (end-i)*4)

		for _, count := range counts[i:end] {
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			chunkArgs = append(chunkArgs, count.TeamID, count.CVE, count.HostCount, count.GlobalStats)
		}

		fullStmt := insertStmt + strings.Join(valueStrings, ", ")
		_, err := tx.ExecContext(ctx, fullStmt, chunkArgs...)
		if err != nil {
			return fmt.Errorf("inserting host counts chunk %d-%d into %s: %w", i, end-1, tableName, err)
		}
	}

	return nil
}

func (ds *Datastore) IsCVEKnownToFleet(ctx context.Context, cve string) (bool, error) {
	var count uint
	err := sqlx.GetContext(ctx, ds.reader(ctx), &count, "SELECT 1 FROM cve_meta WHERE cve = ?", cve)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return count > 0, nil
}
