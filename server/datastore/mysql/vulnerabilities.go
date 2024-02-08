package mysql

import (
	"context"
	"fmt"
	"strings"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	// Define base select statements for EE and Free versions
	eeSelectStmt := `
		SELECT
			vhc.cve,
			MIN(COALESCE(osv.created_at, sc.created_at, NOW())) AS created_at,
			COALESCE(osv.source, sc.source, 0) AS source,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published,
			COALESCE(cm.description, '') AS description,
			vhc.host_count,
			vhc.updated_at as host_count_updated_at
		FROM
			vulnerability_host_counts vhc
		LEFT JOIN cve_meta cm ON cm.cve = vhc.cve
		LEFT JOIN operating_system_vulnerabilities osv ON osv.cve = vhc.cve
		LEFT JOIN software_cve sc ON sc.cve = vhc.cve
		WHERE vhc.host_count > 0
		`
	freeSelectStmt := `
		SELECT
			vhc.cve,
			MIN(COALESCE(osv.created_at, sc.created_at, NOW())) AS created_at,
			COALESCE(osv.source, sc.source, 0) AS source,
			vhc.host_count,
			vhc.updated_at as host_count_updated_at
		FROM
			vulnerability_host_counts vhc
		LEFT JOIN operating_system_vulnerabilities osv ON osv.cve = vhc.cve
		LEFT JOIN software_cve sc ON sc.cve = vhc.cve
		WHERE vhc.host_count > 0
		`

	// Choose the appropriate select statement based on EE or Free
	var selectStmt string
	if opt.IsEE {
		selectStmt = eeSelectStmt
	} else {
		selectStmt = freeSelectStmt
	}

	// Define group by statements for EE and Free
	eeGroupBy := ` GROUP BY 
			vhc.cve, 
			source,
			cm.cvss_score, 
			cm.epss_probability, 
			cm.cisa_known_exploit, 
			cm.published, 
			description, 
			vhc.host_count
	`
	freeGroupBy := " GROUP BY vhc.cve, source, vhc.host_count"

	// Choose the appropriate group by statement based on EE or Free
	var groupBy string
	if opt.IsEE {
		groupBy = eeGroupBy
	} else {
		groupBy = freeGroupBy
	}

	// Prepare arguments for the query
	var args []interface{}
	if opt.TeamID == 0 {
		selectStmt += " AND vhc.team_id = 0"
	} else {
		selectStmt += " AND vhc.team_id = ?"
		args = append(args, opt.TeamID)
	}

	if opt.KnownExploit {
		selectStmt += " AND cm.cisa_known_exploit = 1"
	}

	if match := opt.MatchQuery; match != "" {
		selectStmt, args = searchLike(selectStmt, args, match, "vhc.cve")
	}

	// Append group by statement
	selectStmt += groupBy

	if opt.KnownExploit {
		selectStmt += " AND cm.cisa_known_exploit = 1"
	}

	if match := opt.MatchQuery; match != "" {
		selectStmt, args = searchLike(selectStmt, args, match, "vhc.cve")
	}

	if opt.KnownExploit {
		selectStmt = selectStmt + " AND cm.cisa_known_exploit = 1"
	}

	if match := opt.MatchQuery; match != "" {
		selectStmt, args = searchLike(selectStmt, args, match, "vhc.cve")
	}

	// Append group by statement
	selectStmt += groupBy

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
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.Page > 0}
		if len(vulns) > int(opt.PerPage) {
			metaData.HasNextResults = true
			vulns = vulns[:len(vulns)-1]
		}
	}

	return vulns, metaData, nil
}


func (ds *Datastore) CountVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) (uint, error) {
	selectStmt := `
		SELECT COUNT(*)
		FROM vulnerability_host_counts vhc
		LEFT JOIN cve_meta cm ON cm.cve = vhc.cve
		LEFT JOIN operating_system_vulnerabilities osv ON osv.cve = vhc.cve
		LEFT JOIN software_cve sc ON sc.cve = vhc.cve
		WHERE vhc.host_count > 0
		`
	var args []interface{}
	if opt.TeamID == 0 {
		selectStmt = selectStmt + " AND vhc.team_id = 0"
	} else {
		selectStmt = selectStmt + " AND vhc.team_id = ?"
		args = append(args, opt.TeamID)
	}

	if opt.KnownExploit {
		selectStmt = selectStmt + " AND cm.cisa_known_exploit = 1"
	}

	if match := opt.MatchQuery; match != "" {
		selectStmt, args = searchLike(selectStmt, args, match, "vhc.cve")
	}

	var count uint
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &count, selectStmt, args...); err != nil {
		return 0, ctxerr.Wrap(ctx, err, "count vulnerabilities")
	}

	return count, nil
}

