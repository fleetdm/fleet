package mysql

import (
	"context"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) ListVulnerabilities(ctx context.Context, opt fleet.VulnListOptions) ([]fleet.VulnerabilityWithMetadata, *fleet.PaginationMetadata, error) {
	selectStmt := `
		SELECT DISTINCT
			vhc.cve,
			COALESCE(osv.resolved_in_version, sc.resolved_in_version) as resolved_in_version,
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

	var args []interface{}
	if opt.TeamID == 0 {
		selectStmt = selectStmt + " AND vhc.team_id = 0"
	} else {
		selectStmt = selectStmt + " AND vhc.team_id = ?"
		args = append(args, opt.TeamID)
	}

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
