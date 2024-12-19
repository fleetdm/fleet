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
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) Vulnerability(ctx context.Context, cve string, teamID *uint, includeCVEScores bool) (*fleet.VulnerabilityWithMetadata, error) {
	var vuln fleet.VulnerabilityWithMetadata

	eeSelectStmt := `
		SELECT DISTINCT
			cm.cve,
			COALESCE(LEAST(osv.created_at, sc.created_at), NOW()) AS created_at,
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
			COALESCE(LEAST(osv.created_at, sc.created_at), NOW()) AS created_at,
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
			s.browser,
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

func (ds *Datastore) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	// Define base select statements for EE and Free versions
	eeSelectStmt := `
		SELECT
			combined.cve as cve,
			MIN(combined.created_at) as created_at,
			MIN(combined.source) as source,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published as cve_published,
			cm.description,
			vhc.host_count as hosts_count,
			vhc.updated_at as hosts_count_updated_at
		FROM (
			SELECT cve, created_at, source FROM software_cve
			UNION
			SELECT cve, created_at, source FROM operating_system_vulnerabilities
		) AS combined
		INNER JOIN vulnerability_host_counts vhc ON vhc.cve = combined.cve
		LEFT JOIN cve_meta cm ON cm.cve = combined.cve
		WHERE vhc.host_count > 0
		`
	freeSelectStmt := `
		SELECT
			combined.cve as cve,
			MIN(combined.created_at) as created_at,
			MIN(combined.source) as source,
			vhc.host_count as hosts_count,
			vhc.updated_at as hosts_count_updated_at
		FROM (
			SELECT cve, created_at, source FROM software_cve
			UNION
			SELECT cve, created_at, source FROM operating_system_vulnerabilities
		) AS combined
		INNER JOIN vulnerability_host_counts vhc ON vhc.cve = combined.cve
		WHERE vhc.host_count > 0
		`

	// Choose the appropriate select statement based on EE or Free
	var selectStmt string
	if opt.IsEE {
		selectStmt = eeSelectStmt
	} else {
		selectStmt = freeSelectStmt
	}

	// Prepare arguments for the query
	var args []interface{}
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

	// Append group by statement
	selectStmt += " GROUP BY cve, host_count, updated_at"

	opt.ListOptions.IncludeMetadata = !(opt.ListOptions.UsesCursorPagination())
	selectStmt, args = appendListOptionsWithCursorToSQL(selectStmt, args, &opt.ListOptions)

	// Execute the query
	var vulns []fleet.VulnerabilityWithMetadata
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &vulns, selectStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list vulnerabilities")
	}

	// Prepare metadata
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

func (ds *Datastore) CountVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) (uint, error) {
	selectStmt := `
		SELECT
			COUNT(*)
		FROM (
			SELECT cve, created_at, source FROM software_cve
			UNION
			SELECT cve, created_at, source FROM operating_system_vulnerabilities
		) AS combined
		INNER JOIN vulnerability_host_counts vhc ON vhc.cve = combined.cve
		LEFT JOIN cve_meta cm ON cm.cve = combined.cve
		WHERE vhc.host_count > 0
	`
	var args []interface{}
	if opt.TeamID == nil {
		selectStmt += " AND global_stats = 1"
	} else {
		selectStmt += " AND global_stats = 0 AND vhc.team_id = ?"
		args = append(args, opt.TeamID)
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

func (ds *Datastore) batchFetchVulnerabilityCounts(
	ctx context.Context,
	teamID string,
	globalStats int,
) ([]hostCount, error) {
	const batchSize = 20
	const maxConcurrentBatches = 5

	var allCVEs []string
	var hostCounts []hostCount
	var mu sync.Mutex

	// Fetch distinct CVEs
	allCVEs, err := ds.distinctCVEs(ctx)
	if err != nil {
		return nil, err
	}

	var teamIDCondition string
	switch {
	case teamID == "0" && globalStats == 0: // "No team" counts
		teamIDCondition = "INNER JOIN hosts h ON combined_results.host_id = h.id WHERE h.team_id IS NULL GROUP BY cve"
	case teamID != "0" && globalStats == 0: // Team counts
		teamIDCondition = "INNER JOIN hosts h ON combined_results.host_id = h.id WHERE h.team_id IS NOT NULL GROUP BY h.team_id, combined_results.cve"
	default: // Global counts
		teamIDCondition = " GROUP BY cve"
	}

	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrentBatches)
	errChan := make(chan error, len(allCVEs)/batchSize+1)

	// Process CVEs in batches concurrently
	for i := 0; i < len(allCVEs); i += batchSize {
		end := i + batchSize
		if end > len(allCVEs) {
			end = len(allCVEs)
		}

		cves := allCVEs[i:end]
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(batchCVEs []string) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			selectStmt := fmt.Sprintf(`
				SELECT %s as team_id, %d as global_stats, combined_results.cve, COUNT(*) AS host_count
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
				%s
			`, teamID, globalStats, teamIDCondition)

			query, args, err := sqlx.In(selectStmt, batchCVEs, batchCVEs)
			if err != nil {
				errChan <- err
				return
			}

			var counts []hostCount
			err = sqlx.SelectContext(ctx, ds.reader(ctx), &counts, query, args...)
			if err != nil {
				errChan <- err
				return
			}

			mu.Lock()
			hostCounts = append(hostCounts, counts...)
			mu.Unlock()
		}(cves)
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

func (ds *Datastore) batchFetchGlobalVulnerabilityCounts(ctx context.Context) ([]hostCount, error) {
	return ds.batchFetchVulnerabilityCounts(ctx, "0", 1)
}

func (ds *Datastore) batchFetchNoTeamVulnerabilityCounts(ctx context.Context) ([]hostCount, error) {
	return ds.batchFetchVulnerabilityCounts(ctx, "0", 0)
}

func (ds *Datastore) UpdateVulnerabilityHostCounts(ctx context.Context) error {
	// set all counts to 0 to later identify rows to delete
	_, err := ds.writer(ctx).ExecContext(ctx, "UPDATE vulnerability_host_counts SET host_count = 0")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "initializing vulnerability host counts")
	}

	globalHostCounts, err := ds.batchFetchGlobalVulnerabilityCounts(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching global vulnerability host counts")
	}

	err = ds.batchInsertHostCounts(ctx, globalHostCounts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting global vulnerability host counts")
	}

	teamHostCounts, err := ds.batchFetchVulnerabilityCounts(ctx, "h.team_id", 0)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching team vulnerability host counts")
	}

	err = ds.batchInsertHostCounts(ctx, teamHostCounts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting team vulnerability host counts")
	}

	noTeamHostCounts, err := ds.batchFetchNoTeamVulnerabilityCounts(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching no team vulnerability host counts")
	}

	err = ds.batchInsertHostCounts(ctx, noTeamHostCounts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting team vulnerability host counts")
	}

	err = ds.cleanupVulnerabilityHostCounts(ctx)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "cleaning up vulnerability host counts")
	}

	return nil
}

type hostCount struct {
	TeamID      uint   `db:"team_id"`
	CVE         string `db:"cve"`
	HostCount   uint   `db:"host_count"`
	GlobalStats bool   `db:"global_stats"`
}

func (ds *Datastore) cleanupVulnerabilityHostCounts(ctx context.Context) error {
	_, err := ds.writer(ctx).ExecContext(ctx, "DELETE FROM vulnerability_host_counts WHERE host_count = 0")
	if err != nil {
		return fmt.Errorf("deleting zero host count entries: %w", err)
	}

	return nil
}

func (ds *Datastore) batchInsertHostCounts(ctx context.Context, counts []hostCount) error {
	if len(counts) == 0 {
		return nil
	}

	insertStmt := "INSERT INTO vulnerability_host_counts (team_id, cve, host_count, global_stats) VALUES "
	var insertArgs []interface{}

	chunkSize := 100
	for i := 0; i < len(counts); i += chunkSize {
		end := i + chunkSize
		if end > len(counts) {
			end = len(counts)
		}

		valueStrings := make([]string, 0, chunkSize)
		for _, count := range counts[i:end] {
			valueStrings = append(valueStrings, "(?, ?, ?, ?)")
			insertArgs = append(insertArgs, count.TeamID, count.CVE, count.HostCount, count.GlobalStats)
		}

		insertStmt += strings.Join(valueStrings, ", ")
		insertStmt += " ON DUPLICATE KEY UPDATE host_count = VALUES(host_count);"

		_, err := ds.writer(ctx).ExecContext(ctx, insertStmt, insertArgs...)
		if err != nil {
			return fmt.Errorf("inserting host counts: %w", err)
		}

		insertStmt = "INSERT INTO vulnerability_host_counts (team_id, cve, host_count, global_stats) VALUES "
		insertArgs = nil
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
