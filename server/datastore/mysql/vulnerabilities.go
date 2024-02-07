package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	selectStmt := `
		SELECT
			vhc.cve,
			MIN(COALESCE(osv.created_at, sc.created_at, NOW())) AS created_at,
			cm.cvss_score,
			cm.epss_probability,
			cm.cisa_known_exploit,
			cm.published,
			COALESCE(cm.description, '') AS description,
			vhc.host_count
		FROM
			vulnerability_host_counts vhc
		LEFT JOIN cve_meta cm ON cm.cve = vhc.cve
		LEFT JOIN operating_system_vulnerabilities osv ON osv.cve = vhc.cve
		LEFT JOIN software_cve sc ON sc.cve = vhc.cve
		WHERE vhc.host_count > 0
		`
	groupByAppend := " GROUP BY vhc.cve, cm.cvss_score, cm.epss_probability, cm.cisa_known_exploit, cm.published, description, vhc.host_count"

	var args []interface{}
	if opt.TeamID == 0 {
		selectStmt = selectStmt + " AND vhc.team_id = 0"
	} else {
		selectStmt = selectStmt + " AND vhc.team_id = ?"
		args = append(args, opt.TeamID)
	}
	selectStmt = selectStmt + groupByAppend

	opt.ListOptions.IncludeMetadata = !(opt.ListOptions.UsesCursorPagination())

	selectStmt, args = appendListOptionsWithCursorToSQL(selectStmt, args, &opt.ListOptions)

	var vulns []fleet.VulnerabilityWithMetadata
	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &vulns, selectStmt, args...); err != nil {
		return nil, nil, ctxerr.Wrap(ctx, err, "list vulnerabilities")
	}

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

func (ds *Datastore) UpdateVulnerabilityHostCounts(ctx context.Context) error {
	type hostCount struct {
		TeamID    uint   `db:"team_id"`
		CVE       string `db:"cve"`
		HostCount uint   `db:"host_count"`
	}

	globalSelectStmt := `
    SELECT 0 as team_id, cve, COUNT(DISTINCT host_id) AS host_count
    FROM (
        SELECT sc.cve, hs.host_id
        FROM software_cve sc
        INNER JOIN host_software hs ON sc.software_id = hs.software_id
    
        UNION ALL
    
        SELECT osv.cve, hos.host_id
        FROM operating_system_vulnerabilities osv
        INNER JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id
    ) AS combined_results
    GROUP BY cve;
  `
	var globalHostCounts []hostCount
	err := sqlx.SelectContext(ctx, ds.reader(ctx), &globalHostCounts, globalSelectStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting global host counts")
	}

	insertStmt := "INSERT INTO vulnerability_host_counts (team_id, cve, host_count) VALUES "
	var insertArgs []interface{}
	for _, count := range globalHostCounts {
		insertStmt += "(?, ?, ?),"
		insertArgs = append(insertArgs, count.TeamID, count.CVE, count.HostCount)
	}
	insertStmt = insertStmt[:len(insertStmt)-1] // remove trailing comma
	insertStmt += " ON DUPLICATE KEY UPDATE host_count = VALUES(host_count)"

	_, err = ds.writer(ctx).ExecContext(ctx, insertStmt, insertArgs...)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "inserting global host counts")
	}

	teamSelectStmt := `
		SELECT h.team_id, combined_results.cve, COUNT(DISTINCT h.id) AS host_count
		FROM (
			SELECT hs.host_id, sc.cve
			FROM software_cve sc
			INNER JOIN host_software hs ON sc.software_id = hs.software_id

			UNION ALL

			SELECT hos.host_id, osv.cve
			FROM operating_system_vulnerabilities osv
			INNER JOIN host_operating_system hos ON hos.os_id = osv.operating_system_id
		) AS combined_results
		INNER JOIN hosts h ON combined_results.host_id = h.id
		WHERE h.team_id IS NOT NULL
		GROUP BY h.team_id, combined_results.cve
	`

	var teamHostCounts []hostCount
	err = sqlx.SelectContext(ctx, ds.reader(ctx), &teamHostCounts, teamSelectStmt)
	if err != nil {
		return ctxerr.Wrap(ctx, err, "getting team host counts")
	}

	if len(teamHostCounts) > 0 {

		insertStmt = "INSERT INTO vulnerability_host_counts (team_id, cve, host_count) VALUES "
		insertArgs = []interface{}{}
		for _, count := range teamHostCounts {
			insertStmt += "(?, ?, ?),"
			insertArgs = append(insertArgs, count.TeamID, count.CVE, count.HostCount)
		}
		insertStmt = insertStmt[:len(insertStmt)-1] // remove trailing comma
		insertStmt += " ON DUPLICATE KEY UPDATE host_count = VALUES(host_count)"

		_, err = ds.writer(ctx).ExecContext(ctx, insertStmt, insertArgs...)
		if err != nil {
			return ctxerr.Wrap(ctx, err, "inserting team host counts")
		}
	}

	return nil
}
