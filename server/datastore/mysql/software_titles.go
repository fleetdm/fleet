package mysql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/fleetdm/fleet/v4/server/contexts/ctxerr"
	"github.com/fleetdm/fleet/v4/server/fleet"
	"github.com/jmoiron/sqlx"
)

func (ds *Datastore) SoftwareTitleByID(ctx context.Context, id uint) (*fleet.SoftwareTitle, error) {
	const selectSoftwareTitleStmt = `
SELECT
	st.id,
	st.name,
	st.source,
	COUNT(DISTINCT hs.host_id) AS hosts_count,
	COUNT(DISTINCT s.id) AS versions_count
FROM software_titles st
JOIN software s ON s.title_id = st.id
JOIN host_software hs ON hs.software_id = s.id
WHERE st.id = ?
GROUP BY st.id
	`
	var title fleet.SoftwareTitle
	if err := sqlx.GetContext(ctx, ds.reader(ctx), &title, selectSoftwareTitleStmt, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, notFound("SoftwareTitle").WithID(id)
		}
		return nil, ctxerr.Wrap(ctx, err, "get software title")
	}

	selectSoftwareVersionsStmt, args, err := selectSoftwareVersionsSQL([]uint{id}, 0, true)
	if err != nil {
		return nil, ctxerr.Wrap(ctx, err, "building versions statement")
	}

	if err := sqlx.SelectContext(ctx, ds.reader(ctx), &title.Versions, selectSoftwareVersionsStmt, args...); err != nil {
		return nil, ctxerr.Wrap(ctx, err, "get software title version")
	}

	return &title, nil
}

func (ds *Datastore) ListSoftwareTitles(
	ctx context.Context,
	opt fleet.SoftwareTitleListOptions,
) ([]fleet.SoftwareTitle, int, *fleet.PaginationMetadata, error) {
	dbReader := ds.reader(ctx)
	getTitlesStmt, args := selectSoftwareTitlesSQL(opt)
	// build the count statement before adding the pagination constraints to `getTitlesStmt`
	getTitlesCountStmt := fmt.Sprintf(`SELECT COUNT(DISTINCT s.id) FROM (%s) AS s`, getTitlesStmt)

	// grab titles that match the list options
	var titles []fleet.SoftwareTitle
	getTitlesStmt, args = appendListOptionsWithCursorToSQL(getTitlesStmt, args, &opt.ListOptions)
	if err := sqlx.SelectContext(ctx, dbReader, &titles, getTitlesStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "select software titles")
	}

	// perform a second query to grab the counts
	var counts int
	if !opt.SkipCounts {
		if err := sqlx.GetContext(ctx, dbReader, &counts, getTitlesCountStmt, args...); err != nil {
			return nil, 0, nil, ctxerr.Wrap(ctx, err, "get software titles count")
		}
	}

	// if we don't have any matching titles, there's no point trying to
	// find matching versions. Early return
	if len(titles) == 0 {
		return titles, counts, &fleet.PaginationMetadata{}, nil
	}

	// grab all the IDs to find matching versions below
	titleIDs := make([]uint, len(titles))
	// build an index to quickly access a title by it's ID
	titleIndex := make(map[uint]int, len(titles))
	for i, title := range titles {
		titleIDs[i] = title.ID
		titleIndex[title.ID] = i
	}

	// we grab matching versions separately and build the desired object in
	// the application logic. This is because we need to support MySQL 5.7
	// and there's no good way to do an aggregation that builds a structure
	// (like a JSON) object for nested arrays.
	var teamID uint
	if opt.TeamID != nil {
		teamID = *opt.TeamID
	}
	getVersionsStmt, args, err := selectSoftwareVersionsSQL(titleIDs, teamID, false)
	if err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "build get versions stmt")
	}
	var versions []fleet.SoftwareVersion
	if err := sqlx.SelectContext(ctx, dbReader, &versions, getVersionsStmt, args...); err != nil {
		return nil, 0, nil, ctxerr.Wrap(ctx, err, "get software versions")
	}

	// append matching versions to titles
	for _, version := range versions {
		if i, ok := titleIndex[version.TitleID]; ok {
			titles[i].Versions = append(titles[i].Versions, version)
		}
	}

	var metaData *fleet.PaginationMetadata
	if opt.ListOptions.IncludeMetadata {
		metaData = &fleet.PaginationMetadata{HasPreviousResults: opt.ListOptions.Page > 0}
		if len(titles) > int(opt.ListOptions.PerPage) {
			metaData.HasNextResults = true
			titles = titles[:len(titles)-1]
		}
	}

	return titles, counts, metaData, nil
}

func selectSoftwareTitlesSQL(opt fleet.SoftwareTitleListOptions) (string, []any) {
	stmt := `
SELECT
	st.id,
	st.name,
	st.source,
	COUNT(DISTINCT hs.host_id) AS hosts_count,
	COUNT(DISTINCT s.id) AS versions_count
FROM software_titles st
JOIN software s ON s.title_id = st.id
JOIN host_software hs ON hs.software_id = s.id
-- placeholder for changing the JOIN type to filter vulnerable software
%s JOIN software_cve scve ON s.id = scve.software_id
-- placeholder for potential JOIN on hosts
%s
-- placeholder for WHERE clause
WHERE %s
GROUP BY st.id`

	cveJoinType := "LEFT"
	if opt.VulnerableOnly {
		cveJoinType = "INNER"
	}

	var args []any
	hostsJoin := ""
	whereClause := "TRUE"
	if opt.TeamID != nil {
		hostsJoin = "JOIN hosts h ON h.id = hs.host_id"
		whereClause = "h.team_id = ?"
		args = append(args, opt.TeamID)
	}

	if match := opt.ListOptions.MatchQuery; match != "" {
		whereClause += " AND (st.name LIKE ? OR scve.cve LIKE ?)"
		match = likePattern(match)
		args = append(args, match, match)
	}

	stmt = fmt.Sprintf(stmt, cveJoinType, hostsJoin, whereClause)
	return stmt, args
}

func selectSoftwareVersionsSQL(titleIDs []uint, teamID uint, withCounts bool) (string, []any, error) {
	selectVersionsStmt := `
SELECT
	st.id as title_id,
	s.id, s.version,
	%s -- placeholder for optional host_counts
	CONCAT('[', GROUP_CONCAT(JSON_QUOTE(scve.cve) SEPARATOR ','), ']') as vulnerabilities
FROM software_titles st
JOIN software s ON s.title_id = st.id
LEFT JOIN host_software hs ON hs.software_id = s.id
LEFT JOIN software_cve scve ON s.id = scve.software_id
%s -- placeholder for optional JOIN ON host_counts
WHERE st.id IN (?)
GROUP BY s.id`

	var args []any
	extraSelect := ""
	extraJoin := ""
	if withCounts {
		args = append(args, teamID)
		extraSelect = "MAX(shc.hosts_count) AS hosts_count,"
		extraJoin = `
			JOIN software_host_counts shc
			ON shc.software_id = s.id
				AND shc.hosts_count > 0
				AND shc.team_id = ?
		`
	}

	args = append(args, titleIDs)
	selectVersionsStmt = fmt.Sprintf(selectVersionsStmt, extraSelect, extraJoin)
	selectVersionsStmt, args, err := sqlx.In(selectVersionsStmt, args...)
	if err != nil {
		return "", nil, fmt.Errorf("bulding sqlx.In query: %w", err)
	}
	return selectVersionsStmt, args, nil
}