func (ds *Datastore) UpdateVulnerabilityHostCounts(ctx context.Context) error {
	// set all counts to 0 to later identify rows to delete
	_, err := ds.writer(ctx).ExecContext(ctx, "UPDATE vulnerability_host_counts SET host_count = 0")
	if err != nil {
		return ctxerr.Wrap(ctx, err, "initializing vulnerability host counts")
	}

	globalSelectStmt := `
		SELECT 0 as team_id, cve, COUNT(*) AS host_count
		FROM (
			SELECT sc.cve, hs.host_id
			FROM software_cve sc
			INNER JOIN host_software hs ON sc.software_id = hs.software_id
		
			UNION
		
			SELECT osv.cve, hos.host_id
			FROM operating_system_vulnerabilities osv
			INNER JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id
		) AS combined_results
		GROUP BY cve;
	`

	globalHostCounts, err := ds.fetchHostCounts(ctx, globalSelectStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching global vulnerability host counts")
	}

	err = ds.batchInsertHostCounts(ctx, globalHostCounts)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting global vulnerability host counts")
	}

	teamSelectStmt := `
		SELECT h.team_id, combined_results.cve, COUNT(*) AS host_count
		FROM (
			SELECT hs.host_id, sc.cve
			FROM software_cve sc
			INNER JOIN host_software hs ON sc.software_id = hs.software_id

			UNION

			SELECT hos.host_id, osv.cve
			FROM operating_system_vulnerabilities osv
			INNER JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id
		) AS combined_results
		INNER JOIN hosts h ON combined_results.host_id = h.id
		WHERE h.team_id IS NOT NULL
		GROUP BY h.team_id, combined_results.cve
	`

	teamHostCounts, err := ds.fetchHostCounts(ctx, teamSelectStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "fetching team vulnerability host counts")
	}

	err = ds.batchInsertHostCounts(ctx, teamHostCounts)
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
	TeamID    uint   `db:"team_id"`
	CVE       string `db:"cve"`
	HostCount uint   `db:"host_count"`
}

func (ds *Datastore) cleanupVulnerabilityHostCounts(ctx context.Context) error {
	_, err := ds.writer(ctx).ExecContext(ctx, "DELETE FROM vulnerability_host_counts WHERE host_count = 0")
	if err != nil {
		return fmt.Errorf("deleting zero host count entries: %w", err)
	}

	return nil
}

func (ds *Datastore) fetchHostCounts(ctx context.Context, query string) ([]hostCount, error) {
	var hostCounts []hostCount
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &hostCounts, query)
	if err != nil {
		return nil, err
	}
	return hostCounts, nil
}

func (ds *Datastore) batchInsertHostCounts(ctx context.Context, counts []hostCount) error {
	if len(counts) == 0 {
		return nil
	}

	insertStmt := "INSERT INTO vulnerability_host_counts (team_id, cve, host_count) VALUES "
	var insertArgs []interface{}

	chunkSize := 100
	for i := 0; i < len(counts); i += chunkSize {
		end := i + chunkSize
		if end > len(counts) {
			end = len(counts)
		}

		valueStrings := make([]string, 0, chunkSize)
		for _, count := range counts[i:end] {
			valueStrings = append(valueStrings, "(?, ?, ?)")
			insertArgs = append(insertArgs, count.TeamID, count.CVE, count.HostCount)
		}

		insertStmt += strings.Join(valueStrings, ", ")
		insertStmt += " ON DUPLICATE KEY UPDATE host_count = VALUES(host_count);"

		_, err := ds.writer(ctx).ExecContext(ctx, insertStmt, insertArgs...)
		if err != nil {
			return fmt.Errorf("inserting host counts: %w", err)
		}

		insertStmt = "INSERT INTO vulnerability_host_counts (team_id, cve, host_count) VALUES "
		insertArgs = nil
	}

	return nil
}
